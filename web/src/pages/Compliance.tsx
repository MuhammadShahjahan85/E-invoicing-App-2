// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Tax periods & returns: due dates of the monthly sales tax return and a
// period-close review (what was reported to FBR, what was not, incidents).

import { Link } from 'react-router-dom'
import { useState } from 'react'
import { CalendarClock, CircleCheck, ClipboardList, ExternalLink, FileSpreadsheet, TriangleAlert } from 'lucide-react'
import { api, qs } from '../api'
import { ErrorBox, Spinner, useLoad } from '../components/ui'
import { DeadlineList, HBarList } from '../components/Charts'
import { dateFmt, envLabels, money } from '../format'
import { useCompanyPath, useSession } from '../state'
import type { PeriodReview } from '../types'

function monthOptions(): { value: string; label: string }[] {
  const out: { value: string; label: string }[] = []
  const now = new Date()
  for (let i = 0; i < 18; i++) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1)
    out.push({
      value: `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`,
      label: d.toLocaleDateString('en-GB', { month: 'long', year: 'numeric' }),
    })
  }
  return out
}

export default function Compliance() {
  const s = useSession()
  const cp = useCompanyPath()
  const [period, setPeriod] = useState('')
  const { data, error, loading } = useLoad(() => api.get<PeriodReview>(`${cp}/compliance${qs({ period })}`), [cp, period, s.company?.environment])
  const months = monthOptions()

  return (
    <>
      <div className="page-head">
        <div>
          <div className="eyebrow">
            <CalendarClock size={14} /> Sales tax return
          </div>
          <h1>Tax periods & returns</h1>
          <p>
            Invoices reported through Digital Invoicing populate Annexure-C of your monthly sales tax return on IRIS. Review each period before filing so that
            every supply is reported and the figures agree.
          </p>
        </div>
        <div className="actions">
          <select value={period || data?.period || ''} onChange={(e) => setPeriod(e.target.value)} style={{ width: 'auto' }} aria-label="Tax period">
            {months.map((m) => (
              <option key={m.value} value={m.value}>
                {m.label}
              </option>
            ))}
          </select>
        </div>
      </div>
      {loading && !data ? (
        <Spinner />
      ) : error ? (
        <ErrorBox error={error} />
      ) : data ? (
        <PeriodView r={data} />
      ) : null}
    </>
  )
}

