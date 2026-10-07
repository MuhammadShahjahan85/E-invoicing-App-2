// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Header widgets: global search, notifications (alerts) and the user menu.

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import {
  Bell,
  BookOpen,
  CalendarClock,
  CircleAlert,
  CircleCheck,
  FilePlus2,
  FileText,
  Info,
  KeyRound,
  LifeBuoy,
  Library,
  LogOut,
  Monitor,
  Moon,
  Package,
  Search,
  ShieldCheck,
  Smartphone,
  Sun,
  TriangleAlert,
  Upload,
  Users,
  type LucideIcon,
} from 'lucide-react'
import { api } from '../api'
import { useCompanyPath, useSession } from '../state'
import { applyTheme, getTheme, type Theme } from '../theme'
import { dateFmt, money, statusLabels } from '../format'
import type { Alert, SearchResults } from '../types'
import { usePopover } from './Popover'

// ---- Alerts ----

interface AlertsCtx {
  alerts: Alert[]
  reload: () => void
}
const AlertsContext = createContext<AlertsCtx>({ alerts: [], reload: () => {} })
export const useAlerts = () => useContext(AlertsContext)

/** AlertsProvider loads the current company's alerts and refreshes them regularly. */
export function AlertsProvider({ children }: { children: ReactNode }) {
  const cp = useCompanyPath()
  const { company } = useSession()
  const location = useLocation()
  const [alerts, setAlerts] = useState<Alert[]>([])
  const load = useCallback(() => {
    api
      .get<{ alerts: Alert[] }>(`${cp}/alerts`)
      .then((r) => setAlerts(r.alerts ?? []))
      .catch(() => {})
  }, [cp])
  useEffect(() => {
    load()
    const t = window.setInterval(load, 60_000)
    return () => window.clearInterval(t)
  }, [load, company?.environment])
  // Refresh after navigation (e.g. once an invoice has been corrected).
  useEffect(() => {
    const t = window.setTimeout(load, 800)
    return () => window.clearTimeout(t)
  }, [location.pathname, load])
  const value = useMemo(() => ({ alerts, reload: load }), [alerts, load])
  return <AlertsContext.Provider value={value}>{children}</AlertsContext.Provider>
}

const sevIcon: Record<Alert['severity'], LucideIcon> = { error: CircleAlert, warning: TriangleAlert, info: Info }

export function AlertsBell() {
  const { alerts } = useAlerts()
  const pop = usePopover()
  const urgent = alerts.filter((a) => a.severity !== 'info').length
  return (
    <div className="pop-anchor" ref={pop.ref}>
      <button
        className={'icon-btn' + (pop.open ? ' on' : '')}
        aria-label={`Notifications${alerts.length ? ` (${alerts.length})` : ''}`}
        aria-expanded={pop.open}
        onClick={() => pop.setOpen(!pop.open)}
      >
        <Bell size={19} />
        {urgent > 0 ? <span className="pip">{urgent > 9 ? '9+' : urgent}</span> : alerts.length > 0 ? <span className="pip dot" /> : null}
      </button>
      {pop.open && (
        <div className="popover" role="dialog" aria-label="Notifications">
          <div className="popover-head">
            <h4>Notifications</h4>
            <span className="small faint">{alerts.length ? `${alerts.length} item${alerts.length === 1 ? '' : 's'}` : ''}</span>
          </div>
          <div className="popover-body">
            {alerts.length === 0 ? (
              <div className="all-clear">
                <CircleCheck size={28} />
                <div>
                  <b>All clear.</b>
                </div>
                <div className="small">Nothing needs your attention right now.</div>
              </div>
            ) : (
              alerts.map((a) => {
                const Ic = sevIcon[a.severity]
                const body = (
                  <>
                    <span className="ai-icon">
                      <Ic size={17} />
                    </span>
                    <span>
                      <div className="ai-title">{a.title}</div>
                      {a.detail && <div className="ai-detail">{a.detail}</div>}
                    </span>
                  </>
                )
                return a.link ? (
                  <Link key={a.id} to={a.link} className={'alert-item ' + a.severity} onClick={() => pop.setOpen(false)}>
                    {body}
                  </Link>
                ) : (
                  <div key={a.id} className={'alert-item ' + a.severity}>
                    {body}
                  </div>
                )
              })
            )}
          </div>
        </div>
      )}
    </div>
  )
}

// ---- Global search ----

interface Hit {
  key: string
  group: string
  icon: LucideIcon
  title: string
  sub: string
  to: string
}

