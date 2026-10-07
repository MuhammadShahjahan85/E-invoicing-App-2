// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// FBR reference library: the lists FBR maintains for Digital Invoicing
// (HS codes, sale types, rates, SROs, units, provinces, document types),
// buyer verification against the Active Taxpayer List and a tax calculator.

import { useEffect, useMemo, useState, type FormEvent } from 'react'
import { useSearchParams } from 'react-router-dom'
import {
  BadgeCheck,
  Calculator,
  CircleAlert,
  Database,
  Hash,
  Landmark,
  Layers,
  RefreshCw,
  Ruler,
  Search,
  ShieldCheck,
  TriangleAlert,
  type LucideIcon,
} from 'lucide-react'
import { api, errorMessage, qs } from '../api'
import { Empty, ErrorBox, Field, IssueList, Spinner, useLoad, PageSpinner } from '../components/ui'
import { dateTimeFmt, money, num, todayPK } from '../format'
import { useCompanyPath, useSession, useToast } from '../state'
import type { BuyerStatus, HSCode, Invoice, RateRef, RefStatus, SaleTypeRef } from '../types'

type Tab = 'hs' | 'rates' | 'uom' | 'codes' | 'buyer' | 'calculator' | 'sync'

const tabs: { id: Tab; label: string; icon: LucideIcon }[] = [
  { id: 'hs', label: 'HS codes', icon: Hash },
  { id: 'rates', label: 'Rates & SROs', icon: Layers },
  { id: 'uom', label: 'Units', icon: Ruler },
  { id: 'codes', label: 'Provinces & types', icon: Landmark },
  { id: 'buyer', label: 'Buyer check (ATL)', icon: ShieldCheck },
  { id: 'calculator', label: 'Tax calculator', icon: Calculator },
  { id: 'sync', label: 'Data sync', icon: Database },
]

export default function ReferenceLibrary() {
  const [params, setParams] = useSearchParams()
  const tab = (tabs.find((t) => t.id === params.get('tab'))?.id ?? 'hs') as Tab
  const setTab = (t: Tab) => setParams(t === 'hs' ? {} : { tab: t }, { replace: true })
  return (
    <>
      <div className="page-head">
        <div>
          <div className="eyebrow">
            <Landmark size={14} /> FBR data
          </div>
          <h1>FBR reference library</h1>
          <p>
            The reference lists FBR maintains for Digital Invoicing, downloaded through the PRAL API with your company's security token and kept on this
            server. Invoices are checked against the same lists before they are sent to FBR.
          </p>
        </div>
      </div>
      <div className="tabs" role="tablist">
        {tabs.map((t) => {
          const Ic = t.icon
          return (
            <button key={t.id} role="tab" aria-selected={tab === t.id} className={'tab' + (tab === t.id ? ' active' : '')} onClick={() => setTab(t.id)}>
              <Ic /> {t.label}
            </button>
          )
        })}
      </div>
      {tab === 'hs' && <HSCodes initial={params.get('q') ?? ''} />}
      {tab === 'rates' && <Rates />}
      {tab === 'uom' && <Units />}
      {tab === 'codes' && <Codes />}
      {tab === 'buyer' && <BuyerCheck />}
      {tab === 'calculator' && <TaxCalculator />}
      {tab === 'sync' && <SyncStatus />}
    </>
  )
}

// ---- HS codes ----

