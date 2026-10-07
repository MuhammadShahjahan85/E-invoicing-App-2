// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useState } from 'react'
import { Link } from 'react-router-dom'
import { api, errorMessage } from '../api'
import { ErrorBox, Modal, Spinner, useLoad } from '../components/ui'
import { dateTimeFmt, money, qty } from '../format'
import { useCompanyPath, useSession, useToast } from '../state'
import type { Invoice, ScenarioOverview, ScenarioRun, ScenarioStatus } from '../types'

export default function Scenarios() {
  const s = useSession()
  const cp = useCompanyPath()
  const toast = useToast()
  const { data, error, loading, reload, setData } = useLoad(() => api.get<ScenarioOverview>(`${cp}/scenarios`), [cp])
  const [filter, setFilter] = useState<'assigned' | 'all'>('assigned')
  const [mode, setMode] = useState<'engine' | 'sample'>('engine')
  const [running, setRunning] = useState('')
  const [runAll, setRunAll] = useState(false)
  const [editAssigned, setEditAssigned] = useState(false)
  const [detail, setDetail] = useState<ScenarioStatus | null>(null)
  const company = s.company!
  const canRun = s.can('scenarios')

  if (loading && !data) return <Spinner />
  if (error) return <ErrorBox error={error} />
  if (!data) return null

  const run = async (sn: string, environment: 'sandbox' | 'simulator') => {
    setRunning(sn + environment)
    try {
      const res = await api.post<{ run: ScenarioRun; invoice: Invoice | null }>(`${cp}/scenarios/${sn}/run`, { mode, environment })
      const r = res.run
      toast(r.status === 'passed' || r.status === 'practice' ? 'ok' : 'err', `${sn}: ${r.status} — ${r.message}`)
      return r
    } catch (e) {
      toast('err', `${sn}: ${errorMessage(e)}`)
      return null
    } finally {
      setRunning('')
    }
  }

  const runAllAssigned = async (environment: 'sandbox' | 'simulator') => {
    setRunAll(true)
    for (const sc of data.scenarios.filter((x) => x.assigned && (environment === 'simulator' || !x.passed))) {
      await run(sc.id, environment)
    }
    setRunAll(false)
    reload()
  }

  const list = data.scenarios.filter((x) => filter === 'all' || x.assigned)

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Sandbox scenarios</h1>
          <p>
            FBR assigns test scenarios on IRIS according to your business activity and sector. Every assigned scenario must be submitted successfully in the FBR
            sandbox before IRIS issues the production token.
          </p>
        </div>
        <div className="actions">
          {canRun && (
            <>
              <button className="btn" onClick={() => setEditAssigned(true)}>
                Set assigned scenarios
              </button>
              <button className="btn" disabled={runAll} onClick={() => runAllAssigned('simulator')}>
                Practice all (simulator)
              </button>
              <button className="btn btn-primary" disabled={runAll || !company.hasSandboxToken} onClick={() => runAllAssigned('sandbox')}>
                {runAll ? 'Running…' : 'Run pending in FBR sandbox'}
              </button>
            </>
          )}
        </div>
      </div>

      {!company.hasSandboxToken && (
        <div className="alert alert-warn">
          No sandbox token saved. Register for Digital Invoicing on IRIS, choose your integrator and copy the sandbox token into{' '}
          <Link to="/settings/fbr">FBR integration settings</Link>. Until then you can practise in the training simulator.
        </div>
      )}

      <div className="grid g3">
        <div className="card stat">
          <div className="label">Assigned scenarios passed</div>
          <div className="value">
            {data.passedCount} / {data.assignedCount}
          </div>
          <div className="progress" style={{ marginTop: 8 }}>
            <div style={{ width: `${data.assignedCount ? (100 * data.passedCount) / data.assignedCount : 0}%` }} />
          </div>
        </div>
        <div className="card stat">
          <div className="label">Production readiness</div>
          <div className="value" style={{ color: data.readyForProduction ? 'var(--success)' : 'var(--warning)' }}>
            {data.readyForProduction ? 'Ready' : 'Not yet'}
          </div>
          <div className="sub">{data.readyForProduction ? 'Request the production token on IRIS' : 'Pass all assigned scenarios in the sandbox'}</div>
        </div>
        <div className="card stat">
          <div className="label">Profile</div>
          <div className="sub" style={{ marginTop: 6 }}>
            <b>Activity:</b> {company.businessActivities.join(', ') || '—'}
            <br />
            <b>Sector:</b> {company.sectors.join(', ') || '—'}
            <br />
            <Link to="/settings/company">Change</Link>
          </div>
        </div>
      </div>

      <div className="card mt">
        <div className="card-body row">
          <div className="tabs" style={{ marginBottom: 0 }}>
            <button className={'tab' + (filter === 'assigned' ? ' active' : '')} onClick={() => setFilter('assigned')}>
              Assigned ({data.assignedCount})
            </button>
            <button className={'tab' + (filter === 'all' ? ' active' : '')} onClick={() => setFilter('all')}>
              All 28 scenarios
            </button>
          </div>
          <div className="spacer" />
          <label className="small muted">Amounts</label>
          <select value={mode} onChange={(e) => setMode(e.target.value as 'engine' | 'sample')} style={{ width: 'auto' }}>
            <option value="engine">Computed by the tax engine</option>
            <option value="sample">FBR sample values verbatim</option>
          </select>
        </div>
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>Scenario</th>
                <th>Description</th>
                <th>Sale type / rate</th>
                <th>Status</th>
                <th>Last run</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {list.map((sc) => (
                <tr key={sc.id}>
                  <td className="nowrap">
                    <button className="btn-link mono" onClick={() => setDetail(sc)}>
                      <b>{sc.id}</b>
                    </button>
                    <div>
                      {sc.assigned && <span className="badge b-blue">assigned</span>} {sc.suggested && !sc.assigned && <span className="badge b-gray">suggested</span>}
                    </div>
                  </td>
                  <td>
                    {sc.title}
                    <div className="small faint">{sc.description}</div>
                  </td>
                  <td className="small">
                    {sc.item.saleType}
                    <br />
                    <b>{sc.item.rate}</b>
                  </td>
                  <td>
                    {sc.passed ? (
                      <span className="badge b-green">passed</span>
                    ) : sc.lastRun ? (
                      <span className={'badge ' + (sc.lastRun.status === 'practice' ? 'b-blue' : 'b-red')}>{sc.lastRun.status}</span>
                    ) : (
                      <span className="badge b-gray">not run</span>
                    )}
                  </td>
                  <td className="small" style={{ maxWidth: 360 }}>
                    {sc.lastRun && (
                      <>
                        <span className="faint">
                          {dateTimeFmt(sc.lastRun.runAt)} · {sc.lastRun.mode}
                        </span>
                        <div>
                          {sc.lastRun.invoiceId ? <Link to={`/invoices/${sc.lastRun.invoiceId}`}>{sc.lastRun.fbrInvoiceNumber || 'invoice'}</Link> : null} {sc.lastRun.message}
                        </div>
                      </>
                    )}
                  </td>
                  <td className="nowrap">
                    {canRun && (
                      <>
                        <button className="btn btn-sm" disabled={!!running || runAll} onClick={async () => (await run(sc.id, 'simulator')) && reload()}>
                          {running === sc.id + 'simulator' ? '…' : 'Practice'}
                        </button>{' '}
                        <button
                          className="btn btn-sm btn-primary"
                          disabled={!!running || runAll || !company.hasSandboxToken}
                          onClick={async () => (await run(sc.id, 'sandbox')) && reload()}
                        >
                          {running === sc.id + 'sandbox' ? 'Submitting…' : 'Run in sandbox'}
                        </button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {editAssigned && (
        <AssignModal
          data={data}
          onClose={() => setEditAssigned(false)}
          onSaved={(ov) => {
            setData(ov)
            setEditAssigned(false)
            toast('ok', 'Assigned scenarios saved')
          }}
        />
      )}
      {detail && <ScenarioDetail sc={detail} onClose={() => setDetail(null)} />}
    </>
  )
}

function AssignModal({ data, onClose, onSaved }: { data: ScenarioOverview; onClose: () => void; onSaved: (o: ScenarioOverview) => void }) {
  const cp = useCompanyPath()
  const [sel, setSel] = useState<string[]>(data.scenarios.filter((x) => x.assigned).map((x) => x.id))
  const [error, setError] = useState('')
  const toggle = (id: string) => setSel((x) => (x.includes(id) ? x.filter((y) => y !== id) : [...x, id]))
  const save = async () => {
    try {
      onSaved(await api.put<ScenarioOverview>(`${cp}/scenarios/assigned`, { scenarioIds: sel }))
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  return (
    <Modal
      wide
      title="Scenarios assigned on IRIS"
      onClose={onClose}
      footer={
        <>
          <button className="btn" onClick={() => setSel(data.suggested)}>
            Use suggested ({data.suggested.length})
          </button>
          <div className="spacer" />
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn btn-primary" onClick={save}>
            Save
          </button>
        </>
      }
    >
      <p className="muted">
        Tick exactly the scenarios IRIS lists for your registration (IRIS → Digital Invoicing → Sandbox / Scenarios). The suggestion is based on your business
        activity and sector.
      </p>
      {error && <div className="alert alert-error">{error}</div>}
      <div className="grid g2">
        {data.scenarios.map((sc) => (
          <label key={sc.id} className="check">
            <input type="checkbox" checked={sel.includes(sc.id)} onChange={() => toggle(sc.id)} />
            <span>
              <b className="mono">{sc.id}</b> {sc.title} {sc.suggested && <span className="badge b-gray">suggested</span>}
            </span>
          </label>
        ))}
      </div>
    </Modal>
  )
}

function ScenarioDetail({ sc, onClose }: { sc: ScenarioStatus; onClose: () => void }) {
  const it = sc.item
  return (
    <Modal title={`${sc.id} — ${sc.title}`} onClose={onClose} wide>
      <p>{sc.description}</p>
      <dl className="kv">
        <dt>Sample buyer</dt>
        <dd>
          {sc.buyerName} ({sc.buyerNTNCNIC || 'no NTN/CNIC'}) · {sc.buyerRegistrationType}
        </dd>
        <dt>HS code</dt>
        <dd className="mono">{it.hsCode}</dd>
        <dt>Description</dt>
        <dd>{it.productDescription}</dd>
        <dt>Sale type</dt>
        <dd>{it.saleType}</dd>
        <dt>Rate</dt>
        <dd>{it.rate}</dd>
        <dt>Quantity / UoM</dt>
        <dd>
          {qty(it.quantity)} {it.uoM}
        </dd>
        <dt>Value excl. ST</dt>
        <dd>{money(it.valueSalesExcludingST)}</dd>
        {it.sroScheduleNo && (
          <>
            <dt>SRO / Schedule</dt>
            <dd>
              {it.sroScheduleNo} · S.No {it.sroItemSerialNo}
            </dd>
          </>
        )}
      </dl>
      <p className="small muted mt">
        In the sandbox the invoice is sent with <span className="mono">scenarioId: "{sc.id}"</span> and your own NTN as seller. The scenario passes when FBR returns
        status 00 with an invoice number.
      </p>
    </Modal>
  )
}