function PeriodView({ r }: { r: PeriodReview }) {
  const s = useSession()
  const open = r.checks.filter((c) => !c.ok && c.id !== 'simulator').length
  const sm = r.summary
  // Same definition as the monthly report: debit notes are netted off.
  const net = Number(sm.salesTax) - Number(sm.debitNoteSalesTax)
  return (
    <>
      {r.environment !== 'production' && (
        <div className="alert alert-info">
          <TriangleAlert size={18} />
          <div className="alert-body">
            Showing <b>{envLabels[r.environment]}</b> figures. Only invoices issued in production are reported to FBR and appear in your return.
          </div>
        </div>
      )}
      <div className="grid g4">
        <div className="card stat hero">
          <div className="stat-top">
            <span className="label">Value of supplies · {r.periodLabel}</span>
          </div>
          <div className="value">
            <small>Rs</small>
            {money(sm.valueExclST)}
          </div>
          <div className="sub">
            {sm.saleInvoices} sale invoice{sm.saleInvoices === 1 ? '' : 's'} accepted by FBR
          </div>
        </div>
        <div className="card stat">
          <div className="label">Sales tax charged</div>
          <div className="value">
            <small>Rs</small>
            {money(sm.salesTax)}
          </div>
          <div className="sub">Further tax Rs {money(sm.furtherTax)}</div>
        </div>
        <div className="card stat">
          <div className="label">Debit notes</div>
          <div className="value">
            <small>Rs</small>
            {money(sm.debitNoteValue)}
          </div>
          <div className="sub">
            {sm.debitNotes} note{sm.debitNotes === 1 ? '' : 's'} · tax Rs {money(sm.debitNoteSalesTax)}
          </div>
        </div>
        <div className="card stat">
          <div className="label">Net sales tax (after debit notes)</div>
          <div className="value">
            <small>Rs</small>
            {money(net)}
          </div>
          <div className="sub">Withheld by buyers Rs {money(sm.stWithheld)}</div>
        </div>
      </div>

      <div className="grid g-2-1 mt">
        <div className="card">
          <div className="card-head">
            <h3>
              <ClipboardList size={17} /> Period-close checklist
            </h3>
            {r.environment !== 'production' ? (
              <span className="badge b-purple">Practice figures</span>
            ) : open === 0 ? (
              <span className="badge b-green">Ready to file</span>
            ) : (
              <span className="badge b-amber">{open} to review</span>
            )}
          </div>
          {r.checks.map((c) => (
            <div key={c.id} className="check-row">
              <span className={'ck ' + (c.ok ? 'ok' : 'no')}>{c.ok ? <CircleCheck size={16} /> : <TriangleAlert size={15} />}</span>
              <div style={{ flex: 1, minWidth: 0 }}>
                <div className="ck-title">{c.title}</div>
                {!c.ok && <div className="ck-detail">{c.detail}</div>}
              </div>
              {!c.ok && c.link && (
                <Link to={c.link} className="btn btn-sm">
                  Review
                </Link>
              )}
            </div>
          ))}
        </div>
        <div className="card">
          <div className="card-head">
            <h3>
              <CalendarClock size={17} /> Due dates
            </h3>
            <Link to="/settings/company" className="small">
              Change
            </Link>
          </div>
          <div className="card-body" style={{ paddingTop: 4, paddingBottom: 4 }}>
            <DeadlineList deadlines={r.deadlines} />
          </div>
          <div className="card-foot">
            Generally tax is paid by the {ordinal(s.company?.returnPaymentDay ?? 15)} and the return filed by the {ordinal(s.company?.returnFilingDay ?? 18)} of
            the following month. FBR sometimes extends these dates — check its announcements.
          </div>
        </div>
      </div>

      <div className="card mt">
        <div className="card-head">
          <h3>
            <FileSpreadsheet size={17} /> Supplies by sale type and rate
          </h3>
          <Link to={`/reports`} className="small">
            Sales register & exports <ExternalLink size={12} />
          </Link>
        </div>
        {(r.bySaleType ?? []).length === 0 ? (
          <div className="chart-empty">No accepted invoices in {r.periodLabel}.</div>
        ) : (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Document</th>
                  <th>Sale type</th>
                  <th>Rate</th>
                  <th className="num">Invoices</th>
                  <th className="num">Value excl. ST</th>
                  <th className="num">Sales tax</th>
                  <th className="num">Further tax</th>
                  <th className="num">Extra tax</th>
                  <th className="num">FED</th>
                  <th className="num">ST withheld</th>
                </tr>
              </thead>
              <tbody>
                {(r.bySaleType ?? []).map((x, i) => (
                  <tr key={i}>
                    <td>{x.docType}</td>
                    <td>{x.saleType}</td>
                    <td>{x.rate}</td>
                    <td className="num">{x.invoices}</td>
                    <td className="num">{money(x.valueExclST)}</td>
                    <td className="num">{money(x.salesTax)}</td>
                    <td className="num">{money(x.furtherTax)}</td>
                    <td className="num">{money(x.extraTax)}</td>
                    <td className="num">{money(x.fed)}</td>
                    <td className="num">{money(x.stWithheld)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div className="grid g2 mt">
        <div className="card">
          <div className="card-head">
            <h3>Largest buyers</h3>
            <span className="small faint">Value excl. sales tax</span>
          </div>
          <div className="card-body">
            <HBarList
              rows={(r.topBuyers ?? []).map((b) => ({ label: b.buyerName || 'Walk-in / unnamed', sub: b.buyerNtnCnic || 'Unregistered', value: Number(b.valueExclST) }))}
              empty="No sales in this period."
            />
          </div>
        </div>
        <div className="card">
          <div className="card-head">
            <h3>Largest HS codes</h3>
            <span className="small faint">Value excl. sales tax</span>
          </div>
          <div className="card-body">
            <HBarList rows={(r.topItems ?? []).map((x) => ({ label: x.hsCode, sub: x.description, value: Number(x.valueExclST) }))} empty="No sales in this period." />
          </div>
        </div>
      </div>

      {(r.incidents ?? []).length > 0 && (
        <div className="card mt">
          <div className="card-head">
            <h3>Incidents in this period</h3>
            <Link to="/incidents" className="small">
              Incident register
            </Link>
          </div>
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Started</th>
                  <th>Ended</th>
                  <th>What happened</th>
                  <th>Reported to FBR</th>
                </tr>
              </thead>
              <tbody>
                {(r.incidents ?? []).map((i) => (
                  <tr key={i.id}>
                    <td className="nowrap">{dateFmt(i.startedAt)}</td>
                    <td className="nowrap">{i.endedAt ? dateFmt(i.endedAt) : <span className="badge b-amber">Open</span>}</td>
                    <td>{i.description}</td>
                    <td>{i.reportedAt ? <span className="badge b-green">{i.reportReference || dateFmt(i.reportedAt)}</span> : <span className="badge b-red">Not reported</span>}</td>
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

function ordinal(n: number) {
  const s = ['th', 'st', 'nd', 'rd']
  const v = n % 100
  return n + (s[(v - 20) % 10] || s[v] || s[0])
}
