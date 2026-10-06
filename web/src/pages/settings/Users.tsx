import { Fragment, useState } from 'react'
import { api, errorMessage } from '../../api'
import { ErrorBox, Field, Modal, Spinner, useLoad } from '../../components/ui'
import { dateTimeFmt } from '../../format'
import { useSession, useToast } from '../../state'
import type { User } from '../../types'

const roleInfo: Record<string, string> = {
  admin: 'Everything, including users, licence, backups and API keys',
  manager: 'Company settings, FBR tokens, invoices, cancellations, scenarios, reports, audit',
  accountant: 'Masters, invoices, debit notes, cancellations, reports, incidents',
  operator: 'Create and submit invoices only (counter / POS staff)',
  auditor: 'Read-only access to invoices, reports and the audit trail',
}

interface Form {
  id?: number
  username: string
  fullName: string
  email: string
  role: string
  password: string
  allCompanies: boolean
  companyIds: number[]
  active: boolean
}

export default function Users() {
  const s = useSession()
  const { data, error, loading, reload } = useLoad(() => api.get<User[]>('/users'), [])
  const [editing, setEditing] = useState<Form | null>(null)

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Users & roles</h1>
          <p>Each person should have their own login so that the audit trail shows who issued, changed or cancelled an invoice.</p>
        </div>
        <div className="actions">
          <button
            className="btn btn-primary"
            onClick={() => setEditing({ username: '', fullName: '', email: '', role: 'operator', password: '', allCompanies: true, companyIds: [], active: true })}
          >
            + New user
          </button>
        </div>
      </div>
      <ErrorBox error={error} />
      {loading && !data ? (
        <Spinner />
      ) : (
        data && (
          <div className="card">
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>Username</th>
                    <th>Name</th>
                    <th>Role</th>
                    <th>Companies</th>
                    <th>Last login</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {data.map((u) => (
                    <tr
                      key={u.id}
                      className="clickable"
                      onClick={() =>
                        setEditing({
                          id: u.id,
                          username: u.username,
                          fullName: u.fullName,
                          email: u.email,
                          role: u.role,
                          password: '',
                          allCompanies: u.allCompanies,
                          companyIds: u.companyIds ?? [],
                          active: u.active,
                        })
                      }
                    >
                      <td>
                        <b>{u.username}</b>
                      </td>
                      <td>
                        {u.fullName}
                        <div className="small faint">{u.email}</div>
                      </td>
                      <td>{u.role}</td>
                      <td className="small">
                        {u.allCompanies ? 'All' : (u.companyIds ?? []).map((id) => s.companies.find((c) => c.id === id)?.name ?? `#${id}`).join(', ')}
                      </td>
                      <td className="small">{dateTimeFmt(u.lastLoginAt) || '—'}</td>
                      <td>
                        {u.active ? <span className="badge b-green">active</span> : <span className="badge b-gray">disabled</span>}{' '}
                        {u.mustChangePassword && <span className="badge b-amber">must change password</span>}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )
      )}
      <div className="card card-pad mt">
        <h3>Roles</h3>
        <dl className="kv">
          {Object.entries(roleInfo).map(([r, d]) => (
            <Fragment key={r}>
              <dt>{r}</dt>
              <dd>{d}</dd>
            </Fragment>
          ))}
        </dl>
      </div>
      {editing && (
        <UserForm
          initial={editing}
          onClose={() => setEditing(null)}
          onSaved={() => {
            setEditing(null)
            reload()
          }}
        />
      )}
    </>
  )
}

function UserForm({ initial, onClose, onSaved }: { initial: Form; onClose: () => void; onSaved: () => void }) {
  const s = useSession()
  const toast = useToast()
  const [f, setF] = useState<Form>(initial)
  const [error, setError] = useState('')
  const set = (k: keyof Form, v: unknown) => setF((x) => ({ ...x, [k]: v }))
  const isSelf = f.id === s.user.id

  const save = async () => {
    setError('')
    try {
      const body = { ...f, password: f.password || undefined }
      if (f.id) await api.put(`/users/${f.id}`, body)
      else await api.post('/users', body)
      toast('ok', `User ${f.username} saved`)
      onSaved()
    } catch (e) {
      setError(errorMessage(e))
    }
  }

  return (
    <Modal
      title={f.id ? `Edit user — ${initial.username}` : 'New user'}
      onClose={onClose}
      footer={
        <>
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn btn-primary" onClick={save} disabled={!f.username || (!f.id && !f.password)}>
            Save
          </button>
        </>
      }
    >
      {error && <div className="alert alert-error">{error}</div>}
      <div className="form-grid">
        <Field label="Username *">
          <input value={f.username} onChange={(e) => set('username', e.target.value.trim())} disabled={!!f.id} />
        </Field>
        <Field label="Full name">
          <input value={f.fullName} onChange={(e) => set('fullName', e.target.value)} />
        </Field>
        <Field label="Email" span={2}>
          <input type="email" value={f.email} onChange={(e) => set('email', e.target.value)} />
        </Field>
        <Field label="Role *" hint={roleInfo[f.role]} span={2}>
          <select value={f.role} onChange={(e) => set('role', e.target.value)} disabled={isSelf}>
            {s.meta.roles.map((r) => (
              <option key={r}>{r}</option>
            ))}
          </select>
        </Field>
        <Field
          label={f.id ? 'Reset password' : 'Initial password *'}
          hint={f.id ? 'Leave empty to keep the current password. The user must change a reset password at next login.' : 'The user must change it at first login'}
          span={2}
        >
          <input type="password" autoComplete="new-password" value={f.password} onChange={(e) => set('password', e.target.value)} />
        </Field>
        <label className="check">
          <input type="checkbox" checked={f.allCompanies} onChange={(e) => set('allCompanies', e.target.checked)} /> Access to all companies
        </label>
        {f.id && (
          <label className="check">
            <input type="checkbox" checked={f.active} onChange={(e) => set('active', e.target.checked)} disabled={isSelf} /> Active
          </label>
        )}
        {!f.allCompanies && (
          <div className="field" style={{ gridColumn: 'span 2' }}>
            <label>Companies</label>
            {s.companies.map((c) => (
              <label key={c.id} className="check">
                <input
                  type="checkbox"
                  checked={f.companyIds.includes(c.id)}
                  onChange={(e) => set('companyIds', e.target.checked ? [...f.companyIds, c.id] : f.companyIds.filter((x) => x !== c.id))}
                />{' '}
                {c.name} ({c.ntnCnic})
              </label>
            ))}
          </div>
        )}
      </div>
    </Modal>
  )
}
