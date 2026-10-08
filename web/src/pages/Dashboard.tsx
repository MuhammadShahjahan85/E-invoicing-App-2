// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { Link } from 'react-router-dom'
import {
  BadgeCheck,
  CalendarClock,
  ChartColumnBig,
  CircleAlert,
  CircleCheck,
  Database,
  FilePlus2,
  GraduationCap,
  ListChecks,
  Receipt,
  ShieldCheck,
  TriangleAlert,
  Upload,
  Wifi,
  WifiOff,
} from 'lucide-react'
import { api } from '../api'
import { ErrorBox, StatusBadge, useLoad } from '../components/ui'
import { ColumnChart, DeadlineList, HBarList, Sparkline, StatusBreakdown } from '../components/Charts'
import { dateFmt, dateTimeFmt, money } from '../format'
import { useCompanyPath, useSession } from '../state'
import type { DashboardData } from '../types'

function greeting() {
  const h = Number(new Intl.DateTimeFormat('en-GB', { timeZone: 'Asia/Karachi', hour: 'numeric', hour12: false }).format(new Date()))
  return h < 12 ? 'Good morning' : h < 17 ? 'Good afternoon' : 'Good evening'
}

const todayLong = () => new Date().toLocaleDateString('en-GB', { timeZone: 'Asia/Karachi', weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })

const monthLabel = (p: string, style: 'short' | 'long') => new Date(p + '-01T00:00:00').toLocaleDateString('en-GB', style === 'short' ? { month: 'short' } : { month: 'long', year: 'numeric' })

