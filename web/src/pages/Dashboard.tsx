// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { Link } from 'react-router-dom'
import {
  BadgeCheck,
  CalendarClock,
  ChartColumnBig,
  Check,
  CircleAlert,
  CircleCheck,
  Database,
  FilePlus2,
  ListChecks,
  Receipt,
  ShieldCheck,
  TriangleAlert,
  Upload,
  WifiOff,
} from 'lucide-react'
import { api } from '../api'
import { ErrorBox, StatusBadge, useLoad } from '../components/ui'
import { ColumnChart, DeadlineList, HBarList, StatusBreakdown } from '../components/Charts'
import { dateFmt, dateTimeFmt, envLabels, lakhCrore, money } from '../format'
import { useCompanyPath, useSession } from '../state'
import type { DashboardData, PeriodTie, TieEdge } from '../types'

const todayLong = () => new Date().toLocaleDateString('en-GB', { timeZone: 'Asia/Karachi', weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })

const thisMonth = () => new Date().toLocaleDateString('en-GB', { timeZone: 'Asia/Karachi', month: 'long', year: 'numeric' })

const monthLabel = (p: string, style: 'short' | 'long') => new Date(p + '-01T00:00:00').toLocaleDateString('en-GB', style === 'short' ? { month: 'short' } : { month: 'long', year: 'numeric' })

const plural = (n: number, one: string, many = one + 's') => `${n.toLocaleString('en-PK')} ${n === 1 ? one : many}`

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
  const tie = data.tie

  const readiness = (
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
          <li className={company.softwareRegNo ? 'done' : ''}>
            Record the software registration number and display the “Integrated with FBR” signboard (rule 150R(11)) —{' '}
            <Link to="/settings/fbr">FBR integration</Link>
          </li>
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
  )

  return (
    <>
      <div className="page-head">
        <div>
          <div className="eyebrow">
            {company.name} · {envLabels[data.environment]}
          </div>
          <h1>Sales tax overview</h1>
          <p>
            Assalam-o-Alaikum, {firstName}. {todayLong()} — NTN/CNIC {company.ntnCnic}
            {company.strn ? ` · STRN ${company.strn}` : ''} · {company.province}
          </p>
        </div>
        <div className="actions">
          <Link className="btn" to="/library?tab=buyer">
            <ShieldCheck size={16} /> Verify a buyer
          </Link>
          {st.needsAttention > 0 && (
            <Link className="btn" to="/invoices?status=REJECTED,UNCERTAIN,QUEUED">
              <CircleAlert size={16} /> Review {plural(st.needsAttention, 'item')}
            </Link>
          )}
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
            <b>{plural(st.pendingUpload, 'invoice')} not yet reported to FBR</b> (oldest issued {dateTimeFmt(st.oldestPending)}). Invoices issued while FBR was
            unreachable are marked as issued in the offline mode and must be uploaded within 24 hours of the connection being restored (rule 150XC). Queued
            invoices are sent automatically as soon as FBR responds; any that FBR rejects must be corrected and resubmitted.{' '}
            <Link to="/invoices?status=REJECTED,UNCERTAIN,QUEUED">View them</Link>.
          </div>
        </div>
      )}
      {data.unreportedIncidents > 0 && (
        <div className="alert alert-warn">
          <TriangleAlert size={18} />
          <div className="alert-body">
            {plural(data.unreportedIncidents, 'incident')} not yet reported to FBR. Rule 150XA(c) requires reporting operational failures within 24 hours —{' '}
            <Link to="/incidents">open the incident register</Link>.
          </div>
        </div>
      )}

      <div className="grid g-hero">
        <section className="hero-card" aria-label="Sales this month">
          <div className="hero-top">
            <LivePill data={data} />
            <span className="hero-label">Sales this month · {thisMonth()}</span>
          </div>
          <div className="hero-value" title={`About Rs ${lakhCrore(st.monthValue)}`}>
            {money(st.monthValue)}
            <span className="hero-unit">PKR</span>
          </div>
          <p className="hero-desc">
            {st.monthCount
              ? `${plural(st.monthCount, 'sale invoice')} accepted by FBR this month — about Rs ${lakhCrore(st.monthValue)} before sales tax. Annexure-C of the return is built from these invoices.`
              : sim
                ? 'No invoices yet this month. Practise freely: the training simulator never sends anything to FBR.'
                : 'No invoices accepted by FBR this month yet. Each invoice you issue is reported to FBR in real time.'}
          </p>
          <div className="hero-tiles">
            <div className="hero-tile">
              <span>Sales tax this month</span>
              <b>Rs {money(st.monthSalesTax)}</b>
              <small>on accepted sale invoices</small>
            </div>
            <div className="hero-tile">
              <span>Today</span>
              <b>Rs {money(st.todayValue)}</b>
              <small>{plural(st.todayCount, 'invoice')} accepted</small>
            </div>
            <div className="hero-tile">
              <span>Needs attention</span>
              <b className={st.needsAttention ? 'warn' : 'good'}>
                {st.needsAttention.toLocaleString('en-PK')} {st.needsAttention ? <CircleAlert size={18} /> : <CircleCheck size={18} />}
              </b>
              <small>{st.needsAttention ? 'rejected, queued or uncertain' : 'every invoice is with FBR'}</small>
            </div>
          </div>
          {tie && <CloseSteps tie={tie} />}
        </section>
        {tie ? <TieCard tie={tie} /> : readiness}
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

      <div className={'grid mt ' + (tie ? 'g2' : 'g-2-1')}>
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
        {tie && readiness}
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

