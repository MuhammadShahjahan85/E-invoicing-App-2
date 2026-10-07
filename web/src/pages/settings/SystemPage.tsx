// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useState } from 'react'
import { api, errorMessage } from '../../api'
import { ErrorBox, Spinner, useLoad } from '../../components/ui'
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
          <p className="small muted">Upload the official logo provided by FBR/PRAL. It is printed next to the QR code on every invoice.</p>
          <div className="row">
            <div className="qr-box" style={{ width: 'auto', height: 'auto', padding: 6 }}>
              <img
                key={logoTick}
                src={`/api/v1/system/fbr-logo?t=${logoTick}`}
                alt="not uploaded"
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
    </>
  )
}
