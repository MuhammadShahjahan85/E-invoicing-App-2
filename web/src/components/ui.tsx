// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useCallback, useEffect, useLayoutEffect, useRef, useState, type CSSProperties, type ReactNode, type RefObject } from 'react'
import { createPortal } from 'react-dom'
import { CircleAlert, TriangleAlert, X } from 'lucide-react'
import { api, errorMessage } from '../api'
import { statusLabels } from '../format'
import type { HSCode, Issue } from '../types'

export function Spinner() {
  return <span className="spinner" aria-label="loading" />
}

/** PageSpinner is shown while a whole page loads. */
export function PageSpinner() {
  return (
    <div className="page-loading">
      <span className="spinner" aria-label="loading" />
    </div>
  )
}

export function Empty({ children }: { children: ReactNode }) {
  return <div className="empty">{children}</div>
}

export function Modal({ title, onClose, children, footer, wide }: { title: string; onClose: () => void; children: ReactNode; footer?: ReactNode; wide?: boolean }) {
  useEffect(() => {
    const h = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
    window.addEventListener('keydown', h)
    return () => window.removeEventListener('keydown', h)
  }, [onClose])
  return (
    <div className="modal-back" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className={'modal' + (wide ? ' wide' : '')} role="dialog" aria-modal="true" aria-label={title}>
        <div className="modal-head">
          <h3>{title}</h3>
          <button className="x" onClick={onClose} aria-label="Close">
            <X size={18} />
          </button>
        </div>
        <div className="modal-body">{children}</div>
        {footer && <div className="modal-foot">{footer}</div>}
      </div>
    </div>
  )
}

export function Field({ label, hint, error, children, span }: { label: string; hint?: ReactNode; error?: string; children: ReactNode; span?: number }) {
  return (
    <div className="field" style={span ? { gridColumn: `span ${span}` } : undefined}>
      <label>{label}</label>
      {children}
      {hint && <span className="hint">{hint}</span>}
      {error && <span className="err">{error}</span>}
    </div>
  )
}

const statusClass: Record<string, string> = {
  DRAFT: 'b-gray',
  VALIDATED: 'b-blue',
  QUEUED: 'b-amber',
  SUBMITTING: 'b-blue',
  ACCEPTED: 'b-green',
  REJECTED: 'b-red',
  UNCERTAIN: 'b-purple',
  CANCELLED: 'b-gray',
}

export function StatusBadge({ status }: { status: string }) {
  return <span className={'badge ' + (statusClass[status] ?? 'b-gray')}>{statusLabels[status] ?? status}</span>
}

export function IssueList({ issues, title }: { issues: Issue[] | null | undefined; title?: string }) {
  if (!issues || issues.length === 0) return null
  const errors = issues.filter((i) => i.severity === 'error')
  const warnings = issues.filter((i) => i.severity === 'warning')
  return (
    <>
      {errors.length > 0 && (
        <div className="alert alert-error">
          <CircleAlert size={18} />
          <div className="alert-body">
          <b>{title ?? 'Please correct the following before submitting to FBR'}</b>
          <ul className="issue-list">
            {errors.map((i, k) => (
              <li key={k}>
                {i.line > 0 ? `Line ${i.line} · ` : ''}
                {i.code ? `[${i.code}] ` : ''}
                {i.message}
              </li>
            ))}
          </ul>
          </div>
        </div>
      )}
      {warnings.length > 0 && (
        <div className="alert alert-warn">
          <TriangleAlert size={18} />
          <div className="alert-body">
          <b>Warnings</b>
          <ul className="issue-list">
            {warnings.map((i, k) => (
              <li key={k}>
                {i.line > 0 ? `Line ${i.line} · ` : ''}
                {i.code ? `[${i.code}] ` : ''}
                {i.message}
              </li>
            ))}
          </ul>
          </div>
        </div>
      )}
    </>
  )
}

export function ErrorBox({ error }: { error: unknown }) {
  if (!error) return null
  return (
    <div className="alert alert-error">
      <CircleAlert size={18} />
      <div className="alert-body">{errorMessage(error)}</div>
    </div>
  )
}