export default function Dashboard() {
  const s = useSession()
  const { company } = s
  const cp = useCompanyPath()
  const { data, error, loading } = useLoad(() => api.get<DashboardData>(`${cp}/dashboard`), [cp, company?.environment])
  if (loading && !data)
    return (
      <div className="page-loading">
        <span className="spinner" />
      </div>
    )
  if (error) return <ErrorBox error={error} />
  if (!data || !company) return null
  const st = data.stats
  const counts = Object.fromEntries((st.statusCounts ?? []).map((x) => [x.status, x.count])) as Record<string, number>
  const sc = data.scenarios
  const conn = data.connection
  const firstName = (s.user.fullName || s.user.username).split(' ')[0]
  const trend = data.trend ?? []
  const sim = data.environment === 'simulator'

  return (
    <>
      <div className="page-head">
        <div>
          <div className="eyebrow">{todayLong()}</div>
          <h1>
            {greeting()}, {firstName}
          </h1>
          <p>
            {company.name} · NTN/CNIC {company.ntnCnic}
            {company.strn ? ` · STRN ${company.strn}` : ''} · {company.province}
          </p>
        </div>
        <div className="actions">
          <Link className="btn" to="/library?tab=buyer">
            <ShieldCheck size={16} /> Verify a buyer
          </Link>
          {s.can('invoice.write') && (
            <>
              <Link className="btn hide-mobile" to="/import">
                <Upload size={16} /> Import
              </Link>
              <Link className="btn btn-primary" to="/invoices/new">
                <FilePlus2 size={16} /> New invoice
              </Link>
            </>
          )}
        </div>
      </div>

      {data.license.mode !== 'licensed' && data.license.mode !== 'developer' && (
        <div className="alert alert-warn">
          <TriangleAlert size={18} />
          <div className="alert-body">{data.license.message}</div>
        </div>
      )}
      {data.tokenWarning && (
        <div className="alert alert-warn">
          <TriangleAlert size={18} />
          <div className="alert-body">{data.tokenWarning}</div>
        </div>
      )}
      {!conn.healthy && (
        <div className="alert alert-error">
          <WifiOff size={18} />
          <div className="alert-body">
            <b>{conn.authFailure ? 'FBR rejected the security token.' : 'FBR Digital Invoicing is not reachable.'}</b> Failing since{' '}
            {dateTimeFmt(conn.failingSince)}. Invoices are queued and will be sent automatically once the connection is restored. {conn.lastError}
          </div>
        </div>
      )}
      {st.pendingUpload > 0 && (
        <div className="alert alert-warn">
          <CircleAlert size={18} />
          <div className="alert-body">
            <b>{st.pendingUpload} invoice(s) not yet reported to FBR</b> (oldest issued {dateTimeFmt(st.oldestPending)}). Invoices issued while FBR was
            unreachable must be uploaded within 24 hours of the connection being restored. Queued invoices are sent automatically as soon as FBR responds; any
            that FBR rejects must be corrected and resubmitted. <Link to="/invoices?status=REJECTED,UNCERTAIN,QUEUED">View them</Link>.
          </div>
        </div>
      )}
      {data.unreportedIncidents > 0 && (
        <div className="alert alert-warn">
          <TriangleAlert size={18} />
          <div className="alert-body">
            {data.unreportedIncidents} incident(s) not yet reported to FBR. Rule 150R requires reporting operational failures within 24 hours —{' '}
            <Link to="/incidents">open the incident register</Link>.
          </div>
        </div>
      )}

      <div className="grid g4">
        <div className="card stat hero">
          <div className="stat-top">
            <span className="label">Sales this month</span>
            <span className="stat-icon">
              <Receipt size={18} />
            </span>
          </div>
          <div className="value">
            <small>Rs</small>
            {money(st.monthValue)}
          </div>
          <div className="sub">
            {st.monthCount} invoice{st.monthCount === 1 ? '' : 's'} accepted · value excl. sales tax
          </div>
          {trend.length > 1 && (
            <div style={{ color: 'rgba(255,255,255,.9)' }}>
              <Sparkline values={trend.map((t) => Number(t.valueExclST))} />
            </div>
          )}
        </div>
        <div className="card stat">
          <div className="stat-top">
            <span className="label">Sales tax this month</span>
            <span className="stat-icon gold">
              <ChartColumnBig size={18} />
            </span>
          </div>
          <div className="value">
            <small>Rs</small>
            {money(st.monthSalesTax)}
          </div>
          <div className="sub">
            Today: {st.todayCount} invoice{st.todayCount === 1 ? '' : 's'} · Rs {money(st.todayValue)}
          </div>
        </div>
        <div className="card stat">
          <div className="stat-top">
            <span className="label">Needs attention</span>
            <span className={'stat-icon ' + (st.needsAttention ? 'danger' : 'ok')}>{st.needsAttention ? <CircleAlert size={18} /> : <CircleCheck size={18} />}</span>
          </div>
          <div className="value">{st.needsAttention}</div>
          <div className="sub">
            {st.needsAttention ? <Link to="/invoices?status=REJECTED,UNCERTAIN,QUEUED">Rejected, queued or uncertain — review</Link> : 'Every issued invoice is with FBR'}
          </div>
        </div>
        <div className="card stat">
          <div className="stat-top">
            <span className="label">FBR connection</span>
            <span className={'stat-icon ' + (sim ? 'info' : conn.healthy ? 'ok' : 'danger')}>
              {sim ? <GraduationCap size={18} /> : conn.healthy ? <Wifi size={18} /> : <WifiOff size={18} />}
            </span>
          </div>
          <div className="value" style={{ fontSize: 22 }}>
            {sim ? (
              <span className="state-line">Simulator</span>
            ) : conn.healthy ? (
              <span className="state-line state-ok">
                <CircleCheck size={20} /> Connected
              </span>
            ) : (
              <span className="state-line state-bad">
                <CircleAlert size={20} /> Not reachable
              </span>
            )}
          </div>
          <div className="sub">
            {sim
              ? 'Built-in trainer — nothing is sent to FBR'
              : conn.lastSuccess
                ? 'Last response ' + dateTimeFmt(conn.lastSuccess)
                : conn.failingSince
                  ? 'Failing since ' + dateTimeFmt(conn.failingSince)
                  : `${data.environment === 'production' ? 'Production' : 'Sandbox'} · no calls yet today`}
          </div>
        </div>
      </div>

      <div className="grid g-2-1 mt">
        {trend.length > 0 ? (
          <div className="card">
            <div className="card-head">
              <h3>
                <ChartColumnBig size={17} /> Sales by month
              </h3>
              {s.can('reports') && (
                <Link to="/reports" className="small">
                  Reports
                </Link>
              )}
            </div>
            <div className="card-body">
              <ColumnChart
                caption="Value of sale invoices accepted by FBR, excluding sales tax — last 12 months"
                valueLabel="Value excl. ST"
                data={trend.map((t) => ({
                  key: t.period,
                  label: monthLabel(t.period, 'short'),
                  title: monthLabel(t.period, 'long'),
                  value: Number(t.valueExclST),
                  rows: [
                    { label: 'Sales tax', value: money(t.salesTax) },
                    { label: 'Invoices', value: String(t.saleInvoices) },
                  ],
                }))}
              />
            </div>
          </div>
        ) : (
          <div className="card">
            <div className="card-head">
              <h3>
                <ListChecks size={17} /> Invoices by status
              </h3>
              <Link to="/invoices" className="small">
                View all
              </Link>
            </div>
            <div className="card-body">
              <StatusBreakdown counts={counts} />
            </div>
          </div>
        )}
        <div className="card">
          <div className="card-head">
            <h3>
              <CalendarClock size={17} /> Sales tax return
            </h3>
            {s.can('reports') && (
              <Link to="/compliance" className="small">
                Review period
              </Link>
            )}
          </div>
          <div className="card-body" style={{ paddingTop: 4, paddingBottom: 4 }}>
            <DeadlineList deadlines={data.deadlines ?? []} />
          </div>
          <div className="card-foot">Invoices reported through Digital Invoicing populate Annexure-C of the return on IRIS.</div>
        </div>
      </div>

      {trend.length > 0 && (
        <div className="grid g3 mt">
          <div className="card">
            <div className="card-head">
              <h3>
                <ListChecks size={17} /> Invoices by status
              </h3>
              <Link to="/invoices" className="small">
                View all
              </Link>
            </div>
            <div className="card-body">
              <StatusBreakdown counts={counts} />
            </div>
          </div>
          <div className="card">
            <div className="card-head">
              <h3>Top buyers this month</h3>
            </div>
            <div className="card-body">
              <HBarList
                rows={(data.topBuyers ?? []).map((b) => ({
                  label: b.buyerName || 'Walk-in buyer',
                  sub: b.buyerRegistrationType === 'Registered' ? b.buyerNtnCnic : 'Unregistered',
                  value: Number(b.valueExclST),
                }))}
                empty="No accepted sales this month yet."
              />
            </div>
          </div>
          <div className="card">
            <div className="card-head">
              <h3>Top HS codes this month</h3>
            </div>
            <div className="card-body">
              <HBarList
                rows={(data.topItems ?? []).map((x) => ({ label: x.hsCode, sub: x.description, value: Number(x.valueExclST), to: `/library?q=${encodeURIComponent(x.hsCode)}` }))}
                empty="No accepted sales this month yet."
              />
            </div>
          </div>
        </div>
      )}

      <div className="grid g2 mt">
        <div className="card">
          <div className="card-head">
            <h3>
              <Receipt size={17} /> Recent invoices
            </h3>
            <Link to="/invoices" className="small">
              View all
            </Link>
          </div>
          {(data.recent ?? []).length === 0 ? (
            <div className="chart-empty">
              No invoices yet. <Link to="/invoices/new">Create the first one</Link>.
            </div>
          ) : (
            <ul className="list-plain">
              {(data.recent ?? []).map((i) => (
                <li key={i.id}>
                  <Link to={`/invoices/${i.id}`} className="list-row">
                    <span style={{ flex: 1, minWidth: 0 }}>
                      <div className="truncate" style={{ fontWeight: 600 }}>
                        {i.internalNo} · {i.buyerName || 'Walk-in buyer'}
                      </div>
                      <div className="small faint truncate">
                        {dateFmt(i.invoiceDate)}
                        {i.fbrInvoiceNumber ? ' · ' + i.fbrInvoiceNumber : ''}
                      </div>
                    </span>
                    <span className="right">
                      <div className="tabular" style={{ fontWeight: 650 }}>
                        {money(i.totals?.totalValue)}
                      </div>
                      <StatusBadge status={i.status} />
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </div>

        <div className="card">
          <div className="card-head">
            <h3>
              <BadgeCheck size={17} /> Go-live readiness
            </h3>
            <Link to="/scenarios" className="small">
              Scenarios
            </Link>
          </div>
          <div className="card-body">
            <ol className="steps">
              <li className={company.hasSandboxToken ? 'done' : ''}>Register for Digital Invoicing on IRIS, choose PRAL (or another licensed integrator) and save the sandbox token</li>
              <li className={sc.readyForProduction ? 'done' : ''}>
                Pass the assigned sandbox scenarios ({sc.passedCount}/{sc.assignedCount})
                <div className="progress" style={{ marginTop: 8 }}>
                  <div style={{ width: `${sc.assignedCount ? (100 * sc.passedCount) / sc.assignedCount : 0}%` }} />
                </div>
              </li>
              <li className={company.hasProductionToken ? 'done' : ''}>Save the production token issued on IRIS (whitelist this server's public IP)</li>
              <li className={company.environment === 'production' ? 'done' : ''}>Switch the company to the production environment</li>
            </ol>
          </div>
          {data.reference && (
            <div className="card-foot row" style={{ gap: 8 }}>
              <Database size={14} />
              <span>
                FBR reference data:{' '}
                {data.reference.lastSync ? <>downloaded {dateTimeFmt(data.reference.lastSync)}</> : sim ? 'built-in simulator lists' : 'not downloaded yet'} ·{' '}
                {data.reference.hsCodes.toLocaleString()} HS codes
              </span>
              <Link to="/library?tab=sync" style={{ marginLeft: 'auto' }}>
                Manage
              </Link>
            </div>
          )}
        </div>
      </div>

      {(st.topErrors ?? []).length > 0 && (
        <div className="card mt">
          <div className="card-head">
            <h3>
              <TriangleAlert size={17} /> Most frequent FBR errors
            </h3>
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
