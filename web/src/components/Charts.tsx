// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Small, dependency-free charts for the dashboard and period review.
// Marks: bars ≤ 24px with a 4px rounded data end, hairline solid grid,
// one brand hue (--chart-1, validated for both themes); status colours
// only where a colour means a state, always with an icon and a label.

import { useEffect, useRef, useState, type ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { Ban, CircleCheck, CircleDashed, CircleHelp, CircleX, Clock, FilePen, TriangleAlert, type LucideIcon } from 'lucide-react'
import { money } from '../format'
import type { ReturnDeadline } from '../types'

/** compact formats an amount for axis ticks: 950, 12K, 3.4M, 1.2B. */
export function compact(n: number): string {
  const a = Math.abs(n)
  const f = (v: number, s: string) => (Math.round(v * 10) / 10).toString().replace(/\.0$/, '') + s
  if (a >= 1e9) return f(n / 1e9, 'B')
  if (a >= 1e6) return f(n / 1e6, 'M')
  if (a >= 1e3) return f(n / 1e3, 'K')
  return String(Math.round(n))
}

function niceStep(max: number, ticks: number) {
  const raw = max / ticks
  const mag = Math.pow(10, Math.floor(Math.log10(raw)))
  for (const m of [1, 2, 2.5, 5, 10]) if (raw <= m * mag) return m * mag
  return 10 * mag
}

function useWidth<T extends HTMLElement>(): [React.RefObject<T>, number] {
  const ref = useRef<T>(null)
  const [w, setW] = useState(0)
  useEffect(() => {
    if (!ref.current) return
    const el = ref.current
    setW(el.clientWidth)
    const ro = new ResizeObserver(() => setW(el.clientWidth))
    ro.observe(el)
    return () => ro.disconnect()
  }, [])
  return [ref, w]
}

/** Bar path with a rounded top (data end) and a square base. */
function colPath(x: number, y: number, w: number, h: number, r = 4) {
  if (h <= 0) return ''
  const rr = Math.min(r, h, w / 2)
  const b = y + h
  return `M${x},${b} V${y + rr} Q${x},${y} ${x + rr},${y} H${x + w - rr} Q${x + w},${y} ${x + w},${y + rr} V${b} Z`
}

export interface ColumnDatum {
  key: string
  label: string // axis label, e.g. "Sep"
  title: string // tooltip title, e.g. "September 2026"
  value: number
  rows: { label: string; value: string }[] // extra tooltip rows
}

/** ColumnChart shows one series over time (e.g. monthly sales). */
export function ColumnChart({ data, valueLabel, height = 220, caption }: { data: ColumnDatum[]; valueLabel: string; height?: number; caption: string }) {
  const [ref, width] = useWidth<HTMLDivElement>()
  const [hover, setHover] = useState<number | null>(null)
  const [table, setTable] = useState(false)
  const max = Math.max(0, ...data.map((d) => d.value))
  const padL = 44
  const padR = 8
  const padT = 22
  const axisH = 24
  const plotH = height - padT - axisH
  const step = max > 0 ? niceStep(max, 4) : 1
  const top = max > 0 ? Math.ceil(max / step) * step : 4
  const ticks: number[] = []
  for (let v = 0; v <= top + 1e-9; v += step) ticks.push(v)
  const plotW = Math.max(0, width - padL - padR)
  const band = data.length ? plotW / data.length : 0
  const barW = Math.min(24, band * 0.56)
  const y = (v: number) => padT + plotH - (top ? (v / top) * plotH : 0)
  const last = data.length - 1
  const hv = hover !== null ? data[hover] : null

  return (
    <figure className="chart" style={{ margin: 0 }}>
      <div className="between" style={{ marginBottom: 6 }}>
        <figcaption className="small faint">{caption}</figcaption>
        <button className="btn-link table-toggle" onClick={() => setTable(!table)} aria-pressed={table}>
          {table ? 'Show chart' : 'Show table'}
        </button>
      </div>
      {table ? (
        <div className="table-wrap">
          <table className="table">
            <thead>
              <tr>
                <th>Month</th>
                <th className="num">{valueLabel}</th>
                {data[0]?.rows.map((r) => (
                  <th key={r.label} className="num">
                    {r.label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.map((d) => (
                <tr key={d.key}>
                  <td>{d.title}</td>
                  <td className="num">{money(d.value)}</td>
                  {d.rows.map((r) => (
                    <td key={r.label} className="num">
                      {r.value}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div ref={ref} style={{ position: 'relative' }} onMouseLeave={() => setHover(null)}>
          {width > 0 && (
            <svg width={width} height={height} role="img" aria-label={`${caption}. ${valueLabel} by month.`}>
              {ticks.map((t) => (
                <g key={t}>
                  <line className={t === 0 ? 'base-line' : 'grid-line'} x1={padL} x2={width - padR} y1={Math.round(y(t)) + 0.5} y2={Math.round(y(t)) + 0.5} />
                  <text className="tick" x={padL - 8} y={y(t) + 4} textAnchor="end">
                    {compact(t)}
                  </text>
                </g>
              ))}
              {data.map((d, i) => {
                const cx = padL + band * i + band / 2
                const h = top ? (d.value / top) * plotH : 0
                return (
                  <g key={d.key}>
                    <path className={'bar' + (hover !== null && hover !== i ? ' dim' : '')} d={colPath(cx - barW / 2, y(d.value), barW, h)} />
                    {(i === last || (hover === null && d.value === max && max > 0 && i !== last)) && d.value > 0 && (
                      <text className="val" x={cx} y={y(d.value) - 6} textAnchor="middle">
                        {compact(d.value)}
                      </text>
                    )}
                    <text className="tick" x={cx} y={height - 6} textAnchor="middle">
                      {d.label}
                    </text>
                    <rect
                      className="hit"
                      x={padL + band * i}
                      y={padT - 10}
                      width={band}
                      height={plotH + 10}
                      tabIndex={0}
                      aria-label={`${d.title}: ${valueLabel} ${money(d.value)}`}
                      onMouseEnter={() => setHover(i)}
                      onFocus={() => setHover(i)}
                      onBlur={() => setHover(null)}
                    />
                  </g>
                )
              })}
            </svg>
          )}
          {hv && hover !== null && (
            <div className="chart-tip" style={{ left: Math.min(Math.max(padL + band * hover + band / 2, 90), width - 90), top: y(hv.value) }}>
              <div className="tt-title">{hv.title}</div>
              <div className="tt-row">
                <span className="tt-key">
                  <i /> {valueLabel}
                </span>
                <b>{money(hv.value)}</b>
              </div>
              {hv.rows.map((r) => (
                <div key={r.label} className="tt-row">
                  <span className="tt-key">{r.label}</span>
                  <b>{r.value}</b>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </figure>
  )
}

/** Sparkline: a 12-point trend in the de-emphasis hue, the current period in the accent. */
export function Sparkline({ values, height = 34 }: { values: number[]; height?: number }) {
  const [ref, width] = useWidth<HTMLDivElement>()
  const max = Math.max(1, ...values)
  const pts = values.map((v, i) => [values.length > 1 ? 4 + (i * (width - 8)) / (values.length - 1) : width / 2, 4 + (height - 8) * (1 - v / max)])
  const lastPt = pts[pts.length - 1]
  return (
    <div ref={ref} className="spark" aria-hidden="true">
      {width > 0 && values.length > 0 && (
        <svg width={width} height={height}>
          <polyline points={pts.map((p) => p.join(',')).join(' ')} fill="none" stroke="currentColor" strokeOpacity={0.45} strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" />
          {lastPt && <circle cx={lastPt[0]} cy={lastPt[1]} r={4} fill="#e3b341" stroke="var(--surface)" strokeWidth={2} />}
        </svg>
      )}
    </div>
  )
}

/** HBarList ranks a handful of items (one hue; value at the bar end). */
export function HBarList({ rows, empty }: { rows: { label: string; sub?: string; value: number; to?: string }[]; empty: string }) {
  if (rows.length === 0) return <div className="chart-empty">{empty}</div>
  const max = Math.max(1, ...rows.map((r) => r.value))
  return (
    <div className="hbar">
      {rows.map((r, i) => (
        <div className="hbar-row" key={i}>
          <div className="hbar-label" title={r.sub ? `${r.label} — ${r.sub}` : r.label}>
            {r.to ? <Link to={r.to}>{r.label}</Link> : <b style={{ fontWeight: 600 }}>{r.label}</b>}
            {r.sub && <span className="faint"> · {r.sub}</span>}
          </div>
          <div className="hbar-value">{money(r.value)}</div>
          <div className="hbar-track">
            <div className="hbar-fill" style={{ width: `${Math.max(1, (100 * r.value) / max)}%` }} />
          </div>
        </div>
      ))}
    </div>
  )
}

interface StatusMeta {
  label: string
  cls: string
  icon: LucideIcon
}

const statusMeta: Record<string, StatusMeta> = {
  ACCEPTED: { label: 'Accepted by FBR', cls: 'st-good', icon: CircleCheck },
  QUEUED: { label: 'Queued for FBR', cls: 'st-warning', icon: Clock },
  SUBMITTING: { label: 'Being submitted', cls: 'st-warning', icon: Clock },
  UNCERTAIN: { label: 'Needs reconciliation', cls: 'st-serious', icon: CircleHelp },
  REJECTED: { label: 'Rejected', cls: 'st-critical', icon: CircleX },
  VALIDATED: { label: 'Validated (not issued)', cls: 'st-neutral', icon: CircleDashed },
  DRAFT: { label: 'Draft', cls: 'st-neutral', icon: FilePen },
  CANCELLED: { label: 'Cancelled', cls: 'st-neutral-2', icon: Ban },
}
const statusOrder = ['ACCEPTED', 'QUEUED', 'SUBMITTING', 'UNCERTAIN', 'REJECTED', 'VALIDATED', 'DRAFT', 'CANCELLED']

/** StatusBreakdown: part-to-whole stacked bar plus a legend with icons, labels and counts. */
export function StatusBreakdown({ counts }: { counts: Record<string, number> }) {
  const total = statusOrder.reduce((a, k) => a + (counts[k] ?? 0), 0)
  if (total === 0) return <div className="chart-empty">No invoices yet in this environment.</div>
  const present = statusOrder.filter((k) => counts[k])
  return (
    <>
      <div className="stackbar" role="img" aria-label={present.map((k) => `${statusMeta[k].label} ${counts[k]}`).join(', ')} style={{ borderRadius: '0 4px 4px 0' }}>
        {present.map((k) => (
          <span key={k} className={statusMeta[k].cls} style={{ flexGrow: counts[k], flexBasis: 0 }} title={`${statusMeta[k].label}: ${counts[k]}`} />
        ))}
      </div>
      <div className="legend-rows">
        {present.map((k) => {
          const m = statusMeta[k]
          const Ic = m.icon
          return (
            <Link key={k} to={`/invoices?status=${k}`} className="legend-row">
              <span className={'sw ' + m.cls} />
              <span className="ic">
                <Ic size={15} />
              </span>
              {m.label}
              <span className="cnt">{counts[k].toLocaleString()}</span>
              <span className="faint small" style={{ width: 44, textAlign: 'right' }}>
                {Math.round((100 * counts[k]) / total)}%
              </span>
            </Link>
          )
        })}
      </div>
    </>
  )
}

const monthShort = (iso: string) => new Date(iso + 'T00:00:00').toLocaleDateString('en-GB', { month: 'short' })
const dayNum = (iso: string) => Number(iso.slice(8, 10))
const longDate = (iso: string) => new Date(iso + 'T00:00:00').toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short', year: 'numeric' })

function Countdown({ days }: { days: number }): ReactNode {
  if (days < 0)
    return (
      <span className="countdown" style={{ background: 'var(--surface-3)', color: 'var(--text-2)' }}>
        Passed
      </span>
    )
  if (days === 0)
    return (
      <span className="countdown late">
        <TriangleAlert size={12} /> Today
      </span>
    )
  if (days <= 3)
    return (
      <span className="countdown soon">
        <Clock size={12} /> {days === 1 ? 'Tomorrow' : `In ${days} days`}
      </span>
    )
  return <span className="countdown ok">In {days} days</span>
}

/** DeadlineList shows the payment and filing dates of a return. */
export function DeadlineList({ deadlines }: { deadlines: ReturnDeadline[] }) {
  return (
    <div>
      {deadlines.map((d) => (
        <div key={d.kind + d.period} className="deadline">
          <div className={'date-tile' + (d.kind === 'filing' ? ' gold' : '')}>
            <div className="m">{monthShort(d.due)}</div>
            <div className="d">{dayNum(d.due)}</div>
          </div>
          <div className="dl-main">
            <div className="dl-title">{d.kind === 'payment' ? 'Pay sales tax' : 'File the sales tax return'}</div>
            <div className="dl-sub">
              {d.periodLabel} · due {longDate(d.due)}
            </div>
            {d.originalDue && (
              <div className="dl-sub" title={d.reference}>
                Extended by FBR from {longDate(d.originalDue)}
                {d.reference ? ` — ${d.reference}` : ''}
              </div>
            )}
          </div>
          <Countdown days={d.daysLeft} />
        </div>
      ))}
    </div>
  )
}
