// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useEffect, useState, type FormEvent } from 'react'
import { api, errorMessage } from '../api'
import type { User } from '../types'
import BrandFooter from '../components/BrandFooter'

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
    <div className="center-page">
      <form className="card card-pad auth-card" onSubmit={submit}>
        <h1>{product}</h1>
        <p className="muted">Sales tax invoicing integrated with FBR Digital Invoicing (PRAL).</p>
        {error && <div className="alert alert-error">{error}</div>}
        {license && <div className="alert alert-warn">{license}</div>}
        <div className="stack">
          <div className="field">
            <label htmlFor="u">Username</label>
            <input id="u" autoFocus autoComplete="username" value={username} onChange={(e) => setUsername(e.target.value)} />
          </div>
          <div className="field">
            <label htmlFor="p">Password</label>
            <input id="p" type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} />
          </div>
          <button className="btn btn-primary" disabled={busy || !username || !password}>
            {busy ? 'Signing in…' : 'Sign in'}
          </button>
        </div>
        <BrandFooter />
      </form>
    </div>
  )
}
