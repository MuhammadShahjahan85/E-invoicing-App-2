// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Personal display preferences kept in this browser: Urdu captions and
// how large amounts are grouped (1,234,567 or the lakh/crore style
// 12,34,567).

import { useEffect, useState } from 'react'

export interface Prefs {
  urdu: boolean
  grouping: 'intl' | 'pk'
}

const defaults: Prefs = { urdu: true, grouping: 'intl' }

function read(): Prefs {
  try {
    const raw = localStorage.getItem('prefs')
    if (raw) return { ...defaults, ...JSON.parse(raw) }
  } catch {
    // storage unavailable or damaged: use the defaults
  }
  return { ...defaults }
}

let current = read()

export function getPrefs(): Prefs {
  return current
}

export function setPrefs(p: Partial<Prefs>) {
  current = { ...current, ...p }
  try {
    localStorage.setItem('prefs', JSON.stringify(current))
  } catch {
    // the choice lasts for this visit
  }
  window.dispatchEvent(new Event('prefschange'))
}

/** usePrefs re-renders when a preference changes. */
export function usePrefs(): Prefs {
  const [p, setP] = useState(current)
  useEffect(() => {
    const sync = () => setP(current)
    window.addEventListener('prefschange', sync)
    return () => window.removeEventListener('prefschange', sync)
  }, [])
  return p
}
