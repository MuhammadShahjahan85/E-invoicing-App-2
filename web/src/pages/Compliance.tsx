// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Tax periods & returns: due dates of the monthly sales tax return and a
// period-close review (what was reported to FBR, what was not, incidents).

import { Link } from 'react-router-dom'
import { useState } from 'react'
import { CalendarCheck2, CalendarClock, CircleCheck, ClipboardList, Download, ExternalLink, FileSpreadsheet, Link2, TriangleAlert } from 'lucide-react'
import { api, errorMessage, qs } from '../api'
import { Empty, ErrorBox, Field, Modal, Spinner, useLoad } from '../components/ui'
import { DeadlineList, HBarList } from '../components/Charts'
import { dateFmt, dateTimeFmt, envLabels, money } from '../format'
import { useCompanyPath, useSession, useToast } from '../state'
import type { Closing, PeriodReview } from '../types'

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
  const { data, error, loading, reload } = useLoad(() => api.get<PeriodReview>(`${cp}/compliance${qs({ period })}`), [cp, period, s.company?.environment])
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
        <PeriodView r={data} onChanged={reload} />
      ) : null}
    </>
  )
}

function PeriodView({ r, onChanged }: { r: PeriodReview; onChanged: () => void }) {
  const s = useSession()
  const cp = useCompanyPath()
  const [extending, setExtending] = useState(false)
  const filing = r.deadlines.find((d) => d.kind === 'filing')
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
            the following month. When FBR extends the filing date, record its order here; the payment date stays unchanged.
            {s.can('company.write') && (
              <div className="mt">
                <button className="btn btn-sm" onClick={() => setExtending(true)}>
                  {filing?.originalDue ? 'Change or remove the extension' : 'Record an FBR extension'}
                </button>
              </div>
            )}
          </div>
          {extending && (
            <ExtensionForm
              r={r}
              onClose={() => setExtending(false)}
              onSaved={() => {
                setExtending(false)
                onChanged()
              }}
            />
          )}
        </div>
      </div>

      <div className="card mt">
        <div className="card-head">
          <h3>
            <FileSpreadsheet size={17} /> Supplies by sale type and rate
          </h3>
          <div className="row" style={{ gap: 12 }}>
            <a
              className="btn btn-sm"
              href={`/api/v1${cp}/reports/annex-c${qs({ from: r.from, to: r.to, env: r.environment, format: 'xlsx' })}`}
              title="Every document with an FBR invoice number in the period, to match with Annex-C before filing"
            >
              <Download size={14} /> Annex-C reconciliation
            </a>
            <Link to={`/reports`} className="small">
              Sales register & exports <ExternalLink size={12} />
            </Link>
          </div>
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

      <Closings />

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

function ExtensionForm({ r, onClose, onSaved }: { r: PeriodReview; onClose: () => void; onSaved: () => void }) {
  const cp = useCompanyPath()
  const toast = useToast()
  const filing = r.deadlines.find((d) => d.kind === 'filing')
  const [date, setDate] = useState(filing?.originalDue ? filing.due : '')
  const [reference, setReference] = useState(filing?.reference ?? '')
  const [err, setErr] = useState('')
  const [busy, setBusy] = useState(false)
  const run = async (fn: () => Promise<unknown>, msg: string) => {
    setBusy(true)
    setErr('')
    try {
      await fn()
      toast('ok', msg)
      onSaved()
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Modal
      title={`Filing date extension · ${r.periodLabel}`}
      onClose={onClose}
      footer={
        <>
          {filing?.originalDue && (
            <button
              className="btn btn-ghost"
              style={{ marginRight: 'auto' }}
              disabled={busy}
              onClick={() => run(() => api.del(`${cp}/return-extensions/${r.period}`), 'Extension removed')}
            >
              Remove extension
            </button>
          )}
          <button className="btn" onClick={onClose}>
            Close
          </button>
          <button
            className="btn btn-primary"
            disabled={busy || !date || !reference.trim()}
            onClick={() => run(() => api.put(`${cp}/return-extensions/${r.period}`, { filingDate: date, reference }), 'Extension recorded')}
          >
            Save
          </button>
        </>
      }
    >
      <p className="small muted">
        FBR extends the date for filing the monthly return by notification or circular, usually on condition that the tax is paid by the normal due date. The
        reminders and the period-close checklist will use the extended filing date.
      </p>
      <Field label="Extended filing date">
        <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
      </Field>
      <Field label="FBR reference" hint="Number and date of FBR's notification, circular or order">
        <input value={reference} onChange={(e) => setReference(e.target.value)} placeholder="e.g. C.No.3(4)ST-L&P/2026, dated 16-10-2026" />
      </Field>
      {err && <div className="alert alert-error">{err}</div>}
    </Modal>
  )
}

interface ClosingList {
  closings: Closing[]
  checked: number
  chainOk: boolean
  brokenAt?: number
}

const kindLabels: Record<Closing['kind'], string> = { day: 'Daily', week: 'Weekly', month: 'Monthly' }

function closingPeriod(c: Closing): string {
  if (c.kind === 'day') return dateFmt(c.periodStart)
  if (c.kind === 'month') return new Date(c.periodStart + 'T00:00:00').toLocaleDateString('en-GB', { month: 'long', year: 'numeric' })
  return `${c.periodKey.replace('-W', ' week ')} · ${dateFmt(c.periodStart)} – ${dateFmt(c.periodEnd)}`
}

// Closings: the day, week and month closings the system records under
// rule 150R(4)(f), with the state of their hash chain.
function Closings() {
  const s = useSession()
  const cp = useCompanyPath()
  const [kind, setKind] = useState<Closing['kind']>('day')
  const { data, error, loading } = useLoad(
    () => api.get<ClosingList>(`${cp}/closings${qs({ kind, limit: kind === 'day' ? 31 : kind === 'week' ? 13 : 12 })}`),
    [cp, kind, s.company?.environment],
  )
  return (
    <div className="card mt">
      <div className="card-head">
        <h3>
          <CalendarCheck2 size={17} /> Day, week and month closings
        </h3>
        <div className="row" style={{ gap: 10 }}>
          {data &&
            (data.chainOk ? (
              <span className="badge b-green" title={`${data.checked} closings verified`}>
                <Link2 size={12} /> Chain intact
              </span>
            ) : (
              <span className="badge b-red">Chain broken at closing #{data.brokenAt}</span>
            ))}
          <div className="seg" role="tablist" aria-label="Closing period">
            {(Object.keys(kindLabels) as Closing['kind'][]).map((k) => (
              <button key={k} className={kind === k ? 'on' : ''} onClick={() => setKind(k)} role="tab" aria-selected={kind === k}>
                {kindLabels[k]}
              </button>
            ))}
          </div>
        </div>
      </div>
      <ErrorBox error={error} />
      {loading && !data ? (
        <div className="card-body">
          <Spinner />
        </div>
      ) : data && data.closings.length === 0 ? (
        <Empty>No {kindLabels[kind].toLowerCase()} closings yet. The first is recorded automatically within an hour after the period ends.</Empty>
      ) : (
        data && (
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Period</th>
                  <th className="num">Documents</th>
                  <th className="num">Reported</th>
                  <th className="num">Not reported</th>
                  <th className="num">Cancelled</th>
                  <th className="num">Value excl. ST</th>
                  <th className="num">Sales tax</th>
                  <th>FBR invoice numbers</th>
                  <th>Closed</th>
                </tr>
              </thead>
              <tbody>
                {data.closings.map((c) => {
                  const m = c.summary
                  const open = m.pending + m.unreconciled + m.rejected
                  return (
                    <tr key={c.id}>
                      <td className="nowrap">{closingPeriod(c)}</td>
                      <td className="num">{m.documents}</td>
                      <td className="num">{m.reported}</td>
                      <td className="num">{open > 0 ? <span className="badge b-amber">{open}</span> : 0}</td>
                      <td className="num">{m.cancelled}</td>
                      <td className="num">{money(Number(m.sales.valueExclST) + Number(m.debitNotes.valueExclST))}</td>
                      <td className="num">{money(Number(m.sales.salesTax) + Number(m.debitNotes.salesTax))}</td>
                      <td className="mono small">
                        {m.firstFbrNo ? (
                          <>
                            {m.firstFbrNo}
                            {m.lastFbrNo !== m.firstFbrNo && <div>{m.lastFbrNo}</div>}
                          </>
                        ) : (
                          '—'
                        )}
                      </td>
                      <td className="nowrap small" title={`Hash ${c.hash}`}>
                        {dateTimeFmt(c.createdAt)}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )
      )}
      <div className="card-foot">
        Rule 150R(4)(f) of the Sales Tax Rules, 2006 requires the invoicing system to perform a closing at the close of each day, week and month. Closings are
        recorded automatically, chained to each other by hash and cannot be changed afterwards; the daily integrity check verifies the chain.
      </div>
    </div>
  )
}
