// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useEffect, useRef, useState } from 'react'
import { DatabaseBackup, Download, FileText, History, RefreshCw, Upload } from 'lucide-react'
import { api, errorMessage } from '../../api'
import { ErrorBox, Field, Modal, useLoad } from '../../components/ui'
import { dateTimeFmt } from '../../format'
import { useSession, useToast } from '../../state'

interface Backup {
  name: string
  size: number
  created: string
}

interface BackupSettings {
  hour: number
  retention: number
  dir: string
  defaultDir: string
}

function size(n: number) {
  if (n > 1 << 20) return (n / (1 << 20)).toFixed(1) + ' MB'
  return Math.ceil(n / 1024) + ' KB'
}

const hourLabel = (h: number) => `${String(h).padStart(2, '0')}:00`

const kindLabel: Record<string, string> = {
  auto: 'Automatic',
  manual: 'Made by hand',
  'pre-update': 'Before an update',
  'before-restore': 'Before a restore',
  uploaded: 'Uploaded',
}

/** kind reads the kind of backup from its file name. */
function kind(name: string) {
  const m = /^einvoice-(.+)-\d{8}-\d{6}(-\d+)?\.db$/.exec(name)
  return (m && kindLabel[m[1]]) || ''
}

/** BackupsCard lists, makes, uploads and restores backups, and sets their schedule. */
export default function BackupsCard() {
  const toast = useToast()
  const backups = useLoad(() => api.get<{ items: Backup[] | null; dir: string }>('/system/backups'), [])
  const settings = useLoad(() => api.get<BackupSettings>('/system/backups/settings'), [])
  const [form, setForm] = useState<BackupSettings | null>(null)
  const [busy, setBusy] = useState('')
  const [restoring, setRestoring] = useState<Backup | null>(null)
  const fileRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (settings.data) setForm({ ...settings.data })
  }, [settings.data])

  const backupNow = async () => {
    setBusy('backup')
    try {
      const b = await api.post<Backup>('/system/backups')
      toast('ok', `Backup ${b.name} created`)
      backups.reload()
    } catch (e) {
      toast('err', errorMessage(e))
    } finally {
      setBusy('')
    }
  }

  const upload = async (f: File | undefined) => {
    if (!f) return
    setBusy('upload')
    try {
      const b = await api.upload<Backup>('/system/backups/upload', 'POST', f, 'application/octet-stream')
      toast('ok', `${f.name} checked and added as ${b.name}. You can restore it now.`)
      backups.reload()
    } catch (e) {
      toast('err', errorMessage(e))
    } finally {
      setBusy('')
      if (fileRef.current) fileRef.current.value = ''
    }
  }

  const saveSettings = async () => {
    if (!form) return
    setBusy('settings')
    try {
      const out = await api.put<BackupSettings>('/system/backups/settings', form)
      settings.setData(out)
      toast('ok', 'Backup settings saved')
      backups.reload()
    } catch (e) {
      toast('err', errorMessage(e))
    } finally {
      setBusy('')
    }
  }

  const bs = settings.data
  const changed = !!form && !!bs && (form.hour !== bs.hour || form.retention !== bs.retention || form.dir !== bs.dir)
  return (
    <div className="card mt">
      <div className="card-head">
        <h3>
          <DatabaseBackup size={17} /> Backups
        </h3>
        <span className="row" style={{ gap: 8 }}>
          <input ref={fileRef} type="file" accept=".db" hidden onChange={(e) => upload(e.target.files?.[0])} />
          <button className="btn btn-sm" onClick={() => fileRef.current?.click()} disabled={!!busy} title="Add a backup file kept elsewhere, to restore it">
            <Upload size={14} /> {busy === 'upload' ? 'Checking…' : 'Upload a backup'}
          </button>
          <button className="btn btn-primary btn-sm" onClick={backupNow} disabled={!!busy}>
            {busy === 'backup' ? 'Backing up…' : 'Back up now'}
          </button>
        </span>
      </div>
      <div className="card-body small muted">
        {bs && bs.hour >= 0 ? (
          <>
            A consistent copy of the database is made automatically every day at {hourLabel(bs.hour)} (Pakistan time) in{' '}
            <span className="mono">{backups.data?.dir}</span>; the last {bs.retention} automatic backups are kept.
          </>
        ) : (
          <>
            Automatic backups are <b>off</b>. Backups are kept in <span className="mono">{backups.data?.dir}</span>.
          </>
        )}{' '}
        Copy backups to another disk or cloud storage regularly: records must be kept for six years (rule 150S). Also keep a safe copy of{' '}
        <span className="mono">master.key</span> from the data folder — without it the saved FBR tokens and the invoice signing key cannot be opened (invoices
        remain readable).
      </div>
      <ErrorBox error={backups.error} />
      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>File</th>
              <th>Kind</th>
              <th>Created</th>
              <th className="num">Size</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {(backups.data?.items ?? []).map((b) => (
              <tr key={b.name}>
                <td className="mono small">{b.name}</td>
                <td className="small">{kind(b.name)}</td>
                <td>{dateTimeFmt(b.created)}</td>
                <td className="num">{size(b.size)}</td>
                <td className="right nowrap">
                  <a className="btn btn-sm" href={`/api/v1/system/backups/${encodeURIComponent(b.name)}`}>
                    <Download size={14} /> Download
                  </a>{' '}
                  <button className="btn btn-sm btn-danger" onClick={() => setRestoring(b)} title="Replace all data with this backup">
                    <History size={14} /> Restore
                  </button>
                </td>
              </tr>
            ))}
            {backups.data && (backups.data.items ?? []).length === 0 && (
              <tr>
                <td colSpan={5} className="muted">
                  No backups yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      {form && (
        <details className="card-body backup-settings">
          <summary>Backup settings</summary>
          <div className="form-grid mt-s">
            <Field label="Automatic backup (Pakistan time)">
              <select value={form.hour} onChange={(e) => setForm({ ...form, hour: Number(e.target.value) })}>
                <option value={-1}>Off</option>
                {Array.from({ length: 24 }, (_, h) => (
                  <option key={h} value={h}>
                    Daily at {hourLabel(h)}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Automatic backups kept" hint="Older automatic backups are deleted; backups made by hand are always kept.">
              <input type="number" min={1} max={3650} value={form.retention} onChange={(e) => setForm({ ...form, retention: Number(e.target.value) })} />
            </Field>
            <Field
              label="Backup folder"
              span={2}
              hint={
                <>
                  Leave empty for <span className="mono">{form.defaultDir}</span>. A second disk or a network folder (\\server\share\folder) protects against a
                  failed disk; the background service must be allowed to write there. Backups already made stay where they are.
                </>
              }
            >
              <input value={form.dir} placeholder={form.defaultDir} onChange={(e) => setForm({ ...form, dir: e.target.value })} />
            </Field>
          </div>
          <div className="row mt-s" style={{ justifyContent: 'flex-end' }}>
            <button className="btn btn-primary" onClick={saveSettings} disabled={!!busy || !changed}>
              {busy === 'settings' ? 'Saving…' : 'Save backup settings'}
            </button>
          </div>
        </details>
      )}
      {restoring && <RestoreDialog backup={restoring} onClose={() => setRestoring(null)} />}
    </div>
  )
}

/** RestoreDialog replaces all data with a backup, after a typed confirmation. */
function RestoreDialog({ backup, onClose }: { backup: Backup; onClose: () => void }) {
  const s = useSession()
  const [text, setText] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [done, setDone] = useState<{ safetyBackup: string; dir: string } | null>(null)

  const restore = async () => {
    setBusy(true)
    setError('')
    try {
      setDone(await api.post<{ safetyBackup: string; dir: string }>(`/system/backups/${encodeURIComponent(backup.name)}/restore`, { confirm: text }))
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const signInAgain = async () => {
    await s.logout().catch(() => {})
    window.location.assign('/')
  }

  if (done) {
    return (
      <Modal
        title="Backup restored"
        onClose={signInAgain}
        footer={
          <button className="btn btn-primary" onClick={signInAgain}>
            Sign in again
          </button>
        }
      >
        <div className="alert alert-ok">The data is now as it was on {dateTimeFmt(backup.created)}.</div>
        <p>
          The data from just before the restore was saved as <span className="mono">{done.safetyBackup}</span> in{' '}
          <span className="mono">{done.dir}</span>, so this step can be undone by restoring that file.
        </p>
        <p className="small muted">Everybody signs in again: the sign-ins and passwords are those of the backup.</p>
      </Modal>
    )
  }
  return (
    <Modal
      title={`Restore ${backup.name}`}
      onClose={onClose}
      footer={
        <>
          <button className="btn" onClick={onClose} disabled={busy}>
            Cancel
          </button>
          <button className="btn btn-danger" onClick={restore} disabled={busy || text !== 'RESTORE'}>
            {busy ? (
              <>
                <RefreshCw size={14} className="spin" /> Restoring…
              </>
            ) : (
              'Restore this backup'
            )}
          </button>
        </>
      }
    >
      {error && <div className="alert alert-error">{error}</div>}
      <p>
        All data — companies, customers, products, invoices, users and settings — is replaced with this backup from <b>{dateTimeFmt(backup.created)}</b>.
      </p>
      <div className="alert alert-warn">
        <div className="alert-body">
          <b>Invoices made after this backup will no longer be in the software.</b> FBR keeps those it accepted: check them in the IRIS portal and enter them
          again if needed. The current data is backed up first, so you can go back.
        </div>
      </div>
      <Field label="Type RESTORE to confirm">
        <input value={text} onChange={(e) => setText(e.target.value.trim().toUpperCase())} autoFocus placeholder="RESTORE" />
      </Field>
    </Modal>
  )
}

/** ServerLog shows the end of the server's log file, for troubleshooting. */
export function ServerLog() {
  const [log, setLog] = useState<{ text: string; path: string } | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<unknown>(null)
  const load = async () => {
    setBusy(true)
    setError(null)
    try {
      setLog(await api.get<{ text: string; path: string }>('/system/logs?lines=300'))
    } catch (e) {
      setError(e)
    } finally {
      setBusy(false)
    }
  }
  return (
    <div className="card mt">
      <div className="card-head">
        <h3>
          <FileText size={17} /> Server log
        </h3>
        <span className="row" style={{ gap: 8 }}>
          <button className="btn btn-sm" onClick={load} disabled={busy}>
            <RefreshCw size={14} className={busy ? 'spin' : ''} /> {log ? 'Refresh' : 'Show the latest entries'}
          </button>
          <a className="btn btn-sm" href="/api/v1/system/logs/download">
            <Download size={14} /> Download log
          </a>
        </span>
      </div>
      <div className="card-body">
        <p className="small muted" style={{ marginTop: 0 }}>
          What the server did: start-ups, FBR connection problems, automatic backups, updates and errors. Send the log file to support when asked.
        </p>
        <ErrorBox error={error} />
        {log &&
          (log.text ? (
            <pre className="log-view">{log.text}</pre>
          ) : (
            <p className="small faint">The log is empty.</p>
          ))}
        {log && <p className="small faint mono">{log.path}</p>}
      </div>
    </div>
  )
}
