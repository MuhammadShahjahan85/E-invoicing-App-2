// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { Fragment, useState } from 'react'
import { Link } from 'react-router-dom'
import { FileDown, FileSpreadsheet, FileText } from 'lucide-react'
import { api, qs } from '../api'
import { Empty, ErrorBox, Pager, useLoad, PageSpinner } from '../components/ui'
import { dateFmt, dateTimeFmt, envLabels, money, monthStartPK, qty, todayPK } from '../format'
import { useCompanyPath, useSession } from '../state'
import type { FBRCall, Paged } from '../types'

type Kind = 'register' | 'tax-summary' | 'monthly' | 'customers' | 'annex-c' | 'calls'

const titles: Record<Kind, string> = {
  register: 'Sales register (line level)',
  'tax-summary': 'Tax summary by sale type & rate',
  monthly: 'Monthly summary (tax periods)',
  customers: 'Buyer-wise summary',
  'annex-c': 'Annexure-C reconciliation',
  calls: 'FBR API log',
}

/* eslint-disable @typescript-eslint/no-explicit-any */
type Row = Record<string, any>

export default function Reports() {
  const s = useSession()
  const cp = useCompanyPath()
  const [kind, setKind] = useState<Kind>('register')
  const [from, setFrom] = useState(monthStartPK())
  const [to, setTo] = useState(todayPK())
  const [env, setEnv] = useState(s.company?.environment ?? 'production')
  const params = qs({ from, to, env })

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Reports</h1>
          <p>Only documents accepted by FBR are included. Use the sales register to reconcile with Annexure-C of your sales tax return.</p>
        </div>
      </div>
      <div className="tabs">
        {(Object.keys(titles) as Kind[]).map((k) => (
          <button key={k} className={'tab' + (kind === k ? ' active' : '')} onClick={() => setKind(k)}>
            {titles[k]}
          </button>
        ))}
      </div>
      {kind !== 'calls' && (
        <div className="card mb">
          <div className="card-body row">
            <label className="small muted">From</label>
            <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} style={{ width: 'auto' }} />
            <label className="small muted">To</label>
            <input type="date" value={to} onChange={(e) => setTo(e.target.value)} style={{ width: 'auto' }} />
            <select value={env} onChange={(e) => setEnv(e.target.value as typeof env)} style={{ width: 'auto' }}>
              {s.meta.environments.map((e) => (
                <option key={e.value} value={e.value}>
                  {envLabels[e.value]}
                </option>
              ))}
            </select>
            <div className="spacer" />
            <a className="btn" href={`/api/v1${cp}/reports/${kind}${params}&format=pdf`} title="A formatted PDF document, ready to print or send">
              <FileText size={16} /> PDF
            </a>
            <a className="btn" href={`/api/v1${cp}/reports/${kind}${params}&format=xlsx`} title="Excel workbook with every column">
              <FileSpreadsheet size={16} /> Excel
            </a>
            <a className="btn" href={`/api/v1${cp}/reports/${kind}${params}&format=csv`} title="Comma-separated values for other software">
              <FileDown size={16} /> CSV
            </a>
          </div>
        </div>
      )}
      {kind === 'calls' ? <Calls /> : <ReportTable kind={kind} params={params} />}
    </>
  )
}

