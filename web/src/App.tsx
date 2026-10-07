// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useCallback, useEffect, useState, type ReactElement } from 'react'
import { Navigate, NavLink, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import {
  Building2,
  CalendarClock,
  ChartColumnBig,
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
  PlugZap,
  Plus,
  Printer,
  ScrollText,
  Search,
  Server,
  ShieldAlert,
  Smartphone,
  Upload,
  UserCog,
  Users as UsersIcon,
  type LucideIcon,
} from 'lucide-react'
import { api, ApiError, onUnauthorized, setCsrf } from './api'
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
import { BrandMark } from './components/Brand'
import { AlertsBell, AlertsProvider, GlobalSearch, UserMenu, useAlerts } from './components/Header'

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

function Shell() {
  const s = useSession()
  const c = s.company
  const location = useLocation()
  const [navOpen, setNavOpen] = useState(false)
  const [searchOpen, setSearchOpen] = useState(false)
  useEffect(() => {
    setNavOpen(false)
    setSearchOpen(false)
  }, [location.pathname])

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

  return (
    <AlertsProvider>
      <div className={'app' + (navOpen ? ' nav-open' : '')}>
        <div className="nav-backdrop" onClick={() => setNavOpen(false)} />
        <Sidebar />
        <div className="main">
          <EnvBanner />
          <header className="topbar">
            <button className="icon-btn menu-btn" aria-label="Open menu" onClick={() => setNavOpen(true)}>
              <Menu size={21} />
            </button>
            <div className="topbar-title">
              <BrandMark size={26} />
              <span>{c.name}</span>
            </div>
            <GlobalSearch />
            <div className="spacer hide-mobile" />
            <span className={'env-ribbon env-' + c.environment} title="Working environment of this company">
              {envLabels[c.environment]}
            </span>
            {s.can('invoice.write') && (
              <NavLink to="/invoices/new" className="btn btn-primary hide-mobile">
                <Plus size={16} /> New invoice
              </NavLink>
            )}
            <button className="icon-btn mobile-only" aria-label="Search" onClick={() => setSearchOpen(true)}>
              <Search size={20} />
            </button>
            <AlertsBell />
            <UserMenu />
          </header>
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
          <main className="content">
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/invoices" element={<Invoices />} />
              <Route path="/invoices/new" element={<InvoiceEditor key="new" />} />
              <Route path="/invoices/:id" element={<InvoiceView />} />
              <Route path="/invoices/:id/edit" element={<InvoiceEditor />} />
              <Route path="/import" element={<ImportPage />} />
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
      </div>
    </AlertsProvider>
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

function Sidebar() {
  const s = useSession()
  const c = s.company!
  const { alerts } = useAlerts()
  const attention = alerts.filter((a) => a.id === 'rejected' || a.id === 'uncertain' || a.id === 'pending').length
  const sections: { title?: string; items: NavItem[] }[] = [
    { items: [{ to: '/', label: 'Dashboard', icon: LayoutDashboard, end: true }] },
    {
      title: 'Sales',
      items: [
        { to: '/invoices/new', label: 'New invoice', icon: FilePlus2, perm: 'invoice.write' },
        { to: '/invoices', label: 'Invoices', icon: FileText, end: true, badge: attention },
        { to: '/import', label: 'Import (CSV / Excel)', icon: Upload, perm: 'invoice.write' },
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
  return (
    <aside className="sidebar" aria-label="Main menu">
      <div className="sidebar-brand">
        <NavLink to="/" className="lockup" aria-label="Veridian E-invoicing Pakistan — dashboard">
          <BrandMark size={38} />
          <span className="lockup-text">
            <span className="lockup-name">Veridian</span>
            <span className="lockup-tag">E-invoicing Pakistan</span>
          </span>
        </NavLink>
      </div>
      <div className="workspace">
        <select value={c.id} onChange={(e) => s.setCompanyId(Number(e.target.value))} aria-label="Company">
          {s.companies.map((x) => (
            <option key={x.id} value={x.id}>
              {x.name}
            </option>
          ))}
        </select>
        <ChevronsUpDown size={15} className="chev" />
        <div className={'workspace-meta ' + c.environment}>
          <span className="dot" />
          <span className="truncate">
            {envLabels[c.environment]} · NTN {c.ntnCnic}
          </span>
        </div>
      </div>
      <nav className="nav">
        {sections.map((sec, i) => {
          const items = sec.items.filter((it) => !it.perm || s.can(it.perm))
          if (items.length === 0) return null
          return (
            <div key={i}>
              {sec.title && <div className="nav-section">{sec.title}</div>}
              {items.map((it) => {
                const Ic = it.icon
                return (
                  <NavLink key={it.to} to={it.to} end={it.end}>
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
      <div className="sidebar-foot">
        <b>{s.meta.product}</b> v{s.meta.version}
        <br />
        {s.meta.developedBy}
        <br />
        {s.meta.copyright}
      </div>
    </aside>
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
