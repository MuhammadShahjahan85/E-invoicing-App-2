// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

import { useState } from 'react'
import { api, errorMessage, qs } from '../api'
import { Empty, ErrorBox, Pager, Spinner, useLoad } from '../components/ui'
import { dateTimeFmt } from '../format'
import { useSession } from '../state'
import type { AuditEntry, Paged } from '../types'

export default function Audit() {
  const s = useSession()
  const [entity, setEntity] = useState('')
  const [action, setAction] = useState('')
  const [from, setFrom] = useState('')
  const [to, setTo] = useState('')
  const [onlyCompany, setOnlyCompany] = useState(true)
  const [offset, setOffset] = useState(0)
  const [verify, setVerify] = useState<{ intact: boolean; brokenAt: number; checked: number } | null>(null)
  const [verifyError, setVerifyError] = useState('')
  const limit = 100
  // Users limited to some companies can only read their companies' entries.
  const restricted = !s.user.allCompanies && s.user.role !== 'admin'
  const companyId = onlyCompany || restricted ? s.company?.id : undefined
  const { data, error, loading } = useLoad(
    () => api.get<Paged<AuditEntry>>(`/audit${qs({ entity, action, from, to, companyId, limit, offset })}`),
    [entity, action, from, to, companyId, offset],
  )

  const runVerify = async () => {
    setVerifyError('')
    try {
      setVerify(await api.get(`/audit/verify`))
    } catch (e) {
      setVerifyError(errorMessage(e))
    }
  }

  const reset = (fn: () => void) => {
    fn()
    setOffset(0)
  }

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Audit trail</h1>
          <p>Append-only log of every action. Each entry is chained to the previous one by a SHA-256 hash, so any alteration or deletion is detectable.</p>
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={runVerify}>
            Verify integrity
          </button>
        </div>
      </div>
      {verifyError && <div className="alert alert-error">{verifyError}</div>}
      {verify && (
        <div className={'alert ' + (verify.intact ? 'alert-ok' : 'alert-error')}>
          {verify.intact
            ? `Audit chain intact — ${verify.checked} entries verified.`
            : `Audit chain broken at entry #${verify.brokenAt} (${verify.checked} entries checked). The database may have been altered outside the application.`}
        </div>
      )}
      <div className="card">
        <div className="card-body row">
          <select value={entity} onChange={(e) => reset(() => setEntity(e.target.value))} style={{ width: 'auto' }}>
            <option value="">All records</option>
            <option value="invoice">Invoices</option>
            <option value="customer">Customers</option>
            <option value="product">Products</option>
            <option value="company">Company</option>
            <option value="user">Users</option>
            <option value="api_key">API keys</option>
            <option value="settings">System settings</option>
            <option value="scenario">Scenarios</option>
            <option value="incident">Incidents</option>
            <option value="backup">Backups</option>
            <option value="license">Licence</option>
          </select>
          <input placeholder="Action starts with… (e.g. invoice.)" value={action} onChange={(e) => reset(() => setAction(e.target.value))} style={{ maxWidth: 220 }} />
          <input type="date" value={from} onChange={(e) => reset(() => setFrom(e.target.value))} style={{ width: 'auto' }} aria-label="From" />
          <input type="date" value={to} onChange={(e) => reset(() => setTo(e.target.value))} style={{ width: 'auto' }} aria-label="To" />
          {!restricted && (
            <label className="check">
              <input type="checkbox" checked={onlyCompany} onChange={(e) => reset(() => setOnlyCompany(e.target.checked))} /> Only {s.company?.name}
            </label>
          )}
        </div>
        <ErrorBox error={error} />
        {loading && !data ? (
          <div className="card-body">
            <Spinner />
          </div>
        ) : data && data.items.length === 0 ? (
          <Empty>No entries.</Empty>
        ) : (
          data && (
            <>
              <div className="table-wrap">
                <table className="table">
                  <thead>
                    <tr>
                      <th>#</th>
                      <th>Time</th>
                      <th>User</th>
                      <th>Action</th>
                      <th>Record</th>
                      <th>Details</th>
                      <th>IP</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.items.map((a) => (
                      <tr key={a.id}>
                        <td className="faint small">{a.id}</td>
                        <td className="nowrap">{dateTimeFmt(a.ts)}</td>
                        <td>{a.username || 'system'}</td>
                        <td className="mono small">{a.action}</td>
                        <td className="small">
                          {a.entity} {a.entityId}
                        </td>
                        <td className="small mono" style={{ maxWidth: 520, wordBreak: 'break-all' }}>
                          {a.details}
                        </td>
                        <td className="small faint">{a.ip}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <Pager total={data.total} limit={limit} offset={offset} onChange={setOffset} />
            </>
          )
        )}
      </div>
    </>
  )
}
