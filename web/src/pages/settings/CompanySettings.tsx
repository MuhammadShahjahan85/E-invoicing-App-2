// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useEffect, useState } from 'react'
import { api, errorMessage } from '../../api'
import { Field, Modal } from '../../components/ui'
import { useSession, useToast } from '../../state'
import type { Company } from '../../types'

export default function CompanySettings() {
  const s = useSession()
  const toast = useToast()
  const [c, setC] = useState<Company | null>(s.company)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [adding, setAdding] = useState(false)
  const canWrite = s.can('company.write')

  useEffect(() => setC(s.company), [s.company])
  if (!c) return null
  const set = (k: keyof Company, v: unknown) => setC((x) => (x ? { ...x, [k]: v } : x))
  const toggle = (k: 'businessActivities' | 'sectors', v: string) =>
    setC((x) => (x ? { ...x, [k]: x[k].includes(v) ? x[k].filter((y) => y !== v) : [...x[k], v] } : x))

  const save = async () => {
    setSaving(true)
    setError('')
    try {
      const out = await api.put<Company>(`/companies/${c.id}`, c)
      s.updateCompany(out)
      toast('ok', 'Company saved')
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Company (seller) profile</h1>
          <p>Seller particulars printed on invoices and reported to FBR. They must match your registration on IRIS exactly.</p>
        </div>
        <div className="actions">
          {s.can('company.create') && (
            <button className="btn" onClick={() => setAdding(true)}>
              + Add another company
            </button>
          )}
        </div>
      </div>
      {error && <div className="alert alert-error">{error}</div>}
      <fieldset disabled={!canWrite} style={{ border: 0, padding: 0, margin: 0 }}>
        <div className="card card-pad">
          <h3>Registration</h3>
          <div className="form-grid">
            <Field label="Business name *" span={2}>
              <input value={c.name} onChange={(e) => set('name', e.target.value)} />
            </Field>
            <Field label="NTN / CNIC *" hint="Used as sellerNTNCNIC; FBR invoice numbers start with it">
              <input className="mono" value={c.ntnCnic} onChange={(e) => set('ntnCnic', e.target.value)} />
            </Field>
            <Field label="STRN">
              <input value={c.strn} onChange={(e) => set('strn', e.target.value)} />
            </Field>
            <Field label="Province (origination of supply) *">
              <select value={c.province} onChange={(e) => set('province', e.target.value)}>
                {s.meta.provinces.map((p) => (
                  <option key={p.code}>{p.name}</option>
                ))}
              </select>
            </Field>
            <Field label="City">
              <input value={c.city} onChange={(e) => set('city', e.target.value)} />
            </Field>
            <Field label="Business address *" span={2}>
              <input value={c.address} onChange={(e) => set('address', e.target.value)} />
            </Field>
            <Field label="Phone">
              <input value={c.phone} onChange={(e) => set('phone', e.target.value)} />
            </Field>
            <Field label="Email">
              <input type="email" value={c.email} onChange={(e) => set('email', e.target.value)} />
            </Field>
          </div>
        </div>

        <div className="card card-pad mt">
          <h3>Digital Invoicing profile</h3>
          <div className="field">
            <label>Business activity (as selected on IRIS)</label>
            <div className="pill-list">
              {s.meta.businessActivities.map((a) => (
                <span key={a} className={'pill' + (c.businessActivities.includes(a) ? ' on' : '')} onClick={() => canWrite && toggle('businessActivities', a)}>
                  {a}
                </span>
              ))}
            </div>
          </div>
          <div className="field mt">
            <label>Sector</label>
            <div className="pill-list">
              {s.meta.sectors.map((a) => (
                <span key={a} className={'pill' + (c.sectors.includes(a) ? ' on' : '')} onClick={() => canWrite && toggle('sectors', a)}>
                  {a}
                </span>
              ))}
            </div>
            <span className="hint">Determines the sandbox scenarios suggested on the Scenarios page.</span>
          </div>
        </div>

        <div className="card card-pad mt">
          <h3>Invoicing defaults</h3>
          <div className="form-grid">
            <Field label="Invoice number prefix" hint="Internal numbers look like INV-2627-000001 (fiscal year July–June)">
              <input value={c.invoicePrefix} onChange={(e) => set('invoicePrefix', e.target.value.toUpperCase())} />
            </Field>
            <Field label="Debit note prefix">
              <input value={c.debitNotePrefix} onChange={(e) => set('debitNotePrefix', e.target.value.toUpperCase())} />
            </Field>
            <Field label="Further tax rate % (unregistered buyers)" hint="Section 3(1A) — currently 4%">
              <input type="number" step="0.01" min="0" max="100" value={c.furtherTaxRate} onChange={(e) => set('furtherTaxRate', Number(e.target.value))} />
            </Field>
            <Field label="Withholding fraction" hint="Share of sales tax withheld by withholding agents (0.2 = one-fifth)">
              <input type="number" step="0.01" min="0" max="1" value={c.withholdingFraction} onChange={(e) => set('withholdingFraction', Number(e.target.value))} />
            </Field>
            <label className="check">
              <input type="checkbox" checked={c.validateBeforePost} onChange={(e) => set('validateBeforePost', e.target.checked)} /> Validate with FBR before posting
              (recommended)
            </label>
            <label className="check">
              <input type="checkbox" checked={c.sendInternalRef} onChange={(e) => set('sendInternalRef', e.target.checked)} /> Send our invoice number to FBR as
              invoiceRefNo on sale invoices
            </label>
          </div>
        </div>

        <div className="card card-pad mt">
          <h3>Sales tax return calendar</h3>
          <p className="small muted">
            Used for the due-date reminders on the dashboard, in notifications and on <b>Tax periods & returns</b>. Generally tax is paid by the 15th and the
            return filed by the 18th of the month after the tax period; some sectors have other dates, and FBR sometimes extends them.
          </p>
          <div className="form-grid">
            <Field label="Pay sales tax by (day of the following month)">
              <input type="number" min="1" max="31" value={c.returnPaymentDay || 15} onChange={(e) => set('returnPaymentDay', Number(e.target.value))} />
            </Field>
            <Field label="File the return by (day of the following month)">
              <input type="number" min="1" max="31" value={c.returnFilingDay || 18} onChange={(e) => set('returnFilingDay', Number(e.target.value))} />
            </Field>
          </div>
        </div>
        {canWrite && (
          <div className="form-actions">
            <button className="btn btn-primary" onClick={save} disabled={saving}>
              {saving ? 'Saving…' : 'Save'}
            </button>
          </div>
        )}
      </fieldset>
      {adding && <AddCompany onClose={() => setAdding(false)} />}
    </>
  )
}

function AddCompany({ onClose }: { onClose: () => void }) {
  const s = useSession()
  const toast = useToast()
  const [f, setF] = useState({ name: '', ntnCnic: '', strn: '', province: 'PUNJAB', address: '', city: '' })
  const [error, setError] = useState('')
  const set = (k: string, v: string) => setF((x) => ({ ...x, [k]: v }))
  const save = async () => {
    setError('')
    try {
      const c = await api.post<Company>('/companies', {
        ...f,
        environment: 'simulator',
        businessActivities: ['Manufacturer'],
        sectors: ['All Other Sectors'],
        furtherTaxRate: 4,
        withholdingFraction: 0.2,
        validateBeforePost: true,
      })
      await s.reloadCompanies()
      s.setCompanyId(c.id)
      toast('ok', `${c.name} added — complete its profile`)
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  return (
    <Modal
      title="Add company"
      onClose={onClose}
      footer={
        <>
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn btn-primary" onClick={save} disabled={!f.name || !f.ntnCnic || !f.address}>
            Add
          </button>
        </>
      }
    >
      <p className="muted small">Each company (NTN) has its own customers, products, invoice series, FBR tokens and scenarios. Your licence limits the number of companies.</p>
      {error && <div className="alert alert-error">{error}</div>}
      <div className="form-grid">
        <Field label="Business name *" span={2}>
          <input value={f.name} onChange={(e) => set('name', e.target.value)} />
        </Field>
        <Field label="NTN / CNIC *">
          <input value={f.ntnCnic} onChange={(e) => set('ntnCnic', e.target.value)} />
        </Field>
        <Field label="STRN">
          <input value={f.strn} onChange={(e) => set('strn', e.target.value)} />
        </Field>
        <Field label="Province *">
          <select value={f.province} onChange={(e) => set('province', e.target.value)}>
            {s.meta.provinces.map((p) => (
              <option key={p.code}>{p.name}</option>
            ))}
          </select>
        </Field>
        <Field label="City">
          <input value={f.city} onChange={(e) => set('city', e.target.value)} />
        </Field>
        <Field label="Address *" span={2}>
          <input value={f.address} onChange={(e) => set('address', e.target.value)} />
        </Field>
      </div>
    </Modal>
  )
}
