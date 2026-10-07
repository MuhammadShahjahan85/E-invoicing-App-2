// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useState, type FormEvent } from 'react'
import { api, errorMessage } from '../api'
import { Field } from '../components/ui'
import BrandFooter from '../components/BrandFooter'
import { Lockup } from '../components/Brand'
import { ArrowRight, CircleAlert } from 'lucide-react'

const provinces = ['PUNJAB', 'SINDH', 'KHYBER PAKHTUNKHWA', 'BALOCHISTAN', 'CAPITAL TERRITORY', 'GILGIT BALTISTAN', 'AZAD JAMMU AND KASHMIR']
const activities = ['Manufacturer', 'Importer', 'Distributor', 'Wholesaler', 'Exporter', 'Retailer', 'Service Provider', 'Other']
const sectors = ['All Other Sectors', 'Steel', 'FMCG', 'Textile', 'Telecom', 'Petroleum', 'Electricity Distribution', 'Gas Distribution', 'Services', 'Automobile', 'CNG Stations', 'Pharmaceuticals', 'Wholesale / Retails']

export default function Setup({ onDone }: { onDone: () => void }) {
  const [f, setF] = useState({
    adminUsername: 'admin',
    adminPassword: '',
    confirm: '',
    adminFullName: '',
    name: '',
    ntnCnic: '',
    strn: '',
    province: 'SINDH',
    address: '',
    city: '',
    phone: '',
    email: '',
    activities: ['Manufacturer'] as string[],
    sectors: ['All Other Sectors'] as string[],
  })
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const set = (k: string, v: unknown) => setF((x) => ({ ...x, [k]: v }))
  const toggle = (k: 'activities' | 'sectors', v: string) =>
    setF((x) => ({ ...x, [k]: x[k].includes(v) ? x[k].filter((y) => y !== v) : [...x[k], v] }))

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (f.adminPassword !== f.confirm) {
      setError('Passwords do not match')
      return
    }
    setBusy(true)
    setError('')
    try {
      await api.post('/system/setup', {
        adminUsername: f.adminUsername,
        adminPassword: f.adminPassword,
        adminFullName: f.adminFullName,
        company: {
          name: f.name,
          ntnCnic: f.ntnCnic,
          strn: f.strn,
          province: f.province,
          address: f.address,
          city: f.city,
          phone: f.phone,
          email: f.email,
          businessActivities: f.activities,
          sectors: f.sectors,
          environment: 'simulator',
          validateBeforePost: true,
          furtherTaxRate: 4,
          withholdingFraction: 0.2,
        },
      })
      onDone()
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="setup-wrap">
      <div className="setup-hero">
        <div className="setup-hero-inner">
          <Lockup size={44} />
          <h1>Welcome — let's set up your system</h1>
          <p>
            Create the administrator account and register your business exactly as it appears on FBR's IRIS portal. You start in the offline training
            simulator; connect to the FBR sandbox and production later under <b>FBR integration</b>.
          </p>
        </div>
      </div>
      <div className="setup-card">
      <form className="card" onSubmit={submit}>
        {error && (
          <div className="alert alert-error">
            <CircleAlert size={18} />
            <div className="alert-body">{error}</div>
          </div>
        )}
        <div className="form-section">
          <span className="num">1</span>
          <h3>Administrator account</h3>
        </div>
        <div className="form-grid">
          <Field label="Username">
            <input value={f.adminUsername} onChange={(e) => set('adminUsername', e.target.value)} />
          </Field>
          <Field label="Full name">
            <input value={f.adminFullName} onChange={(e) => set('adminFullName', e.target.value)} />
          </Field>
          <Field label="Password" hint="8+ characters, letters and digits">
            <input type="password" value={f.adminPassword} onChange={(e) => set('adminPassword', e.target.value)} />
          </Field>
          <Field label="Confirm password">
            <input type="password" value={f.confirm} onChange={(e) => set('confirm', e.target.value)} />
          </Field>
        </div>
        <div className="form-section">
          <span className="num">2</span>
          <h3>Your business (the seller on every invoice)</h3>
        </div>
        <div className="form-grid">
          <Field label="Business name (as on IRIS)" span={2}>
            <input value={f.name} onChange={(e) => set('name', e.target.value)} required />
          </Field>
          <Field label="NTN / CNIC" hint="7-digit NTN (no check digit) or 13-digit CNIC">
            <input value={f.ntnCnic} onChange={(e) => set('ntnCnic', e.target.value)} required />
          </Field>
          <Field label="STRN (optional)">
            <input value={f.strn} onChange={(e) => set('strn', e.target.value)} />
          </Field>
          <Field label="Province (origination of supply)">
            <select value={f.province} onChange={(e) => set('province', e.target.value)}>
              {provinces.map((p) => (
                <option key={p}>{p}</option>
              ))}
            </select>
          </Field>
          <Field label="City">
            <input value={f.city} onChange={(e) => set('city', e.target.value)} />
          </Field>
          <Field label="Business address" span={2}>
            <input value={f.address} onChange={(e) => set('address', e.target.value)} required />
          </Field>
          <Field label="Phone">
            <input value={f.phone} onChange={(e) => set('phone', e.target.value)} />
          </Field>
          <Field label="Email">
            <input type="email" value={f.email} onChange={(e) => set('email', e.target.value)} />
          </Field>
        </div>
        <div className="form-section">
          <span className="num">3</span>
          <h3>Digital Invoicing profile</h3>
        </div>
        <div className="field">
          <label>Business activity (as selected on IRIS Digital Invoicing)</label>
          <div className="pill-list">
            {activities.map((a) => (
              <span key={a} className={'pill' + (f.activities.includes(a) ? ' on' : '')} onClick={() => toggle('activities', a)}>
                {a}
              </span>
            ))}
          </div>
        </div>
        <div className="field mt">
          <label>Sector</label>
          <div className="pill-list">
            {sectors.map((a) => (
              <span key={a} className={'pill' + (f.sectors.includes(a) ? ' on' : '')} onClick={() => toggle('sectors', a)}>
                {a}
              </span>
            ))}
          </div>
          <span className="hint">These determine which FBR sandbox scenarios you must pass before production.</span>
        </div>
        <div className="form-actions">
          <button className="btn btn-primary btn-lg" disabled={busy}>
            {busy ? 'Saving…' : 'Complete setup'} {!busy && <ArrowRight size={17} />}
          </button>
        </div>
      </form>
      <BrandFooter />
      </div>
    </div>
  )
}