function HSCodes({ initial }: { initial: string }) {
  const cp = useCompanyPath()
  const [q, setQ] = useState(initial)
  const [items, setItems] = useState<HSCode[] | null>(null)
  const [loading, setLoading] = useState(false)
  const [open, setOpen] = useState<string>('')
  const [uoms, setUoms] = useState<Record<string, string[] | 'error'>>({})
  const status = useLoad(() => api.get<RefStatus>(`${cp}/ref/status`), [cp])

  useEffect(() => {
    const term = q.trim()
    if (term.length < 2) {
      setItems(null)
      return
    }
    setLoading(true)
    const t = window.setTimeout(() => {
      api
        .get<HSCode[]>(`${cp}/ref/hs-codes${qs({ q: term, limit: 100 })}`)
        .then(setItems)
        .catch(() => setItems([]))
        .finally(() => setLoading(false))
    }, 220)
    return () => window.clearTimeout(t)
  }, [q, cp])

  const toggle = (code: string) => {
    setOpen(open === code ? '' : code)
    if (uoms[code]) return
    api
      .get<{ id: number; description: string }[]>(`${cp}/ref/hs-uom${qs({ hsCode: code })}`)
      .then((r) => setUoms((u) => ({ ...u, [code]: (r ?? []).map((x) => x.description) })))
      .catch(() => setUoms((u) => ({ ...u, [code]: 'error' })))
  }

  const st = status.data
  return (
    <div className="card">
      <div className="card-head">
        <div className="input-icon" style={{ flex: 1, maxWidth: 560 }}>
          <Search size={17} />
          <input autoFocus value={q} onChange={(e) => setQ(e.target.value)} placeholder="Search by HS (PCT) code or description, e.g. 3104 or fertilizer" />
        </div>
        {st && (
          <span className={'badge ' + (st.hsSource === 'fbr' ? 'b-green' : 'b-amber')}>
            {st.hsCodes.toLocaleString()} codes · {st.hsSource === 'fbr' ? 'downloaded from FBR' : 'built-in starter list'}
          </span>
        )}
      </div>
      {st && st.hsSource !== 'fbr' && (
        <div className="card-body" style={{ paddingBottom: 0 }}>
          <div className="alert alert-info">
            <CircleAlert size={18} />
            <div className="alert-body">
              Only the built-in starter list is loaded. Once the company has an FBR sandbox or production token, download FBR's complete HS code list
              under <b>Data sync</b>.
            </div>
          </div>
        </div>
      )}
      {items === null ? (
        <Empty>
          <div className="empty-ic">
            <Hash size={24} />
          </div>
          <h3>Find the right HS code</h3>
          Type at least two characters. Select a code to see the unit of measure FBR prescribes for it (HS_UOM).
        </Empty>
      ) : loading && items.length === 0 ? (
        <div className="page-loading" style={{ minHeight: 160 }}>
          <Spinner />
        </div>
      ) : items.length === 0 ? (
        <Empty>No HS code matches “{q.trim()}”.</Empty>
      ) : (
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th style={{ width: 130 }}>HS code</th>
                <th>Description (FBR)</th>
                <th style={{ width: 220 }}>Unit of measure</th>
              </tr>
            </thead>
            <tbody>
              {items.map((h) => {
                const u = uoms[h.code]
                return (
                  <tr key={h.code} className="clickable" onClick={() => toggle(h.code)}>
                    <td className="mono">
                      <b>{h.code}</b>
                    </td>
                    <td>{h.description}</td>
                    <td className="small">
                      {open !== h.code && !u ? (
                        <span className="btn-link">Show FBR unit</span>
                      ) : !u ? (
                        <Spinner />
                      ) : u === 'error' ? (
                        <span className="faint">Not available (needs an FBR connection)</span>
                      ) : u.length === 0 ? (
                        <span className="faint">No unit prescribed</span>
                      ) : (
                        u.map((x) => (
                          <span key={x} className="badge b-green" style={{ marginRight: 4 }}>
                            {x}
                          </span>
                        ))
                      )}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

// ---- Sale types, rates, SRO schedules and items ----

interface SRORef {
  id: number
  serNo?: number
  description: string
}

function Rates() {
  const s = useSession()
  const cp = useCompanyPath()
  const types = useLoad(() => api.get<SaleTypeRef[]>(`${cp}/ref/sale-types`), [cp])
  const [saleType, setSaleType] = useState('')
  const [date, setDate] = useState(todayPK())
  const [rates, setRates] = useState<RateRef[] | null>(null)
  const [rate, setRate] = useState<RateRef | null>(null)
  const [schedules, setSchedules] = useState<SRORef[] | null>(null)
  const [schedule, setSchedule] = useState<SRORef | null>(null)
  const [items, setItems] = useState<SRORef[] | null>(null)
  const [error, setError] = useState('')

  const list = types.data ?? []
  useEffect(() => {
    if (!saleType && list.length) setSaleType(list[0].description)
  }, [list, saleType])

  useEffect(() => {
    setRates(null)
    setRate(null)
    if (!saleType) return
    api
      .get<RateRef[]>(`${cp}/ref/rates${qs({ saleType, date })}`)
      .then((r) => setRates(r ?? []))
      .catch((e) => {
        setRates([])
        setError(errorMessage(e))
      })
  }, [cp, saleType, date])

  useEffect(() => {
    setSchedules(null)
    setSchedule(null)
    if (!rate?.id) return
    api
      .get<SRORef[]>(`${cp}/ref/sro-schedules${qs({ rateId: rate.id, date })}`)
      .then((r) => setSchedules(r ?? []))
      .catch(() => setSchedules([]))
  }, [cp, rate, date])

  useEffect(() => {
    setItems(null)
    if (!schedule?.id) return
    api
      .get<SRORef[]>(`${cp}/ref/sro-items${qs({ sroId: schedule.id, date })}`)
      .then((r) => setItems(r ?? []))
      .catch(() => setItems([]))
  }, [cp, schedule, date])

  if (types.loading && !types.data) return <PageSpinner />
  if (types.error) return <ErrorBox error={types.error} />
  const info = list.find((t) => t.description === saleType)
  const known = info?.known ? info.info : s.meta.saleTypes.find((x) => x.name === saleType)

  return (
    <div className="stack" style={{ gap: 18 }}>
      <div className="card">
        <div className="card-body">
          <div className="form-grid" style={{ gridTemplateColumns: 'minmax(0, 2fr) minmax(0, 1fr)' }}>
            <Field label="Sale type (FBR transaction type)">
              <select value={saleType} onChange={(e) => setSaleType(e.target.value)}>
                {list.map((t) => (
                  <option key={t.description} value={t.description}>
                    {t.description}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="Rates valid on" hint={`For supplies from ${s.company?.province ?? 'your province'}`}>
              <input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </Field>
          </div>
          {known && (
            <div className="row mt" style={{ gap: 8 }}>
              <span className="chip">Category: {known.category}</span>
              <span className="chip">Usual rate: {known.defaultRate}</span>
              <span className="chip">Tax charged on: {known.basis === 'retail_price' ? 'printed retail price (Third Schedule)' : 'value of supply'}</span>
              {known.sroRequired && <span className="badge b-amber">SRO / schedule reference required</span>}
              {known.furtherTaxDefault && <span className="badge b-blue">Further tax applies to unregistered buyers</span>}
              {known.exempt && <span className="badge b-gray">Exempt</span>}
              {known.extraTaxMustBeEmpty && <span className="badge b-gray">Extra tax field must be left empty</span>}
            </div>
          )}
          {known?.note && <p className="small muted mt" style={{ marginBottom: 0 }}>{known.note}</p>}
        </div>
      </div>
      {error && <div className="alert alert-warn">{error}</div>}
      <div className="lib-cols">
        <div className="lib-col">
          <div className="lib-col-head">
            <span>1 · Rates (SaleTypeToRate)</span>
            <span>{rates?.length ?? ''}</span>
          </div>
          <div className="lib-col-body">
            {rates === null ? (
              <div className="chart-empty">
                <Spinner />
              </div>
            ) : rates.length === 0 ? (
              <div className="chart-empty">No rate returned for this sale type.</div>
            ) : (
              rates.map((r) => (
                <button key={r.id + r.description} className={'lib-opt' + (rate?.description === r.description ? ' on' : '')} onClick={() => setRate(r)}>
                  {r.description}
                  <small>{r.id ? `FBR rate id ${r.id}` : 'Built-in default (sync FBR data for the full list)'}</small>
                </button>
              ))
            )}
          </div>
        </div>
        <div className="lib-col">
          <div className="lib-col-head">
            <span>2 · SRO / schedule (SroSchedule)</span>
            <span>{schedules?.length ?? ''}</span>
          </div>
          <div className="lib-col-body">
            {!rate ? (
              <div className="chart-empty">Select a rate to see the SROs and schedules FBR links to it.</div>
            ) : schedules === null ? (
              <div className="chart-empty">
                <Spinner />
              </div>
            ) : schedules.length === 0 ? (
              <div className="chart-empty">No SRO or schedule is linked to this rate — leave the SRO fields empty on the invoice.</div>
            ) : (
              schedules.map((x) => (
                <button key={x.id} className={'lib-opt' + (schedule?.id === x.id ? ' on' : '')} onClick={() => setSchedule(x)}>
                  {x.description}
                  <small>Use as “SRO / schedule no.” on the invoice</small>
                </button>
              ))
            )}
          </div>
        </div>
        <div className="lib-col">
          <div className="lib-col-head">
            <span>3 · Serial no. (SROItem)</span>
            <span>{items?.length ?? ''}</span>
          </div>
          <div className="lib-col-body">
            {!schedule ? (
              <div className="chart-empty">Select an SRO or schedule to see its item serial numbers.</div>
            ) : items === null ? (
              <div className="chart-empty">
                <Spinner />
              </div>
            ) : items.length === 0 ? (
              <div className="chart-empty">No serial numbers listed for this SRO.</div>
            ) : (
              items.map((x) => (
                <div key={x.id} className="lib-opt" style={{ cursor: 'default' }}>
                  {x.description}
                  <small>Use as “SRO item serial no.” on the invoice</small>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
      <p className="small faint">
        Rates, SROs and serial numbers are looked up live from FBR for the selected date and your province, and kept for a day. In the training simulator
        they come from the built-in sample data.
      </p>
    </div>
  )
}

// ---- Units ----

function Units() {
  const cp = useCompanyPath()
  const { data, error, loading } = useLoad(() => api.get<string[]>(`${cp}/ref/uoms`), [cp])
  const [q, setQ] = useState('')
  if (loading && !data) return <PageSpinner />
  if (error) return <ErrorBox error={error} />
  const list = (data ?? []).filter((u) => u.toLowerCase().includes(q.trim().toLowerCase()))
  return (
    <div className="card">
      <div className="card-head">
        <div className="input-icon" style={{ flex: 1, maxWidth: 420 }}>
          <Search size={17} />
          <input value={q} onChange={(e) => setQ(e.target.value)} placeholder="Filter units" />
        </div>
        <span className="small faint">{data?.length ?? 0} units accepted by FBR</span>
      </div>
      <div className="card-body">
        <div className="pill-list">
          {list.map((u) => (
            <span key={u} className="pill" style={{ cursor: 'default' }}>
              {u}
            </span>
          ))}
        </div>
        <p className="small faint mt" style={{ marginBottom: 0 }}>
          Use the unit exactly as FBR spells it. Where FBR prescribes a unit for an HS code (see <b>HS codes</b>), the invoice must use that unit.
        </p>
      </div>
    </div>
  )
}

// ---- Provinces, document types, transaction types ----

function Codes() {
  const cp = useCompanyPath()
  const prov = useLoad(() => api.get<{ code: number; name: string }[]>(`${cp}/ref/provinces`), [cp])
  const docs = useLoad(() => api.get<{ id: number; name: string }[]>(`${cp}/ref/doc-types`), [cp])
  const types = useLoad(() => api.get<SaleTypeRef[]>(`${cp}/ref/sale-types`), [cp])
  return (
    <div className="grid g2">
      <div className="card">
        <div className="card-head">
          <h3>
            <Landmark size={17} /> Provinces
          </h3>
          <span className="small faint">Origin and destination of supply</span>
        </div>
        <CodeTable rows={(prov.data ?? []).map((p) => [String(p.code), p.name])} head={['Code', 'Province']} loading={prov.loading} />
      </div>
      <div className="card">
        <div className="card-head">
          <h3>
            <Layers size={17} /> Document types
          </h3>
          <span className="small faint">Sale invoices and debit notes</span>
        </div>
        <CodeTable rows={(docs.data ?? []).map((d) => [String(d.id), d.name])} head={['Id', 'Document type']} loading={docs.loading} />
      </div>
      <div className="card" style={{ gridColumn: '1 / -1' }}>
        <div className="card-head">
          <h3>
            <Layers size={17} /> Transaction (sale) types
          </h3>
          <span className="small faint">{types.data?.length ?? 0} types</span>
        </div>
        <CodeTable
          rows={(types.data ?? []).map((t) => [t.id ? String(t.id) : '—', t.description, t.known ? t.info.defaultRate : '—', t.known ? t.info.category : 'New in FBR list'])}
          head={['Id', 'Sale type', 'Usual rate', 'Category']}
          loading={types.loading}
        />
      </div>
    </div>
  )
}

function CodeTable({ rows, head, loading }: { rows: string[][]; head: string[]; loading: boolean }) {
  if (loading && rows.length === 0) return <div className="card-body"><Spinner /></div>
  return (
    <div className="table-wrap" style={{ maxHeight: 460, overflowY: 'auto' }}>
      <table className="table">
        <thead>
          <tr>
            {head.map((h) => (
              <th key={h}>{h}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={i}>
              {r.map((c, j) => (
                <td key={j} className={j === 0 ? 'mono' : ''}>
                  {c}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// ---- Buyer verification ----

function BuyerCheck() {
  const cp = useCompanyPath()
  const s = useSession()
  const [regNo, setRegNo] = useState('')
  const [busy, setBusy] = useState(false)
  const [res, setRes] = useState<BuyerStatus | null>(null)
  const [error, setError] = useState('')
  const [history, setHistory] = useState<BuyerStatus[]>([])

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    const n = regNo.replace(/[\s-]/g, '')
    if (!n) return
    setBusy(true)
    setError('')
    setRes(null)
    try {
      const r = await api.post<BuyerStatus>(`${cp}/buyer-check`, { regNo: n })
      setRes(r)
      setHistory((h) => [r, ...h.filter((x) => x.regNo !== r.regNo)].slice(0, 8))
    } catch (err) {
      setError(errorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  const ftr = s.company?.furtherTaxRate ?? 4
  const verdict = res && !res.error ? buyerVerdict(res, ftr) : null
  return (
    <div className="grid g-2-1">
      <div className="card">
        <div className="card-head">
          <h3>
            <ShieldCheck size={17} /> Check a buyer with FBR
          </h3>
        </div>
        <div className="card-body">
          <form onSubmit={submit} className="row" style={{ alignItems: 'flex-end' }}>
            <Field label="Buyer NTN or CNIC" hint="NTN: 7 digits (without the check digit). CNIC: 13 digits. Dashes are ignored.">
              <input value={regNo} onChange={(e) => setRegNo(e.target.value)} placeholder="e.g. 1234567 or 3520212345671" inputMode="numeric" autoFocus />
            </Field>
            <button className="btn btn-primary" disabled={busy || !regNo.trim()} style={{ marginBottom: 22 }}>
              {busy ? <span className="spinner" style={{ width: 14, height: 14, borderTopColor: '#fff' }} /> : <BadgeCheck size={16} />} Verify
            </button>
          </form>
          {error && <div className="alert alert-error">{error}</div>}
          {res?.error && (
            <div className="alert alert-warn">
              <TriangleAlert size={18} />
              <div className="alert-body">
                <b>FBR could not be asked.</b> {res.error}
              </div>
            </div>
          )}
          {verdict && res && (
            <div className={'verdict ' + verdict.tone}>
              <span className="v-ic">{verdict.tone === 'ok' ? <BadgeCheck size={22} /> : <TriangleAlert size={22} />}</span>
              <div>
                <div style={{ fontWeight: 700, fontSize: 15 }}>{verdict.title}</div>
                <div className="row small" style={{ gap: 6, margin: '6px 0 8px' }}>
                  <span className={'badge ' + (res.statlActive ? 'b-green' : 'b-red')}>ATL: {res.statlStatus || (res.statlActive ? 'Active' : 'Not active')}</span>
                  <span className={'badge ' + (res.registered ? 'b-green' : 'b-amber')}>Registration: {res.registrationType || (res.registered ? 'Registered' : 'Unregistered')}</span>
                  <span className="faint">Checked {dateTimeFmt(res.checkedAt)}</span>
                </div>
                <ul className="issue-list small" style={{ color: 'var(--text-2)' }}>
                  {verdict.points.map((p) => (
                    <li key={p}>{p}</li>
                  ))}
                </ul>
              </div>
            </div>
          )}
          {!res && !error && (
            <p className="small muted" style={{ margin: 0 }}>
              Asks FBR's <b>Sales Tax Active Taxpayer List</b> (STATL) and <b>registration type</b> services in real time. Save buyers under Customers to
              record their status with the date checked.
            </p>
          )}
        </div>
      </div>
      <div className="card">
        <div className="card-head">
          <h3>Recent checks</h3>
        </div>
        {history.length === 0 ? (
          <div className="chart-empty">Checks you make appear here.</div>
        ) : (
          <ul className="list-plain">
            {history.map((h) => (
              <li key={h.regNo} className="list-row">
                <span className="mono" style={{ flex: 1 }}>
                  {h.regNo}
                </span>
                {h.error ? (
                  <span className="badge b-gray">Not checked</span>
                ) : (
                  <span className={'badge ' + (h.statlActive && h.registered ? 'b-green' : 'b-amber')}>
                    {h.statlActive ? 'Active' : 'Not active'} · {h.registered ? 'Registered' : 'Unregistered'}
                  </span>
                )}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}

function buyerVerdict(r: BuyerStatus, furtherTaxRate: number): { tone: 'ok' | 'warn' | 'bad'; title: string; points: string[] } {
  if (r.registered && r.statlActive) {
    return {
      tone: 'ok',
      title: 'Registered and active — a registered buyer',
      points: [
        'Enter the buyer as “Registered” with this NTN/CNIC on the invoice.',
        'No further tax is charged on supplies to an active registered buyer.',
        'The buyer can claim input tax on your invoice once it is reported to FBR.',
      ],
    }
  }
  if (r.registered && !r.statlActive) {
    return {
      tone: 'warn',
      title: 'Registered but not on the Active Taxpayer List',
      points: [
        'FBR lists the buyer as registered but inactive (for example, returns not filed).',
        'Ask the buyer to resolve their status; an inactive buyer may not be able to claim input tax on your invoice.',
        'Check whether further tax applies to supplies to inactive persons before issuing the invoice.',
      ],
    }
  }
  return {
    tone: 'bad',
    title: 'Not registered for sales tax — an unregistered buyer',
    points: [
      'Enter the buyer as “Unregistered”.',
      `Further tax (currently set to ${furtherTaxRate}% in Company settings) is added on taxable supplies to unregistered persons, unless the sale type is excluded.`,
      'Record the buyer’s CNIC or NTN where the law requires it for supplies to unregistered persons.',
    ],
  }
}

// ---- Tax calculator ----

function TaxCalculator() {
  const s = useSession()
  const cp = useCompanyPath()
  const saleTypes = s.meta.saleTypes
  const [saleType, setSaleType] = useState(saleTypes[0]?.name ?? '')
  const [rate, setRate] = useState(saleTypes[0]?.defaultRate ?? '18%')
  const [rates, setRates] = useState<RateRef[]>([])
  const [qty, setQty] = useState('1')
  const [price, setPrice] = useState('100000')
  const [retail, setRetail] = useState('0')
  const [fed, setFed] = useState('0')
  const [extra, setExtra] = useState('0')
  const [regType, setRegType] = useState('Registered')
  const [province, setProvince] = useState(s.company?.province ?? '')
  const [result, setResult] = useState<Invoice | null>(null)
  const [error, setError] = useState('')
  const st = saleTypes.find((x) => x.name === saleType)
  const retailBasis = st?.basis === 'retail_price'

  useEffect(() => {
    api
      .get<RateRef[]>(`${cp}/ref/rates${qs({ saleType, date: todayPK() })}`)
      .then((r) => setRates(r ?? []))
      .catch(() => setRates([]))
  }, [cp, saleType])

  const body = useMemo(
    () => ({
      docType: 'Sale Invoice',
      invoiceDate: todayPK(),
      buyer: {
        ntnCnic: regType === 'Registered' ? '1000000' : '',
        name: 'Calculator',
        province,
        address: '-',
        registrationType: regType,
      },
      items: [
        {
          hsCode: '0000.0000',
          description: 'Calculation',
          uom: 'Numbers, pieces, units',
          quantity: num(qty),
          unitPrice: num(price),
          saleType,
          rate,
          retailPrice: num(retail),
          furtherTaxMode: 'auto',
          extraTaxRate: num(extra),
          fedRate: num(fed),
          sroScheduleNo: '',
          sroItemSerialNo: '',
        },
      ],
    }),
    [regType, province, qty, price, saleType, rate, retail, extra, fed],
  )

  useEffect(() => {
    const t = window.setTimeout(() => {
      api
        .post<Invoice>(`${cp}/invoices/compute`, body)
        .then((r) => {
          setResult(r)
          setError('')
        })
        .catch((e) => setError(errorMessage(e)))
    }, 300)
    return () => window.clearTimeout(t)
  }, [body, cp])

  const it = result?.items?.[0]
  const tot = result?.totals
  const rateChoices = rates.length ? rates.map((r) => r.description) : [st?.defaultRate ?? rate]
  // Only show the checks that concern the amounts, not the placeholder buyer/HS code.
  const issues = (result?.validation ?? []).filter((i) => !['hsCode', 'buyerNTNCNIC', 'buyerNtnCnic', 'buyerBusinessName', 'productDescription'].includes(i.field))
  return (
    <div className="grid g2">
      <div className="card">
        <div className="card-head">
          <h3>
            <Calculator size={17} /> Calculate sales tax
          </h3>
          <span className="small faint">Uses the same engine as invoices</span>
        </div>
        <div className="card-body">
          <div className="form-grid" style={{ gridTemplateColumns: 'repeat(2, minmax(0, 1fr))' }}>
            <Field label="Sale type" span={2}>
              <select
                value={saleType}
                onChange={(e) => {
                  const t = saleTypes.find((x) => x.name === e.target.value)
                  setSaleType(e.target.value)
                  setRate(t?.defaultRate ?? rate)
                }}
              >
                {saleTypes.map((t) => (
                  <option key={t.name}>{t.name}</option>
                ))}
              </select>
            </Field>
            <Field label="Rate">
              <select value={rate} onChange={(e) => setRate(e.target.value)}>
                {Array.from(new Set([...rateChoices, rate])).map((r) => (
                  <option key={r}>{r}</option>
                ))}
              </select>
            </Field>
            <Field label="Buyer">
              <select value={regType} onChange={(e) => setRegType(e.target.value)}>
                <option value="Registered">Registered (active)</option>
                <option value="Unregistered">Unregistered</option>
              </select>
            </Field>
            <Field label="Quantity">
              <input value={qty} onChange={(e) => setQty(e.target.value)} inputMode="decimal" />
            </Field>
            <Field label={retailBasis ? 'Unit price (excl. tax)' : 'Unit price (excl. sales tax)'}>
              <input value={price} onChange={(e) => setPrice(e.target.value)} inputMode="decimal" />
            </Field>
            {retailBasis && (
              <Field label="Printed retail price per unit" hint="Third Schedule: tax is charged on the retail price printed on the pack">
                <input value={retail} onChange={(e) => setRetail(e.target.value)} inputMode="decimal" />
              </Field>
            )}
            <Field label="FED rate %">
              <input value={fed} onChange={(e) => setFed(e.target.value)} inputMode="decimal" />
            </Field>
            <Field label="Extra tax rate %">
              <input value={extra} onChange={(e) => setExtra(e.target.value)} inputMode="decimal" disabled={st?.extraTaxMustBeEmpty} />
            </Field>
            <Field label="Buyer province">
              <select value={province} onChange={(e) => setProvince(e.target.value)}>
                {s.meta.provinces.map((p) => (
                  <option key={p.code}>{p.name}</option>
                ))}
              </select>
            </Field>
          </div>
        </div>
      </div>
      <div className="card">
        <div className="card-head">
          <h3>Result</h3>
          <span className="small faint">{saleType}</span>
        </div>
        <div className="card-body">
          {error && <div className="alert alert-error">{error}</div>}
          {tot && it ? (
            <>
              <div className="calc-out">
                <div>
                  <div className="k">Value excl. sales tax</div>
                  <div className="v">{money(tot.valueExclST)}</div>
                </div>
                {retailBasis && (
                  <div>
                    <div className="k">Retail value (tax base)</div>
                    <div className="v">{money(tot.retailValue)}</div>
                  </div>
                )}
                <div>
                  <div className="k">Sales tax ({rate})</div>
                  <div className="v">{money(tot.salesTax)}</div>
                </div>
                <div>
                  <div className="k">Further tax</div>
                  <div className="v">{money(tot.furtherTax)}</div>
                </div>
                <div>
                  <div className="k">Extra tax</div>
                  <div className="v">{money(tot.extraTax)}</div>
                </div>
                <div>
                  <div className="k">FED</div>
                  <div className="v">{money(tot.fed)}</div>
                </div>
                <div>
                  <div className="k">Sales tax withheld by buyer</div>
                  <div className="v">{money(tot.stWithheld)}</div>
                </div>
                <div className="grand">
                  <div className="k">Invoice total</div>
                  <div className="v">{money(tot.totalValue)}</div>
                </div>
                <div className="grand">
                  <div className="k">Amount payable by buyer</div>
                  <div className="v">{money(tot.amountPayable)}</div>
                </div>
              </div>
              {(it.warnings ?? []).length > 0 && (
                <div className="alert alert-info mt">
                  <CircleAlert size={18} />
                  <div className="alert-body">
                    {(it.warnings ?? []).map((w) => (
                      <div key={w}>{w}</div>
                    ))}
                  </div>
                </div>
              )}
              <div className="mt">
                <IssueList issues={issues} title="This combination would be refused" />
              </div>
            </>
          ) : (
            !error && <Spinner />
          )}
          <p className="small faint" style={{ margin: '12px 0 0' }}>
            Amounts are rounded the way FBR expects. Withholding follows the buyer's withholding setting; on a real invoice it depends on the customer
            master.
          </p>
        </div>
      </div>
    </div>
  )
}

// ---- Sync status ----

const kindLabels: Record<string, string> = {
  provinces: 'Provinces',
  doctypecode: 'Document types',
  transtypecode: 'Sale (transaction) types',
  uom: 'Units of measure',
  sroitemcode: 'SRO item codes',
  rates: 'Rates (live lookups)',
  sroschedule: 'SRO schedules (live lookups)',
  sroitem: 'SRO items (live lookups)',
  hsuom: 'HS code units (live lookups)',
}

function SyncStatus() {
  const s = useSession()
  const cp = useCompanyPath()
  const toast = useToast()
  const { data, error, loading, reload } = useLoad(() => api.get<RefStatus>(`${cp}/ref/status`), [cp])
  const [busy, setBusy] = useState(false)
  const [report, setReport] = useState<{ counts: Record<string, number>; errors: Record<string, string> } | null>(null)
  const env = s.company?.environment ?? 'simulator'

  const sync = async () => {
    setBusy(true)
    setReport(null)
    try {
      const r = await api.post<{ counts: Record<string, number>; errors: Record<string, string> }>(`${cp}/sync-reference`)
      setReport(r)
      toast(Object.keys(r.errors ?? {}).length ? 'err' : 'ok', Object.keys(r.errors ?? {}).length ? 'Some lists could not be downloaded' : 'FBR reference data updated')
      reload()
    } catch (e) {
      toast('err', errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  if (loading && !data) return <PageSpinner />
  if (error) return <ErrorBox error={error} />
  // Group cached entries of this environment by kind.
  const groups = new Map<string, { count: number; latest: string; source: string }>()
  for (const e of data?.entries ?? []) {
    if (!e.key.startsWith(env + ':')) continue
    const g = groups.get(e.kind) ?? { count: 0, latest: '', source: e.source }
    g.count++
    if (e.fetchedAt > g.latest) g.latest = e.fetchedAt
    groups.set(e.kind, g)
  }
  return (
    <div className="grid g-2-1">
      <div className="card">
        <div className="card-head">
          <h3>
            <Database size={17} /> Reference data on this server
          </h3>
          {s.can('company.write') && (
            <button className="btn btn-primary btn-sm" onClick={sync} disabled={busy}>
              <RefreshCw size={14} className={busy ? 'spin' : ''} /> {busy ? 'Downloading…' : 'Download from FBR now'}
            </button>
          )}
        </div>
        {report && (
          <div className="card-body" style={{ paddingBottom: 0 }}>
            <div className={'alert ' + (Object.keys(report.errors ?? {}).length ? 'alert-warn' : 'alert-ok')}>
              <div className="alert-body">
                {Object.entries(report.counts ?? {}).map(([k, n]) => (
                  <div key={k}>
                    {kindLabels[k] ?? k}: <b>{n}</b>
                  </div>
                ))}
                {Object.entries(report.errors ?? {}).map(([k, m]) => (
                  <div key={k}>
                    {kindLabels[k] ?? k}: {m}
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>List</th>
                <th className="num">Entries</th>
                <th>Source</th>
                <th>Last downloaded</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>HS codes (itemdesccode)</td>
                <td className="num">{data?.hsCodes.toLocaleString()}</td>
                <td>{data?.hsSource === 'fbr' ? <span className="badge b-green">FBR</span> : <span className="badge b-amber">Starter list</span>}</td>
                <td>—</td>
              </tr>
              {Array.from(groups.entries()).map(([kind, g]) => (
                <tr key={kind}>
                  <td>{kindLabels[kind] ?? kind}</td>
                  <td className="num">{g.count}</td>
                  <td>{g.source === 'fbr' ? <span className="badge b-green">FBR</span> : <span className="badge b-gray">{g.source}</span>}</td>
                  <td>{dateTimeFmt(g.latest)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
      <div className="card card-pad">
        <h3>How FBR data is used</h3>
        <ul className="issue-list small muted">
          <li>Downloads use the {env === 'production' ? 'production' : env === 'sandbox' ? 'sandbox' : 'simulator'} environment of this company and its security token.</li>
          <li>Sale types, units, provinces and HS codes are checked on every invoice before it is sent to FBR.</li>
          <li>Rates, SROs, serial numbers and HS units are asked live when you pick them, and kept for a day.</li>
          <li>Refresh the lists after FBR announces new sale types, rates or SROs (for example after the budget).</li>
        </ul>
        <p className="small faint" style={{ marginBottom: 0 }}>
          Last full download: <b>{data?.lastSync ? dateTimeFmt(data.lastSync) : 'never'}</b>
        </p>
      </div>
    </div>
  )
}
