// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Light / dark theme. "system" follows the device setting.

export type Theme = 'system' | 'light' | 'dark'

export function getTheme(): Theme {
  try {
    const t = localStorage.getItem('theme')
    return t === 'light' || t === 'dark' ? t : 'system'
  } catch {
    return 'system'
  }
}

export function applyTheme(t: Theme) {
  const root = document.documentElement
  if (t === 'system') root.removeAttribute('data-theme')
  else root.setAttribute('data-theme', t)
  try {
    if (t === 'system') localStorage.removeItem('theme')
    else localStorage.setItem('theme', t)
  } catch {
    // storage unavailable (private window): the choice lasts for this visit
  }
}
