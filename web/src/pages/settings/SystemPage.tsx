// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useEffect, useState } from 'react'
import { Mail, Send } from 'lucide-react'
import { api, errorMessage } from '../../api'
import { ErrorBox, Field, Spinner, useLoad } from '../../components/ui'
import { dateFmt, dateTimeFmt } from '../../format'
import { useToast } from '../../state'
import type { LicenseStatus } from '../../types'

interface SystemInfo {
  product: string
  version: string
  buildDate: string
  go: string
  os: string
  dataDir: string
  backupDir: string
  fbrBaseUrl: string
  uptimeSeconds: number
}

interface Backup {
  name: string
  size: number
  created: string
}

function size(n: number) {
  if (n > 1 << 20) return (n / (1 << 20)).toFixed(1) + ' MB'
  return Math.ceil(n / 1024) + ' KB'
}

function uptime(s: number) {
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  return (d ? `${d}d ` : '') + `${h}h ${m}m`
}

export default function SystemPage() {
  const toast = useToast()
  const info = useLoad(() => api.get<SystemInfo>('/system/info'), [])
  const lic = useLoad(() => api.get<LicenseStatus>('/system/license'), [])
  const backups = useLoad(() => api.get<{ items: Backup[] | null; dir: string }>('/system/backups'), [])
  const [backingUp, setBackingUp] = useState(false)
  const [licError, setLicError] = useState('')
  const [logoTick, setLogoTick] = useState(0)

  const installLicense = async (f: File | undefined) => {
    if (!f) return
    setLicError('')
    try {
      const st = await api.upload<LicenseStatus>('/system/license', 'POST', await f.text(), 'text/plain')
      lic.setData(st)
      toast('ok', `Licence installed for ${st.licensee}`)
    } catch (e) {
      setLicError(errorMessage(e))
    }
  }

  const backupNow = async () => {
    setBackingUp(true)
    try {
      const b = await api.post<Backup>('/system/backups')
      toast('ok', `Backup ${b.name} created`)
      backups.reload()
    } catch (e) {
      toast('err', errorMessage(e))
    } finally {
      setBackingUp(false)
    }
  }

  const uploadFBRLogo = async (f: File | undefined) => {
    if (!f) return
    try {
      await api.upload('/system/fbr-logo', 'PUT', f, f.type)
      setLogoTick((t) => t + 1)
      toast('ok', 'FBR Digital Invoicing logo saved')
    } catch (e) {
      toast('err', errorMessage(e))
    }
  }

  const l = lic.data
  return (
    <>
      <div className="page-head">
        <div>
          <h1>System</h1>
          <p>Licence, backups and server information.</p>
        </div>
      </div>
      <div className="grid g2">
        <div className="card card-pad">
          <h3>Licence</h3>
          <ErrorBox error={lic.error} />
          {l ? (
            <>
              <div className={'alert ' + (l.mode === 'licensed' || l.mode === 'developer' ? 'alert-ok' : 'alert-warn')}>{l.message}</div>
              <dl className="kv">
                <dt>Status</dt>
                <dd>{l.mode}</dd>
                {l.licensee && (
                  <>
                    <dt>Licensee</dt>
                    <dd>{l.licensee}</dd>
                    <dt>Licence ID</dt>
                    <dd className="mono">{l.licenseId}</dd>
                    <dt>Edition</dt>
                    <dd>{l.edition}</dd>
                    <dt>Licensed NTNs</dt>
                    <dd className="mono">{(l.sellerNtns ?? []).join(', ') || '—'}</dd>
                    <dt>Companies / users</dt>
                    <dd>
                      {l.maxCompanies || 'unlimited'} / {l.maxUsers || 'unlimited'}
                    </dd>
                  </>
                )}
                {l.expiresAt && (
                  <>
                    <dt>Valid until</dt>
                    <dd>
                      {dateFmt(l.expiresAt)} ({l.daysLeft} days)
                    </dd>
                  </>
                )}
                {l.supportUntil && (
                  <>
                    <dt>Support until</dt>
                    <dd>{dateFmt(l.supportUntil)}</dd>
                  </>
                )}
                <dt>Production use</dt>
                <dd>{l.production ? 'Allowed' : 'Not allowed'}</dd>
              </dl>
            </>
          ) : (
            <Spinner />
          )}
          {licError && <div className="alert alert-error">{licError}</div>}
          <div className="field mt">
            <label>Install licence file (license.lic)</label>
            <input type="file" accept=".lic,.txt,text/plain" onChange={(e) => installLicense(e.target.files?.[0])} />
          </div>
        </div>

        <div className="card card-pad">
          <h3>Server</h3>
          <ErrorBox error={info.error} />
          {info.data ? (
            <dl className="kv">
              <dt>Product</dt>
              <dd>
                {info.data.product} {info.data.version}
              </dd>
              <dt>Build</dt>
              <dd>
                {info.data.buildDate} · {info.data.go} · {info.data.os}
              </dd>
              <dt>Uptime</dt>
              <dd>{uptime(info.data.uptimeSeconds)}</dd>
              <dt>Data folder</dt>
              <dd className="mono small">{info.data.dataDir}</dd>
              <dt>FBR gateway</dt>
              <dd className="mono small">{info.data.fbrBaseUrl}</dd>
            </dl>
          ) : (
            <Spinner />
          )}
          <hr />
          <h3>FBR Digital Invoicing logo</h3>
          <p className="small muted">
            Printed next to the QR code on every invoice, as section 6 of the DI technical specification requires. The official logo from the specification is
            built in; upload a file only if FBR or PRAL gives you a newer version.
          </p>
          <div className="row">
            <div className="qr-box" style={{ width: 'auto', height: 'auto', padding: 6 }}>
              <img
                key={logoTick}
                src={`/api/v1/system/fbr-logo?t=${logoTick}`}
                alt="FBR Digital Invoicing System logo"
                style={{ maxHeight: 60 }}
                onError={(e) => ((e.target as HTMLImageElement).style.visibility = 'hidden')}
              />
            </div>
            <input type="file" accept="image/png,image/jpeg,image/svg+xml,image/webp" onChange={(e) => uploadFBRLogo(e.target.files?.[0])} />
          </div>
        </div>
      </div>

      <div className="card mt">
        <div className="card-head">
          <h3>Backups</h3>
          <button className="btn btn-primary btn-sm" onClick={backupNow} disabled={backingUp}>
            {backingUp ? 'Backing up…' : 'Back up now'}
          </button>
        </div>
        <div className="card-body small muted">
          A consistent copy of the database is written automatically every night to <span className="mono">{backups.data?.dir}</span>. Copy backups to another disk or
          cloud storage regularly: records must be kept for six years (rule 150S). Also keep a safe copy of <span className="mono">master.key</span> from the data folder —
          without it the saved FBR tokens cannot be decrypted (invoices remain readable).
        </div>
        <ErrorBox error={backups.error} />
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>File</th>
                <th>Created</th>
                <th className="num">Size</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {(backups.data?.items ?? []).map((b) => (
                <tr key={b.name}>
                  <td className="mono small">{b.name}</td>
                  <td>{dateTimeFmt(b.created)}</td>
                  <td className="num">{size(b.size)}</td>
                  <td>
                    <a className="btn btn-sm" href={`/api/v1/system/backups/${encodeURIComponent(b.name)}`}>
                      Download
                    </a>
                  </td>
                </tr>
              ))}
              {backups.data && (backups.data.items ?? []).length === 0 && (
                <tr>
                  <td colSpan={4} className="muted">
                    No backups yet.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      <EmailNotifications />
    </>
  )
}

interface NotifySettings {
  enabled: boolean
  smtpHost: string
  smtpPort: number
  security: 'starttls' | 'tls' | 'none'
  username: string
  hasPassword: boolean
  from: string
  to: string[]
  immediate: boolean
  digestHour: number
  appUrl: string
  lastSent: string
  lastError: string
}

const presets: { label: string; host: string; port: number; security: NotifySettings['security'] }[] = [
  { label: 'Microsoft 365 / Outlook', host: 'smtp.office365.com', port: 587, security: 'starttls' },
  { label: 'Gmail / Google Workspace', host: 'smtp.gmail.com', port: 587, security: 'starttls' },
  { label: 'Other (SSL on port 465)', host: '', port: 465, security: 'tls' },
]

/** EmailNotifications configures alert e-mails and the daily summary. */
function EmailNotifications() {
  const toast = useToast()
  const { data, error, loading, setData } = useLoad(() => api.get<NotifySettings>('/system/notifications'), [])
  const [form, setForm] = useState<(NotifySettings & { toText: string }) | null>(null)
  const [password, setPassword] = useState<string | null>(null)
  const [busy, setBusy] = useState('')
  const [err, setErr] = useState('')
  useEffect(() => {
    if (data) setForm({ ...data, toText: (data.to ?? []).join(', ') })
  }, [data])
  if (loading && !data) return <Spinner />
  if (error) return <ErrorBox error={error} />
  if (!form) return null
  const set = <K extends keyof NotifySettings | 'toText'>(k: K, v: (NotifySettings & { toText: string })[K]) => setForm({ ...form, [k]: v })

  const save = async (): Promise<boolean> => {
    setBusy('save')
    setErr('')
    try {
      const { toText, ...rest } = form
      const out = await api.put<NotifySettings>('/system/notifications', {
        ...rest,
        to: toText.split(/[,;\n]/).map((x) => x.trim()).filter(Boolean),
        ...(password !== null ? { password } : {}),
      })
      setData(out)
      setPassword(null)
      toast('ok', 'E-mail settings saved')
      return true
    } catch (e) {
      setErr(errorMessage(e))
      return false
    } finally {
      setBusy('')
    }
  }
  const test = async () => {
    if (!(await save())) return
    setBusy('test')
    try {
      await api.post('/system/notifications/test')
      toast('ok', 'Test e-mail sent — check the inbox')
      setData(await api.get<NotifySettings>('/system/notifications'))
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setBusy('')
    }
  }

  return (
    <div className="card mt">
      <div className="card-head">
        <h3>
          <Mail size={17} /> E-mail notifications
        </h3>
        <label className="check">
          <input type="checkbox" checked={form.enabled} onChange={(e) => set('enabled', e.target.checked)} /> Send e-mails
        </label>
      </div>
      <div className="card-body">
        <p className="small muted" style={{ marginTop: 0 }}>
          Tell the people responsible about problems that need action — FBR not reachable, a rejected security token, rejected invoices, invoices not reported
          within 24 hours, unreported incidents, return deadlines, backups — even when nobody has the system open. Each problem is e-mailed at most once a day
          while it lasts. Uses your own mail server; nothing passes through Veridian.
        </p>
        {err && (
          <div className="alert alert-error">
            <div className="alert-body">{err}</div>
          </div>
        )}
        <div className="row small" style={{ gap: 6, marginBottom: 12 }}>
          <span className="faint">Quick setup:</span>
          {presets.map((p) => (
            <button key={p.label} type="button" className="pill" onClick={() => setForm({ ...form, smtpHost: p.host || form.smtpHost, smtpPort: p.port, security: p.security })}>
              {p.label}
            </button>
          ))}
        </div>
        <div className="form-grid">
          <Field label="Mail server (SMTP)">
            <input value={form.smtpHost} onChange={(e) => set('smtpHost', e.target.value)} placeholder="smtp.office365.com" />
          </Field>
          <Field label="Port">
            <input type="number" value={form.smtpPort} onChange={(e) => set('smtpPort', Number(e.target.value))} />
          </Field>
          <Field label="Connection security">
            <select value={form.security} onChange={(e) => set('security', e.target.value as NotifySettings['security'])}>
              <option value="starttls">STARTTLS (usually port 587)</option>
              <option value="tls">SSL/TLS (usually port 465)</option>
              <option value="none">None (internal relay without sign-in)</option>
            </select>
          </Field>
          <Field label="User name" hint="Usually the full e-mail address">
            <input value={form.username} onChange={(e) => set('username', e.target.value)} autoComplete="off" />
          </Field>
          <Field label="Password" hint={form.hasPassword && password === null ? 'Saved (encrypted). Type to replace it.' : 'Gmail and Microsoft 365 may need an app password'}>
            <input type="password" value={password ?? ''} onChange={(e) => setPassword(e.target.value)} autoComplete="new-password" placeholder={form.hasPassword ? '••••••••' : ''} />
          </Field>
          <Field label="Send from" hint='e.g. "E-invoicing <accounts@yourcompany.pk>"'>
            <input value={form.from} onChange={(e) => set('from', e.target.value)} />
          </Field>
          <Field label="Send to" hint="Separate several addresses with commas" span={2}>
            <input value={form.toText} onChange={(e) => set('toText', e.target.value)} placeholder="accounts@yourcompany.pk, owner@yourcompany.pk" />
          </Field>
          <Field label="Daily summary">
            <select value={form.digestHour} onChange={(e) => set('digestHour', Number(e.target.value))}>
              <option value={-1}>Off</option>
              {Array.from({ length: 24 }, (_, h) => (
                <option key={h} value={h}>
                  Every day at {String(h).padStart(2, '0')}:00 (Pakistan time)
                </option>
              ))}
            </select>
          </Field>
          <Field label="Link to the system (optional)" hint="Added to e-mails, e.g. https://192.168.1.10:8443">
            <input value={form.appUrl} onChange={(e) => set('appUrl', e.target.value)} placeholder="https://" />
          </Field>
          <label className="check" style={{ gridColumn: '1 / -1' }}>
            <input type="checkbox" checked={form.immediate} onChange={(e) => set('immediate', e.target.checked)} /> E-mail problems as soon as they are found
            (checked every 5 minutes)
          </label>
        </div>
        <div className="row small mt" style={{ justifyContent: 'space-between' }}>
          <span className="faint">
            {form.lastError ? (
              <span style={{ color: 'var(--danger)' }}>Last attempt failed — {form.lastError}</span>
            ) : form.lastSent ? (
              <>Last e-mail sent {dateTimeFmt(form.lastSent)}</>
            ) : (
              'No e-mail sent yet'
            )}
          </span>
          <span className="row" style={{ gap: 8 }}>
            <button className="btn" onClick={test} disabled={!!busy}>
              <Send size={15} /> {busy === 'test' ? 'Sending…' : 'Save & send test e-mail'}
            </button>
            <button className="btn btn-primary" onClick={save} disabled={!!busy}>
              {busy === 'save' ? 'Saving…' : 'Save'}
            </button>
          </span>
        </div>
      </div>
    </div>
  )
}
