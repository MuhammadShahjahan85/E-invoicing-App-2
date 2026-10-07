// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { Link } from 'react-router-dom'
import { api } from '../api'
import { ErrorBox, Spinner, useLoad } from '../components/ui'
import { dateTimeFmt, money, statusLabels } from '../format'
import { useCompanyPath, useSession } from '../state'
import type { DashboardData } from '../types'

export default function Dashboard() {
  const { company } = useSession()
  const cp = useCompanyPath()
  const { data, error, loading } = useLoad(() => api.get<DashboardData>(`${cp}/dashboard`), [cp, company?.environment])
  if (loading && !data) return <Spinner />
  if (error) return <ErrorBox error={error} />
  if (!data || !company) return null
  const st = data.stats
  const counts = Object.fromEntries((st.statusCounts ?? []).map((s) => [s.status, s.count]))
  const sc = data.scenarios
  const conn = data.connection

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{company.name}</h1>
          <p>
            NTN/CNIC {company.ntnCnic} · {company.province}
          </p>
        </div>
        <div className="actions">
          <Link className="btn btn-primary" to="/invoices/new">
            + New invoice
          </Link>
          <Link className="btn" to="/import">
            Import
          </Link>
        </div>
      </div>

      {data.license.mode !== 'licensed' && data.license.mode !== 'developer' && <div className="alert alert-warn">{data.license.message}</div>}
      {data.tokenWarning && <div className="alert alert-warn">{data.tokenWarning}</div>}
      {!conn.healthy && (
        <div className="alert alert-error">
          <b>{conn.authFailure ? 'FBR rejected the security token.' : 'FBR Digital Invoicing is not reachable.'}</b> Failing since {dateTimeFmt(conn.failingSince)}.
          Invoices are queued and will be sent automatically once the connection is restored. {conn.lastError}
        </div>
      )}
      {st.pendingUpload > 0 && (
        <div className="alert alert-warn">
          <b>{st.pendingUpload} invoice(s) not yet reported to FBR</b> (oldest issued {dateTimeFmt(st.oldestPending)}). Invoices issued while FBR was unreachable must be uploaded
          within 24 hours of the connection being restored. Queued invoices are sent automatically as soon as FBR responds; any that FBR rejects must be corrected
          and resubmitted. <Link to="/invoices?status=REJECTED,UNCERTAIN,QUEUED">View them</Link>.
        </div>
      )}
      {data.unreportedIncidents > 0 && (
        <div className="alert alert-warn">
          {data.unreportedIncidents} incident(s) not yet reported to FBR. Rule 150R requires reporting operational failures within 24 hours —{' '}
          <Link to="/incidents">open the incident register</Link>.
        </div>
      )}

      <div className="grid g4">
        <div className="card stat">
          <div className="label">Invoices accepted today</div>
          <div className="value">{st.todayCount}</div>
          <div className="sub">
            Value {money(st.todayValue)} · Tax {money(st.todaySalesTax)}
          </div>
        </div>
        <div className="card stat">
          <div className="label">This month (tax period)</div>
          <div className="value">{money(st.monthValue)}</div>
          <div className="sub">
            {st.monthCount} invoices · Sales tax {money(st.monthSalesTax)}
          </div>
        </div>
        <div className="card stat">
          <div className="label">Needs attention</div>
          <div className="value" style={{ color: st.needsAttention ? 'var(--danger)' : undefined }}>
            {st.needsAttention}
          </div>
          <div className="sub">
            <Link to="/invoices?status=REJECTED,UNCERTAIN,QUEUED">Rejected, queued or uncertain</Link>
          </div>
        </div>
        <div className="card stat">
          <div className="label">FBR connection ({data.environment})</div>
          <div className="value" style={{ color: conn.healthy ? 'var(--success)' : 'var(--danger)' }}>
            {conn.healthy ? 'OK' : 'Down'}
          </div>
          <div className="sub">
            {data.environment === 'simulator'
              ? 'Built-in simulator — nothing is sent to FBR'
              : conn.lastSuccess
                ? 'Last success ' + dateTimeFmt(conn.lastSuccess)
                : conn.failingSince
                  ? 'Failing since ' + dateTimeFmt(conn.failingSince)
                  : 'No calls yet'}
          </div>
        </div>
      </div>

      <div className="grid g2 mt">
        <div className="card">
          <div className="card-head">
            <h3>Invoices by status</h3>
            <Link to="/invoices" className="small">
              View all
            </Link>
          </div>
          <div className="card-body">
            {(st.statusCounts ?? []).length === 0 ? (
              <p className="muted">No invoices yet in this environment.</p>
            ) : (
              <table className="table">
                <tbody>
                  {Object.entries(statusLabels).map(([k, label]) =>
                    counts[k] ? (
                      <tr key={k}>
                        <td>
                          <Link to={`/invoices?status=${k}`}>{label}</Link>
                        </td>
                        <td className="num">{counts[k]}</td>
                      </tr>
                    ) : null,
                  )}
                </tbody>
              </table>
            )}
          </div>
        </div>

        <div className="card">
          <div className="card-head">
            <h3>Go-live readiness</h3>
            <Link to="/scenarios" className="small">
              Scenarios
            </Link>
          </div>
          <div className="card-body">
            <ol className="steps">
              <li className={company.hasSandboxToken ? 'done' : ''}>Register for Digital Invoicing on IRIS, choose PRAL (or another licensed integrator) and save the sandbox token</li>
              <li className={sc.readyForProduction ? 'done' : ''}>
                Pass the assigned sandbox scenarios ({sc.passedCount}/{sc.assignedCount})
                <div className="progress mt" style={{ marginTop: 6 }}>
                  <div style={{ width: `${sc.assignedCount ? (100 * sc.passedCount) / sc.assignedCount : 0}%` }} />
                </div>
              </li>
              <li className={company.hasProductionToken ? 'done' : ''}>Save the production token issued on IRIS (whitelist this server's public IP)</li>
              <li className={company.environment === 'production' ? 'done' : ''}>Switch the company to the production environment</li>
            </ol>
          </div>
        </div>
      </div>

      {(st.topErrors ?? []).length > 0 && (
        <div className="card mt">
          <div className="card-head">
            <h3>Most frequent FBR errors</h3>
            <Link to="/help" className="small">
              Error code guide
            </Link>
          </div>
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Code</th>
                  <th>Message</th>
                  <th className="num">Count</th>
                </tr>
              </thead>
              <tbody>
                {st.topErrors!.map((e, i) => (
                  <tr key={i}>
                    <td className="mono">{e.code || '—'}</td>
                    <td>{e.message}</td>
                    <td className="num">{e.count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </>
  )
}
