// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useEffect, useState } from 'react'
import { api, errorMessage, qs } from '../api'
import { Empty, ErrorBox, Field, HSCodeInput, Modal, Pager, Spinner, useLoad } from '../components/ui'
import { money, todayPK } from '../format'
import { useCompanyPath, useSession, useToast } from '../state'
import type { Paged, Product, RateRef } from '../types'

interface SRORef {
  id: number
  serNo?: number
  description: string
}

const blank: Partial<Product> = {
  code: '',
  description: '',
  hsCode: '',
  uom: 'Numbers, pieces, units',
  saleType: 'Goods at standard rate (default)',
  rate: '18%',
  sroScheduleNo: '',
  sroItemSerialNo: '',
  unitPrice: 0,
  retailPrice: 0,
  furtherTaxMode: 'auto',
  extraTaxRate: 0,
  fedRate: 0,
  active: true,
}

export default function Products() {
  const s = useSession()
  const cp = useCompanyPath()
  const [q, setQ] = useState('')
  const [search, setSearch] = useState('')
  const [offset, setOffset] = useState(0)
  const [editing, setEditing] = useState<Partial<Product> | null>(null)
  const limit = 50
  const { data, error, loading, reload } = useLoad(() => api.get<Paged<Product>>(`${cp}/products${qs({ q: search, limit, offset })}`), [cp, search, offset])
  const canWrite = s.can('masters.write')

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Products & services</h1>
          <p>Item master with HS (PCT) code, unit of measure, FBR sale type and rate — used to fill invoice lines correctly.</p>
        </div>
        <div className="actions">
          {canWrite && (
            <button className="btn btn-primary" onClick={() => setEditing({ ...blank })}>
              + New product
            </button>
          )}
        </div>
      </div>
      <div className="card">
        <form
          className="card-body row"
          onSubmit={(e) => {
            e.preventDefault()
            setOffset(0)
            setSearch(q)
          }}
        >
          <input placeholder="Search description, code, HS code…" value={q} onChange={(e) => setQ(e.target.value)} style={{ maxWidth: 360 }} />
          <button className="btn">Search</button>
        </form>
        <ErrorBox error={error} />
        {loading && !data ? (
          <div className="card-body">
            <Spinner />
          </div>
        ) : data && data.items.length === 0 ? (
          <Empty>No products yet.</Empty>
        ) : (
          data && (
            <>
              <div className="table-wrap">
                <table className="table">
                  <thead>
                    <tr>
                      <th>Code</th>
                      <th>Description</th>
                      <th>HS code</th>
                      <th>UoM</th>
                      <th>Sale type</th>
                      <th>Rate</th>
                      <th className="num">Unit price</th>
                      <th className="num">Retail price</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.items.map((p) => (
                      <tr key={p.id} className={canWrite ? 'clickable' : ''} onClick={() => canWrite && setEditing(p)}>
                        <td className="mono small">{p.code}</td>
                        <td>
                          <b>{p.description}</b>
                          {p.sroScheduleNo && (
                            <div className="small faint">
                              {p.sroScheduleNo} · S.No {p.sroItemSerialNo}
                            </div>
                          )}
                        </td>
                        <td className="mono">{p.hsCode}</td>
                        <td className="small">{p.uom}</td>
                        <td className="small">{p.saleType}</td>
                        <td>{p.rate}</td>
                        <td className="num">{money(p.unitPrice)}</td>
                        <td className="num">{p.retailPrice ? money(p.retailPrice) : '—'}</td>
                        <td>{!p.active && <span className="badge b-gray">inactive</span>}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <Pager total={data.total} limit={limit} offset={offset} onChange={setOffset} />
            </>
          )
        )}
      </div>
      {editing && (
        <ProductForm
          initial={editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null)
            reload()
          }}
        />
      )}
    </>
  )
}

