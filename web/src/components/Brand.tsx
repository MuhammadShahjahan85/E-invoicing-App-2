// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useId } from 'react'

/** BrandMark is the Veridian logo mark: a viridian tile with a white check-mark "V" and a gold seal. */
export function BrandMark({ size = 36, title }: { size?: number; title?: string }) {
  const id = useId().replace(/:/g, '')
  return (
    <svg className="brand-mark" width={size} height={size} viewBox="0 0 100 100" role={title ? 'img' : undefined} aria-label={title} aria-hidden={title ? undefined : true}>
      <defs>
        <linearGradient id={`vg${id}`} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#1E8E72" />
          <stop offset="1" stopColor="#0A3A31" />
        </linearGradient>
      </defs>
      <rect width="100" height="100" rx="22" fill={`url(#vg${id})`} />
      <path d="M26 35 L46 70 L72 30" fill="none" stroke="#FFFFFF" strokeWidth="11" strokeLinecap="round" strokeLinejoin="round" />
      <circle cx="80.5" cy="18.5" r="6.5" fill="#E3B341" />
    </svg>
  )
}

/** Lockup combines the mark with the product name ("Veridian" / "E-invoicing Pakistan"). */
export function Lockup({ size = 38, light, className }: { size?: number; light?: boolean; className?: string }) {
  return (
    <span className={'lockup' + (light ? ' on-light' : '') + (className ? ' ' + className : '')}>
      <BrandMark size={size} title="Veridian" />
      <span className="lockup-text">
        <span className="lockup-name">Veridian</span>
        <span className="lockup-tag">E-invoicing Pakistan</span>
      </span>
    </span>
  )
}

/** VMark is the bare check-mark "V", used as a large watermark. */
export function VWatermark({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 100 100" aria-hidden="true">
      <path d="M26 35 L46 70 L72 30" fill="none" stroke="#FFFFFF" strokeWidth="11" strokeLinecap="round" strokeLinejoin="round" />
      <circle cx="80.5" cy="18.5" r="6.5" fill="#E3B341" />
    </svg>
  )
}
