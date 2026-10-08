// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api'
import { ErrorBox, StatusBadge } from '../components/ui'
import { money } from '../format'
import { useCompanyPath, useSession, useToast } from '../state'
import type { Issue } from '../types'

interface ImportResult {
  ref: string
  rows: number[]
  invoiceId?: number
  status: string
  errors?: string[]
  issues?: Issue[]
  total: number
}

interface ImportSummary {
  invoices: number
  ok: number
  failed: number
  results: ImportResult[]
  preview: boolean
}

const columns: [string, string][] = [
  ['invoice_ref', 'Your ERP invoice number. Rows with the same value form one invoice; it also prevents importing the same invoice twice.'],
  ['invoice_date', 'YYYY-MM-DD or DD/MM/YYYY (Excel dates are accepted).'],
  ['doc_type', '"Sale Invoice" (default) or "Debit Note".'],
  ['original_fbr_invoice_no', 'Debit notes only: the FBR invoice number of the original sale invoice.'],
  ['scenario_id', 'Sandbox only: SN001–SN028.'],
  ['buyer_ntn_cnic', '7-digit NTN (without check digit), 9-digit or 13-digit CNIC. Required for registered buyers.'],
  ['buyer_name / buyer_province / buyer_address', 'Buyer particulars. Province is the destination of supply.'],
  ['buyer_registration_type', 'Registered or Unregistered. Leave blank to use the customer master (matched by NTN/CNIC); unknown buyers count as Unregistered.'],
  ['withholding_mode', 'Blank, fraction or full — when the buyer is a sales tax withholding agent.'],
  ['product_code', 'Optional: code from the Products master; fills HS code, UoM, sale type and rate.'],
  ['hs_code / description / uom', 'PCT code (NNNN.NNNN), product description and FBR unit of measure.'],
  ['quantity / unit_price / discount', 'Quantity, price excluding tax and discount amount.'],
  ['value_excl_st', 'Optional override of the value excluding sales tax.'],
  ['sale_type / rate', 'FBR sale type text and rate, e.g. "Goods at standard rate (default)" and "18%".'],
  ['retail_price', 'Third Schedule goods: retail price printed on the pack per unit, including sales tax (the tax is price × rate ÷ (100 + rate)).'],
  ['sro_schedule_no / sro_item_serial_no', 'Required for reduced-rate, exempt and other SRO-based supplies.'],
  ['sales_tax / further_tax / extra_tax / fed / st_withheld', 'Optional overrides — leave blank to let the tax engine compute them.'],
]

export default function ImportPage() {
  const s = useSession()
  const cp = useCompanyPath()
  const toast = useToast()
  const [file, setFile] = useState<File | null>(null)
  const [submit, setSubmit] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<unknown>()
  const [summary, setSummary] = useState<ImportSummary | null>(null)

  const run = async (preview: boolean) => {
    if (!file) return
    setBusy(true)
    setError(undefined)
    setSummary(null)
    try {
      const fd = new FormData()
      fd.append('file', file)
      const q = preview ? '?preview=1' : submit ? '?submit=1' : ''
      const res = await api.upload<ImportSummary>(`${cp}/import${q}`, 'POST', fd, '')
      setSummary(res)
      if (!preview) toast(res.failed ? 'err' : 'ok', `${res.ok} of ${res.invoices} invoice(s) imported`)
    } catch (e) {
      setError(e)
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Import invoices (CSV / Excel)</h1>
          <p>Bulk-load invoices from any accounting system or POS export. Each file is checked line by line before anything is saved.</p>
        </div>
        <div className="actions">
          <a className="btn" href="/api/v1/import/template.xlsx">
            Excel template
          </a>
          <a className="btn" href="/api/v1/import/template.csv">
            CSV template
          </a>
        </div>
      </div>

      <div className="grid g2">
        <div className="card card-pad">
          <h3>1. Choose file</h3>
          <input type="file" accept=".csv,.xlsx,.xls,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" onChange={(e) => setFile(e.target.files?.[0] ?? null)} />
          <label className="check mt">
            <input type="checkbox" checked={submit} onChange={(e) => setSubmit(e.target.checked)} /> Submit each imported invoice to FBR immediately (
            {s.company?.environment})
          </label>
          <div className="row mt">
            <button className="btn" disabled={!file || busy} onClick={() => run(true)}>
              {busy ? 'Checking…' : '2. Preview & check'}
            </button>
            <button className="btn btn-primary" disabled={!file || busy || !s.can('invoice.write')} onClick={() => run(false)}>
              {busy ? 'Importing…' : submit ? '3. Import & submit' : '3. Import as drafts'}
            </button>
          </div>
          <p className="small muted mt">
            Invoices whose invoice_ref was imported before are skipped, so a file can safely be uploaded again after correcting errors.
          </p>
        </div>
        <div className="card card-pad">
          <h3>Columns</h3>
          <div className="table-wrap" style={{ maxHeight: 280, overflow: 'auto' }}>
            <table className="table">
              <tbody>
                {columns.map(([c, d]) => (
                  <tr key={c}>
                    <td className="mono small nowrap">{c}</td>
                    <td className="small">{d}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div className="mt">
        <ErrorBox error={error} />
      </div>

      {summary && (
        <div className="card mt">
          <div className="card-head">
            <h3>
              {summary.preview ? 'Preview' : 'Import result'}: {summary.ok} ready / {summary.failed} with errors, {summary.invoices} invoice(s)
            </h3>
          </div>
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Invoice ref</th>
                  <th>File rows</th>
                  <th>Status</th>
                  <th className="num">Total</th>
                  <th>Messages</th>
                </tr>
              </thead>
              <tbody>
                {summary.results.map((r, i) => (
                  <tr key={i}>
                    <td className="nowrap">{r.invoiceId ? <Link to={`/invoices/${r.invoiceId}`}>{r.ref}</Link> : r.ref}</td>
                    <td className="small">{r.rows.join(', ')}</td>
                    <td>{/^[A-Z]+$/.test(r.status) ? <StatusBadge status={r.status} /> : <span className={'badge ' + (r.status === 'ok' ? 'b-green' : 'b-red')}>{r.status === 'ok' ? 'ready' : r.status}</span>}</td>
                    <td className="num">{money(r.total)}</td>
                    <td className="small">
                      {(r.errors ?? []).map((e, k) => (
                        <div key={k} style={{ color: 'var(--danger)' }}>
                          {e}
                        </div>
                      ))}
                      {(r.issues ?? []).map((e, k) => (
                        <div key={'i' + k} style={{ color: e.severity === 'error' ? 'var(--danger)' : 'var(--warning)' }}>
                          {e.line > 0 && `Line ${e.line}: `}
                          {e.code && `[${e.code}] `}
                          {e.message}
                        </div>
                      ))}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </>
  )
}
