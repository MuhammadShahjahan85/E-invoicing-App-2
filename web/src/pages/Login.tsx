// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useEffect, useState, type FormEvent } from 'react'
import { api, errorMessage } from '../api'
import type { User } from '../types'
import BrandFooter from '../components/BrandFooter'
import { Lockup, VWatermark } from '../components/Brand'
import { ArrowRight, BadgeCheck, CircleAlert, Library, LockKeyhole, ShieldCheck, TriangleAlert, UserRound, WifiOff } from 'lucide-react'

interface Me {
  user: User
  csrf: string
  permissions: string[]
}

export default function Login({ onLogin }: { onLogin: (me: Me) => void }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [product, setProduct] = useState('Veridian E-invoicing Pakistan')
  const [license, setLicense] = useState('')

  useEffect(() => {
    api
      .get<{ product: string; license: { message: string; mode: string } }>('/system/status')
      .then((s) => {
        setProduct(s.product)
        if (s.license.mode !== 'licensed' && s.license.mode !== 'developer') setLicense(s.license.message)
      })
      .catch(() => {})
  }, [])

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setBusy(true)
    setError('')
    try {
      const me = await api.post<Me>('/auth/login', { username, password })
      onLogin(me)
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="auth-split">
      <AuthBrandPanel />
      <div className="auth-panel">
        <form className="auth-form" onSubmit={submit}>
          <Lockup light className="mobile-only" size={44} />
          <h1>Sign in</h1>
          <p className="muted" style={{ marginBottom: 22 }}>
            Welcome to {product}. Use the account your administrator created for you.
          </p>
          <div className="auth-form-card">
            {error && (
              <div className="alert alert-error">
                <CircleAlert size={18} />
                <div className="alert-body">{error}</div>
              </div>
            )}
            {license && (
              <div className="alert alert-warn">
                <TriangleAlert size={18} />
                <div className="alert-body">{license}</div>
              </div>
            )}
            <div className="stack" style={{ gap: 14 }}>
              <div className="field">
                <label htmlFor="u">Username</label>
                <div className="input-icon">
                  <UserRound size={17} />
                  <input id="u" autoFocus autoComplete="username" value={username} onChange={(e) => setUsername(e.target.value)} />
                </div>
              </div>
              <div className="field">
                <label htmlFor="p">Password</label>
                <div className="input-icon">
                  <LockKeyhole size={17} />
                  <input id="p" type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} />
                </div>
              </div>
              <button className="btn btn-primary btn-lg btn-block" disabled={busy || !username || !password}>
                {busy ? 'Signing in…' : 'Sign in'} {!busy && <ArrowRight size={17} />}
              </button>
            </div>
          </div>
          <p className="small faint" style={{ marginTop: 16, display: 'flex', gap: 8, alignItems: 'center' }}>
            <ShieldCheck size={15} /> Your data stays on your own server. Sessions sign out after a period of inactivity.
          </p>
          <BrandFooter />
        </form>
      </div>
    </div>
  )
}

/** AuthBrandPanel is the branded half of the sign-in and setup screens. */
export function AuthBrandPanel() {
  return (
    <aside className="auth-brand" aria-label="About Veridian E-invoicing Pakistan">
      <VWatermark className="auth-watermark" />
      <Lockup size={46} />
      <div>
        <h2 className="auth-headline">
          FBR Digital Invoicing, <em>done right</em> — for every business in Pakistan.
        </h2>
        <p className="auth-lede">Issue sales tax invoices that are reported to FBR in real time, with the FBR invoice number and QR code printed on every copy.</p>
        <ul className="auth-points">
          <li>
            <BadgeCheck size={19} />
            <span>
              <b>Integrated with FBR (PRAL)</b> — sandbox scenarios, production reporting, cancellations and debit notes.
            </span>
          </li>
          <li>
            <Library size={19} />
            <span>
              <b>FBR reference data built in</b> — HS codes, sale types, rates, SROs, units and buyer ATL checks.
            </span>
          </li>
          <li>
            <WifiOff size={19} />
            <span>
              <b>Keeps working during outages</b> — invoices are queued and uploaded automatically within FBR's 24-hour window.
            </span>
          </li>
          <li>
            <ShieldCheck size={19} />
            <span>
              <b>On your premises</b> — tamper-evident records, audit trail, backups, and access from desktop, web and phone.
            </span>
          </li>
        </ul>
      </div>
      <div className="auth-legal">
        <b>Veridian Partners Consultancy Private Limited</b>
        <br />© 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
      </div>
    </aside>
  )
}
