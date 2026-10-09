// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { BadgeCheck } from 'lucide-react'
import { api, errorMessage, qs } from '../api'
import { Empty, ErrorBox, Field, Modal, Pager, Spinner, useLoad } from '../components/ui'
import { dateTimeFmt } from '../format'
import { useCompanyPath, useSession, useToast } from '../state'
import { pkCities, provinceForCity, regNoHint } from '../pkcities'
import type { Customer, Paged } from '../types'

interface BuyerStatus {
  regNo: string
  statlActive: boolean
  statlStatus: string
  registrationType: string
  registered: boolean
  checkedAt: string
  error?: string
}

const blank: Partial<Customer> = {
  code: '',
  name: '',
  ntnCnic: '',
  strn: '',
  registrationType: 'Registered',
  province: '',
  address: '',
  city: '',
  phone: '',
  email: '',
  withholdingMode: '',
  notes: '',
  active: true,
}

export default function Customers() {
  const s = useSession()
  const cp = useCompanyPath()
  const [params] = useSearchParams()
  const urlQ = params.get('q') ?? ''
  const [q, setQ] = useState(urlQ)
  const [search, setSearch] = useState(urlQ)
  // Opening the page from the global search fills the search box.
  useEffect(() => {
    setQ(urlQ)
    setSearch(urlQ)
  }, [urlQ])
  const [offset, setOffset] = useState(0)
  const [editing, setEditing] = useState<Partial<Customer> | null>(null)
  const limit = 50
  const { data, error, loading, reload } = useLoad(() => api.get<Paged<Customer>>(`${cp}/customers${qs({ q: search, limit, offset })}`), [cp, search, offset])
  const canWrite = s.can('masters.write')
  const toast = useToast()
  const [bulk, setBulk] = useState<{ done: number; total: number } | null>(null)

  // Re-checks every active buyer with an NTN/CNIC against FBR (ATL status and
  // registration type), one at a time.
  const checkAll = async () => {
    try {
      const all = await api.get<Paged<Customer>>(`${cp}/customers${qs({ limit: 1000, active: 1 })}`)
      const list = all.items.filter((c) => [7, 9, 13].includes(c.ntnCnic.replace(/\D/g, '').length))
      if (list.length === 0) {
        toast('info', 'No buyers with an NTN or CNIC to check')
        return
      }
      let failed = 0
      setBulk({ done: 0, total: list.length })
      for (let i = 0; i < list.length; i++) {
        try {
          const r = await api.post<{ error?: string }>(`${cp}/customers/${list[i].id}/check`)
          if (r.error) failed++
        } catch {
          failed++
        }
        setBulk({ done: i + 1, total: list.length })
      }
      toast(failed ? 'err' : 'ok', failed ? `${list.length - failed} of ${list.length} buyers checked; ${failed} could not be checked` : `${list.length} buyers checked with FBR`)
    } catch (e) {
      toast('err', errorMessage(e))
    } finally {
      setBulk(null)
      reload()
    }
  }

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Customers (buyers)</h1>
          <p>Buyer master used on invoices. FBR checks the buyer's NTN/CNIC, registration type and province of destination.</p>
        </div>
        <div className="actions">
          {canWrite && (
            <button className="btn" onClick={checkAll} disabled={!!bulk} title="Check every buyer's Active Taxpayer List status and registration type with FBR">
              <BadgeCheck size={16} /> {bulk ? `Checking ${bulk.done} / ${bulk.total}…` : 'Check all with FBR'}
            </button>
          )}
          {canWrite && (
            <button className="btn btn-primary" onClick={() => setEditing({ ...blank, province: s.company?.province ?? '' })}>
              + New customer
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
          <input placeholder="Search name, code, NTN/CNIC…" value={q} onChange={(e) => setQ(e.target.value)} style={{ maxWidth: 360 }} />
          <button className="btn">Search</button>
        </form>
        <ErrorBox error={error} />
        {loading && !data ? (
          <div className="card-body">
            <Spinner />
          </div>
        ) : data && data.items.length === 0 ? (
          <Empty>No customers yet.</Empty>
        ) : (
          data && (
            <>
              <div className="table-wrap">
                <table className="table">
                  <thead>
                    <tr>
                      <th>Code</th>
                      <th>Name</th>
                      <th>NTN / CNIC</th>
                      <th>Registration</th>
                      <th>Province</th>
                      <th>Withholding agent</th>
                      <th>FBR status</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.items.map((c) => (
                      <tr key={c.id} className={canWrite ? 'clickable' : ''} onClick={() => canWrite && setEditing(c)}>
                        <td className="mono small">{c.code}</td>
                        <td>
                          <b>{c.name}</b>
                          {c.city && <div className="small faint">{c.city}</div>}
                        </td>
                        <td className="mono">{c.ntnCnic || '—'}</td>
                        <td>{c.registrationType}</td>
                        <td>{c.province}</td>
                        <td className="small">{c.withholdingMode === 'full' ? 'Full ST' : c.withholdingMode === 'fraction' ? 'Fraction of ST' : '—'}</td>
                        <td className="small">
                          {c.statlStatus ? (
                            <>
                              <span className={'badge ' + (c.statlStatus.toLowerCase().includes('in') ? 'b-red' : 'b-green')}>{c.statlStatus}</span>{' '}
                              {c.fbrRegType}
                              {c.fbrRegType && c.fbrRegType !== c.registrationType && (
                                <div>
                                  <span className="badge b-amber" title="The registration type in the customer master differs from FBR's record">
                                    FBR says {c.fbrRegType}
                                  </span>
                                </div>
                              )}
                              <div className="faint">{dateTimeFmt(c.statusCheckedAt)}</div>
                            </>
                          ) : (
                            <span className="faint">not checked</span>
                          )}
                        </td>
                        <td>{!c.active && <span className="badge b-gray">inactive</span>}</td>
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
        <CustomerForm
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

export function CustomerForm({ initial, onClose, onSaved }: { initial: Partial<Customer>; onClose: () => void; onSaved: (c: Customer) => void }) {
  const s = useSession()
  const cp = useCompanyPath()
  const toast = useToast()
  const [c, setC] = useState<Partial<Customer>>(initial)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [check, setCheck] = useState<BuyerStatus | null>(null)
  const [checking, setChecking] = useState(false)
  const set = (k: keyof Customer, v: unknown) => setC((x) => ({ ...x, [k]: v }))

  const save = async () => {
    setSaving(true)
    setError('')
    try {
      const out = c.id ? await api.put<Customer>(`${cp}/customers/${c.id}`, c) : await api.post<Customer>(`${cp}/customers`, c)
      toast('ok', `Customer ${out.name} saved`)
      onSaved(out)
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setSaving(false)
    }
  }

  const checkFBR = async () => {
    setChecking(true)
    setCheck(null)
    try {
      const res = c.id
        ? await api.post<BuyerStatus>(`${cp}/customers/${c.id}/check`)
        : await api.post<BuyerStatus>(`${cp}/buyer-check`, { regNo: c.ntnCnic })
      setCheck(res)
      if (!res.error && res.registrationType) {
        set('registrationType', res.registered ? 'Registered' : 'Unregistered')
      }
    } catch (e) {
      setCheck({ regNo: c.ntnCnic ?? '', statlActive: false, statlStatus: '', registrationType: '', registered: false, checkedAt: '', error: errorMessage(e) })
    } finally {
      setChecking(false)
    }
  }

  return (
    <Modal
      wide
      title={c.id ? `Edit customer — ${initial.name}` : 'New customer'}
      onClose={onClose}
      footer={
        <>
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn btn-primary" onClick={save} disabled={saving || !c.name}>
            {saving ? 'Saving…' : 'Save'}
          </button>
        </>
      }
    >
      {error && <div className="alert alert-error">{error}</div>}
      <div className="form-grid">
        <Field label="Name *" span={2}>
          <input value={c.name ?? ''} onChange={(e) => set('name', e.target.value)} autoFocus />
        </Field>
        <Field label="Customer code">
          <input value={c.code ?? ''} onChange={(e) => set('code', e.target.value)} />
        </Field>
        <Field label="NTN / CNIC" hint={regNoHint(c.ntnCnic ?? '') || 'NTN 7 digits (without check digit) or CNIC 13 digits'}>
          <div className="row" style={{ flexWrap: 'nowrap' }}>
            <input className="mono" value={c.ntnCnic ?? ''} onChange={(e) => set('ntnCnic', e.target.value.replace(/[\s-]/g, ''))} />
            <button type="button" className="btn btn-sm" onClick={checkFBR} disabled={checking || !c.ntnCnic}>
              {checking ? '…' : 'Check FBR'}
            </button>
          </div>
        </Field>
        <Field label="Registration type *" hint="Unregistered buyers attract further tax (4%) on taxable supplies">
          <select value={c.registrationType} onChange={(e) => set('registrationType', e.target.value)}>
            <option>Registered</option>
            <option>Unregistered</option>
          </select>
        </Field>
        <Field label="STRN">
          <input value={c.strn ?? ''} onChange={(e) => set('strn', e.target.value)} />
        </Field>
        <Field label="Province (destination of supply) *">
          <select value={c.province} onChange={(e) => set('province', e.target.value)}>
            <option value="">— select —</option>
            {s.meta.provinces.map((p) => (
              <option key={p.code}>{p.name}</option>
            ))}
          </select>
        </Field>
        <Field label="City" hint={provinceForCity(c.city ?? '') ? `In ${provinceForCity(c.city ?? '')} — the province is filled in` : undefined}>
          <input
            value={c.city ?? ''}
            list="pk-cities"
            onChange={(e) => {
              set('city', e.target.value)
              const p = provinceForCity(e.target.value)
              if (p && s.meta.provinces.some((x) => x.name === p)) set('province', p)
            }}
          />
          <datalist id="pk-cities">
            {pkCities.map((x) => (
              <option key={x} value={x} />
            ))}
          </datalist>
        </Field>
        <Field label="Address" span={2}>
          <input value={c.address ?? ''} onChange={(e) => set('address', e.target.value)} />
        </Field>
        <Field label="Phone">
          <input value={c.phone ?? ''} onChange={(e) => set('phone', e.target.value)} />
        </Field>
        <Field label="Email">
          <input type="email" value={c.email ?? ''} onChange={(e) => set('email', e.target.value)} />
        </Field>
        <Field label="Sales tax withholding agent" hint="Buyer withholds sales tax under the Eleventh Schedule" span={2}>
          <select value={c.withholdingMode ?? ''} onChange={(e) => set('withholdingMode', e.target.value)}>
            <option value="">Not a withholding agent</option>
            <option value="fraction">Withholds a fraction of sales tax (default 1/5)</option>
            <option value="full">Withholds the full sales tax</option>
          </select>
        </Field>
        <Field label="Notes" span={2}>
          <input value={c.notes ?? ''} onChange={(e) => set('notes', e.target.value)} />
        </Field>
        {c.id && (
          <label className="check">
            <input type="checkbox" checked={!!c.active} onChange={(e) => set('active', e.target.checked)} /> Active
          </label>
        )}
      </div>
      {check && (
        <div className={'alert mt ' + (check.error ? 'alert-error' : check.statlActive ? 'alert-ok' : 'alert-warn')}>
          {check.error ? (
            check.error
          ) : (
            <>
              <b>{check.regNo}</b>: Active Taxpayer List status <b>{check.statlStatus || (check.statlActive ? 'Active' : 'Not active')}</b>; FBR registration type{' '}
              <b>{check.registrationType || '—'}</b>.
              {!check.statlActive && ' Supplies to buyers not on the ATL may attract further tax and input tax restrictions — treat as unregistered unless confirmed.'}
            </>
          )}
        </div>
      )}
    </Modal>
  )
}
