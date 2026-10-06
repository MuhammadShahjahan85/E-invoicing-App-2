import { useState } from 'react'
import { api, errorMessage } from '../../api'
import { Confirm, ErrorBox, Field, Modal, Spinner, useLoad } from '../../components/ui'
import { dateTimeFmt, todayPK } from '../../format'
import { useCompanyPath, useSession, useToast } from '../../state'
import type { APIKey } from '../../types'

export default function ApiKeys() {
  const s = useSession()
  const cp = useCompanyPath()
  const toast = useToast()
  const { data, error, loading, reload } = useLoad(() => api.get<APIKey[]>(`${cp}/api-keys`), [cp])
  const [creating, setCreating] = useState(false)
  const [name, setName] = useState('')
  const [created, setCreated] = useState<string | null>(null)
  const [revoke, setRevoke] = useState<APIKey | null>(null)
  const [formError, setFormError] = useState('')
  const base = `${window.location.origin}/api/v1${cp}`

  const create = async () => {
    setFormError('')
    try {
      const res = await api.post<{ key: string }>(`${cp}/api-keys`, { name })
      setCreated(res.key)
      setCreating(false)
      setName('')
      reload()
    } catch (e) {
      setFormError(errorMessage(e))
    }
  }

  const doRevoke = async (k: APIKey) => {
    try {
      await api.del(`${cp}/api-keys/${k.id}`)
      toast('ok', `Key ${k.name} revoked`)
      reload()
    } catch (e) {
      toast('err', errorMessage(e))
    }
  }

  const example = `curl -k -X POST "${base}/invoices" \\
  -H "X-API-Key: eik_…" -H "Content-Type: application/json" \\
  -d '{
  "externalRef": "ERP-1001",
  "invoiceDate": "${todayPK()}",
  "buyer": {"ntnCnic": "2046004", "name": "ABC Traders", "province": "PUNJAB",
            "address": "Lahore", "registrationType": "Registered"},
  "items": [{"hsCode": "0101.2100", "description": "Product", "uom": "Numbers, pieces, units",
             "quantity": 10, "unitPrice": 150, "saleType": "Goods at standard rate (default)", "rate": "18%"}],
  "submit": true
}'`

  return (
    <>
      <div className="page-head">
        <div>
          <h1>ERP / POS API keys</h1>
          <p>
            Connect your ERP, accounting software or POS to {s.company?.name}. The system reports each invoice to FBR and returns the FBR invoice number for
            printing in your own system.
          </p>
        </div>
        <div className="actions">
          <button className="btn btn-primary" onClick={() => setCreating(true)}>
            + New API key
          </button>
        </div>
      </div>
      {created && (
        <div className="alert alert-ok">
          <b>Copy this key now — it will not be shown again:</b>
          <pre className="code" style={{ marginTop: 6 }}>
            {created}
          </pre>
          <button className="btn btn-sm" onClick={() => navigator.clipboard?.writeText(created).then(() => toast('ok', 'Copied'))}>
            Copy
          </button>{' '}
          <button className="btn btn-sm" onClick={() => setCreated(null)}>
            Done
          </button>
        </div>
      )}
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
                    <th>Name</th>
                    <th>Key prefix</th>
                    <th>Created</th>
                    <th>Last used</th>
                    <th>Status</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {data.length === 0 && (
                    <tr>
                      <td colSpan={6} className="muted">
                        No API keys.
                      </td>
                    </tr>
                  )}
                  {data.map((k) => (
                    <tr key={k.id}>
                      <td>{k.name}</td>
                      <td className="mono">{k.prefix}…</td>
                      <td className="small">{dateTimeFmt(k.createdAt)}</td>
                      <td className="small">{dateTimeFmt(k.lastUsedAt) || 'never'}</td>
                      <td>{k.revokedAt ? <span className="badge b-gray">revoked</span> : <span className="badge b-green">active</span>}</td>
                      <td>
                        {!k.revokedAt && (
                          <button className="btn btn-sm btn-danger" onClick={() => setRevoke(k)}>
                            Revoke
                          </button>
                        )}
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
        <h3>Integration quick reference</h3>
        <dl className="kv">
          <dt>Base URL</dt>
          <dd className="mono">{base}</dd>
          <dt>Authentication</dt>
          <dd>
            Header <span className="mono">X-API-Key: eik_…</span> (or <span className="mono">Authorization: Bearer eik_…</span>)
          </dd>
          <dt>Create & submit</dt>
          <dd className="mono">POST /invoices</dd>
          <dt>Look up by ERP ref</dt>
          <dd className="mono">GET /invoice-by-ref/&#123;externalRef&#125;</dd>
          <dt>Print / QR</dt>
          <dd className="mono">GET /invoices/&#123;id&#125;/print · /qr.png · /qr.svg</dd>
          <dt>Raw FBR payload</dt>
          <dd>
            ERPs that already produce FBR's JSON can send it unchanged as <span className="mono">{'{"fbrPayload": {...}, "externalRef": "...", "submit": true}'}</span>
          </dd>
        </dl>
        <p className="small muted">
          Re-sending the same <span className="mono">externalRef</span> never creates a second invoice — the existing one is returned, so retries after a network error are
          safe. The response contains <span className="mono">status</span> and, once accepted, <span className="mono">fbrInvoiceNumber</span>. See the integration guide in
          the documentation folder for every field.
        </p>
        <pre className="code">{example}</pre>
      </div>

      {creating && (
        <Modal
          title="New API key"
          onClose={() => setCreating(false)}
          footer={
            <>
              <button className="btn" onClick={() => setCreating(false)}>
                Cancel
              </button>
              <button className="btn btn-primary" onClick={create} disabled={!name}>
                Create
              </button>
            </>
          }
        >
          {formError && <div className="alert alert-error">{formError}</div>}
          <Field label="Name" hint="e.g. SAP Business One, POS counter 1">
            <input value={name} onChange={(e) => setName(e.target.value)} autoFocus />
          </Field>
        </Modal>
      )}
      {revoke && (
        <Confirm
          danger
          label="Revoke"
          text={`Revoke API key "${revoke.name}"? Systems using it will stop reporting invoices until given a new key.`}
          onConfirm={() => doRevoke(revoke)}
          onClose={() => setRevoke(null)}
        />
      )}
    </>
  )
}
