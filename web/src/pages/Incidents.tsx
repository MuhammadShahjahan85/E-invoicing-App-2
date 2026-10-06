import { useState } from 'react'
import { api, errorMessage } from '../api'
import { Empty, ErrorBox, Field, Modal, Spinner, useLoad } from '../components/ui'
import { dateTimeFmt } from '../format'
import { useCompanyPath, useSession, useToast } from '../state'
import type { Incident } from '../types'

export const incidentKinds: Record<string, string> = {
  fbr_unreachable: 'Loss of connectivity with FBR',
  auth_failure: 'Security token / authorisation failure',
  system_failure: 'Hardware or software failure',
  power_failure: 'Power failure',
  tampering: 'Suspected tampering',
  other: 'Other disruption',
}

function toLocalInput(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return ''
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`
}

function fromLocalInput(v: string): string {
  if (!v) return ''
  const d = new Date(v)
  return isNaN(d.getTime()) ? '' : d.toISOString()
}

export default function Incidents() {
  const s = useSession()
  const cp = useCompanyPath()
  const { data, error, loading, reload } = useLoad(() => api.get<Incident[]>(`${cp}/incidents`), [cp])
  const [editing, setEditing] = useState<Partial<Incident> | null>(null)
  const canWrite = s.can('incidents')

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Incident register</h1>
          <p>
            Rule 150R of the Sales Tax Rules, 2006 requires a registered person to report any failure, disruption or tampering of the electronic invoicing system
            to the Commissioner within 24 hours. Connection and token failures are detected automatically.
          </p>
        </div>
        <div className="actions">
          {canWrite && (
            <button className="btn btn-primary" onClick={() => setEditing({ kind: 'system_failure', description: '', startedAt: new Date().toISOString() })}>
              + Record incident
            </button>
          )}
        </div>
      </div>
      <ErrorBox error={error} />
      {loading && !data ? (
        <Spinner />
      ) : data && data.length === 0 ? (
        <div className="card">
          <Empty>No incidents recorded.</Empty>
        </div>
      ) : (
        data && (
          <div className="card">
            <div className="table-wrap">
              <table className="table">
                <thead>
                  <tr>
                    <th>Nature</th>
                    <th>Started</th>
                    <th>Ended</th>
                    <th>Details</th>
                    <th>Reported to FBR</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {data.map((i) => {
                    const overdue = !i.reportedAt && Date.now() - new Date(i.startedAt).getTime() > 24 * 3600_000
                    return (
                      <tr key={i.id}>
                        <td>
                          {incidentKinds[i.kind] ?? i.kind}
                          {i.autoDetected && (
                            <div>
                              <span className="badge b-blue">auto-detected</span>
                            </div>
                          )}
                        </td>
                        <td className="nowrap">{dateTimeFmt(i.startedAt)}</td>
                        <td className="nowrap">{i.endedAt ? dateTimeFmt(i.endedAt) : <span className="badge b-amber">ongoing</span>}</td>
                        <td className="small" style={{ maxWidth: 380 }}>
                          {i.description}
                        </td>
                        <td className="small">
                          {i.reportedAt ? (
                            <>
                              <span className="badge b-green">reported</span> {dateTimeFmt(i.reportedAt)}
                              <div className="faint">{i.reportReference}</div>
                            </>
                          ) : (
                            <span className={'badge ' + (overdue ? 'b-red' : 'b-amber')}>{overdue ? 'overdue (24 h)' : 'not reported'}</span>
                          )}
                        </td>
                        <td className="nowrap">
                          <a className="btn btn-sm" href={`/api/v1${cp}/incidents/${i.id}/letter`} target="_blank" rel="noopener">
                            Letter
                          </a>{' '}
                          {canWrite && (
                            <button className="btn btn-sm" onClick={() => setEditing(i)}>
                              Edit
                            </button>
                          )}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )
      )}
      <div className="card card-pad mt">
        <h3>How to report</h3>
        <ol className="small">
          <li>Open the incident and print the intimation letter (it lists invoices issued during the disruption).</li>
          <li>Send it to your Commissioner Inland Revenue (RTO/LTO/CTO) within 24 hours — by email, IRIS correspondence or by hand — and keep the acknowledgement.</li>
          <li>Record the date and the reference of the intimation here. Invoices queued during an outage are sent automatically once FBR is reachable again.</li>
        </ol>
      </div>
      {editing && (
        <IncidentForm
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

function IncidentForm({ initial, onClose, onSaved }: { initial: Partial<Incident>; onClose: () => void; onSaved: () => void }) {
  const cp = useCompanyPath()
  const toast = useToast()
  const [i, setI] = useState<Partial<Incident>>(initial)
  const [error, setError] = useState('')
  const set = (k: keyof Incident, v: unknown) => setI((x) => ({ ...x, [k]: v }))
  const save = async () => {
    setError('')
    try {
      if (i.id) await api.put(`${cp}/incidents/${i.id}`, i)
      else await api.post(`${cp}/incidents`, i)
      toast('ok', 'Incident saved')
      onSaved()
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  return (
    <Modal
      wide
      title={i.id ? 'Edit incident' : 'Record incident'}
      onClose={onClose}
      footer={
        <>
          <button className="btn" onClick={onClose}>
            Cancel
          </button>
          <button className="btn btn-primary" onClick={save} disabled={!i.kind || !i.description}>
            Save
          </button>
        </>
      }
    >
      {error && <div className="alert alert-error">{error}</div>}
      <div className="form-grid">
        <Field label="Nature of incident *">
          <select value={i.kind} onChange={(e) => set('kind', e.target.value)} disabled={i.autoDetected}>
            {Object.entries(incidentKinds).map(([k, v]) => (
              <option key={k} value={k}>
                {v}
              </option>
            ))}
          </select>
        </Field>
        <div />
        <Field label="Started *">
          <input type="datetime-local" value={toLocalInput(i.startedAt ?? '')} onChange={(e) => set('startedAt', fromLocalInput(e.target.value))} />
        </Field>
        <Field label="Ended" hint="Leave empty while the disruption continues">
          <input type="datetime-local" value={toLocalInput(i.endedAt ?? '')} onChange={(e) => set('endedAt', fromLocalInput(e.target.value))} />
        </Field>
        <Field label="Details *" span={2}>
          <textarea rows={3} value={i.description ?? ''} onChange={(e) => set('description', e.target.value)} />
        </Field>
        <Field label="Reported to Commissioner on">
          <input type="datetime-local" value={toLocalInput(i.reportedAt ?? '')} onChange={(e) => set('reportedAt', fromLocalInput(e.target.value))} />
        </Field>
        <Field label="Intimation reference (letter / email / IRIS)">
          <input value={i.reportReference ?? ''} onChange={(e) => set('reportReference', e.target.value)} />
        </Field>
      </div>
    </Modal>
  )
}