const commands: { label: string; keywords: string; to: string; icon: LucideIcon; perm?: string }[] = [
  { label: 'New invoice', keywords: 'create sale invoice issue', to: '/invoices/new', icon: FilePlus2 },
  { label: 'Import invoices (CSV / Excel)', keywords: 'upload bulk erp', to: '/import', icon: Upload },
  { label: 'Tax periods & returns', keywords: 'calendar deadline due date annexure c return filing payment', to: '/compliance', icon: CalendarClock, perm: 'reports' },
  { label: 'FBR reference library', keywords: 'hs code pct uom unit sro schedule rate sale type province', to: '/library', icon: Library },
  { label: 'Verify a buyer (ATL / registration)', keywords: 'atl active taxpayer ntn cnic strn check buyer', to: '/library?tab=buyer', icon: ShieldCheck },
  { label: 'Sales tax calculator', keywords: 'calculator compute tax retail price third schedule', to: '/library?tab=calculator', icon: BookOpen },
  { label: 'Help & FBR error codes', keywords: 'help error code guide support', to: '/help', icon: LifeBuoy },
  { label: 'Mobile app & access', keywords: 'phone install android iphone qr', to: '/mobile', icon: Smartphone },
]

function isTyping(e: KeyboardEvent) {
  const t = e.target as HTMLElement | null
  return !!t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.tagName === 'SELECT' || t.isContentEditable)
}

