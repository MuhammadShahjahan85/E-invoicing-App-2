// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useCallback, useEffect, useState, type ReactElement } from 'react'
import { Link, Navigate, NavLink, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import {
  Building2,
  CalendarClock,
  CalendarDays,
  ChartColumnBig,
  Check,
  ChevronsUpDown,
  FilePlus2,
  FileText,
  FlaskConical,
  GraduationCap,
  KeyRound,
  LayoutDashboard,
  Library,
  LifeBuoy,
  Menu,
  Package,
  PanelLeftClose,
  PanelLeftOpen,
  PlugZap,
  Plus,
  Printer,
  RadioTower,
  ScrollText,
  Search,
  Server,
  ShieldAlert,
  Smartphone,
  Truck,
  Upload,
  UserCog,
  Users as UsersIcon,
  type LucideIcon,
} from 'lucide-react'
import { api, ApiError, onBusy, onUnauthorized, setCsrf } from './api'
import { SessionProvider, useSession } from './state'
import { envLabels } from './format'
import type { User } from './types'
import Login from './pages/Login'
import Setup from './pages/Setup'
import ChangePassword from './pages/ChangePassword'
import Dashboard from './pages/Dashboard'
import Invoices from './pages/Invoices'
import InvoiceEditor from './pages/InvoiceEditor'
import InvoiceView from './pages/InvoiceView'
import Customers from './pages/Customers'
import Products from './pages/Products'
import Scenarios from './pages/Scenarios'
import Reports from './pages/Reports'
import ImportPage from './pages/ImportPage'
import Incidents from './pages/Incidents'
import Help from './pages/Help'
import Audit from './pages/Audit'
import CompanySettings from './pages/settings/CompanySettings'
import FBRSettings from './pages/settings/FBRSettings'
import PrintSettingsPage from './pages/settings/PrintSettingsPage'
import Users from './pages/settings/Users'
import ApiKeys from './pages/settings/ApiKeys'
import SystemPage from './pages/settings/SystemPage'
import MobileAccess from './pages/MobileAccess'
import ReferenceLibrary from './pages/ReferenceLibrary'
import Compliance from './pages/Compliance'
import StockTransfers from './pages/StockTransfers'
import { BrandMark } from './components/Brand'
import { AlertsBell, AlertsProvider, GlobalSearch, StatusChip, ThemeToggle, UserMenu, useAlerts } from './components/Header'
import { usePopover } from './components/Popover'
import { usePrefs } from './prefs'
import { urdu } from './urdu'
import { ShortcutsHelp, useShortcuts } from './components/Shortcuts'

interface Me {
  user: User
  csrf: string
  permissions: string[]
}

type Phase = { kind: 'loading' } | { kind: 'setup' } | { kind: 'login' } | { kind: 'app'; me: Me } | { kind: 'error'; message: string }

export default function App() {
  const [phase, setPhase] = useState<Phase>({ kind: 'loading' })

  const boot = useCallback(async () => {
    try {
      const st = await api.get<{ needsSetup: boolean }>('/system/status')
      if (st.needsSetup) {
        setPhase({ kind: 'setup' })
        return
      }
      try {
        const me = await api.get<Me>('/auth/me')
        setCsrf(me.csrf)
        setPhase({ kind: 'app', me })
      } catch (e) {
        if (e instanceof ApiError && e.status === 401) setPhase({ kind: 'login' })
        else throw e
      }
    } catch (e) {
      setPhase({ kind: 'error', message: e instanceof Error ? e.message : String(e) })
    }
  }, [])

  useEffect(() => {
    boot()
    onUnauthorized(() => setPhase({ kind: 'login' }))
  }, [boot])

  switch (phase.kind) {
    case 'loading':
      return (
        <div className="center-page">
          <div className="stack" style={{ alignItems: 'center', gap: 16 }}>
            <BrandMark size={56} title="Veridian" />
            <span className="spinner" />
          </div>
        </div>
      )
    case 'error':
      return (
        <div className="center-page">
          <div className="card card-pad auth-card">
            <BrandMark size={44} title="Veridian" />
            <h2 style={{ marginTop: 14 }}>Cannot reach the server</h2>
            <p className="muted">{phase.message}</p>
            <button className="btn" onClick={boot}>
              Retry
            </button>
          </div>
        </div>
      )
    case 'setup':
      return <Setup onDone={() => setPhase({ kind: 'login' })} />
    case 'login':
      return (
        <Login
          onLogin={(me) => {
            setCsrf(me.csrf)
            setPhase({ kind: 'app', me })
          }}
        />
      )
    case 'app':
      if (phase.me.user.mustChangePassword) {
        return <ChangePassword forced onDone={() => boot()} />
      }
      return (
        <SessionProvider me={phase.me} onLogout={() => setPhase({ kind: 'login' })}>
          <Shell />
        </SessionProvider>
      )
  }
}

interface NavItem {
  to: string
  label: string
  icon: LucideIcon
  end?: boolean
  perm?: string
  badge?: number
}

function readCollapsed() {
  try {
    return localStorage.getItem('sidebar') === 'collapsed'
  } catch {
    return false
  }
}

function Shell() {
  const s = useSession()
  const c = s.company
  const location = useLocation()
  const [navOpen, setNavOpen] = useState(false)
  const [searchOpen, setSearchOpen] = useState(false)
  const [collapsed, setCollapsed] = useState(readCollapsed)
  const prefs = usePrefs()
  const shortcuts = useShortcuts()
  useEffect(() => {
    setNavOpen(false)
    setSearchOpen(false)
  }, [location.pathname])
  const toggleCollapsed = () => {
    setCollapsed((v) => {
      try {
        localStorage.setItem('sidebar', v ? 'open' : 'collapsed')
      } catch {
        // storage unavailable: the choice lasts for this visit
      }
      return !v
    })
  }

  if (!c) {
    return (
      <div className="center-page">
        <div className="card card-pad auth-card">
          <h2>No company available</h2>
          <p className="muted">Your account has no access to any company. Ask an administrator to grant access.</p>
          <button className="btn" onClick={s.logout}>
            Log out
          </button>
        </div>
      </div>
    )
  }

  const page = pageTitle(location.pathname)
  return (
    <AlertsProvider>
      <div className={'app' + (navOpen ? ' nav-open' : '') + (collapsed ? ' collapsed' : '')}>
        <div className="nav-backdrop" onClick={() => setNavOpen(false)} />
        <Sidebar collapsed={collapsed} onCollapse={toggleCollapsed} />
        <div className="main">
          <EnvBanner />
          <header className="topbar">
            <button className="icon-btn menu-btn" aria-label="Open menu" onClick={() => setNavOpen(true)}>
              <Menu size={21} />
            </button>
            <div className="tb-title">
              <span className="tb-crumb">
                {c.name} · {page.section}
              </span>
              <span className="tb-h">
                {page.title}
                {prefs.urdu && urdu[page.title] && (
                  <span className="tb-ur" lang="ur">
                    {urdu[page.title]}
                  </span>
                )}
              </span>
            </div>
            <div className="spacer" />
            <span className="tb-chip" title="Today's date in Pakistan (PKT)">
              <CalendarDays size={16} /> As at <b>{asAtToday()}</b>
            </span>
            <StatusChip />
            <GlobalSearch />
            <button className="icon-btn mobile-only" aria-label="Search" onClick={() => setSearchOpen(true)}>
              <Search size={20} />
            </button>
            <ThemeToggle />
            <AlertsBell />
            <UserMenu />
          </header>
          <Flowbar />
          {searchOpen && (
            <div className="search-sheet">
              <div className="row" style={{ flexWrap: 'nowrap' }}>
                <GlobalSearch autoFocus onDone={() => setSearchOpen(false)} />
                <button className="btn btn-ghost" onClick={() => setSearchOpen(false)}>
                  Cancel
                </button>
              </div>
            </div>
          )}
          <main className="content" key={prefs.grouping}>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/invoices" element={<Invoices />} />
              <Route path="/invoices/new" element={<InvoiceEditor key="new" />} />
              <Route path="/invoices/:id" element={<InvoiceView />} />
              <Route path="/invoices/:id/edit" element={<InvoiceEditor />} />
              <Route path="/import" element={<ImportPage />} />
              <Route path="/stock-transfers" element={<StockTransfers />} />
              <Route path="/customers" element={<Customers />} />
              <Route path="/products" element={<Products />} />
              <Route path="/library" element={<ReferenceLibrary />} />
              <Route path="/compliance" element={<RequirePerm perm="reports"><Compliance /></RequirePerm>} />
              <Route path="/scenarios" element={<Scenarios />} />
              <Route path="/reports" element={<RequirePerm perm="reports"><Reports /></RequirePerm>} />
              <Route path="/incidents" element={<Incidents />} />
              <Route path="/audit" element={<RequirePerm perm="audit"><Audit /></RequirePerm>} />
              <Route path="/help" element={<Help />} />
              <Route path="/mobile" element={<MobileAccess />} />
              <Route path="/password" element={<ChangePasswordPage />} />
              <Route path="/settings/company" element={<CompanySettings />} />
              <Route path="/settings/fbr" element={<FBRSettings />} />
              <Route path="/settings/printing" element={<PrintSettingsPage />} />
              <Route path="/settings/users" element={<RequirePerm perm="users"><Users /></RequirePerm>} />
              <Route path="/settings/api-keys" element={<RequirePerm perm="apikeys"><ApiKeys /></RequirePerm>} />
              <Route path="/settings/system" element={<RequirePerm perm="system"><SystemPage /></RequirePerm>} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </main>
        </div>
        <TabBar onMenu={() => setNavOpen(true)} />
        {shortcuts.open && <ShortcutsHelp onClose={() => shortcuts.setOpen(false)} />}
      </div>
    </AlertsProvider>
  )
}

/** asAtToday is today's date in Pakistan, e.g. "09 Oct 2026". */
function asAtToday() {
  return new Date().toLocaleDateString('en-GB', { timeZone: 'Asia/Karachi', day: '2-digit', month: 'short', year: 'numeric' })
}

/** Flowbar is the thin line under the top bar; it flows while the app is talking to the server. */
function Flowbar() {
  const [busy, setBusy] = useState(false)
  useEffect(() => {
    let t: number | undefined
    const off = onBusy((b) => {
      window.clearTimeout(t)
      // Short requests do not flicker the line; it lingers briefly once they finish.
      t = window.setTimeout(() => setBusy(b), b ? 180 : 300)
    })
    return () => {
      off()
      window.clearTimeout(t)
    }
  }, [])
  return (
    <div className={'flowbar' + (busy ? ' busy' : '')} aria-hidden="true">
      <span />
      <span />
      <span />
      <span />
    </div>
  )
}

function ChangePasswordPage() {
  const navigate = useNavigate()
  return <ChangePassword onDone={() => navigate('/')} />
}

function EnvBanner() {
  const { company } = useSession()
  const env = company?.environment ?? 'simulator'
  if (env === 'production') return null
  return (
    <div className={'env-banner ' + env}>
      {env === 'sandbox' ? <FlaskConical size={15} /> : <GraduationCap size={15} />}
      {env === 'sandbox'
        ? 'FBR SANDBOX — invoices are test submissions for scenario certification, not tax invoices.'
        : 'TRAINING SIMULATOR — nothing is sent to FBR. Switch to production in FBR integration settings when ready.'}
    </div>
  )
}

const envWorkspace: Record<string, { title: string; icon: LucideIcon }> = {
  production: { title: 'FBR production', icon: RadioTower },
  sandbox: { title: 'FBR sandbox', icon: FlaskConical },
  simulator: { title: 'Training simulator', icon: GraduationCap },
}

function navSections(attention: number): { title?: string; items: NavItem[] }[] {
  return [
    { items: [{ to: '/', label: 'Dashboard', icon: LayoutDashboard, end: true }] },
    {
      title: 'Sales',
      items: [
        { to: '/invoices/new', label: 'New invoice', icon: FilePlus2, perm: 'invoice.write' },
        { to: '/invoices', label: 'Invoices', icon: FileText, end: true, badge: attention },
        { to: '/import', label: 'Import data', icon: Upload, perm: 'invoice.write' },
        { to: '/stock-transfers', label: 'Stock transfer notes', icon: Truck },
      ],
    },
    {
      title: 'Masters',
      items: [
        { to: '/customers', label: 'Customers (buyers)', icon: UsersIcon },
        { to: '/products', label: 'Products & services', icon: Package },
      ],
    },
    {
      title: 'FBR & compliance',
      items: [
        { to: '/compliance', label: 'Tax periods & returns', icon: CalendarClock, perm: 'reports' },
        { to: '/library', label: 'FBR reference library', icon: Library },
        { to: '/scenarios', label: 'Sandbox scenarios', icon: FlaskConical },
        { to: '/reports', label: 'Reports', icon: ChartColumnBig, perm: 'reports' },
        { to: '/incidents', label: 'Incident register', icon: ShieldAlert },
        { to: '/audit', label: 'Audit trail', icon: ScrollText, perm: 'audit' },
      ],
    },
    {
      title: 'Settings',
      items: [
        { to: '/settings/company', label: 'Company', icon: Building2 },
        { to: '/settings/fbr', label: 'FBR integration', icon: PlugZap },
        { to: '/settings/printing', label: 'Invoice printing', icon: Printer },
        { to: '/settings/users', label: 'Users & roles', icon: UserCog, perm: 'users' },
        { to: '/settings/api-keys', label: 'ERP API keys', icon: KeyRound, perm: 'apikeys' },
        { to: '/settings/system', label: 'System & licence', icon: Server, perm: 'system' },
      ],
    },
    {
      title: 'Support',
      items: [
        { to: '/mobile', label: 'Mobile app & access', icon: Smartphone },
        { to: '/help', label: 'Help & error codes', icon: LifeBuoy },
      ],
    },
  ]
}

/** pageTitle names the page shown in the top bar, with its menu section. */
function pageTitle(path: string): { section: string; title: string } {
  if (path === '/') return { section: 'Overview', title: 'Sales tax overview' }
  if (/^\/invoices\/\d+\/edit$/.test(path)) return { section: 'Sales', title: 'Edit invoice' }
  if (/^\/invoices\/\d+$/.test(path)) return { section: 'Sales', title: 'Invoice' }
  if (path === '/password') return { section: 'Account', title: 'Change password' }
  for (const sec of navSections(0)) {
    for (const it of sec.items) {
      if (it.to === path) return { section: sec.title ?? 'Overview', title: it.label }
    }
  }
  return { section: 'Veridian', title: 'E-invoicing Pakistan' }
}

/** coInitials gives the two-letter tile of a company, e.g. "Acme Textiles (Pvt) Ltd" → "AT". */
function coInitials(name: string) {
  const words = name
    .replace(/\(.*?\)/g, ' ')
    .split(/[\s.&,-]+/)
    .filter((w) => w && !/^(m\/s|the|pvt|private|ltd|limited|llc|co|and|of)$/i.test(w))
  if (words.length === 0) return name.slice(0, 2).toUpperCase()
  return (words[0][0] + (words[1]?.[0] ?? words[0][1] ?? '')).toUpperCase()
}

function Sidebar({ collapsed, onCollapse }: { collapsed: boolean; onCollapse: () => void }) {
  const s = useSession()
  const prefs = usePrefs()
  const c = s.company!
  const { alerts } = useAlerts()
  const attention = alerts.filter((a) => a.id === 'rejected' || a.id === 'uncertain' || a.id === 'pending').length
  const ws = envWorkspace[c.environment] ?? envWorkspace.simulator
  const WsIcon = ws.icon
  return (
    <aside className="sidebar" aria-label="Main menu">
      <div className="sidebar-brand">
        <NavLink to="/" className="lockup" aria-label="Veridian E-invoicing Pakistan — dashboard">
          <BrandMark size={40} />
          <span className="lockup-text">
            <span className="lockup-name">Veridian</span>
            <span className="lockup-tag">E-invoicing Pakistan</span>
          </span>
        </NavLink>
      </div>
      <Link to="/settings/fbr" className={'ws-card ' + c.environment} title={`Working environment: ${envLabels[c.environment]}`}>
        <span className="ws-ic">
          <WsIcon size={17} />
        </span>
        <span className="ws-text">
          <span className="ws-title">{ws.title}</span>
          <span className="ws-sub">Digital Invoicing workspace</span>
        </span>
      </Link>
      <nav className="nav">
        {navSections(attention).map((sec, i) => {
          const items = sec.items.filter((it) => !it.perm || s.can(it.perm))
          if (items.length === 0) return null
          return (
            <div key={i}>
              {sec.title && (
                <div className="nav-section">
                  {sec.title}
                  {prefs.urdu && urdu[sec.title] && (
                    <span className="nav-ur" lang="ur">
                      {urdu[sec.title]}
                    </span>
                  )}
                </div>
              )}
              {items.map((it) => {
                const Ic = it.icon
                return (
                  <NavLink key={it.to} to={it.to} end={it.end} title={collapsed ? it.label : undefined}>
                    <Ic size={18} />
                    <span>{it.label}</span>
                    {it.badge ? <span className="count">{it.badge}</span> : null}
                  </NavLink>
                )
              })}
            </div>
          )
        })}
      </nav>
      <div className="sidebar-bottom">
        <CompanySwitcher />
        <div className="sidebar-credit">
          {s.meta.developedBy}
          <br />
          <b>{s.meta.product}</b> · v{s.meta.version}
        </div>
        <button className="collapse-btn" onClick={onCollapse} aria-label={collapsed ? 'Expand the menu' : 'Collapse the menu'}>
          {collapsed ? <PanelLeftOpen size={16} /> : <PanelLeftClose size={16} />}
          <span>Collapse</span>
        </button>
      </div>
    </aside>
  )
}

/** CompanySwitcher shows the current company at the foot of the menu and switches between companies. */
function CompanySwitcher() {
  const s = useSession()
  const c = s.company!
  const pop = usePopover()
  return (
    <div ref={pop.ref} style={{ position: 'relative' }}>
      {pop.open && (
        <div className="co-menu" role="menu" aria-label="Companies">
          {s.companies.map((x) => (
            <button
              key={x.id}
              role="menuitem"
              className={x.id === c.id ? 'on' : ''}
              onClick={() => {
                s.setCompanyId(x.id)
                pop.setOpen(false)
              }}
            >
              <span className="co-avatar">{coInitials(x.name)}</span>
              <span style={{ flex: 1, minWidth: 0 }}>
                <b className="truncate" style={{ display: 'block' }}>
                  {x.name}
                </b>
                <small>
                  NTN {x.ntnCnic} · {envLabels[x.environment]}
                </small>
              </span>
              {x.id === c.id && <Check size={16} color="#16a34a" />}
            </button>
          ))}
          <div className="menu-sep" />
          <Link to="/settings/company" className="co-link" onClick={() => pop.setOpen(false)}>
            <Building2 size={15} /> Company details
          </Link>
        </div>
      )}
      <button
        className="co-card"
        aria-haspopup="menu"
        aria-expanded={pop.open}
        aria-label={`Company: ${c.name}. Switch company`}
        title={c.name}
        onClick={() => pop.setOpen(!pop.open)}
      >
        <span className="co-avatar">{coInitials(c.name)}</span>
        <span className="co-text">
          <b>{c.name}</b>
          <small>
            NTN {c.ntnCnic}
            {c.city ? ' · ' + c.city : c.province ? ' · ' + c.province : ''}
          </small>
        </span>
        <ChevronsUpDown size={16} />
      </button>
    </div>
  )
}

function TabBar({ onMenu }: { onMenu: () => void }) {
  const s = useSession()
  return (
    <nav className="tabbar" aria-label="Quick navigation">
      <NavLink to="/" end>
        <LayoutDashboard size={21} />
        Home
      </NavLink>
      <NavLink to="/invoices" end>
        <FileText size={21} />
        Invoices
      </NavLink>
      {s.can('invoice.write') ? (
        <NavLink to="/invoices/new" aria-label="New invoice">
          <span className="fab">
            <Plus size={24} />
          </span>
        </NavLink>
      ) : (
        <NavLink to="/customers">
          <UsersIcon size={21} />
          Buyers
        </NavLink>
      )}
      <NavLink to="/library">
        <Library size={21} />
        FBR data
      </NavLink>
      <button onClick={onMenu}>
        <Menu size={21} />
        Menu
      </button>
    </nav>
  )
}

/** RequirePerm hides pages the user's role cannot use (e.g. after another user's URL is reused). */
function RequirePerm({ perm, children }: { perm: string; children: ReactElement }) {
  const s = useSession()
  if (s.can(perm)) return children
  return (
    <div className="card card-pad">
      <h2>Not available for your role</h2>
      <p className="muted">
        Your role ({s.user.role}) does not have access to this page. Ask an administrator if you need it.
      </p>
      <NavLink to="/" className="btn">
        Go to dashboard
      </NavLink>
    </div>
  )
}
