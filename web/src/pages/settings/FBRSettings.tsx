// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Copy, ExternalLink, Printer } from 'lucide-react'
import { api, errorMessage } from '../../api'
import { Confirm, Field, useLoad } from '../../components/ui'
import { dateTimeFmt, envLabels } from '../../format'
import { useCompanyPath, useSession, useToast } from '../../state'
import type { Company, Env } from '../../types'

interface ConnectionTest {
  environment: Env
  ok: boolean
  steps: { name: string; ok: boolean; message: string }[]
}

interface SyncReport {
  environment: Env
  counts: Record<string, number>
  errors: Record<string, string> | null
  at: string
}

export default function FBRSettings() {
  const s = useSession()
  const cp = useCompanyPath()
  const toast = useToast()
  const c = s.company!
  const canWrite = s.can('company.write')
  const [confirmEnv, setConfirmEnv] = useState<Env | null>(null)
  const [envError, setEnvError] = useState('')
  const [test, setTest] = useState<ConnectionTest | null>(null)
  const [testing, setTesting] = useState<Env | ''>('')
  const [sync, setSync] = useState<SyncReport | null>(null)
  const [syncing, setSyncing] = useState(false)
  const { data: refStatus, reload: reloadRef } = useLoad(
    () => api.get<{ entries: { key: string; kind: string; source: string; fetchedAt: string }[] | null; hsCodes: number }>(`${cp}/ref/status`),
    [cp],
  )

  const switchEnv = async (env: Env) => {
    setEnvError('')
    try {
      const out = await api.put<Company>(`/companies/${c.id}`, { ...c, environment: env })
      s.updateCompany(out)
      toast('ok', `Now working in: ${envLabels[env]}`)
    } catch (e) {
      setEnvError(errorMessage(e))
    }
  }

  const runTest = async (env: Env) => {
    setTesting(env)
    setTest(null)
    try {
      setTest(await api.post<ConnectionTest>(`${cp}/test-connection?env=${env}`))
    } catch (e) {
      setTest({ environment: env, ok: false, steps: [{ name: 'Request', ok: false, message: errorMessage(e) }] })
    } finally {
      setTesting('')
    }
  }

  const runSync = async () => {
    setSyncing(true)
    try {
      const r = await api.post<SyncReport>(`${cp}/sync-reference`)
      setSync(r)
      reloadRef()
      toast(r.errors && Object.keys(r.errors).length ? 'err' : 'ok', 'Reference data synchronised')
    } catch (e) {
      toast('err', errorMessage(e))
    } finally {
      setSyncing(false)
    }
  }

  return (
    <>
      <div className="page-head">
        <div>
          <h1>FBR integration</h1>
          <p>Connection to FBR's Digital Invoicing API (gw.fbr.gov.pk) through PRAL or your licensed integrator.</p>
        </div>
      </div>

      <div className="card card-pad">
        <h3>Working environment</h3>
        <div className="grid g3">
          {(['simulator', 'sandbox', 'production'] as Env[]).map((env) => (
            <div key={env} className={'card card-pad' + (c.environment === env ? ' selected' : '')} style={c.environment === env ? { outline: '2px solid var(--primary)' } : undefined}>
              <b>{envLabels[env]}</b>
              <p className="small muted">
                {env === 'simulator' && 'Offline practice with a built-in FBR simulator. Nothing is sent to FBR; printouts say TRAINING.'}
                {env === 'sandbox' && 'FBR test system for the assigned scenarios. Needs the sandbox token. Printouts are marked as not valid.'}
                {env === 'production' && 'Live reporting of tax invoices. Needs the production token, a whitelisted IP and a licence for this NTN.'}
              </p>
              {c.environment === env ? (
                <span className="badge b-green">current</span>
              ) : (
                canWrite && (
                  <button className={'btn btn-sm' + (env === 'production' ? ' btn-primary' : '')} onClick={() => setConfirmEnv(env)}>
                    Switch to {envLabels[env].toLowerCase()}
                  </button>
                )
              )}
            </div>
          ))}
        </div>
        {envError && <div className="alert alert-error mt">{envError}</div>}
      </div>

      <div className="grid g2 mt">
        <TokenCard env="sandbox" company={c} canWrite={canWrite} onTest={() => runTest('sandbox')} testing={testing === 'sandbox'} />
        <TokenCard env="production" company={c} canWrite={canWrite} onTest={() => runTest('production')} testing={testing === 'production'} />
      </div>

      {test && (
        <div className={'alert mt ' + (test.ok ? 'alert-ok' : 'alert-error')}>
          <b>
            Connection test — {envLabels[test.environment]}: {test.ok ? 'passed' : 'failed'}
          </b>
          <ul className="issue-list">
            {test.steps.map((st, i) => (
              <li key={i}>
                {st.ok ? '✔' : '✘'} <b>{st.name}</b> — {st.message}
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="grid g2 mt">
        <div className="card card-pad">
          <h3>Reference data</h3>
          <p className="small muted">
            Provinces, units of measure, sale types, SRO items and HS codes are downloaded from FBR's reference APIs and cached. Rates and SRO schedules are fetched
            per invoice date and cached for 24 hours. Built-in lists are used when FBR is unreachable.
          </p>
          <p className="small">
            HS codes available: <b>{refStatus?.hsCodes ?? '…'}</b>
            {refStatus?.entries?.length ? (
              <>
                {' '}
                · last download {dateTimeFmt(refStatus.entries.map((e) => e.fetchedAt).sort().slice(-1)[0])}
              </>
            ) : null}
          </p>
          {canWrite && (
            <button className="btn" onClick={runSync} disabled={syncing}>
              {syncing ? 'Downloading…' : `Download from FBR (${envLabels[c.environment].toLowerCase()})`}
            </button>
          )}
          {sync && (
            <div className="small mt">
              {Object.entries(sync.counts ?? {}).map(([k, v]) => (
                <span key={k} className="badge b-blue" style={{ marginRight: 4 }}>
                  {k}: {v}
                </span>
              ))}
              {Object.entries(sync.errors ?? {}).map(([k, v]) => (
                <div key={k} style={{ color: 'var(--danger)' }}>
                  {k}: {v}
                </div>
              ))}
            </div>
          )}
        </div>
        <div className="card card-pad">
          <h3>Go-live requirements</h3>
          <ol className="small">
            <li>
              On IRIS open <b>Digital Invoicing</b>, register, and choose PRAL (free) or your licensed integrator. Record your business activity and sector.
            </li>
            <li>Copy the sandbox security token into the sandbox card above and run every assigned scenario on the <Link to="/scenarios">Scenarios</Link> page.</li>
            <li>
              FBR only accepts production calls from a <b>whitelisted static public IP</b>. Give the public IP of this server's internet connection (ask your ISP for a
              static IP) to PRAL / your integrator to whitelist.
            </li>
            <li>Generate the production token on IRIS (valid for five years), save it above with its expiry date and run the connection test.</li>
            <li>Switch the working environment to production. From then on every invoice is reported to FBR in real time.</li>
          </ol>
        </div>
      </div>

      <div className="grid g2 mt">
        <IrisDetails company={c} canWrite={canWrite} />
        <SignboardCard company={c} />
      </div>

      {confirmEnv && (
        <Confirm
          danger={confirmEnv !== 'production'}
          label={`Switch to ${envLabels[confirmEnv].toLowerCase()}`}
          text={
            confirmEnv === 'production' ? (
              <p>
                From now on invoices for <b>{c.name}</b> will be reported to FBR as real tax invoices. Make sure all assigned scenarios have passed and the production token
                is saved.
              </p>
            ) : (
              <p>
                Invoices created in the {envLabels[confirmEnv].toLowerCase()} are <b>not</b> reported to FBR as tax invoices.
                {c.environment === 'production' && ' Do not issue real sales from this environment.'}
              </p>
            )
          }
          onConfirm={() => switchEnv(confirmEnv)}
          onClose={() => setConfirmEnv(null)}
        />
      )}
    </>
  )
}

function TokenCard({ env, company, canWrite, onTest, testing }: { env: 'sandbox' | 'production'; company: Company; canWrite: boolean; onTest: () => void; testing: boolean }) {
  const s = useSession()
  const toast = useToast()
  const [token, setToken] = useState('')
  const [expiry, setExpiry] = useState(env === 'sandbox' ? company.sandboxTokenExpiry : company.productionTokenExpiry)
  const [error, setError] = useState('')
  const has = env === 'sandbox' ? company.hasSandboxToken : company.hasProductionToken
  const curExpiry = env === 'sandbox' ? company.sandboxTokenExpiry : company.productionTokenExpiry

  const save = async (clear: boolean) => {
    setError('')
    try {
      const out = await api.post<Company>(`/companies/${company.id}/token`, { environment: env, token, expiry, clear })
      s.updateCompany(out)
      setToken('')
      toast('ok', clear ? 'Token removed' : 'Token saved (encrypted)')
    } catch (e) {
      setError(errorMessage(e))
    }
  }

  return (
    <div className="card card-pad">
      <h3>{env === 'sandbox' ? 'Sandbox token' : 'Production token'}</h3>
      <p className="small">
        {has ? <span className="badge b-green">saved</span> : <span className="badge b-gray">not saved</span>} {curExpiry && <span className="muted">expires {curExpiry}</span>}
      </p>
      {error && <div className="alert alert-error">{error}</div>}
      {canWrite && (
        <div className="stack">
          <Field label="Security token from IRIS" hint="Stored encrypted with this server's master key; never displayed again">
            <input type="password" autoComplete="off" value={token} onChange={(e) => setToken(e.target.value)} placeholder={has ? '•••••••• (enter to replace)' : ''} />
          </Field>
          <Field label="Token expiry date">
            <input type="date" value={expiry} onChange={(e) => setExpiry(e.target.value)} />
          </Field>
          <div className="row">
            <button className="btn btn-primary" disabled={!token} onClick={() => save(false)}>
              Save token
            </button>
            <button className="btn" disabled={!has || testing} onClick={onTest}>
              {testing ? 'Testing…' : 'Test connection'}
            </button>
            {has && (
              <button className="btn btn-danger" onClick={() => save(true)}>
                Remove
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

function CopyValue({ value }: { value: string }) {
  const toast = useToast()
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(value)
      toast('ok', 'Copied')
    } catch {
      toast('err', 'Copy is not available here — select the text and copy it')
    }
  }
  return (
    <span className="row" style={{ gap: 6, flexWrap: 'nowrap' }}>
      <code style={{ whiteSpace: 'normal' }}>{value}</code>
      <button className="icon-btn" aria-label={`Copy ${value}`} title="Copy" onClick={copy}>
        <Copy size={14} />
      </button>
    </span>
  )
}

// IrisDetails lists what to enter on IRIS (Digital Invoicing → API
// Integration → Technical Details and IP Whitelisting, PRAL's DI user
// manual v1.5) and keeps the software registration number FBR issues.
function IrisDetails({ company, canWrite }: { company: Company; canWrite: boolean }) {
  const s = useSession()
  const toast = useToast()
  const [regNo, setRegNo] = useState(company.softwareRegNo ?? '')
  const [ip, setIp] = useState<{ ip: string; source: string } | null>(null)
  const [ipError, setIpError] = useState('')
  const [busy, setBusy] = useState(false)
  const provider = `${s.meta.vendor || 'Veridian Partners Consultancy Private Limited'} — ${s.meta.product || 'Veridian E-invoicing Pakistan'}`

  const saveRegNo = async () => {
    try {
      const out = await api.put<Company>(`/companies/${company.id}`, { ...company, softwareRegNo: regNo })
      s.updateCompany(out)
      toast('ok', 'Software registration number saved')
    } catch (e) {
      toast('err', errorMessage(e))
    }
  }
  const detectIp = async () => {
    setBusy(true)
    setIpError('')
    try {
      setIp(await api.get<{ ip: string; source: string }>('/system/public-ip'))
    } catch (e) {
      setIpError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="card card-pad">
      <h3>Registration on IRIS</h3>
      <p className="small muted">
        On IRIS open <b>Digital Invoicing → API Integration</b>. Enter these technical details, then the IP whitelisting details. PRAL accepts or rejects the IP
        addresses within about two working hours; with another licensed integrator, the integrator whitelists them.
      </p>
      <table className="table kv-table small">
        <tbody>
          <tr>
            <th>ERP / System Provider</th>
            <td>
              <CopyValue value={provider} />
            </td>
          </tr>
          <tr>
            <th>Software Type</th>
            <td>
              <CopyValue value="On Premises" />
            </td>
          </tr>
          <tr>
            <th>Software Version</th>
            <td>
              <CopyValue value={s.meta.version || '1.0'} />
            </td>
          </tr>
          <tr>
            <th>Business Nature</th>
            <td>{company.businessActivities.join(', ') || '—'}</td>
          </tr>
          <tr>
            <th>Sector (one only)</th>
            <td>{company.sectors[0] || '—'}</td>
          </tr>
          <tr>
            <th>CRM user ID</th>
            <td>
              The e-mail address you will use on PRAL's support portal,{' '}
              <a href="https://dicrm.pral.com.pk" target="_blank" rel="noopener noreferrer">
                dicrm.pral.com.pk <ExternalLink size={11} />
              </a>
            </td>
          </tr>
          <tr>
            <th>IP address (1 to 3)</th>
            <td>
              {ip ? (
                <>
                  <CopyValue value={ip.ip} />
                  <div className="faint">as seen by {ip.source}</div>
                </>
              ) : (
                canWrite && (
                  <button className="btn btn-sm" onClick={detectIp} disabled={busy} title="Asks a public IP service which address this server's internet traffic comes from">
                    {busy ? 'Checking…' : "Find this server's public IP"}
                  </button>
                )
              )}
              {ipError && <div style={{ color: 'var(--danger)' }}>{ipError}</div>}
              <div className="faint">The address must be a static public IP; ask your internet provider for one.</div>
            </td>
          </tr>
        </tbody>
      </table>
      <Field
        label="Software registration number"
        hint="The registration number of this invoicing software issued on FBR's system. Rule 150R(13)(c) requires it on every invoice; it is also printed on the signboard."
      >
        <div className="row" style={{ flexWrap: 'nowrap' }}>
          <input value={regNo} onChange={(e) => setRegNo(e.target.value)} disabled={!canWrite} maxLength={60} />
          {canWrite && (
            <button className="btn" disabled={regNo === (company.softwareRegNo ?? '')} onClick={saveRegNo}>
              Save
            </button>
          )}
        </div>
      </Field>
    </div>
  )
}

function SignboardCard({ company }: { company: Company }) {
  const cp = useCompanyPath()
  const [outlet, setOutlet] = useState('')
  return (
    <div className="card card-pad">
      <h3>“Integrated with FBR” signboard</h3>
      <p className="small muted">
        Rule 150R(11) requires a signboard bearing FBR's official logo, the text “Integrated with FBR” and the software registration number to be displayed
        prominently at each notified outlet, point of sale or invoicing machine. Print one per outlet (A4 landscape) and display it where buyers can see it.
      </p>
      {!company.softwareRegNo && (
        <div className="alert alert-warn small">Enter the software registration number first; the signboard must show it.</div>
      )}
      <Field label="Outlet or point of sale (optional)">
        <input value={outlet} onChange={(e) => setOutlet(e.target.value)} placeholder="e.g. Head office sales counter" />
      </Field>
      <a className="btn btn-primary mt" href={`/api/v1${cp}/signboard${outlet.trim() ? '?outlet=' + encodeURIComponent(outlet.trim()) : ''}`} target="_blank" rel="noopener">
        <Printer size={15} /> Print signboard
      </a>
    </div>
  )
}
