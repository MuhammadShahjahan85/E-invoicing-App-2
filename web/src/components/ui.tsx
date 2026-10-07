// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import { api, errorMessage } from '../api'
import { statusLabels } from '../format'
import type { HSCode, Issue } from '../types'

export function Spinner() {
  return <span className="spinner" aria-label="loading" />
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
            ×
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
      )}
      {warnings.length > 0 && (
        <div className="alert alert-warn">
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
      )}
    </>
  )
}

export function ErrorBox({ error }: { error: unknown }) {
  if (!error) return null
  return <div className="alert alert-error">{errorMessage(error)}</div>
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

/** HSCodeInput: free text with suggestions from the HS code list. */
export function HSCodeInput({ value, onChange, companyPath, onPick }: { value: string; onChange: (v: string) => void; companyPath: string; onPick?: (h: HSCode) => void }) {
  const [open, setOpen] = useState(false)
  const [items, setItems] = useState<HSCode[]>([])
  const timer = useRef<number>()
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
        <div className="suggest">
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
            </div>
          ))}
        </div>
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