/** LivePill shows whether invoices are going to FBR right now. */
function LivePill({ data }: { data: DashboardData }) {
  if (data.environment === 'simulator')
    return (
      <span className="live-pill sim">
        <i /> Simulator
      </span>
    )
  if (!data.connection.healthy)
    return (
      <span className="live-pill err">
        <i /> FBR offline
      </span>
    )
  if (data.environment === 'sandbox')
    return (
      <span className="live-pill warn">
        <i /> Sandbox
      </span>
    )
  return (
    <span className="live-pill">
      <i /> Live
    </span>
  )
}

const edge = (tie: PeriodTie, id: TieEdge['id']) => tie.edges.find((e) => e.id === id)

/** CloseSteps walks through closing the tax period whose return is due next. */
function CloseSteps({ tie }: { tie: PeriodTie }) {
  const unsent = edge(tie, 'books-fbr')?.count ?? 0
  const refused = edge(tie, 'fbr-annexc')?.count ?? 0
  const month = tie.periodLabel.split(' ')[0]
  const due = new Date(tie.filingDue + 'T00:00:00').toLocaleDateString('en-GB', { day: 'numeric', month: 'short' })
  const steps: { state: 'done' | 'warn' | 'todo'; title: string; sub: string; to: string }[] = [
    {
      state: tie.issued > 0 ? 'done' : 'todo',
      title: 'Invoices issued',
      sub: tie.issued > 0 ? `${plural(tie.issued, 'document')} in ${month}` : `None dated in ${month}`,
      to: `/invoices?from=${tie.period}-01`,
    },
    {
      state: unsent + refused ? 'warn' : 'done',
      title: 'Reported to FBR',
      sub: unsent + refused ? `${unsent + refused} to resolve` : tie.issued ? `${plural(tie.accepted, 'document')} accepted` : 'Nothing to report yet',
      to: (unsent ? edge(tie, 'books-fbr') : edge(tie, 'fbr-annexc'))?.link ?? '/invoices',
    },
    {
      state: tie.checksOpen ? 'warn' : 'done',
      title: 'Period checks',
      sub: tie.checksOpen ? `${tie.checksOpen} of ${tie.checksTotal} to review` : `All ${tie.checksTotal} passed`,
      to: `/compliance?period=${tie.period}`,
    },
    {
      state: 'todo',
      title: 'File the return',
      sub: tie.daysLeft < 0 ? `Was due ${due}` : tie.daysLeft === 0 ? 'Due today' : `Due ${due} · ${plural(tie.daysLeft, 'day')} to go`,
      to: `/compliance?period=${tie.period}`,
    },
  ]
  return (
    <>
      <div className="hero-steps-label">Closing {tie.periodLabel}</div>
      <div className="hero-steps">
        {steps.map((st, i) => (
          <Link key={st.title} to={st.to} className={'hstep' + (st.state === 'todo' ? '' : ' ' + st.state)}>
            <span className="hs-ic">{st.state === 'done' ? <Check size={14} strokeWidth={3} /> : i + 1}</span>
            <span style={{ minWidth: 0 }}>
              <b>{st.title}</b>
              <small>{st.sub}</small>
            </span>
          </Link>
        ))}
      </div>
    </>
  )
}