/** useLoad fetches data for a component and exposes reload. */
export function useLoad<T>(fn: () => Promise<T>, deps: unknown[]): { data: T | undefined; error: unknown; loading: boolean; reload: () => void; setData: (d: T) => void } {
  const [data, setData] = useState<T>()
  const [error, setError] = useState<unknown>()
  const [loading, setLoading] = useState(true)
  const [tick, setTick] = useState(0)
  useEffect(() => {
    let alive = true
    setLoading(true)
    fn()
      .then((d) => {
        if (alive) {
          setData(d)
          setError(undefined)
        }
      })
      .catch((e) => alive && setError(e))
      .finally(() => alive && setLoading(false))
    return () => {
      alive = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...deps, tick])
  const reload = useCallback(() => setTick((t) => t + 1), [])
  return { data, error, loading, reload, setData }
}

export function Pager({ total, limit, offset, onChange }: { total: number; limit: number; offset: number; onChange: (o: number) => void }) {
  if (total <= limit) return <div className="small faint" style={{ padding: '8px 10px' }}>{total} record(s)</div>
  return (
    <div className="row small" style={{ padding: '8px 10px', justifyContent: 'space-between' }}>
      <span className="faint">
        {offset + 1}–{Math.min(offset + limit, total)} of {total}
      </span>
      <span className="row">
        <button className="btn btn-sm" disabled={offset === 0} onClick={() => onChange(Math.max(0, offset - limit))}>
          ‹ Previous
        </button>
        <button className="btn btn-sm" disabled={offset + limit >= total} onClick={() => onChange(offset + limit)}>
          Next ›
        </button>
      </span>
    </div>
  )
}

/** FloatingList shows a suggestion list under its anchor (above it near the
 * bottom of the window), on top of the page, so that scrolling tables and
 * dialogs do not cut it off. */
export function FloatingList({ anchor, children }: { anchor: RefObject<HTMLElement>; children: ReactNode }) {
  const [pos, setPos] = useState<CSSProperties | null>(null)
  useLayoutEffect(() => {
    // Follows the anchor every frame while open: scrolling, resizing and
    // content appearing above it (e.g. validation messages) all move it.
    let frame = 0
    let last = ''
    const place = () => {
      const el = anchor.current
      if (el) {
        const r = el.getBoundingClientRect()
        const vw = window.innerWidth
        const vh = window.innerHeight
        const key = [r.top, r.left, r.width, r.height, vw, vh].join('|')
        if (key !== last) {
          last = key
          if (r.bottom < 0 || r.top > vh) {
            // The field being typed in was pushed out of view: bring it back.
            if (document.activeElement === el) el.scrollIntoView({ block: 'nearest' })
            setPos(null)
          } else {
            const width = Math.min(Math.max(r.width, 320), vw - 16)
            const left = Math.max(8, Math.min(r.left, vw - width - 8))
            const below = vh - r.bottom - 12
            const above = r.top - 12
            const up = below < 180 && above > below
            const maxHeight = Math.max(120, Math.min(300, up ? above : below))
            setPos(up ? { left, width, maxHeight, top: 'auto', bottom: vh - r.top + 4 } : { left, width, maxHeight, top: r.bottom + 4 })
          }
        }
      }
      frame = window.requestAnimationFrame(place)
    }
    place()
    return () => window.cancelAnimationFrame(frame)
  }, [anchor])
  if (!pos) return null
  return createPortal(
    // Keeps the focus in the input while the list is scrolled or clicked.
    <div className="suggest floating" style={pos} onMouseDown={(e) => e.preventDefault()}>
      {children}
    </div>,
    document.body,
  )
}

/** HSCodeInput: free text with suggestions from the HS code list. */
export function HSCodeInput({ value, onChange, companyPath, onPick }: { value: string; onChange: (v: string) => void; companyPath: string; onPick?: (h: HSCode) => void }) {
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState<HSCode[]>([])
  const timer = useRef<number>()
  const input = useRef<HTMLInputElement>(null)
  const search = (q: string) => {
    window.clearTimeout(timer.current)
    timer.current = window.setTimeout(async () => {
      if (q.trim().length < 2) {
        setItems([])
        return
      }
      try {
        setItems(await api.get<HSCode[]>(`${companyPath}/ref/hs-codes?q=${encodeURIComponent(q)}&limit=25`))
      } catch {
        setItems([])
      }
    }, 220)
  }
  return (
    <div className="dropdown">
      <input
        ref={input}
        value={value}
        placeholder="0101.2100"
        onChange={(e) => {
          onChange(e.target.value)
          search(e.target.value)
          setOpen(true)
        }}
        onFocus={() => setOpen(true)}
        onBlur={() => setTimeout(() => setOpen(false), 180)}
      />
      {open && items.length > 0 && (
        <FloatingList anchor={input}>
          {items.map((h) => (
            <div
              key={h.code}
              onMouseDown={() => {
                onChange(h.code)
                onPick?.(h)
                setOpen(false)
              }}
            >
              <b className="mono">{h.code}</b> <span className="muted">{h.description}</span>
              {h.context && <span className="hs-ctx">{h.context}</span>}
            </div>
          ))}
        </FloatingList>
      )}
    </div>
  )
}

export function Confirm({ text, onConfirm, onClose, danger, label }: { text: ReactNode; onConfirm: () => void; onClose: () => void; danger?: boolean; label?: string }) {
  return (
    <Modal
      title="Please confirm"
      onClose={onClose}
      footer={
        <>
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button
            className={'btn ' + (danger ? 'btn-danger' : 'btn-primary')}
            onClick={() => {
              onConfirm()
              onClose()
            }}
          >
            {label ?? 'Confirm'}
          </button>
        </>
      }
    >
      {text}
    </Modal>
  )
}
