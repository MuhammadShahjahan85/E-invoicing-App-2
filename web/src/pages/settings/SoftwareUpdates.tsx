// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useEffect, useRef, useState } from 'react'
import { CircleCheck, Download, RefreshCw, Rocket, ShieldCheck, TriangleAlert } from 'lucide-react'
import { api, errorMessage } from '../../api'
import { ErrorBox, Field, Spinner, useLoad } from '../../components/ui'
import { dateFmt, dateTimeFmt } from '../../format'
import { useToast } from '../../state'
import type { UpdateStatus } from '../../types'

const hourLabel = (h: number) => `${String(h).padStart(2, '0')}:00`

function mb(n: number) {
  return (n / (1 << 20)).toFixed(1) + ' MB'
}

/** Notes shows release notes; lines starting with "- " become a list. */
export function ReleaseNotes({ text }: { text: string }) {
  const lines = text.split('\n').map((l) => l.trim()).filter(Boolean)
  const items = lines.filter((l) => l.startsWith('- ') || l.startsWith('* '))
  if (items.length && items.length === lines.length) {
    return (
      <ul className="release-notes">
        {items.map((l, i) => (
          <li key={i}>{l.slice(2)}</li>
        ))}
      </ul>
    )
  }
  return <div className="release-notes pre">{text}</div>
}

