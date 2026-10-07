import { useCallback, useEffect, useState, type ReactElement } from 'react'
import { Navigate, NavLink, Route, Routes, useNavigate } from 'react-router-dom'
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
          <span className="spinner" />
        </div>
      )
    case 'error':
      return (
        <div className="center-page">
          <div className="card card-pad auth-card">
            <h2>Cannot reach the server</h2>
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

function Shell() {
  const s = useSession()
  const navigate = useNavigate()
  const c = s.company
  const env = c?.environment ?? 'simulator'

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
    <div className="app">
      <aside className="sidebar">
        <div className="brand">
          <div className="logo">e</div>
          <div>
            {s.meta.product}
            <small>FBR Digital Invoicing</small>
          </div>
        </div>
        <nav className="nav">
          <NavLink to="/" end>
            Dashboard
          </NavLink>
          <div className="nav-section">Sales</div>
          <NavLink to="/invoices/new">New invoice</NavLink>
          <NavLink to="/invoices" end>
            Invoices
          </NavLink>
          <NavLink to="/import">Import (CSV / Excel)</NavLink>
          <div className="nav-section">Masters</div>
          <NavLink to="/customers">Customers (buyers)</NavLink>
          <NavLink to="/products">Products & services</NavLink>
          <div className="nav-section">Compliance</div>
          <NavLink to="/scenarios">Sandbox scenarios</NavLink>
          {s.can('reports') && <NavLink to="/reports">Reports</NavLink>}
          <NavLink to="/incidents">Incident register</NavLink>
          {s.can('audit') && <NavLink to="/audit">Audit trail</NavLink>}
          <div className="nav-section">Settings</div>
          <NavLink to="/settings/company">Company</NavLink>
          <NavLink to="/settings/fbr">FBR integration</NavLink>
          <NavLink to="/settings/printing">Invoice printing</NavLink>
          {s.can('users') && <NavLink to="/settings/users">Users & roles</NavLink>}
          {s.can('apikeys') && <NavLink to="/settings/api-keys">ERP API keys</NavLink>}
          {s.can('system') && <NavLink to="/settings/system">System</NavLink>}
          <NavLink to="/help">Help & error codes</NavLink>
        </nav>
        <div className="sidebar-foot">
          v{s.meta.version}
          <br />
          {s.meta.vendor}
        </div>
      </aside>
      <div className="main">
        {env !== 'production' && (
          <div className={'env-banner ' + env}>
            {env === 'sandbox'
              ? 'FBR SANDBOX — invoices are test submissions for scenario certification, not tax invoices.'
              : 'TRAINING SIMULATOR — nothing is sent to FBR. Switch to production in FBR integration settings when ready.'}
          </div>
        )}
        <header className="topbar">
          <select value={c.id} onChange={(e) => s.setCompanyId(Number(e.target.value))} style={{ width: 'auto', maxWidth: 360 }} aria-label="Company">
            {s.companies.map((x) => (
              <option key={x.id} value={x.id}>
                {x.name} — {x.ntnCnic}
              </option>
            ))}
          </select>
          <span className={'env-ribbon env-' + env}>{envLabels[env]}</span>
          <div className="spacer" />
          <button className="btn btn-primary btn-sm" onClick={() => navigate('/invoices/new')}>
            + New invoice
          </button>
          <span className="muted small">
            {s.user.fullName || s.user.username} · {s.user.role}
          </span>
          <NavLink to="/password" className="btn btn-sm">
            Password
          </NavLink>
          <button className="btn btn-sm" onClick={s.logout}>
            Log out
          </button>
        </header>
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
            <Route path="/scenarios" element={<Scenarios />} />
            <Route path="/reports" element={<RequirePerm perm="reports"><Reports /></RequirePerm>} />
            <Route path="/incidents" element={<Incidents />} />
            <Route path="/audit" element={<RequirePerm perm="audit"><Audit /></RequirePerm>} />
            <Route path="/help" element={<Help />} />
            <Route path="/password" element={<ChangePassword onDone={() => navigate('/')} />} />
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
    </div>
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