function ProductForm({ initial, onClose, onSaved }: { initial: Partial<Product>; onClose: () => void; onSaved: (p: Product) => void }) {
  const s = useSession()
  const cp = useCompanyPath()
  const toast = useToast()
  const [p, setP] = useState<Partial<Product>>(initial)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [rates, setRates] = useState<RateRef[]>([])
  const [rateId, setRateId] = useState(0)
  const [schedules, setSchedules] = useState<SRORef[]>([])
  const [sroId, setSroId] = useState(0)
  const [sroItems, setSroItems] = useState<SRORef[]>([])
  const [hsUoms, setHsUoms] = useState<string[]>([])
  const set = (k: keyof Product, v: unknown) => setP((x) => ({ ...x, [k]: v }))
  const st = s.meta.saleTypes.find((x) => x.name === p.saleType)
  const today = todayPK()

  useEffect(() => {
    if (!p.saleType) return
    let alive = true
    api
      .get<RateRef[]>(`${cp}/ref/rates${qs({ saleType: p.saleType, date: today })}`)
      .then((r) => alive && setRates(r ?? []))
      .catch(() => alive && setRates([]))
    return () => {
      alive = false
    }
  }, [cp, p.saleType, today])

  useEffect(() => {
    const r = rates.find((x) => x.description === p.rate)
    setRateId(r?.id ?? 0)
  }, [rates, p.rate])

  useEffect(() => {
    if (!rateId || !(st?.sroRequired || st?.sroTypical)) {
      setSchedules([])
      return
    }
    api
      .get<SRORef[]>(`${cp}/ref/sro-schedules${qs({ rateId, date: today })}`)
      .then((r) => setSchedules(r ?? []))
      .catch(() => setSchedules([]))
  }, [cp, rateId, st?.sroRequired, st?.sroTypical, today])

  useEffect(() => {
    if (!sroId) {
      setSroItems([])
      return
    }
    api
      .get<SRORef[]>(`${cp}/ref/sro-items${qs({ sroId, date: today })}`)
      .then((r) => setSroItems(r ?? []))
      .catch(() => setSroItems([]))
  }, [cp, sroId, today])

  useEffect(() => {
    if (!/^\d{4}\.\d{4}$/.test(p.hsCode ?? '')) {
      setHsUoms([])
      return
    }
    api
      .get<{ id: number; description: string }[]>(`${cp}/ref/hs-uom${qs({ hsCode: p.hsCode })}`)
      .then((r) => setHsUoms((r ?? []).map((x) => x.description)))
      .catch(() => setHsUoms([]))
  }, [cp, p.hsCode])

  const changeSaleType = (name: string) => {
    const t = s.meta.saleTypes.find((x) => x.name === name)
    setP((x) => ({ ...x, saleType: name, rate: t?.defaultRate ?? x.rate }))
  }

  const save = async () => {
    setSaving(true)
    setError('')
    try {
      const out = p.id ? await api.put<Product>(`${cp}/products/${p.id}`, p) : await api.post<Product>(`${cp}/products`, p)
      toast('ok', `Product ${out.description} saved`)
      onSaved(out)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal
      wide
      title={p.id ? `Edit product — ${initial.description}` : 'New product / service'}
      onClose={onClose}
      footer={
        <>
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn btn-primary" onClick={save} disabled={saving || !p.description}>
            {saving ? 'Saving…' : 'Save'}
          </button>
        </>
      }
    >
      {error && <div className="alert alert-error">{error}</div>}
      <div className="form-grid">
        <Field label="Description *" span={2}>
          <input value={p.description ?? ''} onChange={(e) => set('description', e.target.value)} autoFocus />
        </Field>
        <Field label="Item code">
          <input value={p.code ?? ''} onChange={(e) => set('code', e.target.value)} />
        </Field>
        <Field label="HS code (PCT) *" hint="Format NNNN.NNNN">
          <HSCodeInput value={p.hsCode ?? ''} onChange={(v) => set('hsCode', v)} companyPath={cp} />
        </Field>
        <Field label="Unit of measure *" hint={hsUoms.length ? `FBR expects for this HS code: ${hsUoms.join(', ')}` : undefined}>
          <select value={p.uom} onChange={(e) => set('uom', e.target.value)}>
            {!s.meta.uoms.includes(p.uom ?? '') && p.uom && <option>{p.uom}</option>}
            {s.meta.uoms.map((u) => (
              <option key={u}>{u}</option>
            ))}
          </select>
        </Field>
        <Field label="Sale type *" span={2} hint={st?.note}>
          <select value={p.saleType} onChange={(e) => changeSaleType(e.target.value)}>
            {s.meta.saleTypes.map((t) => (
              <option key={t.name} value={t.name}>
                {t.name}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Rate *" hint={rates.length ? 'Pick from FBR SaleTypeToRate or type' : 'e.g. 18%, 5%, Exempt, Rs.200, 18% along with rupees 60 per kilogram'}>
          <input list="product-rates" value={p.rate ?? ''} onChange={(e) => set('rate', e.target.value)} disabled={st?.exempt} />
          <datalist id="product-rates">
            {rates.map((r) => (
              <option key={r.id} value={r.description} />
            ))}
          </datalist>
        </Field>
        <Field label="Further tax">
          <select value={p.furtherTaxMode ?? 'auto'} onChange={(e) => set('furtherTaxMode', e.target.value)}>
            <option value="auto">Automatic (unregistered buyers)</option>
            <option value="yes">Always charge</option>
            <option value="no">Never charge</option>
          </select>
        </Field>
        <Field label={'SRO / Schedule no.' + (st?.sroRequired ? ' *' : '')} hint={schedules.length ? undefined : st?.sroRequired ? 'Required for this sale type' : undefined}>
          {schedules.length > 0 ? (
            <select
              value={p.sroScheduleNo ?? ''}
              onChange={(e) => {
                const sc = schedules.find((x) => x.description === e.target.value)
                set('sroScheduleNo', e.target.value)
                setSroId(sc?.id ?? 0)
              }}
            >
              <option value="">— select —</option>
              {p.sroScheduleNo && !schedules.some((x) => x.description === p.sroScheduleNo) && <option>{p.sroScheduleNo}</option>}
              {schedules.map((x) => (
                <option key={x.id} value={x.description}>
                  {x.description}
                </option>
              ))}
            </select>
          ) : (
            <input value={p.sroScheduleNo ?? ''} onChange={(e) => set('sroScheduleNo', e.target.value)} placeholder="e.g. EIGHTH SCHEDULE Table 1" />
          )}
        </Field>
        <Field label={'SRO item serial no.' + (st?.sroRequired ? ' *' : '')}>
          {sroItems.length > 0 ? (
            <select value={p.sroItemSerialNo ?? ''} onChange={(e) => set('sroItemSerialNo', e.target.value)}>
              <option value="">— select —</option>
              {sroItems.map((x) => (
                <option key={x.id} value={x.description}>
                  {x.description}
                </option>
              ))}
            </select>
          ) : (
            <input value={p.sroItemSerialNo ?? ''} onChange={(e) => set('sroItemSerialNo', e.target.value)} placeholder="e.g. 82" />
          )}
        </Field>
        <Field label="Default unit price (excl. tax)">
          <input type="number" step="0.01" min="0" value={p.unitPrice ?? 0} onChange={(e) => set('unitPrice', Number(e.target.value))} />
        </Field>
        <Field
          label={'Printed retail price (incl. sales tax)' + (st?.basis === 'retail_price' ? ' *' : '')}
          hint={st?.basis === 'retail_price' ? 'Third Schedule: tax = printed price × rate ÷ (100 + rate)' : undefined}
        >
          <input type="number" step="0.01" min="0" value={p.retailPrice ?? 0} onChange={(e) => set('retailPrice', Number(e.target.value))} />
        </Field>
        {!st?.extraTaxMustBeEmpty && (
          <Field label="Extra tax rate %">
            <input type="number" step="0.01" min="0" value={p.extraTaxRate ?? 0} onChange={(e) => set('extraTaxRate', Number(e.target.value))} />
          </Field>
        )}
        <Field label="FED rate % (FED in sales tax mode)">
          <input type="number" step="0.01" min="0" value={p.fedRate ?? 0} onChange={(e) => set('fedRate', Number(e.target.value))} />
        </Field>
        {p.id && (
          <label className="check">
            <input type="checkbox" checked={!!p.active} onChange={(e) => set('active', e.target.checked)} /> Active
          </label>
        )}
      </div>
    </Modal>
  )
}