/** SoftwareUpdates checks for, downloads and installs new versions. */
export default function SoftwareUpdates() {
  const toast = useToast()
  const { data, error, setData } = useLoad(() => api.get<UpdateStatus>('/system/updates'), [])
  const [busy, setBusy] = useState('')
  const [err, setErr] = useState('')
  const [form, setForm] = useState<UpdateStatus['settings'] | null>(null)
  const [installing, setInstalling] = useState<string | null>(null)

  useEffect(() => {
    if (data && !form) setForm({ ...data.settings })
  }, [data, form])

  // Follow a download in progress.
  const state = data?.state
  useEffect(() => {
    if (state !== 'downloading') return
    const t = window.setInterval(() => {
      api.get<UpdateStatus>('/system/updates').then(setData).catch(() => {})
    }, 1000)
    return () => window.clearInterval(t)
  }, [state, setData])

  if (error) return <ErrorBox error={error} />
  if (!data || !form) return <Spinner />

  const run = async (what: string, fn: () => Promise<UpdateStatus>) => {
    setBusy(what)
    setErr('')
    try {
      setData(await fn())
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setBusy('')
    }
  }
  const check = () =>
    run('check', async () => {
      const st = await api.post<UpdateStatus>('/system/updates/check')
      if (!st.lastError) toast('ok', st.available ? `Version ${st.available.version} is available` : 'You have the latest version')
      return st
    })
  const download = () => run('download', () => api.post<UpdateStatus>('/system/updates/start', { install: false }))
  const install = async () => {
    setErr('')
    try {
      const st = await api.post<UpdateStatus>('/system/updates/start', { install: true })
      setData(st)
      setInstalling(st.available?.version ?? '')
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  const save = () =>
    run('save', async () => {
      const st = await api.put<UpdateStatus>('/system/updates/settings', form)
      setForm({ ...st.settings })
      toast('ok', 'Update settings saved')
      return st
    })

  const a = data.available
  const changed = form.mode !== data.settings.mode || form.hour !== data.settings.hour || form.feed !== data.settings.feed
  return (
    <div className="card mt-0">
      <div className="card-head">
        <h3>
          <Rocket size={17} /> Software updates
        </h3>
        <span className="badge b-blue">Version {data.current}</span>
      </div>
      <div className="card-body">
        {err && <div className="alert alert-error">{err}</div>}
        {!data.configured ? (
          <div className="alert alert-info">
            This copy of the software was built without automatic updates. Ask Veridian Partners Consultancy for the latest installer, or for a copy that updates
            itself.
          </div>
        ) : a ? (
          <div className={'update-offer' + (a.mandatory ? ' required' : '')}>
            <div className="row" style={{ justifyContent: 'space-between', alignItems: 'flex-start' }}>
              <div>
                <div className="update-title">
                  Version {a.version} is available {a.mandatory && <span className="badge b-amber">Required update</span>}
                </div>
                <div className="small muted">
                  {a.released && <>Released {dateFmt(a.released)} · </>}You have version {data.current}
                </div>
              </div>
              {data.canInstall && !data.blocked && (
                <button className="btn btn-primary" onClick={install} disabled={!!busy || data.state === 'downloading' || data.state === 'installing'}>
                  <Rocket size={15} /> Update now
                </button>
              )}
            </div>
            <h4 className="mt">What's new</h4>
            <ReleaseNotes text={a.notes} />
            {data.blocked && (
              <div className="alert alert-warn mt">
                <TriangleAlert size={16} /> {data.blocked}
              </div>
            )}
            {data.state === 'downloading' && (
              <div className="mt">
                <div className="small muted">
                  Downloading… {mb(data.downloaded)} of {mb(data.size)}
                </div>
                <div className="progress">
                  <div style={{ width: `${data.size ? Math.min(100, (100 * data.downloaded) / data.size) : 0}%` }} />
                </div>
              </div>
            )}
            {data.state === 'failed' && data.message && <div className="alert alert-error mt">{data.message}</div>}
            {!data.canInstall && !data.blocked && (
              <div className="alert alert-info mt">
                <div className="alert-body">
                  {data.manualNote}
                  <div className="row mt" style={{ gap: 8 }}>
                    {data.ready ? (
                      <a className="btn btn-primary btn-sm" href="/api/v1/system/updates/file">
                        <Download size={14} /> Download the installer (checked)
                      </a>
                    ) : (
                      <button className="btn btn-sm" onClick={download} disabled={!!busy || data.state === 'downloading'}>
                        <Download size={14} /> {data.state === 'downloading' ? 'Downloading…' : 'Download the update'}
                      </button>
                    )}
                  </div>
                </div>
              </div>
            )}
            {data.nextAuto && data.state !== 'installing' && (
              <p className="small muted mt">
                {a.mandatory && data.settings.mode !== 'auto' ? 'This required update' : 'This update'} installs by itself on {dateTimeFmt(data.nextAuto)} (Pakistan time),
                or click Update now.
              </p>
            )}
            <p className="small faint mt">
              <ShieldCheck size={13} /> Signed by Veridian Partners Consultancy and checked before it is installed. The database is backed up first; the software
              restarts by itself and is back in about a minute.
            </p>
          </div>
        ) : (
          <div className="alert alert-ok">
            <CircleCheck size={16} /> You have the latest version{data.lastCheck ? '' : ' that is known'} ({data.current}).
          </div>
        )}

        {data.configured && (
          <div className="row small" style={{ justifyContent: 'space-between', marginTop: 10 }}>
            <span className="faint">
              {data.lastError ? (
                <span style={{ color: 'var(--danger)' }}>The last check failed — {data.lastError}</span>
              ) : data.lastCheck ? (
                <>Last checked {dateTimeFmt(data.lastCheck)}</>
              ) : (
                'Not checked yet'
              )}
            </span>
            <button className="btn btn-sm" onClick={check} disabled={!!busy}>
              <RefreshCw size={14} className={busy === 'check' ? 'spin' : ''} /> {busy === 'check' ? 'Checking…' : 'Check for updates'}
            </button>
          </div>
        )}

        {data.last && (
          <div className={'alert mt ' + (data.last.ok ? 'alert-ok' : 'alert-warn')}>
            {data.last.ok ? (
              <>
                Updated from version {data.last.from} to {data.last.to} on {dateTimeFmt(data.last.at)}
                {data.last.by && data.last.by !== 'system' ? ` by ${data.last.by}` : ' (automatic)'}.
              </>
            ) : (
              data.last.message
            )}
          </div>
        )}

        {data.configured && (
          <>
            <hr />
            <div className="form-grid">
              <Field label="How should updates be installed?" span={2}>
                <select value={form.mode} onChange={(e) => setForm({ ...form, mode: e.target.value as UpdateStatus['settings']['mode'] })}>
                  <option value="notify">Tell administrators — they click Update now</option>
                  <option value="auto">Automatically, at the time below</option>
                  <option value="off">Don't look for updates</option>
                </select>
              </Field>
              <Field label="Update time (Pakistan time)" hint="Choose a time when nobody is invoicing and the computer is on. Required updates also install at this time.">
                <select value={form.hour} onChange={(e) => setForm({ ...form, hour: Number(e.target.value) })} disabled={form.mode === 'off'}>
                  {Array.from({ length: 24 }, (_, h) => (
                    <option key={h} value={h}>
                      {hourLabel(h)}
                    </option>
                  ))}
                </select>
              </Field>
            </div>
            <details className="mt small">
              <summary className="faint">Advanced: update address</summary>
              <Field label="Update address" hint="Leave empty to use the address built into the software. Only updates signed by Veridian Partners Consultancy are ever installed.">
                <input value={form.feed} placeholder={data.builtInFeed || 'https://'} onChange={(e) => setForm({ ...form, feed: e.target.value })} />
              </Field>
            </details>
            <div className="row mt" style={{ justifyContent: 'flex-end' }}>
              <button className="btn btn-primary" onClick={save} disabled={!!busy || !changed}>
                {busy === 'save' ? 'Saving…' : 'Save'}
              </button>
            </div>
          </>
        )}
      </div>
      {installing !== null && <UpdateProgress target={installing} onClose={() => setInstalling(null)} />}
    </div>
  )
}

type Phase = 'working' | 'restarting' | 'done' | 'failed' | 'slow'

/** UpdateProgress follows an installation until the new version answers. */
function UpdateProgress({ target, onClose }: { target: string; onClose: () => void }) {
  const [phase, setPhase] = useState<Phase>('working')
  const [st, setSt] = useState<UpdateStatus | null>(null)
  const [message, setMessage] = useState('')
  const since = useRef(Date.now())

  useEffect(() => {
    let stop = false
    let restarted = false
    const tick = async () => {
      if (stop) return
      try {
        // Sessions survive the restart, so the new version answers this too.
        const s = await api.get<UpdateStatus>('/system/updates')
        setSt(s)
        if (s.current === target) {
          setPhase('done')
          window.setTimeout(() => window.location.reload(), 1800)
          return
        }
        const failed = s.state === 'failed' || (restarted && !s.pending && s.last && !s.last.ok && s.last.to === target)
        if (failed) {
          setMessage(s.message || s.last?.message || 'The update did not complete.')
          setPhase('failed')
          return
        }
      } catch {
        // The service stopped for the installer: wait for it to come back.
        restarted = true
        setPhase((p) => (p === 'working' ? 'restarting' : p))
      }
      if (Date.now() - since.current > 6 * 60_000) setPhase((p) => (p === 'done' || p === 'failed' ? p : 'slow'))
      window.setTimeout(tick, restarted ? 2500 : 1000)
    }
    tick()
    return () => {
      stop = true
    }
  }, [target])

  const step =
    phase === 'restarting'
      ? 'Installing and restarting…'
      : st?.state === 'downloading'
        ? `Downloading… ${mb(st.downloaded)} of ${mb(st.size)}`
        : st?.message || 'Preparing…'
  return (
    <div className="modal-back update-screen">
      <div className="modal" role="dialog" aria-modal="true" aria-label="Updating the software">
        <div className="modal-body" style={{ textAlign: 'center', padding: '30px 26px' }}>
          {phase === 'done' ? (
            <>
              <CircleCheck size={42} className="ok-icon" />
              <h3>Updated to version {target}</h3>
              <p className="muted">Opening the new version…</p>
            </>
          ) : phase === 'failed' ? (
            <>
              <TriangleAlert size={42} className="warn-icon" />
              <h3>The update did not complete</h3>
              <p className="muted">{message}</p>
              <p className="small faint">The software keeps working on version {st?.current}. Nothing was lost; the database was backed up first.</p>
              <button className="btn btn-primary mt" onClick={onClose}>
                Close
              </button>
            </>
          ) : (
            <>
              <RefreshCw size={38} className="spin" />
              <h3>Updating to version {target}</h3>
              <p className="muted">{step}</p>
              {st?.state === 'downloading' && st.size > 0 && (
                <div className="progress" style={{ margin: '12px 0' }}>
                  <div style={{ width: `${Math.min(100, (100 * st.downloaded) / st.size)}%` }} />
                </div>
              )}
              <p className="small faint">
                Please keep this page open. Invoices waiting to be sent to FBR are kept and sent after the restart.
              </p>
              {phase === 'slow' && (
                <div className="alert alert-warn mt" style={{ textAlign: 'left' }}>
                  This is taking longer than expected. If the software does not come back, open it from the desktop icon (it starts the service again) or
                  contact support. The database was backed up before the update.
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}
