// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Keyboard shortcuts for frequent tasks, and the "?" help that lists them.

import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Modal } from './ui'

const goTo: { code: string; key: string; to: string; label: string }[] = [
  { code: 'KeyN', key: 'N', to: '/invoices/new', label: 'New invoice' },
  { code: 'KeyI', key: 'I', to: '/invoices', label: 'Invoices' },
  { code: 'KeyD', key: 'D', to: '/', label: 'Dashboard' },
  { code: 'KeyU', key: 'U', to: '/import', label: 'Import data' },
  { code: 'KeyB', key: 'B', to: '/customers', label: 'Customers (buyers)' },
  { code: 'KeyP', key: 'P', to: '/products', label: 'Products & services' },
  { code: 'KeyT', key: 'T', to: '/compliance', label: 'Tax periods & returns' },
  { code: 'KeyR', key: 'R', to: '/reports', label: 'Reports' },
  { code: 'KeyL', key: 'L', to: '/library', label: 'FBR reference library' },
]

function typing(e: KeyboardEvent) {
  const t = e.target as HTMLElement | null
  return !!t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.tagName === 'SELECT' || t.isContentEditable)
}

/** useShortcuts listens for Alt+letter shortcuts and "?" for the list. */
export function useShortcuts() {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.altKey && !e.ctrlKey && !e.metaKey) {
        const hit = goTo.find((g) => g.code === e.code)
        if (hit) {
          e.preventDefault()
          navigate(hit.to)
        }
        return
      }
      if (e.key === '?' && !typing(e)) {
        e.preventDefault()
        setOpen(true)
      }
    }
    const onOpen = () => setOpen(true)
    window.addEventListener('keydown', onKey)
    window.addEventListener('showshortcuts', onOpen)
    return () => {
      window.removeEventListener('keydown', onKey)
      window.removeEventListener('showshortcuts', onOpen)
    }
  }, [navigate])
  return { open, setOpen }
}

/** showShortcuts opens the list from anywhere (e.g. the user menu). */
export function showShortcuts() {
  window.dispatchEvent(new Event('showshortcuts'))
}

export function ShortcutsHelp({ onClose }: { onClose: () => void }) {
  return (
    <Modal title="Keyboard shortcuts" onClose={onClose}>
      <div className="shortcut-grid">
        <div className="shortcut">
          <span>Search invoices, buyers, products and HS codes</span>
          <span>
            <kbd className="kbd">Ctrl</kbd> <kbd className="kbd">K</kbd> or <kbd className="kbd">/</kbd>
          </span>
        </div>
        {goTo.map((g) => (
          <div key={g.code} className="shortcut">
            <span>{g.label}</span>
            <span>
              <kbd className="kbd">Alt</kbd> <kbd className="kbd">{g.key}</kbd>
            </span>
          </div>
        ))}
        <div className="shortcut">
          <span>Show this list</span>
          <span>
            <kbd className="kbd">?</kbd>
          </span>
        </div>
      </div>
    </Modal>
  )
}
