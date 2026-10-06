import { useState } from 'react'
import { Link } from 'react-router-dom'
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
