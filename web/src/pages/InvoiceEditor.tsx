import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api, ApiError, errorMessage } from '../api'
import { ErrorBox, Field, HSCodeInput, IssueList, Spinner } from '../components/ui'
import { money, num, todayPK } from '../format'
import { useCompanyPath, useSession, useToast } from '../state'
import type { Customer, Invoice, InvoiceItem, Issue, Paged, Product, RateRef, SaleType } from '../types'

interface Line extends InvoiceItem {
  key: number
  showMore?: boolean
}

let keySeq = 1
const blankLine = (st?: SaleType): Line => ({
  key: keySeq++,
  hsCode: '',
  description: '',
  uom: 'Numbers, pieces, units',
  quantity: 1,
  unitPrice: 0,
  discountPercent: 0,
  discountAmount: 0,
  saleType: st?.name ?? 'Goods at standard rate (default)',
  rate: st?.defaultRate ?? '18%',
  retailPrice: 0,
  furtherTaxMode: 'auto',
  extraTaxRate: 0,
  fedRate: 0,
  sroScheduleNo: '',
  sroItemSerialNo: '',
})

interface Buyer {
  customerId: number | null
  ntnCnic: string
  name: string
  province: string
  address: string
  registrationType: string
  withholdingMode: string
}

const emptyBuyer: Buyer = { customerId: null, ntnCnic: '', name: '', province: '', address: '', registrationType: 'Unregistered', withholdingMode: '' }

function toPayloadItem(l: Line) {
  return {
    productId: l.productId ?? null,
    hsCode: l.hsCode,
    description: l.description,
    uom: l.uom,
    quantity: Number(l.quantity) || 0,
    unitPrice: Number(l.unitPrice) || 0,
    discountPercent: Number(l.discountPercent) || 0,
    discountAmount: Number(l.discountAmount) || 0,
    value: l.valueOverride ?? null,
    saleType: l.saleType,
    rate: l.rate,
    retailPrice: Number(l.retailPrice) || 0,
    retailValue: l.retailValueOverride ?? null,
    furtherTaxMode: l.furtherTaxMode,
    furtherTax: l.furtherTaxOverride ?? null,
    extraTaxRate: Number(l.extraTaxRate) || 0,
    extraTax: l.extraTaxOverride ?? null,
    fedRate: Number(l.fedRate) || 0,
    fed: l.fedOverride ?? null,
    stWithheld: l.withholdingOverride ?? null,
    salesTax: l.salesTaxOverride ?? null,
    sroScheduleNo: l.sroScheduleNo,
    sroItemSerialNo: l.sroItemSerialNo,
  }
}