type Pt = { x: number; y: number }

/** TieCard shows whether the books, FBR's records and Annexure-C of the return agree. */
function TieCard({ tie }: { tie: PeriodTie }) {
  const open = tie.edges.filter((e) => e.count > 0)
  const ok = (id: TieEdge['id']) => (edge(tie, id)?.count ?? 0) === 0
  const nodes: Record<'books' | 'fbr' | 'annexc', Pt & { label: string }> = {
    books: { x: 160, y: 44, label: 'BOOKS' },
    fbr: { x: 52, y: 206, label: 'FBR' },
    annexc: { x: 268, y: 206, label: 'ANNEX-C' },
  }
  const lines: { id: TieEdge['id']; a: Pt; b: Pt }[] = [
    { id: 'books-fbr', a: nodes.books, b: nodes.fbr },
    { id: 'books-annexc', a: nodes.books, b: nodes.annexc },
    { id: 'fbr-annexc', a: nodes.fbr, b: nodes.annexc },
  ]
  return (
    <div className="card tie-card">
      <div className="between" style={{ alignItems: 'flex-start' }}>
        <div>
          <h3>Books, FBR and Annexure-C tie</h3>
          <p className="tie-sub">
            {tie.periodLabel} return · due {dateFmt(tie.filingDue)}
          </p>
        </div>
        <span className={'badge ' + (open.length ? 'b-amber' : 'b-green')}>
          {open.length ? (
            <>
              <CircleAlert /> {plural(open.length, 'difference')}
            </>
          ) : (
            <>
              <CircleCheck /> All tie
            </>
          )}
        </span>
      </div>
      <svg className="tie-diagram" viewBox="0 0 320 252" role="img" aria-label={open.length ? `${open.length} of 3 sides do not tie` : 'All three sides tie'}>
        {lines.map((l) => (
          <line key={l.id} className={'edge ' + (ok(l.id) ? 'ok' : 'warn')} x1={l.a.x} y1={l.a.y} x2={l.b.x} y2={l.b.y} />
        ))}
        {Object.entries(nodes).map(([k, n]) => (
          <g key={k} className="node">
            <circle cx={n.x} cy={n.y} r={36} />
            <text x={n.x} y={n.y} style={n.label.length > 5 ? { fontSize: 11 } : undefined}>
              {n.label}
            </text>
          </g>
        ))}
        {lines.map((l) => {
          const x = (l.a.x + l.b.x) / 2
          const y = (l.a.y + l.b.y) / 2
          const good = ok(l.id)
          return (
            <g key={l.id} className={'mark ' + (good ? 'ok' : 'warn')}>
              <circle cx={x} cy={y} r={13} />
              {good ? <path d={`M${x - 5.5} ${y + 0.5} l3.8 3.8 l7 -7.6`} /> : <path d={`M${x} ${y - 6} v7 M${x} ${y + 5} v0.6`} />}
            </g>
          )
        })}
      </svg>
      <ul className="tie-list">
        {tie.edges.map((e) => (
          <li key={e.id}>
            <span className={'tl-ic ' + (e.count ? 'warn' : 'ok')}>{e.count ? <CircleAlert size={15} /> : <Check size={15} strokeWidth={3} />}</span>
            <span className="tl-main">
              <b>{e.title}</b>
              <small>{e.count ? <Link to={e.link}>{plural(e.count, 'document')} to review</Link> : e.detail}</small>
            </span>
            <span className="tl-val">{money(e.value)}</span>
          </li>
        ))}
      </ul>
    </div>
  )
}
