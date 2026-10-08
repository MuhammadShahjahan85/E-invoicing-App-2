// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Formatting helpers (amounts in PKR, Pakistan time).

const amountFmt = new Intl.NumberFormat('en-PK', { minimumFractionDigits: 2, maximumFractionDigits: 2 })
const qtyFmt = new Intl.NumberFormat('en-PK', { maximumFractionDigits: 4 })

export function money(n: number | null | undefined): string {
  return amountFmt.format(Number(n ?? 0))
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