/** GlobalSearch finds invoices, buyers, products, HS codes and commands. */
export function GlobalSearch({ autoFocus, onDone }: { autoFocus?: boolean; onDone?: () => void }) {
  const cp = useCompanyPath()
  const s = useSession()
  const navigate = useNavigate()
  const [q, setQ] = useState('')
  const [res, setRes] = useState<SearchResults | null>(null)
  const [open, setOpen] = useState(false)
  const [hl, setHl] = useState(0)
  const [loading, setLoading] = useState(false)
  const input = useRef<HTMLInputElement>(null)
  const wrap = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (autoFocus) {
      input.current?.focus()
      return
    }
    const h = (e: KeyboardEvent) => {
      if ((e.key.toLowerCase() === 'k' && (e.ctrlKey || e.metaKey)) || (e.key === '/' && !isTyping(e))) {
        e.preventDefault()
        input.current?.focus()
        setOpen(true)
      }
    }
    window.addEventListener('keydown', h)
    return () => window.removeEventListener('keydown', h)
  }, [autoFocus])

  useEffect(() => {
    const onDown = (e: MouseEvent) => {
      if (wrap.current && !wrap.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    return () => document.removeEventListener('mousedown', onDown)
  }, [])

  useEffect(() => {
    const term = q.trim()
    if (term.length < 2) {
      setRes(null)
      return
    }
    setLoading(true)
    const t = window.setTimeout(async () => {
      try {
        setRes(await api.get<SearchResults>(`${cp}/search?q=${encodeURIComponent(term)}`))
        setHl(0)
      } catch {
        setRes(null)
      } finally {
        setLoading(false)
      }
    }, 200)
    return () => window.clearTimeout(t)
  }, [q, cp])

  const hits = useMemo<Hit[]>(() => {
    const term = q.trim().toLowerCase()
    const out: Hit[] = []
    if (term.length >= 1) {
      for (const c of commands) {
        if (c.perm && !s.can(c.perm)) continue
        if ((c.label + ' ' + c.keywords).toLowerCase().includes(term)) {
          out.push({ key: 'c' + c.to, group: 'Go to', icon: c.icon, title: c.label, sub: '', to: c.to })
        }
      }
    }
    if (res) {
      for (const i of res.invoices)
        out.push({
          key: 'i' + i.id,
          group: 'Invoices',
          icon: FileText,
          title: `${i.internalNo} · ${i.buyerName || 'Walk-in buyer'}`,
          sub: `${dateFmt(i.invoiceDate)} · Rs ${money(i.totals?.totalValue)} · ${statusLabels[i.status] ?? i.status}${i.fbrInvoiceNumber ? ' · ' + i.fbrInvoiceNumber : ''}`,
          to: `/invoices/${i.id}`,
        })
      for (const c of res.customers)
        out.push({
          key: 'b' + c.id,
          group: 'Buyers',
          icon: Users,
          title: c.name,
          sub: [c.ntnCnic, c.registrationType, c.province].filter(Boolean).join(' · '),
          to: `/customers?q=${encodeURIComponent(c.ntnCnic || c.name)}`,
        })
      for (const p of res.products)
        out.push({
          key: 'p' + p.id,
          group: 'Products & services',
          icon: Package,
          title: p.description,
          sub: [p.hsCode, p.saleType, p.rate].filter(Boolean).join(' · '),
          to: `/products?q=${encodeURIComponent(p.code || p.description)}`,
        })
      for (const h of res.hsCodes)
        out.push({
          key: 'h' + h.code,
          group: 'FBR HS codes',
          icon: KeyRound,
          title: h.code,
          sub: h.description,
          to: `/library?tab=hs&q=${encodeURIComponent(h.code)}`,
        })
    }
    return out
  }, [q, res, s])

  const go = (h: Hit) => {
    navigate(h.to)
    setOpen(false)
    setQ('')
    input.current?.blur()
    onDone?.()
  }

  const onKey = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setHl((h) => Math.min(h + 1, hits.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setHl((h) => Math.max(h - 1, 0))
    } else if (e.key === 'Enter' && hits[hl]) {
      e.preventDefault()
      go(hits[hl])
    } else if (e.key === 'Escape') {
      setOpen(false)
      input.current?.blur()
      onDone?.()
    }
  }

  let lastGroup = ''
  const showPanel = open && q.trim().length >= 1
  return (
    <div className="search" ref={wrap}>
      <div className="search-box">
        <Search size={17} />
        <input
          ref={input}
          value={q}
          placeholder="Search invoices, buyers, products, HS codes…"
          aria-label="Search"
          onChange={(e) => {
            setQ(e.target.value)
            setOpen(true)
          }}
          onFocus={() => setOpen(true)}
          onKeyDown={onKey}
        />
        {loading ? <span className="spinner" style={{ width: 14, height: 14 }} /> : <span className="kbd">Ctrl K</span>}
      </div>
      {showPanel && (
        <div className="search-results" role="listbox">
          {hits.length === 0 ? (
            <div className="sr-empty">{loading || q.trim().length < 2 ? 'Keep typing to search…' : `Nothing found for “${q.trim()}”.`}</div>
          ) : (
            hits.map((h, i) => {
              const head = h.group !== lastGroup ? <div className="sr-group">{h.group}</div> : null
              lastGroup = h.group
              const Ic = h.icon
              return (
                <div key={h.key}>
                  {head}
                  <div
                    className={'sr-item' + (i === hl ? ' hl' : '')}
                    role="option"
                    aria-selected={i === hl}
                    onMouseEnter={() => setHl(i)}
                    onMouseDown={(e) => {
                      e.preventDefault()
                      go(h)
                    }}
                  >
                    <span className="sr-ic">
                      <Ic size={16} />
                    </span>
                    <span className="sr-main">
                      <div className="sr-title">{h.title}</div>
                      {h.sub && <div className="sr-sub">{h.sub}</div>}
                    </span>
                  </div>
                </div>
              )
            })
          )}
        </div>
      )}
    </div>
  )
}

// ---- User menu ----

function initials(name: string) {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  return (parts[0][0] + (parts.length > 1 ? parts[parts.length - 1][0] : '')).toUpperCase()
}

const roleLabels: Record<string, string> = {
  admin: 'Administrator',
  manager: 'Manager',
  accountant: 'Accountant',
  operator: 'Operator',
  auditor: 'Auditor',
}

export function UserMenu() {
  const s = useSession()
  const pop = usePopover()
  const [theme, setTheme] = useState<Theme>(getTheme())
  const name = s.user.fullName || s.user.username
  const pick = (t: Theme) => {
    setTheme(t)
    applyTheme(t)
  }
  return (
    <div className="pop-anchor" ref={pop.ref}>
      <button className="avatar-btn" aria-label="Account menu" aria-expanded={pop.open} onClick={() => pop.setOpen(!pop.open)}>
        <span className="avatar sm">{initials(name)}</span>
      </button>
      {pop.open && (
        <div className="popover" style={{ width: 300 }} role="menu">
          <div className="user-chip">
            <span className="avatar">{initials(name)}</span>
            <span style={{ minWidth: 0 }}>
              <div className="truncate" style={{ fontWeight: 650 }}>
                {name}
              </div>
              <div className="small faint truncate">
                {roleLabels[s.user.role] ?? s.user.role} · {s.user.username}
              </div>
            </span>
          </div>
          <div style={{ padding: '12px 16px 4px' }}>
            <div className="small faint" style={{ marginBottom: 6 }}>
              Appearance
            </div>
            <div className="seg" role="group" aria-label="Theme">
              <button className={theme === 'light' ? 'on' : ''} onClick={() => pick('light')}>
                <Sun size={14} /> Light
              </button>
              <button className={theme === 'dark' ? 'on' : ''} onClick={() => pick('dark')}>
                <Moon size={14} /> Dark
              </button>
              <button className={theme === 'system' ? 'on' : ''} onClick={() => pick('system')}>
                <Monitor size={14} /> Auto
              </button>
            </div>
          </div>
          <div className="menu-list">
            <Link to="/password" onClick={() => pop.setOpen(false)}>
              <KeyRound size={16} /> Change password
            </Link>
            <Link to="/mobile" onClick={() => pop.setOpen(false)}>
              <Smartphone size={16} /> Mobile app & access
            </Link>
            <Link to="/help" onClick={() => pop.setOpen(false)}>
              <LifeBuoy size={16} /> Help & FBR error codes
            </Link>
            <div className="menu-sep" />
            <button onClick={s.logout}>
              <LogOut size={16} /> Log out
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
