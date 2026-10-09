// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Formatting helpers (amounts in PKR, Pakistan time).

import { getPrefs } from './prefs'

// Amounts use international grouping (1,234,567.00) or, by preference,
// the lakh and crore grouping used in Pakistan (12,34,567.00).
const amountFmt = new Intl.NumberFormat('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const amountFmtPK = new Intl.NumberFormat('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const qtyFmt = new Intl.NumberFormat('en-US', { maximumFractionDigits: 4 })

export function money(n: number | null | undefined): string {
  return (getPrefs().grouping === 'pk' ? amountFmtPK : amountFmt).format(Number(n ?? 0))
}

/**
 * lakhCrore expresses an amount the way it is spoken in Pakistan:
 * 4,343,659 → "43.44 lakh", 125,000,000 → "12.5 crore".
 */
export function lakhCrore(n: number | null | undefined): string {
  const v = Number(n ?? 0)
  const a = Math.abs(v)
  const f = (x: number) => x.toLocaleString('en-PK', { maximumFractionDigits: 2 })
  if (a >= 1e7) return f(v / 1e7) + ' crore'
  if (a >= 1e5) return f(v / 1e5) + ' lakh'
  if (a >= 1e3) return f(v / 1e3) + ' thousand'
  return f(v)
}

export function qty(n: number | null | undefined): string {
  return qtyFmt.format(Number(n ?? 0))
}

export function dateFmt(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = /^\d{4}-\d{2}-\d{2}$/.test(iso) ? new Date(iso + 'T00:00:00') : new Date(iso)
  if (isNaN(d.getTime())) return iso
  return d.toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
}

export function dateTimeFmt(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return iso
  return d.toLocaleString('en-GB', { timeZone: 'Asia/Karachi', day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

export function todayPK(): string {
  const parts = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Karachi', year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date())
  return parts // YYYY-MM-DD
}

export function monthStartPK(): string {
  return todayPK().slice(0, 8) + '01'
}

export const statusLabels: Record<string, string> = {
  DRAFT: 'Draft',
  VALIDATED: 'Validated',
  QUEUED: 'Queued',
  SUBMITTING: 'Submitting',
  ACCEPTED: 'Accepted by FBR',
  REJECTED: 'Rejected',
  UNCERTAIN: 'Needs reconciliation',
  CANCELLED: 'Cancelled',
}

export const envLabels: Record<string, string> = {
  simulator: 'Training simulator',
  sandbox: 'FBR sandbox',
  production: 'FBR production',
}

export function hoursSince(iso: string): number {
  const t = new Date(iso).getTime()
  if (isNaN(t)) return Infinity
  return (Date.now() - t) / 3_600_000
}

export function num(v: string): number {
  const n = parseFloat(String(v).replace(/,/g, ''))
  return isNaN(n) ? 0 : n
}
