// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { api, setCsrf } from './api'
import type { Company, Meta, User } from './types'

// ---- Toasts ----
type ToastKind = 'ok' | 'err' | 'info'
interface Toast {
  id: number
  kind: ToastKind
  text: string
}
const ToastCtx = createContext<(kind: ToastKind, text: string) => void>(() => {})
export const useToast = () => useContext(ToastCtx)

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])
  const push = useCallback((kind: ToastKind, text: string) => {
    const id = Date.now() + Math.random()
    setToasts((t) => [...t, { id, kind, text }])
    setTimeout(() => setToasts((t) => t.filter((x) => x.id !== id)), kind === 'err' ? 8000 : 4000)
  }, [])
  return (
    <ToastCtx.Provider value={push}>
      {children}
      <div className="toasts" role="status">
        {toasts.map((t) => (
          <div key={t.id} className={'toast ' + t.kind}>
            {t.text}
          </div>
        ))}
      </div>
    </ToastCtx.Provider>
  )
}

// ---- Session / company ----
export interface Session {
  user: User
  permissions: string[]
  meta: Meta
  companies: Company[]
  company: Company | null
  setCompanyId: (id: number) => void
  reloadCompanies: () => Promise<void>
  updateCompany: (c: Company) => void
  can: (perm: string) => boolean
  logout: () => Promise<void>
}

const SessionCtx = createContext<Session | null>(null)

export function useSession(): Session {
  const s = useContext(SessionCtx)
  if (!s) throw new Error('no session')
  return s
}

/** Company id path prefix for API calls. */
export function useCompanyPath(): string {
  const { company } = useSession()
  return company ? `/companies/${company.id}` : '/companies/0'
}

interface MeResponse {
  user: User
  csrf: string
  permissions: string[]
}

export function SessionProvider({ me, onLogout, children }: { me: MeResponse; onLogout: () => void; children: ReactNode }) {
  const [meta, setMeta] = useState<Meta | null>(null)
  const [companies, setCompanies] = useState<Company[]>([])
  const [companyId, setCompanyIdState] = useState<number>(() => Number(localStorage.getItem('companyId') || 0))

  useEffect(() => {
    setCsrf(me.csrf)
  }, [me.csrf])

  const reloadCompanies = useCallback(async () => {
    const list = await api.get<Company[]>('/companies')
    setCompanies(list)
  }, [])

  useEffect(() => {
    api.get<Meta>('/meta').then(setMeta).catch(() => {})
    reloadCompanies().catch(() => {})
  }, [reloadCompanies])

  const company = useMemo(() => companies.find((c) => c.id === companyId) ?? companies[0] ?? null, [companies, companyId])

  const value: Session | null = meta
    ? {
        user: me.user,
        permissions: me.permissions,
        meta,
        companies,
        company,
        setCompanyId: (id: number) => {
          localStorage.setItem('companyId', String(id))
          setCompanyIdState(id)
        },
        reloadCompanies,
        updateCompany: (c: Company) => setCompanies((list) => list.map((x) => (x.id === c.id ? c : x))),
        can: (p: string) => me.permissions.includes(p),
        logout: async () => {
          try {
            await api.post('/auth/logout')
          } finally {
            onLogout()
          }
        },
      }
    : null

  if (!value) {
    return (
      <div className="center-page">
        <span className="spinner" />
      </div>
    )
  }
  return <SessionCtx.Provider value={value}>{children}</SessionCtx.Provider>
}
