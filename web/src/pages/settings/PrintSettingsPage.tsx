// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useEffect, useState } from 'react'
import { api, errorMessage } from '../../api'
import { Field } from '../../components/ui'
import { useSession, useToast } from '../../state'
import type { Company, PrintSettings } from '../../types'

const defaults: PrintSettings = {
  paperSize: 'A4',
  wordsStyle: 'south_asian',
  showAmountWords: true,
  showSignature: true,
  footerText: '',
  termsText: '',
  showUnitPrice: true,
  showHsCode: true,
  showDiscount: true,
  copies: 1,
}

export default function PrintSettingsPage() {
  const s = useSession()
  const toast = useToast()
  const c = s.company!
  const canWrite = s.can('company.write')
  const [p, setP] = useState<PrintSettings>({ ...defaults, ...(c.printSettings ?? {}) })
  const [error, setError] = useState('')
  const [logoTick, setLogoTick] = useState(0)
  useEffect(() => setP({ ...defaults, ...(c.printSettings ?? {}) }), [c])
  const set = (k: keyof PrintSettings, v: unknown) => setP((x) => ({ ...x, [k]: v }))

  const save = async () => {
    setError('')
    try {
      const out = await api.put<Company>(`/companies/${c.id}`, { ...c, printSettings: p })
      s.updateCompany(out)
      toast('ok', 'Print settings saved')
    } catch (e) {
      setError(errorMessage(e))
    }
  }

  const uploadLogo = async (f: File | undefined) => {
    if (!f) return
    try {
      await api.upload(`/companies/${c.id}/logo`, 'PUT', f, f.type)
      s.updateCompany({ ...c, hasLogo: true })
      setLogoTick((t) => t + 1)
      toast('ok', 'Logo uploaded')
    } catch (e) {
      toast('err', errorMessage(e))
    }
  }

  const removeLogo = async () => {
    try {
      await api.del(`/companies/${c.id}/logo`)
      s.updateCompany({ ...c, hasLogo: false })
      toast('ok', 'Logo removed')
    } catch (e) {
      toast('err', errorMessage(e))
    }
  }

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Invoice printing</h1>
          <p>
            Every printed invoice carries the FBR invoice number, the FBR Digital Invoicing logo and a QR code (version 2.0, 25×25 modules, printed at 1 × 1 inch) as
            required by FBR. These elements cannot be switched off.
          </p>
        </div>
      </div>
      {error && <div className="alert alert-error">{error}</div>}
      <div className="grid g2">
        <fieldset disabled={!canWrite} className="card card-pad" style={{ border: 0, margin: 0 }}>
          <h3>Layout</h3>
          <div className="form-grid">
            <Field label="Default format">
              <select value={p.paperSize} onChange={(e) => set('paperSize', e.target.value)}>
                <option value="A4">A4 tax invoice</option>
                <option value="thermal80">80 mm thermal receipt (POS)</option>
              </select>
            </Field>
            <Field label="Amount in words">
              <select value={p.wordsStyle} onChange={(e) => set('wordsStyle', e.target.value)}>
                <option value="south_asian">Lakh / crore</option>
                <option value="international">Million / billion</option>
              </select>
            </Field>
            <label className="check">
              <input type="checkbox" checked={p.showAmountWords} onChange={(e) => set('showAmountWords', e.target.checked)} /> Print amount in words
            </label>
            <label className="check">
              <input type="checkbox" checked={p.showSignature} onChange={(e) => set('showSignature', e.target.checked)} /> Signature / stamp boxes
            </label>
            <label className="check">
              <input type="checkbox" checked={p.showHsCode} onChange={(e) => set('showHsCode', e.target.checked)} /> HS code column
            </label>
            <label className="check">
              <input type="checkbox" checked={p.showUnitPrice} onChange={(e) => set('showUnitPrice', e.target.checked)} /> Unit price column
            </label>
            <label className="check">
              <input type="checkbox" checked={p.showDiscount} onChange={(e) => set('showDiscount', e.target.checked)} /> Discount column
            </label>
            <Field label="Copies per print">
              <input type="number" min={1} max={5} value={p.copies} onChange={(e) => set('copies', Number(e.target.value))} />
            </Field>
            <Field label="Terms & conditions" span={2}>
              <textarea rows={3} value={p.termsText} onChange={(e) => set('termsText', e.target.value)} />
            </Field>
            <Field label="Footer text" span={2}>
              <input value={p.footerText} onChange={(e) => set('footerText', e.target.value)} placeholder="e.g. Thank you for your business" />
            </Field>
          </div>
          {canWrite && (
            <div className="form-actions">
              <button type="button" className="btn btn-primary" onClick={save}>
                Save
              </button>
            </div>
          )}
        </fieldset>
        <div className="card card-pad">
          <h3>Company logo</h3>
          {c.hasLogo ? (
            <div className="qr-box" style={{ width: 'auto', height: 'auto', padding: 8 }}>
              <img src={`/api/v1/companies/${c.id}/logo?t=${logoTick}`} alt="Company logo" style={{ maxHeight: 90, maxWidth: 260 }} />
            </div>
          ) : (
            <p className="muted small">No logo uploaded.</p>
          )}
          {canWrite && (
            <div className="row mt">
              <input type="file" accept="image/png,image/jpeg,image/gif,image/webp,image/svg+xml" onChange={(e) => uploadLogo(e.target.files?.[0])} />
              {c.hasLogo && (
                <button className="btn btn-sm btn-danger" onClick={removeLogo}>
                  Remove
                </button>
              )}
            </div>
          )}
          <p className="small muted mt">PNG, JPEG, GIF, WebP or SVG up to 2 MB. Printed at the top left of the invoice.</p>
          <hr />
          <h3>FBR Digital Invoicing logo</h3>
          <p className="small muted">
            FBR requires its Digital Invoicing logo next to the QR code. Download the official logo from FBR / PRAL and upload it once under <b>System</b>; it is used for
            every company. Until then a text mark is printed.
          </p>
          <div className="qr-box" style={{ width: 'auto', height: 'auto', padding: 8 }}>
            <img
              src="/api/v1/system/fbr-logo"
              alt="FBR DI logo not uploaded"
              style={{ maxHeight: 70 }}
              onError={(e) => ((e.target as HTMLImageElement).style.display = 'none')}
            />
          </div>
        </div>
      </div>
    </>
  )
}