export default function InvoiceEditor() {
  const { id } = useParams()
  const editing = !!id
  const s = useSession()
  const cp = useCompanyPath()
  const toast = useToast()
  const nav = useNavigate()
  const company = s.company!
  const saleTypes = s.meta.saleTypes
  const provinces = s.meta.provinces.map((p) => p.name)
  const env = company.environment

  const [loading, setLoading] = useState(editing)
  const [loadError, setLoadError] = useState<unknown>()
  const [docType, setDocType] = useState('Sale Invoice')
  const [date, setDate] = useState(todayPK())
  const [buyer, setBuyer] = useState<Buyer>(emptyBuyer)
  const [invoiceRefNo, setInvoiceRefNo] = useState('')
  const [scenarioId, setScenarioId] = useState('SN001')
  const [notes, setNotes] = useState('')
  const [externalRef, setExternalRef] = useState('')
  const [lines, setLines] = useState<Line[]>([blankLine(saleTypes[0])])
  const [computed, setComputed] = useState<Invoice | null>(null)
  const [computeError, setComputeError] = useState('')
  const [saving, setSaving] = useState(false)
  const [saveIssues, setSaveIssues] = useState<Issue[] | null>(null)
  const [saveError, setSaveError] = useState('')
  const [uoms, setUoms] = useState<string[]>(s.meta.uoms)
  const [rateOptions, setRateOptions] = useState<Record<string, RateRef[]>>({})

  // Load reference lists for the company environment.
  useEffect(() => {
    api.get<string[]>(`${cp}/ref/uoms`).then(setUoms).catch(() => {})
  }, [cp])

  // Load an existing invoice.
  useEffect(() => {
    if (!editing) return
    api
      .get<{ invoice: Invoice }>(`${cp}/invoices/${id}`)
      .then(({ invoice }) => {
        if (!['DRAFT', 'VALIDATED', 'REJECTED'].includes(invoice.status)) {
          nav(`/invoices/${id}`, { replace: true })
          return
        }
        setDocType(invoice.docType)
        setDate(invoice.invoiceDate)
        setBuyer({
          customerId: invoice.customerId,
          ntnCnic: invoice.buyerNtnCnic,
          name: invoice.buyerName,
          province: invoice.buyerProvince,
          address: invoice.buyerAddress,
          registrationType: invoice.buyerRegistrationType,
          withholdingMode: invoice.withholdingMode,
        })
        setInvoiceRefNo(invoice.invoiceRefNo)
        setScenarioId(invoice.scenarioId || 'SN001')
        setNotes(invoice.notes)
        setExternalRef(invoice.externalRef)
        setLines((invoice.items ?? []).map((it) => ({ ...it, key: keySeq++ })))
      })
      .catch(setLoadError)
      .finally(() => setLoading(false))
  }, [editing, id, cp, nav])

  const body = useMemo(
    () => ({
      docType,
      invoiceDate: date,
      customerId: buyer.customerId,
      buyer: { ntnCnic: buyer.ntnCnic, name: buyer.name, province: buyer.province, address: buyer.address, registrationType: buyer.registrationType },
      withholdingMode: buyer.withholdingMode,
      invoiceRefNo,
      scenarioId: env === 'sandbox' ? scenarioId : '',
      notes,
      externalRef,
      items: lines.map(toPayloadItem),
    }),
    [docType, date, buyer, invoiceRefNo, scenarioId, notes, externalRef, lines, env],
  )

  // Live computation (debounced) using the server's tax engine.
  const timer = useRef<number>()
  useEffect(() => {
    if (loading) return
    window.clearTimeout(timer.current)
    timer.current = window.setTimeout(async () => {
      try {
        const inv = await api.post<Invoice>(`${cp}/invoices/compute`, body)
        setComputed(inv)
        setComputeError('')
      } catch (e) {
        setComputeError(errorMessage(e))
      }
    }, 350)
    return () => window.clearTimeout(timer.current)
  }, [body, cp, loading])

  const loadRates = useCallback(
    async (saleType: string) => {
      if (rateOptions[saleType]) return
      try {
        const list = await api.get<RateRef[]>(`${cp}/ref/rates?saleType=${encodeURIComponent(saleType)}&date=${date}`)
        setRateOptions((r) => ({ ...r, [saleType]: list }))
      } catch {
        /* rates are optional suggestions */
      }
    },
    [cp, date, rateOptions],
  )

  const updateLine = (key: number, patch: Partial<Line>) => setLines((ls) => ls.map((l) => (l.key === key ? { ...l, ...patch } : l)))

  const onSaleType = (l: Line, name: string) => {
    const st = saleTypes.find((x) => x.name === name)
    updateLine(l.key, {
      saleType: name,
      rate: st?.defaultRate ?? l.rate,
      showMore: l.showMore || st?.sroRequired || st?.basis === 'retail_price',
    })
    loadRates(name)
  }

  const save = async (submit: boolean) => {
    setSaving(true)
    setSaveIssues(null)
    setSaveError('')
    try {
      const payload = { ...body, submit }
      const inv = editing ? await api.put<Invoice>(`${cp}/invoices/${id}`, payload) : await api.post<Invoice>(`${cp}/invoices`, payload)
      if (submit) {
        if (inv.status === 'ACCEPTED') toast('ok', `Accepted by FBR: ${inv.fbrInvoiceNumber}`)
        else if (inv.status === 'QUEUED') toast('info', 'FBR not reachable — the invoice is queued and will be submitted automatically.')
        else if (inv.status === 'REJECTED') toast('err', 'FBR rejected the invoice — see the errors.')
        else if (inv.status === 'UNCERTAIN') toast('err', 'No definitive answer from FBR — reconcile with IRIS before resubmitting.')
      } else toast('ok', `Saved ${inv.internalNo}`)
      nav(`/invoices/${inv.id}`)
    } catch (e) {
      if (e instanceof ApiError && e.issues) {
        setSaveIssues(e.issues)
        const saved = (e.data as { invoice?: Invoice } | undefined)?.invoice
        if (saved) nav(`/invoices/${saved.id}`)
      } else setSaveError(errorMessage(e))
      window.scrollTo({ top: 0, behavior: 'smooth' })
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <Spinner />
  if (loadError) return <ErrorBox error={loadError} />

  const totals = computed?.totals
  const compItem = (i: number) => computed?.items?.[i]
  const issues = saveIssues ?? computed?.validation ?? null

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{editing ? (docType === 'Debit Note' ? 'Edit debit note' : 'Edit invoice') : docType === 'Debit Note' ? 'New debit note' : 'New sales tax invoice'}</h1>
          <p>
            {company.name} · {env === 'production' ? 'FBR production' : env === 'sandbox' ? 'FBR sandbox (scenario testing)' : 'Training simulator'}
          </p>
        </div>
        <div className="actions">
          <button className="btn" onClick={() => nav(-1)}>
            Cancel
          </button>
          <button className="btn" disabled={saving} onClick={() => save(false)}>
            Save draft
          </button>
          <button className="btn btn-primary" disabled={saving || !!computeError} onClick={() => save(true)}>
            {saving ? 'Submitting…' : 'Save & submit to FBR'}
          </button>
        </div>
      </div>

      {saveError && <div className="alert alert-error">{saveError}</div>}
      {computeError && <div className="alert alert-error">{computeError}</div>}
      <IssueList issues={issues} />

      <div className="grid g2">
        <div className="card">
          <div className="card-head">
            <h3>Document</h3>
          </div>
          <div className="card-body form-grid">
            <Field label="Document type">
              <select value={docType} onChange={(e) => setDocType(e.target.value)} disabled={editing}>
                <option>Sale Invoice</option>
                <option>Debit Note</option>
              </select>
            </Field>
            <Field label="Invoice date" hint="Report at the time of supply (real time)">
              <input type="date" value={date} max={todayPK()} onChange={(e) => setDate(e.target.value)} />
            </Field>
            {docType === 'Debit Note' && (
              <Field label="Original FBR invoice no." hint="The FBR number of the invoice being adjusted" span={2}>
                <input className="mono" value={invoiceRefNo} onChange={(e) => setInvoiceRefNo(e.target.value.trim())} placeholder="0786909DI1747119701593" />
              </Field>
            )}
            {env === 'sandbox' && (
              <Field label="Sandbox scenario" hint="Required by FBR for sandbox submissions">
                <select value={scenarioId} onChange={(e) => setScenarioId(e.target.value)}>
                  {s.meta.scenarios.map((sc) => (
                    <option key={sc.id} value={sc.id}>
                      {sc.id} — {sc.title.slice(0, 50)}
                    </option>
                  ))}
                </select>
              </Field>
            )}
            <Field label="Your reference (optional)" hint="e.g. ERP / order number; prevents duplicates">
              <input value={externalRef} onChange={(e) => setExternalRef(e.target.value)} />
            </Field>
            <Field label="Notes (printed)" span={2}>
              <input value={notes} onChange={(e) => setNotes(e.target.value)} />
            </Field>
          </div>
        </div>
        <BuyerCard buyer={buyer} setBuyer={setBuyer} provinces={provinces} cp={cp} />
      </div>

      <div className="card mt">
        <div className="card-head">
          <h3>Lines</h3>
          <button className="btn btn-sm" onClick={() => setLines((ls) => [...ls, blankLine(saleTypes[0])])}>
            + Add line
          </button>
        </div>
        <div className="table-wrap">
          <table className="table lines-table">
            <thead>
              <tr>
                <th style={{ width: 28 }}>#</th>
                <th style={{ minWidth: 230 }}>Product / description</th>
                <th style={{ width: 120 }}>HS code</th>
                <th style={{ width: 150 }}>UoM</th>
                <th style={{ width: 90 }} className="num">
                  Qty
                </th>
                <th style={{ width: 110 }} className="num">
                  Unit price
                </th>
                <th style={{ minWidth: 200 }}>Sale type</th>
                <th style={{ width: 110 }}>Rate</th>
                <th className="num">Value excl. ST</th>
                <th className="num">Taxes</th>
                <th className="num">Total</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {lines.map((l, i) => {
                const c = compItem(i)
                const st = saleTypes.find((x) => x.name === l.saleType)
                return (
                  <LineRow
                    key={l.key}
                    n={i + 1}
                    l={l}
                    c={c}
                    st={st}
                    cp={cp}
                    uoms={uoms}
                    saleTypes={saleTypes}
                    rates={rateOptions[l.saleType]}
                    onFocusRate={() => loadRates(l.saleType)}
                    update={(p) => updateLine(l.key, p)}
                    onSaleType={(name) => onSaleType(l, name)}
                    remove={lines.length > 1 ? () => setLines((ls) => ls.filter((x) => x.key !== l.key)) : undefined}
                  />
                )
              })}
            </tbody>
          </table>
        </div>
      </div>

      <div className="row mt" style={{ alignItems: 'flex-start', justifyContent: 'space-between' }}>
        <div className="muted small" style={{ maxWidth: 560 }}>
          Taxes are computed by the built-in engine: sales tax on the value of supply (or printed retail price for Third Schedule goods), further tax
          {' ' + company.furtherTaxRate}% for unregistered buyers where applicable (section 3(1A)), extra tax and FED where entered, and sales tax
          withheld when the buyer is a withholding agent.
        </div>
        <div className="card card-pad totals-box">
          <div className="t-row">
            <span>Value excluding sales tax</span>
            <span>{money(totals?.valueExclST)}</span>
          </div>
          {!!totals?.retailValue && (
            <div className="t-row">
              <span>Retail price (3rd Schedule)</span>
              <span>{money(totals.retailValue)}</span>
            </div>
          )}
          <div className="t-row">
            <span>Sales tax</span>
            <span>{money(totals?.salesTax)}</span>
          </div>
          {!!totals?.furtherTax && (
            <div className="t-row">
              <span>Further tax</span>
              <span>{money(totals.furtherTax)}</span>
            </div>
          )}
          {!!totals?.extraTax && (
            <div className="t-row">
              <span>Extra tax</span>
              <span>{money(totals.extraTax)}</span>
            </div>
          )}
          {!!totals?.fed && (
            <div className="t-row">
              <span>FED</span>
              <span>{money(totals.fed)}</span>
            </div>
          )}
          <div className="t-row grand">
            <span>Total incl. taxes</span>
            <span>{money(totals?.totalValue)}</span>
          </div>
          {!!totals?.stWithheld && (
            <>
              <div className="t-row">
                <span>Less: ST withheld at source</span>
                <span>({money(totals.stWithheld)})</span>
              </div>
              <div className="t-row grand">
                <span>Amount payable</span>
                <span>{money(totals.amountPayable)}</span>
              </div>
            </>
          )}
        </div>
      </div>
    </>
  )
}

function BuyerCard({ buyer, setBuyer, provinces, cp }: { buyer: Buyer; setBuyer: (b: Buyer) => void; provinces: string[]; cp: string }) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<Customer[]>([])
  const [open, setOpen] = useState(false)
  const t = useRef<number>()
  const search = (q: string) => {
    window.clearTimeout(t.current)
    t.current = window.setTimeout(async () => {
      try {
        const r = await api.get<Paged<Customer>>(`${cp}/customers?q=${encodeURIComponent(q)}&limit=15&active=1`)
        setResults(r.items)
      } catch {
        setResults([])
      }
    }, 200)
  }
  const pick = (c: Customer) => {
    setBuyer({ customerId: c.id, ntnCnic: c.ntnCnic, name: c.name, province: c.province, address: c.address, registrationType: c.registrationType, withholdingMode: c.withholdingMode })
    setQuery('')
    setOpen(false)
  }
  const set = (k: keyof Buyer, v: string) => setBuyer({ ...buyer, [k]: v, ...(k !== 'withholdingMode' ? { customerId: buyer.customerId } : {}) })

  return (
    <div className="card">
      <div className="card-head">
        <h3>Buyer</h3>
        {buyer.customerId && (
          <button className="btn btn-sm" onClick={() => setBuyer(emptyBuyer)}>
            Clear
          </button>
        )}
      </div>
      <div className="card-body">
        <div className="dropdown mb">
          <input
            placeholder="Search customer master by name, NTN or code…"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value)
              search(e.target.value)
              setOpen(true)
            }}
            onFocus={() => {
              search(query)
              setOpen(true)
            }}
            onBlur={() => setTimeout(() => setOpen(false), 180)}
          />
          {open && results.length > 0 && (
            <div className="suggest">
              {results.map((c) => (
                <div key={c.id} onMouseDown={() => pick(c)}>
                  <b>{c.name}</b> <span className="muted">
                    {c.ntnCnic || 'no NTN'} · {c.registrationType} · {c.province}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
        <div className="form-grid">
          <Field label="Buyer name">
            <input value={buyer.name} onChange={(e) => set('name', e.target.value)} placeholder="Walk-in customer" />
          </Field>
          <Field label="NTN / CNIC" hint="Required for registered buyers">
            <input value={buyer.ntnCnic} onChange={(e) => set('ntnCnic', e.target.value)} />
          </Field>
          <Field label="Registration type">
            <select value={buyer.registrationType} onChange={(e) => set('registrationType', e.target.value)}>
              <option>Registered</option>
              <option>Unregistered</option>
            </select>
          </Field>
          <Field label="Province (destination)">
            <select value={buyer.province} onChange={(e) => set('province', e.target.value)}>
              <option value="">— select —</option>
              {provinces.map((p) => (
                <option key={p}>{p}</option>
              ))}
            </select>
          </Field>
          <Field label="Address" span={2}>
            <input value={buyer.address} onChange={(e) => set('address', e.target.value)} />
          </Field>
          <Field label="Withholding agent?" hint="Sales tax withheld at source by the buyer">
            <select value={buyer.withholdingMode} onChange={(e) => set('withholdingMode', e.target.value)}>
              <option value="">No</option>
              <option value="fraction">Yes — withholds a fraction (default 1/5th)</option>
              <option value="full">Yes — withholds the full sales tax</option>
            </select>
          </Field>
        </div>
      </div>
    </div>
  )
}

interface LineRowProps {
  n: number
  l: Line
  c?: InvoiceItem
  st?: SaleType
  cp: string
  uoms: string[]
  saleTypes: SaleType[]
  rates?: RateRef[]
  onFocusRate: () => void
  update: (p: Partial<Line>) => void
  onSaleType: (name: string) => void
  remove?: () => void
}

function LineRow({ n, l, c, st, cp, uoms, saleTypes, rates, onFocusRate, update, onSaleType, remove }: LineRowProps) {
  const [pq, setPq] = useState('')
  const [products, setProducts] = useState<Product[]>([])
  const [open, setOpen] = useState(false)
  const t = useRef<number>()
  const searchProducts = (q: string) => {
    window.clearTimeout(t.current)
    t.current = window.setTimeout(async () => {
      try {
        const r = await api.get<Paged<Product>>(`${cp}/products?q=${encodeURIComponent(q)}&limit=15&active=1`)
        setProducts(r.items)
      } catch {
        setProducts([])
      }
    }, 200)
  }
  const pickProduct = (p: Product) => {
    update({
      productId: p.id,
      description: p.description,
      hsCode: p.hsCode,
      uom: p.uom,
      saleType: p.saleType,
      rate: p.rate,
      unitPrice: p.unitPrice,
      retailPrice: p.retailPrice,
      sroScheduleNo: p.sroScheduleNo,
      sroItemSerialNo: p.sroItemSerialNo,
      furtherTaxMode: p.furtherTaxMode,
      extraTaxRate: p.extraTaxRate,
      fedRate: p.fedRate,
    })
    setPq('')
    setOpen(false)
  }
  const taxes = (c?.salesTax ?? 0) + (c?.furtherTax ?? 0) + (c?.extraTax ?? 0) + (c?.fed ?? 0)
  const listId = `rates-${l.key}`
  const optNum = (v: string) => (v.trim() === '' ? null : num(v))

  return (
    <>
      <tr>
        <td className="faint">{n}</td>
        <td>
          <div className="dropdown">
            <input
              value={pq || l.description}
              placeholder="Type to search products, or enter a description"
              onChange={(e) => {
                setPq('')
                update({ description: e.target.value, productId: null })
                searchProducts(e.target.value)
                setOpen(true)
              }}
              onFocus={() => {
                searchProducts(l.description)
                setOpen(true)
              }}
              onBlur={() => setTimeout(() => setOpen(false), 180)}
            />
            {open && products.length > 0 && (
              <div className="suggest">
                {products.map((p) => (
                  <div key={p.id} onMouseDown={() => pickProduct(p)}>
                    <b>{p.description}</b> <span className="muted">
                      {p.code} · {p.hsCode} · {p.rate} · {money(p.unitPrice)}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
          <button className="btn-link small" onClick={() => update({ showMore: !l.showMore })}>
            {l.showMore ? '▾ fewer details' : '▸ SRO, retail price, further/extra tax, discount…'}
          </button>
          {c?.warnings?.map((w, i) => (
            <div key={i} className="small" style={{ color: 'var(--warning)' }}>
              ⚠ {w}
            </div>
          ))}
        </td>
        <td>
          <HSCodeInput value={l.hsCode} onChange={(v) => update({ hsCode: v })} companyPath={cp} />
        </td>
        <td>
          <select value={l.uom} onChange={(e) => update({ uom: e.target.value })}>
            {!uoms.includes(l.uom) && <option>{l.uom}</option>}
            {uoms.map((u) => (
              <option key={u}>{u}</option>
            ))}
          </select>
        </td>
        <td>
          <input className="right" type="number" step="any" min="0" value={l.quantity} onChange={(e) => update({ quantity: num(e.target.value) })} />
        </td>
        <td>
          <input className="right" type="number" step="any" min="0" value={l.unitPrice} onChange={(e) => update({ unitPrice: num(e.target.value) })} />
        </td>
        <td>
          <select value={l.saleType} onChange={(e) => onSaleType(e.target.value)}>
            {!saleTypes.find((x) => x.name === l.saleType) && <option>{l.saleType}</option>}
            {saleTypes.map((x) => (
              <option key={x.name} value={x.name}>
                {x.name}
              </option>
            ))}
          </select>
        </td>
        <td>
          <input list={listId} value={l.rate} onFocus={onFocusRate} onChange={(e) => update({ rate: e.target.value })} />
          <datalist id={listId}>
            {(rates ?? []).map((r) => (
              <option key={r.id + r.description} value={r.description} />
            ))}
          </datalist>
        </td>
        <td className="num">{money(c?.valueExclST)}</td>
        <td className="num">{money(taxes)}</td>
        <td className="num">
          <b>{money(c?.totalValue)}</b>
        </td>
        <td>
          {remove && (
            <button className="btn btn-sm btn-danger" onClick={remove} title="Remove line">
              ×
            </button>
          )}
        </td>
      </tr>
      {l.showMore && (
        <tr>
          <td></td>
          <td colSpan={11}>
            <div className="form-grid" style={{ background: 'var(--surface-2)', padding: 10, borderRadius: 6 }}>
              {st?.note && <div className="small muted" style={{ gridColumn: '1 / -1' }}>ℹ {st.note}</div>}
              <Field label={'SRO / Schedule no.' + (st?.sroRequired ? ' *' : '')} hint="e.g. EIGHTH SCHEDULE Table 1">
                <input value={l.sroScheduleNo} onChange={(e) => update({ sroScheduleNo: e.target.value })} />
              </Field>
              <Field label={'SRO item serial no.' + (st?.sroRequired ? ' *' : '')}>
                <input value={l.sroItemSerialNo} onChange={(e) => update({ sroItemSerialNo: e.target.value })} />
              </Field>
              {st?.basis === 'retail_price' && (
                <Field label="Printed retail price per unit *" hint="Third Schedule: tax on retail price">
                  <input type="number" step="any" min="0" value={l.retailPrice} onChange={(e) => update({ retailPrice: num(e.target.value) })} />
                </Field>
              )}
              <Field label="Discount amount">
                <input type="number" step="any" min="0" value={l.discountAmount} onChange={(e) => update({ discountAmount: num(e.target.value) })} />
              </Field>
              <Field label="Discount %">
                <input type="number" step="any" min="0" value={l.discountPercent} onChange={(e) => update({ discountPercent: num(e.target.value) })} />
              </Field>
              <Field label="Further tax">
                <select value={l.furtherTaxMode} onChange={(e) => update({ furtherTaxMode: e.target.value })}>
                  <option value="auto">Automatic (by sale type & buyer)</option>
                  <option value="yes">Charge (unregistered buyer)</option>
                  <option value="no">Do not charge</option>
                </select>
              </Field>
              {!st?.extraTaxMustBeEmpty && (
                <Field label="Extra tax rate %">
                  <input type="number" step="any" min="0" value={l.extraTaxRate} onChange={(e) => update({ extraTaxRate: num(e.target.value) })} />
                </Field>
              )}
              <Field label="FED rate % (charged separately)">
                <input type="number" step="any" min="0" value={l.fedRate} onChange={(e) => update({ fedRate: num(e.target.value) })} />
              </Field>
              <Field label="Value override" hint="Value excl. ST from your ERP">
                <input type="number" step="any" value={l.valueOverride ?? ''} onChange={(e) => update({ valueOverride: optNum(e.target.value) })} />
              </Field>
              <Field label="ST withheld override">
                <input type="number" step="any" value={l.withholdingOverride ?? ''} onChange={(e) => update({ withholdingOverride: optNum(e.target.value) })} />
              </Field>
              <div className="small muted" style={{ gridColumn: '1 / -1' }}>
                Computed: sales tax {money(c?.salesTax)} · further tax {money(c?.furtherTax)} · extra tax {c?.extraTaxEmpty ? '(empty)' : money(c?.extraTax)} · FED {money(c?.fed)} · withheld{' '}
                {money(c?.stWithheld)}
                {c?.retailValue ? ` · retail value ${money(c.retailValue)}` : ''}
              </div>
            </div>
          </td>
        </tr>
      )}
    </>
  )
}
