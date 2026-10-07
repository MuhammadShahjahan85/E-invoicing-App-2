// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

import { useState, type FormEvent } from 'react'
import { api, errorMessage } from '../api'
import { Field } from '../components/ui'

export default function ChangePassword({ forced, onDone }: { forced?: boolean; onDone: () => void }) {
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [ok, setOk] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')
    if (next !== confirm) {
      setError('New passwords do not match')
      return
    }
    try {
      await api.post('/auth/password', { current, new: next })
      setOk(true)
      setTimeout(onDone, 800)
    } catch (err) {
      setError(errorMessage(err))
    }
  }

  const form = (
    <form className="card card-pad auth-card" onSubmit={submit}>
      <h2>{forced ? 'Set a new password' : 'Change password'}</h2>
      {forced && <p className="muted">Your password was set by an administrator. Choose your own password to continue.</p>}
      {error && <div className="alert alert-error">{error}</div>}
      {ok && <div className="alert alert-ok">Password changed.</div>}
      <div className="stack">
        <Field label="Current password">
          <input type="password" value={current} onChange={(e) => setCurrent(e.target.value)} autoComplete="current-password" />
        </Field>
        <Field label="New password" hint="8+ characters with letters and digits">
          <input type="password" value={next} onChange={(e) => setNext(e.target.value)} autoComplete="new-password" />
        </Field>
        <Field label="Confirm new password">
          <input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" />
        </Field>
        <button className="btn btn-primary" disabled={!current || !next}>
          Save
        </button>
      </div>
    </form>
  )
  return forced ? <div className="center-page">{form}</div> : form
}