function ReportTable({ kind, params }: { kind: Exclude<Kind, 'calls'>; params: string }) {
  const cp = useCompanyPath()
  const { data, error, loading } = useLoad(() => api.get<Row[]>(`${cp}/reports/${kind}${params}`), [cp, kind, params])
  if (loading && !data) return <PageSpinner />
  if (error) return <ErrorBox error={error} />
  if (!data || data.length === 0) return <Empty>No accepted documents in this period.</Empty>
  const sum = (k: string) => data.reduce((a, r) => a + Number(r[k] ?? 0), 0)

  if (kind === 'register') {
    return (
      <div className="card">
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Document</th>
                <th>FBR invoice no.</th>
                <th>Buyer</th>
                <th>HS code</th>
                <th>Description</th>
                <th>Sale type / rate</th>
                <th className="num">Qty</th>
                <th className="num">Value excl. ST</th>
                <th className="num">Sales tax</th>
                <th className="num">Further</th>
                <th className="num">Extra</th>
                <th className="num">FED</th>
                <th className="num">Withheld</th>
                <th className="num">Total</th>
              </tr>
            </thead>
            <tbody>
              {data.map((r, i) => (
                <tr key={i}>
                  <td className="nowrap">{dateFmt(r.invoiceDate)}</td>
                  <td className="nowrap">
                    <Link to={`/invoices/${r.invoiceId}`}>{r.internalNo}</Link>
                    <div className="small faint">
                      {r.docType}
                      {r.status === 'CANCELLED' && ' · cancelled'}
                    </div>
                  </td>
                  <td className="mono small">{r.fbrInvoiceNumber}</td>
                  <td>
                    {r.buyerName}
                    <div className="small faint">
                      {r.buyerNtnCnic || '—'} · {r.buyerRegistrationType} · {r.buyerProvince}
                    </div>
                  </td>
                  <td className="mono">{r.hsCode}</td>
                  <td>{r.description}</td>
                  <td className="small">
                    {r.saleType}
                    <br />
                    <b>{r.rate}</b>
                  </td>
                  <td className="num">
                    {qty(r.quantity)} <span className="small faint">{r.uom}</span>
                  </td>
                  <td className="num">{money(r.valueExclST)}</td>
                  <td className="num">{money(r.salesTax)}</td>
                  <td className="num">{money(r.furtherTax)}</td>
                  <td className="num">{money(r.extraTax)}</td>
                  <td className="num">{money(r.fed)}</td>
                  <td className="num">{money(r.stWithheld)}</td>
                  <td className="num">{money(r.totalValue)}</td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr>
                <td colSpan={8}>Totals ({data.length} lines)</td>
                <td className="num">{money(sum('valueExclST'))}</td>
                <td className="num">{money(sum('salesTax'))}</td>
                <td className="num">{money(sum('furtherTax'))}</td>
                <td className="num">{money(sum('extraTax'))}</td>
                <td className="num">{money(sum('fed'))}</td>
                <td className="num">{money(sum('stWithheld'))}</td>
                <td className="num">{money(sum('totalValue'))}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>
    )
  }

  if (kind === 'tax-summary') {
    return (
      <div className="card">
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>Document</th>
                <th>Sale type</th>
                <th>Rate</th>
                <th className="num">Documents</th>
                <th className="num">Lines</th>
                <th className="num">Value excl. ST</th>
                <th className="num">Retail value</th>
                <th className="num">Sales tax</th>
                <th className="num">Further</th>
                <th className="num">Extra</th>
                <th className="num">FED</th>
                <th className="num">Withheld</th>
              </tr>
            </thead>
            <tbody>
              {data.map((r, i) => (
                <tr key={i}>
                  <td>{r.docType}</td>
                  <td>{r.saleType}</td>
                  <td>{r.rate}</td>
                  <td className="num">{r.invoices}</td>
                  <td className="num">{r.lines}</td>
                  <td className="num">{money(r.valueExclST)}</td>
                  <td className="num">{money(r.retailValue)}</td>
                  <td className="num">{money(r.salesTax)}</td>
                  <td className="num">{money(r.furtherTax)}</td>
                  <td className="num">{money(r.extraTax)}</td>
                  <td className="num">{money(r.fed)}</td>
                  <td className="num">{money(r.stWithheld)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="small muted card-body">Debit notes are listed separately; they adjust (reduce) the output tax of the referenced sale invoices.</p>
      </div>
    )
  }

  if (kind === 'annex-c') {
    const active = data.filter((r) => r.status === 'ACCEPTED')
    const total = (k: string) => active.reduce((a, r) => a + Number(r[k] ?? 0), 0)
    return (
      <div className="card">
        <p className="small muted card-body">
          Every document that received an FBR invoice number in the period, including those cancelled later. Rule 150XD(2) (as amended by SRO 1666(I)/2026)
          lets FBR recover tax on any invoice transmitted with an FBR number but not accounted for in Annexure-C or the return, unless it was cancelled through
          the approved mechanism. Match each line with Annexure-C on IRIS before filing.
        </p>
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>FBR invoice no.</th>
                <th>Date</th>
                <th>Document</th>
                <th>Buyer</th>
                <th className="num">Value excl. ST</th>
                <th className="num">Sales tax</th>
                <th className="num">Further</th>
                <th className="num">Extra</th>
                <th className="num">FED</th>
                <th className="num">Withheld</th>
                <th className="num">Total</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {data.map((r, i) => (
                <tr key={i} className={r.status === 'CANCELLED' ? 'faint' : undefined}>
                  <td className="mono small">{r.fbrInvoiceNumber}</td>
                  <td className="nowrap">{dateFmt(r.invoiceDate)}</td>
                  <td className="nowrap small">
                    {r.internalNo}
                    <div className="faint">
                      {r.docType}
                      {r.invoiceRefNo && <> · against {r.invoiceRefNo}</>}
                    </div>
                  </td>
                  <td>
                    {r.buyerName}
                    <div className="small faint">
                      {r.buyerNtnCnic || '—'} · {r.buyerRegistrationType}
                    </div>
                  </td>
                  <td className="num">{money(r.valueExclST)}</td>
                  <td className="num">{money(r.salesTax)}</td>
                  <td className="num">{money(r.furtherTax)}</td>
                  <td className="num">{money(r.extraTax)}</td>
                  <td className="num">{money(r.fed)}</td>
                  <td className="num">{money(r.stWithheld)}</td>
                  <td className="num">{money(r.totalValue)}</td>
                  <td className="small">
                    {r.status === 'CANCELLED' ? <span className="badge b-gray">Cancelled {r.cancelReference}</span> : <span className="badge b-green">Reported</span>}
                    {r.offlineMode && <div className="faint">offline mode</div>}
                  </td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr>
                <td colSpan={4}>Reported, not cancelled ({active.length})</td>
                <td className="num">{money(total('valueExclST'))}</td>
                <td className="num">{money(total('salesTax'))}</td>
                <td className="num">{money(total('furtherTax'))}</td>
                <td className="num">{money(total('extraTax'))}</td>
                <td className="num">{money(total('fed'))}</td>
                <td className="num">{money(total('stWithheld'))}</td>
                <td className="num">{money(total('totalValue'))}</td>
                <td></td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>
    )
  }

  if (kind === 'monthly') {
    return (
      <div className="card">
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>Tax period</th>
                <th className="num">Sale invoices</th>
                <th className="num">Value excl. ST</th>
                <th className="num">Sales tax</th>
                <th className="num">Further tax</th>
                <th className="num">Debit notes</th>
                <th className="num">DN value</th>
                <th className="num">DN sales tax</th>
                <th className="num">Net sales tax</th>
                <th className="num">Withheld by buyers</th>
              </tr>
            </thead>
            <tbody>
              {data.map((r, i) => (
                <tr key={i}>
                  <td>{r.period}</td>
                  <td className="num">{r.saleInvoices}</td>
                  <td className="num">{money(r.valueExclST)}</td>
                  <td className="num">{money(r.salesTax)}</td>
                  <td className="num">{money(r.furtherTax)}</td>
                  <td className="num">{r.debitNotes}</td>
                  <td className="num">{money(r.debitNoteValue)}</td>
                  <td className="num">{money(r.debitNoteSalesTax)}</td>
                  <td className="num">
                    <b>{money(Number(r.salesTax) - Number(r.debitNoteSalesTax))}</b>
                  </td>
                  <td className="num">{money(r.stWithheld)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    )
  }

  return (
    <div className="card">
      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>Buyer</th>
              <th>NTN / CNIC</th>
              <th>Registration</th>
              <th className="num">Invoices</th>
              <th className="num">Value excl. ST</th>
              <th className="num">Sales tax</th>
              <th className="num">Further tax</th>
              <th className="num">Withheld</th>
              <th className="num">Total</th>
            </tr>
          </thead>
          <tbody>
            {data.map((r, i) => (
              <tr key={i}>
                <td>{r.buyerName}</td>
                <td className="mono">{r.buyerNtnCnic || '—'}</td>
                <td>{r.buyerRegistrationType}</td>
                <td className="num">{r.invoices}</td>
                <td className="num">{money(r.valueExclST)}</td>
                <td className="num">{money(r.salesTax)}</td>
                <td className="num">{money(r.furtherTax)}</td>
                <td className="num">{money(r.stWithheld)}</td>
                <td className="num">{money(r.totalValue)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

function Calls() {
  const cp = useCompanyPath()
  const [offset, setOffset] = useState(0)
  const [open, setOpen] = useState<number | null>(null)
  const limit = 100
  const { data, error, loading } = useLoad(() => api.get<Paged<FBRCall>>(`${cp}/calls${qs({ limit, offset })}`), [cp, offset])
  if (loading && !data) return <PageSpinner />
  if (error) return <ErrorBox error={error} />
  if (!data || data.items.length === 0) return <Empty>No calls to FBR yet.</Empty>
  return (
    <div className="card">
      <p className="small muted card-body">Every request to FBR's Digital Invoicing API, with the response, is kept for evidence and troubleshooting. Security tokens are never logged.</p>
      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>Time</th>
              <th>Environment</th>
              <th>Operation</th>
              <th>Invoice</th>
              <th>HTTP</th>
              <th className="num">ms</th>
              <th>Outcome</th>
            </tr>
          </thead>
          <tbody>
            {data.items.map((c) => (
              <Fragment key={c.id}>
                <tr className="clickable" onClick={() => setOpen(open === c.id ? null : c.id)}>
                  <td className="nowrap">{dateTimeFmt(c.createdAt)}</td>
                  <td>{envLabels[c.environment] ?? c.environment}</td>
                  <td>{c.operation}</td>
                  <td>{c.invoiceId ? <Link to={`/invoices/${c.invoiceId}`}>#{c.invoiceId}</Link> : '—'}</td>
                  <td>{c.httpStatus || '—'}</td>
                  <td className="num">{c.durationMs}</td>
                  <td>{c.errorKind ? <span className="badge b-red">{c.errorKind}</span> : <span className="badge b-green">ok</span>}</td>
                </tr>
                {open === c.id && (
                  <tr>
                    <td colSpan={7}>
                      <div className="small mono">
                        {c.method} {c.url}
                      </div>
                      {c.error && <div className="alert alert-error small">{c.error}</div>}
                      <div className="grid g2">
                        <pre className="code">{c.requestBody}</pre>
                        <pre className="code">{c.responseBody}</pre>
                      </div>
                    </td>
                  </tr>
                )}
              </Fragment>
            ))}
          </tbody>
        </table>
      </div>
      <Pager total={data.total} limit={limit} offset={offset} onChange={setOffset} />
    </div>
  )
}
