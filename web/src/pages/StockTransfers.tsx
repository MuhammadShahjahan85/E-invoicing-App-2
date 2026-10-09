// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Download, Printer, Truck } from 'lucide-react'
import { api, errorMessage, qs } from '../api'
import { Empty, ErrorBox, Field, Modal, Pager, Spinner, useLoad } from '../components/ui'
import { dateTimeFmt, hoursSince, money, num, qty } from '../format'
import { useCompanyPath, useSession, useToast } from '../state'
import type { Paged, Product, StockTransfer } from '../types'

const statusText: Record<string, [string, string]> = {
  DISPATCHED: ['In transit', 'b-amber'],
  RECEIVED: ['Received', 'b-green'],
  CANCELLED: ['Cancelled', 'b-gray'],
}

function localNow(): string {
  const d = new Date()
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`
}

// The browser's local time with its offset, so the server records the
// moment the user meant whatever the PC's time zone.
function withOffset(v: string): string {
  if (!v) return ''
  const d = new Date(v)
  return isNaN(d.getTime()) ? v : d.toISOString()
}

function csvCell(v: unknown): string {
  const s = String(v ?? '')
  return /[",\n]/.test(s) ? '"' + s.replace(/"/g, '""') + '"' : s
}

export default function StockTransfers() {
  const s = useSession()
  const cp = useCompanyPath()
  const toast = useToast()
  const [params, setParams] = useSearchParams()
  const [q, setQ] = useState(params.get('q') ?? '')
  const status = params.get('status') ?? ''
  const from = params.get('from') ?? ''
  const to = params.get('to') ?? ''
  const offset = Number(params.get('offset') ?? 0)
  const limit = 50
  const { data, error, loading, reload } = useLoad(
    () => api.get<{ transfers: StockTransfer[]; total: number }>(`${cp}/stock-transfers${qs({ status, from, to, q: params.get('q'), limit, offset })}`),
    [cp, status, from, to, params.get('q'), offset],
  )
  const [creating, setCreating] = useState(false)
  const [receiving, setReceiving] = useState<StockTransfer | null>(null)
  const [cancelling, setCancelling] = useState<StockTransfer | null>(null)
  const canWrite = s.can('invoice.write')
  const canManage = s.can('invoice.manage')

  const setParam = (k: string, v: string) => {
    const p = new URLSearchParams(params)
    if (k !== 'offset') p.delete('offset')
    if (v) p.set(k, v)
    else p.delete(k)
    setParams(p)
  }

  const exportRegister = async () => {
    try {
      const r = await api.get<{ transfers: StockTransfer[]; total: number }>(`${cp}/stock-transfers${qs({ status, from, to, q: params.get('q'), limit: 500 })}`)
      const head = ['Note No.', 'Dispatched', 'From', 'To (warehouse)', 'Vehicle', 'Driver CNIC', 'Lines', 'Value at cost', 'Status', 'Received by', 'Received at', 'Remarks']
      const rows = r.transfers.map((t) => [
        t.number,
        dateTimeFmt(t.dispatchedAt),
        t.fromName,
        t.toName,
        t.vehicleNo,
        t.driverCnic,
        t.itemCount,
        t.totalValue,
        statusText[t.status]?.[0] ?? t.status,
        t.receivedBy,
        dateTimeFmt(t.receivedAt),
        t.status === 'CANCELLED' ? 'Cancelled: ' + t.cancelReason : t.notes,
      ])
      const csv = '﻿' + [head, ...rows].map((row) => row.map(csvCell).join(',')).join('\r\n')
      const a = document.createElement('a')
      a.href = URL.createObjectURL(new Blob([csv], { type: 'text/csv' }))
      a.download = `stock-transfer-register${from ? '-' + from : ''}${to ? '-to-' + to : ''}.csv`
      a.click()
      URL.revokeObjectURL(a.href)
    } catch (e) {
      toast('err', errorMessage(e))
    }
  }

  const inTransit = (data?.transfers ?? []).filter((t) => t.status === 'DISPATCHED' && hoursSince(t.dispatchedAt) > 48).length

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Stock transfer notes</h1>
          <p>
            Goods moved from your factory to your own warehouse registered under the same STRN are not a supply, so no sales tax invoice is reported to FBR.
            Under Sales Tax General Order 25 of 2026 each consignment must travel with a serially numbered stock transfer note, be acknowledged by the
            warehouse, reconciled monthly with stock records and kept for six years.
          </p>
        </div>
        <div className="actions">
          <button className="btn" onClick={exportRegister}>
            <Download size={15} /> Register (CSV)
          </button>
          {canWrite && (
            <button className="btn btn-primary" onClick={() => setCreating(true)}>
              <Truck size={16} /> New transfer note
            </button>
          )}
        </div>
      </div>
      {inTransit > 0 && (
        <div className="alert alert-warn">
          {inTransit} note(s) on this page have been in transit for more than 48 hours. Record the warehouse's acknowledgement when the goods arrive.
        </div>
      )}
      <div className="card">
        <div className="card-body row">
          <form
            className="row"
            style={{ flex: 1 }}
            onSubmit={(e) => {
              e.preventDefault()
              setParam('q', q)
            }}
          >
            <input placeholder="Search note no., warehouse, vehicle, goods…" value={q} onChange={(e) => setQ(e.target.value)} style={{ maxWidth: 340 }} />
            <button className="btn">Search</button>
          </form>
          <select value={status} onChange={(e) => setParam('status', e.target.value)} style={{ width: 'auto' }} aria-label="Status">
            <option value="">All notes</option>
            <option value="DISPATCHED">In transit</option>
            <option value="RECEIVED">Received</option>
            <option value="CANCELLED">Cancelled</option>
          </select>
          <input type="date" value={from} onChange={(e) => setParam('from', e.target.value)} style={{ width: 'auto' }} aria-label="From" />
          <input type="date" value={to} onChange={(e) => setParam('to', e.target.value)} style={{ width: 'auto' }} aria-label="To" />
        </div>
        <ErrorBox error={error} />
        {loading && !data ? (
          <div className="card-body">
            <Spinner />
          </div>
        ) : data && data.transfers.length === 0 ? (
          <Empty>No stock transfer notes{params.toString() ? ' match' : ' yet'}.</Empty>
        ) : (
          data && (
            <>
              <div className="table-wrap">
                <table className="table">
                  <thead>
                    <tr>
                      <th>Note No.</th>
                      <th>Dispatched</th>
                      <th>To (warehouse)</th>
                      <th>Vehicle</th>
                      <th className="num">Lines</th>
                      <th className="num">Value at cost</th>
                      <th>Status</th>
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.transfers.map((t) => (
                      <tr key={t.id}>
                        <td className="nowrap">
                          <b>{t.number}</b>
                        </td>
                        <td className="nowrap">{dateTimeFmt(t.dispatchedAt)}</td>
                        <td>
                          {t.toName}
                          <div className="small faint">{t.toAddress}</div>
                        </td>
                        <td className="nowrap">
                          {t.vehicleNo}
                          {t.driverCnic && <div className="small faint">{t.driverCnic}</div>}
                        </td>
                        <td className="num">{t.itemCount}</td>
                        <td className="num">{money(t.totalValue)}</td>
                        <td className="small">
                          <span className={'badge ' + (statusText[t.status]?.[1] ?? '')}>{statusText[t.status]?.[0] ?? t.status}</span>
                          {t.status === 'RECEIVED' && (
                            <div className="faint">
                              {t.receivedBy}, {dateTimeFmt(t.receivedAt)}
                            </div>
                          )}
                          {t.status === 'CANCELLED' && <div className="faint">{t.cancelReason}</div>}
                        </td>
                        <td className="nowrap">
                          <a className="btn btn-sm" href={`/api/v1${cp}/stock-transfers/${t.id}/print`} target="_blank" rel="noopener">
                            <Printer size={14} /> Print
                          </a>{' '}
                          {t.status === 'DISPATCHED' && canWrite && (
                            <button className="btn btn-sm" onClick={() => setReceiving(t)}>
                              Received
                            </button>
                          )}{' '}
                          {t.status === 'DISPATCHED' && canManage && (
                            <button className="btn btn-sm btn-ghost" onClick={() => setCancelling(t)}>
                              Cancel
                            </button>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <Pager total={data.total} limit={limit} offset={offset} onChange={(o) => setParam('offset', String(o))} />
            </>
          )
        )}
      </div>
      <div className="card card-pad mt">
        <h3>When to use a stock transfer note</h3>
        <ul className="small">
          <li>
            <b>Same STRN:</b> goods sent from the factory to your own warehouse or depot registered under the same STRN — issue a stock transfer note here. It is
            not reported to FBR's Digital Invoicing system.
          </li>
          <li>
            <b>Different STRN:</b> a warehouse registered separately is another registered person; the movement is a taxable supply — issue a sales tax invoice.
          </li>
          <li>Print two copies: the despatch copy travels with the goods, the warehouse copy is signed on receipt and returned. Record the receipt here.</li>
          <li>Reconcile the month's notes with your stock register; cancelled notes stay in the register with the reason.</li>
        </ul>
      </div>
      {creating && (
        <TransferForm
          onClose={() => setCreating(false)}
          onSaved={(t) => {
            setCreating(false)
            toast('ok', `Stock transfer note ${t.number} recorded`)
            window.open(`/api/v1${cp}/stock-transfers/${t.id}/print`, '_blank', 'noopener')
            reload()
          }}
        />
      )}
      {receiving && (
        <ReceiveForm
          t={receiving}
          onClose={() => setReceiving(null)}
          onSaved={() => {
            setReceiving(null)
            reload()
          }}
        />
      )}
      {cancelling && (
        <CancelForm
          t={cancelling}
          onClose={() => setCancelling(null)}
          onSaved={() => {
            setCancelling(null)
            reload()
          }}
        />
      )}
    </>
  )
}

interface Line {
  key: number
  productId: number | null
  description: string
  hsCode: string
  quantity: string
  uom: string
  valueAtCost: string
}

let lineKey = 0
const newLine = (): Line => ({ key: ++lineKey, productId: null, description: '', hsCode: '', quantity: '', uom: '', valueAtCost: '' })

function TransferForm({ onClose, onSaved }: { onClose: () => void; onSaved: (t: StockTransfer) => void }) {
  const s = useSession()
  const cp = useCompanyPath()
  const c = s.company!
  const [f, setF] = useState({
    dispatchedAt: localNow(),
    fromName: '',
    fromAddress: '',
    toName: '',
    toAddress: '',
    vehicleNo: '',
    driverCnic: '',
    authorisedBy: '',
    notes: '',
    sameStrn: false,
  })
  const [lines, setLines] = useState<Line[]>([newLine()])
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState('')
  const set = (k: keyof typeof f, v: string | boolean) => setF({ ...f, [k]: v })
  const update = (key: number, patch: Partial<Line>) => setLines((ls) => ls.map((l) => (l.key === key ? { ...l, ...patch } : l)))
  const total = lines.reduce((sum, l) => sum + num(l.valueAtCost), 0)

  const save = async () => {
    setBusy(true)
    setErr('')
    try {
      const t = await api.post<StockTransfer>(`${cp}/stock-transfers`, {
        ...f,
        dispatchedAt: withOffset(f.dispatchedAt),
        items: lines
          .filter((l) => l.productId || l.description.trim() || l.quantity)
          .map((l) => ({
            productId: l.productId,
            description: l.description,
            hsCode: l.hsCode,
            quantity: num(l.quantity),
            uom: l.uom,
            valueAtCost: num(l.valueAtCost),
          })),
      })
      onSaved(t)
    } catch (e) {
      setErr(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal
      title="New stock transfer note"
      wide
      onClose={onClose}
      footer={
        <>
          <span className="muted small" style={{ marginRight: 'auto' }}>
            Total value at cost: <b>{money(total)}</b>
          </span>
          <button className="btn" onClick={onClose}>
            Close
          </button>
          <button className="btn btn-primary" disabled={busy || !f.sameStrn} onClick={save}>
            {busy ? 'Saving…' : 'Save and print'}
          </button>
        </>
      }
    >
      <div className="form-grid">
        <Field label="Dispatch date and time">
          <input type="datetime-local" value={f.dispatchedAt} onChange={(e) => set('dispatchedAt', e.target.value)} />
        </Field>
        <Field label="Authorised by" hint="Name and designation of the officer releasing the goods">
          <input value={f.authorisedBy} onChange={(e) => set('authorisedBy', e.target.value)} />
        </Field>
        <Field label="From (factory / premises)" hint="Leave blank to use the company's name">
          <input value={f.fromName} placeholder={c.name} onChange={(e) => set('fromName', e.target.value)} />
        </Field>
        <Field label="From address">
          <input value={f.fromAddress} placeholder={[c.address, c.city].filter(Boolean).join(', ')} onChange={(e) => set('fromAddress', e.target.value)} />
        </Field>
        <Field label="To (warehouse) *">
          <input value={f.toName} onChange={(e) => set('toName', e.target.value)} />
        </Field>
        <Field label="Warehouse address *">
          <input value={f.toAddress} onChange={(e) => set('toAddress', e.target.value)} />
        </Field>
        <Field label="Vehicle registration no.">
          <input value={f.vehicleNo} onChange={(e) => set('vehicleNo', e.target.value)} />
        </Field>
        <Field label="Driver's CNIC" hint="13 digits">
          <input value={f.driverCnic} inputMode="numeric" onChange={(e) => set('driverCnic', e.target.value)} />
        </Field>
      </div>
      <div className="table-wrap mt">
        <table className="table">
          <thead>
            <tr>
              <th>#</th>
              <th style={{ minWidth: 220 }}>Goods</th>
              <th>HS code</th>
              <th className="num">Quantity</th>
              <th>Unit</th>
              <th className="num">Value at cost</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {lines.map((l, i) => (
              <TransferLine
                key={l.key}
                n={i + 1}
                l={l}
                cp={cp}
                update={(p) => update(l.key, p)}
                remove={lines.length > 1 ? () => setLines((ls) => ls.filter((x) => x.key !== l.key)) : undefined}
              />
            ))}
          </tbody>
        </table>
      </div>
      <button className="btn btn-sm mt" onClick={() => setLines((ls) => [...ls, newLine()])}>
        + Add line
      </button>
      <Field label="Remarks">
        <textarea rows={2} value={f.notes} onChange={(e) => set('notes', e.target.value)} />
      </Field>
      <label className="check mt">
        <input type="checkbox" checked={f.sameStrn} onChange={(e) => set('sameStrn', e.target.checked)} /> The receiving warehouse is registered under our own STRN
        {c.strn ? ` (${c.strn})` : ''}. (If it has a separate STRN, issue a sales tax invoice instead.)
      </label>
      {err && <div className="alert alert-error mt">{err}</div>}
    </Modal>
  )
}

function TransferLine({ n, l, cp, update, remove }: { n: number; l: Line; cp: string; update: (p: Partial<Line>) => void; remove?: () => void }) {
  const [products, setProducts] = useState<Product[]>([])
  const [open, setOpen] = useState(false)
  const t = useRef<number>()
  const search = (q: string) => {
    window.clearTimeout(t.current)
    t.current = window.setTimeout(async () => {
      try {
        const r = await api.get<Paged<Product>>(`${cp}/products?q=${encodeURIComponent(q)}&limit=12&active=1`)
        setProducts(r.items)
      } catch {
        setProducts([])
      }
    }, 200)
  }
  return (
    <tr>
      <td className="faint">{n}</td>
      <td>
        <div className="dropdown">
          <input
            value={l.description}
            placeholder="Search products or describe the goods"
            onChange={(e) => {
              update({ description: e.target.value, productId: null })
              search(e.target.value)
              setOpen(true)
            }}
            onFocus={() => {
              search(l.description)
              setOpen(true)
            }}
            onBlur={() => setTimeout(() => setOpen(false), 180)}
          />
          {open && products.length > 0 && (
            <div className="suggest">
              {products.map((p) => (
                <div
                  key={p.id}
                  onMouseDown={() => {
                    update({ productId: p.id, description: p.description, hsCode: p.hsCode, uom: p.uom })
                    setOpen(false)
                  }}
                >
                  <b>{p.description}</b> <span className="muted">
                    {p.code} · {p.hsCode} · {p.uom}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </td>
      <td>
        <input value={l.hsCode} onChange={(e) => update({ hsCode: e.target.value })} style={{ width: 110 }} />
      </td>
      <td>
        <input className="right" type="number" step="any" min="0" value={l.quantity} onChange={(e) => update({ quantity: e.target.value })} style={{ width: 100 }} />
      </td>
      <td>
        <input value={l.uom} onChange={(e) => update({ uom: e.target.value })} style={{ width: 120 }} />
      </td>
      <td>
        <input className="right" type="number" step="any" min="0" value={l.valueAtCost} onChange={(e) => update({ valueAtCost: e.target.value })} style={{ width: 130 }} />
      </td>
      <td>
        {remove && (
          <button className="btn btn-sm btn-ghost" aria-label={`Remove line ${n}`} onClick={remove}>
            ✕
          </button>
        )}
      </td>
    </tr>
  )
}

function ReceiveForm({ t, onClose, onSaved }: { t: StockTransfer; onClose: () => void; onSaved: () => void }) {
  const cp = useCompanyPath()
  const toast = useToast()
  const [by, setBy] = useState('')
  const [at, setAt] = useState(localNow())
  const [err, setErr] = useState('')
  const save = async () => {
    try {
      await api.post(`${cp}/stock-transfers/${t.id}/receive`, { receivedBy: by, receivedAt: withOffset(at) })
      toast('ok', `${t.number} marked received`)
      onSaved()
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  return (
    <Modal
      title={`Receipt of ${t.number}`}
      onClose={onClose}
      footer={
        <>
          <button className="btn" onClick={onClose}>
            Close
          </button>
          <button className="btn btn-primary" disabled={!by.trim()} onClick={save}>
            Record receipt
          </button>
        </>
      }
    >
      <p className="small muted">
        {qty(t.itemCount)} line(s), value at cost {money(t.totalValue)}, dispatched {dateTimeFmt(t.dispatchedAt)} to {t.toName}.
      </p>
      <Field label="Received by" hint="Name and designation of the warehouse in-charge who signed the warehouse copy">
        <input value={by} onChange={(e) => setBy(e.target.value)} autoFocus />
      </Field>
      <Field label="Received at">
        <input type="datetime-local" value={at} onChange={(e) => setAt(e.target.value)} />
      </Field>
      {err && <div className="alert alert-error">{err}</div>}
    </Modal>
  )
}

function CancelForm({ t, onClose, onSaved }: { t: StockTransfer; onClose: () => void; onSaved: () => void }) {
  const cp = useCompanyPath()
  const toast = useToast()
  const [reason, setReason] = useState('')
  const [err, setErr] = useState('')
  const save = async () => {
    try {
      await api.post(`${cp}/stock-transfers/${t.id}/cancel`, { reason })
      toast('ok', `${t.number} cancelled`)
      onSaved()
    } catch (e) {
      setErr(errorMessage(e))
    }
  }
  return (
    <Modal
      title={`Cancel ${t.number}`}
      onClose={onClose}
      footer={
        <>
          <button className="btn" onClick={onClose}>
            Keep the note
          </button>
          <button className="btn btn-danger" disabled={!reason.trim()} onClick={save}>
            Cancel the note
          </button>
        </>
      }
    >
      <p className="small">A cancelled note stays in the register with its number and the reason, so the series has no gaps.</p>
      <Field label="Reason">
        <input value={reason} onChange={(e) => setReason(e.target.value)} autoFocus />
      </Field>
      {err && <div className="alert alert-error">{err}</div>}
    </Modal>
  )
}
