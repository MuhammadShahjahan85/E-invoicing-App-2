// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { Fragment, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api, ApiError, errorMessage } from '../api'
import { Confirm, ErrorBox, Field, IssueList, Modal, Spinner, StatusBadge, useLoad } from '../components/ui'
import { dateFmt, dateTimeFmt, envLabels, hoursSince, money, qty } from '../format'
import { useCompanyPath, useSession, useToast } from '../state'
import type { AuditEntry, FBRCall, Invoice, Paged } from '../types'

interface Detail {
  invoice: Invoice
  sealValid: boolean
  issuedAt?: string
  original?: { id: number; internalNo: string; fbrInvoiceNumber: string }
  debitNotes?: { value: number; salesTax: number }
}

export default function InvoiceView() {
  const { id } = useParams()
  const s = useSession()
  const cp = useCompanyPath()
  const toast = useToast()
  const nav = useNavigate()
  const { data, error, loading, reload, setData } = useLoad(() => api.get<Detail>(`${cp}/invoices/${id}`), [cp, id])
  const [tab, setTab] = useState<'lines' | 'payload' | 'calls' | 'history'>('lines')
  const [busy, setBusy] = useState('')
  const [actionError, setActionError] = useState<unknown>()
  const [issues, setIssues] = useState<Invoice['validation']>(null)
  const [showCancel, setShowCancel] = useState(false)
  const [showResolve, setShowResolve] = useState(false)
  const [confirmDelete, setConfirmDelete] = useState(false)

  if (loading && !data) return <Spinner />
  if (error) return <ErrorBox error={error} />
  if (!data) return null
  const inv = data.invoice
  const errorCatalogue = Object.fromEntries(s.meta.errorCatalogue.map((e) => [e.code, e]))
  const editable = ['DRAFT', 'VALIDATED', 'REJECTED'].includes(inv.status)
  // The 72-hour window runs from FBR's issue time (falls back to acceptance).
  const issuedAt = data.issuedAt || inv.acceptedAt
  const hours = issuedAt ? hoursSince(issuedAt) : Infinity
  const withinCancel = hours <= s.meta.cancelWindowHours

  const act = async (name: string, fn: () => Promise<Invoice>, okMsg?: (i: Invoice) => string) => {
    setBusy(name)
    setActionError(undefined)
    setIssues(null)
    try {
      const updated = await fn()
      setData({ ...data, invoice: updated })
      if (okMsg) toast('ok', okMsg(updated))
      reload()
    } catch (e) {
      if (e instanceof ApiError && e.issues) setIssues(e.issues)
      else setActionError(e)
    } finally {
      setBusy('')
    }
  }

  const submit = () =>
    act('submit', () => api.post<Invoice>(`${cp}/invoices/${inv.id}/submit`), (i) =>
      i.status === 'ACCEPTED' ? `Accepted by FBR: ${i.fbrInvoiceNumber}` : `Status: ${i.status}`,
    )
  const validate = () =>
    act('validate', () => api.post<Invoice>(`${cp}/invoices/${inv.id}/validate`), (i) => (i.status === 'VALIDATED' ? 'FBR validation passed' : 'FBR validation reported errors'))
  const retry = () => act('retry', () => api.post<Invoice>(`${cp}/invoices/${inv.id}/retry`), (i) => `Status: ${i.status}`)
  const debitNote = async () => {
    setBusy('dn')
    try {
      const dn = await api.post<Invoice>(`${cp}/invoices/${inv.id}/debit-note`)
      toast('ok', `Draft debit note ${dn.internalNo} created — adjust the lines and submit`)
      nav(`/invoices/${dn.id}/edit`)
    } catch (e) {
      setActionError(e)
    } finally {
      setBusy('')
    }
  }
  const del = async () => {
    try {
      await api.del(`${cp}/invoices/${inv.id}`)
      toast('ok', 'Draft deleted')
      nav('/invoices')
    } catch (e) {
      setActionError(e)
    }
  }
  const print = (format: 'a4' | 'thermal') => {
    window.open(`/api/v1${cp}/invoices/${inv.id}/print?format=${format}&autoprint=1`, '_blank', 'noopener')
    setTimeout(reload, 1500)
  }

  return (
    <>
      <div className="page-head">
        <div>
          <h1>
            {inv.docType === 'Debit Note' ? 'Debit note' : 'Invoice'} {inv.internalNo} <StatusBadge status={inv.status} />
          </h1>
          <p>
            {dateFmt(inv.invoiceDate)} · {envLabels[inv.environment]} · source {inv.source}
            {inv.externalRef && <> · ERP ref {inv.externalRef}</>}
            {inv.scenarioId && <> · scenario {inv.scenarioId}</>}
          </p>
        </div>
        <div className="actions">
          {editable && (
            <>
              <Link className="btn" to={`/invoices/${inv.id}/edit`}>
                Edit
              </Link>
              <button className="btn" disabled={!!busy} onClick={validate}>
                {busy === 'validate' ? 'Validating…' : 'Validate with FBR'}
              </button>
              <button className="btn btn-primary" disabled={!!busy} onClick={submit}>
                {busy === 'submit' ? 'Submitting…' : 'Submit to FBR'}
              </button>
              <button className="btn btn-danger" onClick={() => setConfirmDelete(true)}>
                Delete
              </button>
            </>
          )}
          {inv.status === 'QUEUED' && (
            <button className="btn btn-primary" disabled={!!busy} onClick={retry}>
              {busy === 'retry' ? 'Submitting…' : 'Retry now'}
            </button>
          )}
          {inv.status === 'UNCERTAIN' && s.can('invoice.manage') && (
            <button className="btn btn-primary" onClick={() => setShowResolve(true)}>
              Reconcile with IRIS
            </button>
          )}
          {(inv.status === 'ACCEPTED' || inv.status === 'CANCELLED') && (
            <>
              <button className="btn btn-primary" onClick={() => print('a4')}>
                Print A4
              </button>
              <button className="btn" onClick={() => print('thermal')}>
                Print receipt (80 mm)
              </button>
            </>
          )}
          {!editable && inv.status !== 'ACCEPTED' && inv.status !== 'CANCELLED' && (
            <a className="btn" href={`/api/v1${cp}/invoices/${inv.id}/print?preview=1`} target="_blank" rel="noopener">
              {inv.status === 'QUEUED' || inv.offlineSince ? 'Print provisional copy' : 'Preview'}
            </a>
          )}
          {editable && (
            <a className="btn" href={`/api/v1${cp}/invoices/${inv.id}/print?preview=1`} target="_blank" rel="noopener">
              {inv.offlineSince ? 'Print provisional copy' : 'Preview'}
            </a>
          )}
          {inv.status === 'ACCEPTED' && inv.docType === 'Sale Invoice' && s.can('invoice.manage') && (
            <button className="btn" disabled={!!busy} onClick={debitNote}>
              Debit note
            </button>
          )}
          {inv.status === 'ACCEPTED' && s.can('invoice.manage') && (
            <button className="btn btn-danger" onClick={() => setShowCancel(true)}>
              Cancel invoice
            </button>
          )}
        </div>
      </div>

      <ErrorBox error={actionError} />
      <IssueList issues={issues} />

      {inv.status === 'QUEUED' && (
        <div className="alert alert-warn">
          <b>Not yet reported to FBR.</b> The invoice is sent automatically as soon as FBR responds; invoices issued offline must be uploaded within 24 hours of the
          connection being restored. A provisional copy can be printed now; print the final copy (with the FBR number and QR code) once it is accepted.{' '}
          {inv.lastError} {inv.nextAttemptAt && <>Next automatic attempt: {dateTimeFmt(inv.nextAttemptAt)}.</>}
        </div>
      )}
      {inv.status === 'UNCERTAIN' && (
        <div className="alert alert-warn">
          <b>No definitive answer was received from FBR.</b> The invoice may already be recorded. Log in to IRIS → Digital Invoicing and search for this invoice
          (date {dateFmt(inv.invoiceDate)}, buyer {inv.buyerName}, value {money(inv.totals.valueExclST)}). Then use <b>Reconcile with IRIS</b>. Do not
          resubmit blindly — that could report the sale twice. <div className="small">{inv.lastError}</div>
        </div>
      )}
      {inv.offlineSince && inv.status !== 'QUEUED' && inv.status !== 'ACCEPTED' && inv.status !== 'CANCELLED' && (
        <div className="alert alert-warn">
          Issued while FBR was unreachable ({dateTimeFmt(inv.offlineSince)}) and not yet accepted by FBR. It must be reported within 24 hours of the connection being
          restored — correct it if needed and submit it again.
        </div>
      )}
      {inv.status === 'REJECTED' && inv.lastError && <div className="alert alert-error">{inv.lastError}</div>}
      {editable && inv.lastError && inv.status !== 'REJECTED' && <div className="alert alert-warn">{inv.lastError}</div>}
      {inv.status === 'CANCELLED' && (
        <div className="alert alert-info">
          Cancelled on {dateTimeFmt(inv.cancelledAt)} — {inv.cancelReason}. Reference: {inv.cancelReference}
        </div>
      )}

      {(inv.fbrErrors?.length ?? 0) > 0 && editable && (
        <div className="card mb">
          <div className="card-head">
            <h3>FBR validation errors</h3>
          </div>
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Line</th>
                  <th>Code</th>
                  <th>FBR message</th>
                  <th>How to fix</th>
                </tr>
              </thead>
              <tbody>
                {inv.fbrErrors!.map((e, i) => (
                  <tr key={i}>
                    <td>{e.item || 'Header'}</td>
                    <td className="mono">{e.code}</td>
                    <td>{e.message}</td>
                    <td className="small muted">{errorCatalogue[e.code]?.fix ?? ''}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
      {editable && <IssueList issues={inv.validation} title="Local validation" />}

      <div className="grid g3">
        <div className="card card-pad">
          <h3>FBR</h3>
          {inv.fbrInvoiceNumber ? (
            <div className="row" style={{ alignItems: 'flex-start' }}>
              <div className="qr-box">
                <img src={`/api/v1${cp}/invoices/${inv.id}/qr.svg`} alt="FBR QR code" />
              </div>
              <dl className="kv" style={{ gridTemplateColumns: '110px 1fr' }}>
                <dt>Invoice no.</dt>
                <dd className="fbr-no">{inv.fbrInvoiceNumber}</dd>
                <dt>FBR time</dt>
                <dd>{inv.fbrDated}</dd>
                <dt>Integrity</dt>
                <dd>{data.sealValid ? <span className="badge b-green">Seal verified</span> : <span className="badge b-red">Seal mismatch</span>}</dd>
                <dt>Printed</dt>
                <dd>{inv.printCount} time(s)</dd>
                {inv.status === 'ACCEPTED' && (
                  <>
                    <dt>Edit window</dt>
                    <dd>{withinCancel ? `${Math.max(0, s.meta.cancelWindowHours - hours).toFixed(1)} h left (STGO 01/2026)` : 'Closed — Commissioner approval needed'}</dd>
                  </>
                )}
              </dl>
            </div>
          ) : (
            <p className="muted">Not yet accepted by FBR. Submit the invoice to obtain the FBR invoice number and QR code.</p>
          )}
          {data.original && (
            <p className="small">
              Adjusts invoice <Link to={`/invoices/${data.original.id}`}>{data.original.internalNo}</Link> ({data.original.fbrInvoiceNumber})
            </p>
          )}
          {data.debitNotes && data.debitNotes.value > 0 && (
            <p className="small">
              Debit notes accepted against this invoice: value {money(data.debitNotes.value)}, sales tax {money(data.debitNotes.salesTax)}
            </p>
          )}
        </div>
        <div className="card card-pad">
          <h3>Seller</h3>
          <dl className="kv" style={{ gridTemplateColumns: '90px 1fr' }}>
            <dt>Name</dt>
            <dd>{inv.sellerName}</dd>
            <dt>NTN/CNIC</dt>
            <dd>{inv.sellerNtnCnic}</dd>
            <dt>Province</dt>
            <dd>{inv.sellerProvince}</dd>
            <dt>Address</dt>
            <dd>{inv.sellerAddress}</dd>
          </dl>
        </div>
        <div className="card card-pad">
          <h3>Buyer</h3>
          <dl className="kv" style={{ gridTemplateColumns: '90px 1fr' }}>
            <dt>Name</dt>
            <dd>{inv.buyerName}</dd>
            <dt>NTN/CNIC</dt>
            <dd>{inv.buyerNtnCnic || '—'}</dd>
            <dt>Type</dt>
            <dd>{inv.buyerRegistrationType}</dd>
            <dt>Province</dt>
            <dd>{inv.buyerProvince}</dd>
            <dt>Address</dt>
            <dd>{inv.buyerAddress || '—'}</dd>
            {inv.withholdingMode && (
              <>
                <dt>Withholding</dt>
                <dd>{inv.withholdingMode === 'full' ? 'Full sales tax' : 'Fraction of sales tax'}</dd>
              </>
            )}
          </dl>
        </div>
      </div>

      <div className="card mt">
        <div className="tabs" style={{ padding: '0 12px', marginBottom: 0 }}>
          {(['lines', 'payload', 'calls', 'history'] as const).map((t) => (
            <button key={t} className={'tab' + (tab === t ? ' active' : '')} onClick={() => setTab(t)}>
              {{ lines: 'Lines & totals', payload: 'FBR payload (JSON)', calls: 'FBR call log', history: 'History' }[t]}
            </button>
          ))}
        </div>
        {tab === 'lines' && <Lines inv={inv} />}
        {tab === 'payload' && <Payload cp={cp} id={inv.id} />}
        {tab === 'calls' && <Calls cp={cp} id={inv.id} />}
        {tab === 'history' && <History companyId={inv.companyId} id={inv.id} />}
      </div>

      {confirmDelete && <Confirm danger label="Delete" text={`Delete draft ${inv.internalNo}? This cannot be undone.`} onConfirm={del} onClose={() => setConfirmDelete(false)} />}
      {showCancel && (
        <CancelModal
          inv={inv}
          withinWindow={withinCancel}
          onClose={() => setShowCancel(false)}
          onDone={(i) => {
            setData({ ...data, invoice: i })
            setShowCancel(false)
            toast('ok', 'Invoice cancellation recorded')
          }}
        />
      )}
      {showResolve && (
        <ResolveModal
          inv={inv}
          onClose={() => setShowResolve(false)}
          onDone={(i) => {
            setData({ ...data, invoice: i })
            setShowResolve(false)
            toast('ok', `Invoice is now ${i.status}`)
          }}
        />
      )}
    </>
  )
}

function Lines({ inv }: { inv: Invoice }) {
  const t = inv.totals
  return (
    <div className="table-wrap">
      <table className="table">
        <thead>
          <tr>
            <th>#</th>
            <th>Description</th>
            <th>HS code</th>
            <th>Sale type / rate</th>
            <th className="num">Qty</th>
            <th className="num">Value excl. ST</th>
            <th className="num">Sales tax</th>
            <th className="num">Further</th>
            <th className="num">Extra / FED</th>
            <th className="num">Withheld</th>
            <th className="num">Total</th>
            <th>FBR item no. / error</th>
          </tr>
        </thead>
        <tbody>
          {(inv.items ?? []).map((it) => (
            <tr key={it.id}>
              <td>{it.lineNo}</td>
              <td>
                {it.description}
                {it.sroScheduleNo && (
                  <div className="small faint">
                    {it.sroScheduleNo} {it.sroItemSerialNo && `S.No ${it.sroItemSerialNo}`}
                  </div>
                )}
                {!!it.retailValue && <div className="small faint">Retail value {money(it.retailValue)}</div>}
              </td>
              <td className="mono">{it.hsCode}</td>
              <td className="small">
                {it.saleType}
                <br />
                <b>{it.rate}</b>
              </td>
              <td className="num">
                {qty(it.quantity)} <span className="faint small">{it.uom}</span>
              </td>
              <td className="num">{money(it.valueExclST)}</td>
              <td className="num">{money(it.salesTax)}</td>
              <td className="num">{money(it.furtherTax)}</td>
              <td className="num">
                {it.extraTaxEmpty ? '—' : money(it.extraTax)} / {money(it.fed)}
              </td>
              <td className="num">{money(it.stWithheld)}</td>
              <td className="num">
                <b>{money(it.totalValue)}</b>
              </td>
              <td className="small">
                {it.fbrItemInvoiceNo && <span className="mono">{it.fbrItemInvoiceNo}</span>}
                {it.fbrError && (
                  <span style={{ color: 'var(--danger)' }}>
                    [{it.fbrErrorCode}] {it.fbrError}
                  </span>
                )}
              </td>
            </tr>
          ))}
        </tbody>
        <tfoot>
          <tr>
            <td colSpan={5}>Totals</td>
            <td className="num">{money(t.valueExclST)}</td>
            <td className="num">{money(t.salesTax)}</td>
            <td className="num">{money(t.furtherTax)}</td>
            <td className="num">
              {money(t.extraTax)} / {money(t.fed)}
            </td>
            <td className="num">{money(t.stWithheld)}</td>
            <td className="num">{money(t.totalValue)}</td>
            <td>Payable {money(t.amountPayable)}</td>
          </tr>
        </tfoot>
      </table>
    </div>
  )
}

function Payload({ cp, id }: { cp: string; id: number }) {
  const { data, error } = useLoad(async () => {
    const res = await fetch(`/api/v1${cp}/invoices/${id}/payload`, { credentials: 'same-origin' })
    return res.text()
  }, [cp, id])
  return (
    <div className="card-body">
      <p className="small muted">The exact JSON body sent (or to be sent) to FBR's postinvoicedata API. The security token is never stored here.</p>
      <ErrorBox error={error} />
      <pre className="code">{data}</pre>
    </div>
  )
}

function Calls({ cp, id }: { cp: string; id: number }) {
  const { data, error } = useLoad(() => api.get<FBRCall[]>(`${cp}/invoices/${id}/calls`), [cp, id])
  const [open, setOpen] = useState<number | null>(null)
  if (error) return <ErrorBox error={error} />
  if (!data) return <Spinner />
  if (data.length === 0) return <div className="empty">No exchanges with FBR yet.</div>
  return (
    <div className="table-wrap">
      <table className="table">
        <thead>
          <tr>
            <th>Time</th>
            <th>Operation</th>
            <th>HTTP</th>
            <th>ms</th>
            <th>Outcome</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {data.map((c) => (
            <Fragment key={c.id}>
              <tr>
                <td className="nowrap">{dateTimeFmt(c.createdAt)}</td>
                <td>{c.operation}</td>
                <td>{c.httpStatus || '—'}</td>
                <td>{c.durationMs}</td>
                <td>{c.errorKind ? <span className="badge b-red">{c.errorKind}</span> : <span className="badge b-green">ok</span>}</td>
                <td>
                  <button className="btn-link" onClick={() => setOpen(open === c.id ? null : c.id)}>
                    {open === c.id ? 'hide' : 'details'}
                  </button>
                </td>
              </tr>
              {open === c.id && (
                <tr>
                  <td colSpan={6}>
                    <div className="small mono">{c.url}</div>
                    {c.error && <div className="alert alert-error small">{c.error}</div>}
                    <div className="grid g2">
                      <pre className="code">{pretty(c.requestBody)}</pre>
                      <pre className="code">{pretty(c.responseBody)}</pre>
                    </div>
                  </td>
                </tr>
              )}
            </Fragment>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function pretty(s: string) {
  try {
    return JSON.stringify(JSON.parse(s), null, 2)
  } catch {
    return s
  }
}

function History({ companyId, id }: { companyId: number; id: number }) {
  const { data, error } = useLoad(() => api.get<Paged<AuditEntry>>(`/audit?companyId=${companyId}&entity=invoice&entityId=${id}&limit=200`), [companyId, id])
  if (error) return <div className="card-body muted">History is available to users with audit permission.</div>
  if (!data) return <Spinner />
  return (
    <div className="table-wrap">
      <table className="table">
        <thead>
          <tr>
            <th>Time</th>
            <th>User</th>
            <th>Action</th>
            <th>Details</th>
          </tr>
        </thead>
        <tbody>
          {data.items.map((a) => (
            <tr key={a.id}>
              <td className="nowrap">{dateTimeFmt(a.ts)}</td>
              <td>{a.username}</td>
              <td>{a.action}</td>
              <td className="small mono" style={{ maxWidth: 560, wordBreak: 'break-all' }}>
                {a.details}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function CancelModal({ inv, withinWindow, onClose, onDone }: { inv: Invoice; withinWindow: boolean; onClose: () => void; onDone: (i: Invoice) => void }) {
  const cp = useCompanyPath()
  const s = useSession()
  const apiAvailable = !!s.meta.cancelApi?.[inv.environment]
  const [useApi, setUseApi] = useState(apiAvailable)
  const [reason, setReason] = useState('')
  const [reference, setReference] = useState('')
  const [approval, setApproval] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const submit = async () => {
    setError('')
    setBusy(true)
    try {
      onDone(await api.post<Invoice>(`${cp}/invoices/${inv.id}/cancel`, { reason, reference, commissionerApproval: approval, useApi: apiAvailable && useApi }))
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  const referenceOptional = inv.environment === 'simulator' || (apiAvailable && useApi)
  return (
    <Modal
      title={`Cancel ${inv.internalNo}`}
      onClose={onClose}
      footer={
        <>
          <button className="btn" onClick={onClose}>
            Close
          </button>
          <button className="btn btn-danger" onClick={submit} disabled={!reason || busy}>
            {busy ? 'Cancelling…' : apiAvailable && useApi ? 'Cancel with FBR' : 'Record cancellation'}
          </button>
        </>
      }
    >
      <div className="alert alert-info">
        Under Sales Tax General Order 01 of 2026 an electronic invoice may be cancelled, deleted or edited only through FBR's system within 72 hours of issue;
        afterwards prior approval of the Commissioner Inland Revenue is required.{' '}
        {apiAvailable && useApi ? (
          <>The cancellation is sent to FBR now and recorded here only if FBR confirms it.</>
        ) : (
          <>
            First cancel the invoice on <b>IRIS → Digital Invoicing</b>, then record the cancellation here so your records match FBR.
          </>
        )}
      </div>
      {!withinWindow && <div className="alert alert-warn">This invoice was issued more than 72 hours ago — enter the Commissioner's approval reference.</div>}
      {error && <div className="alert alert-error">{error}</div>}
      <div className="stack">
        <Field label="Reason for cancellation *">
          <input value={reason} onChange={(e) => setReason(e.target.value)} placeholder="e.g. Invoice issued in error / duplicate" />
        </Field>
        {apiAvailable && (
          <label className="row small" style={{ gap: 8 }}>
            <input type="checkbox" checked={useApi} onChange={(e) => setUseApi(e.target.checked)} style={{ width: 'auto' }} />
            Cancel through FBR's cancellation service (configured on this server)
          </label>
        )}
        <Field label={'IRIS cancellation reference' + (inv.environment === 'simulator' ? ' (optional in training)' : referenceOptional ? ' (optional)' : ' *')}>
          <input value={reference} onChange={(e) => setReference(e.target.value)} />
        </Field>
        {!withinWindow && (
          <Field label="Commissioner approval reference *">
            <input value={approval} onChange={(e) => setApproval(e.target.value)} />
          </Field>
        )}
      </div>
    </Modal>
  )
}

function ResolveModal({ inv, onClose, onDone }: { inv: Invoice; onClose: () => void; onDone: (i: Invoice) => void }) {
  const cp = useCompanyPath()
  const [mode, setMode] = useState<'accepted' | 'retry' | 'draft'>('accepted')
  const [fbrNo, setFbrNo] = useState('')
  const [dated, setDated] = useState('')
  const [note, setNote] = useState('')
  const [error, setError] = useState('')
  const submit = async () => {
    setError('')
    try {
      onDone(await api.post<Invoice>(`${cp}/invoices/${inv.id}/resolve`, { action: mode, fbrInvoiceNumber: fbrNo, fbrDated: dated, note }))
    } catch (e) {
      setError(errorMessage(e))
    }
  }
  return (
    <Modal
      title="Reconcile with IRIS"
      onClose={onClose}
      footer={
        <>
          <button className="btn" onClick={onClose}>
            Close
          </button>
          <button className="btn btn-primary" onClick={submit} disabled={mode === 'accepted' && !fbrNo}>
            Apply
          </button>
        </>
      }
    >
      {error && <div className="alert alert-error">{error}</div>}
      <div className="stack">
        <label className="check">
          <input type="radio" checked={mode === 'accepted'} onChange={() => setMode('accepted')} /> IRIS shows this invoice — record its FBR invoice number
        </label>
        {mode === 'accepted' && (
          <div className="form-grid" style={{ paddingLeft: 24 }}>
            <Field label="FBR invoice number" span={2}>
              <input className="mono" value={fbrNo} onChange={(e) => setFbrNo(e.target.value.trim())} placeholder={inv.sellerNtnCnic + 'DI…'} />
            </Field>
            <Field label="FBR date/time shown on IRIS *">
              <input value={dated} onChange={(e) => setDated(e.target.value)} placeholder="YYYY-MM-DD HH:MM:SS" />
            </Field>
          </div>
        )}
        <label className="check">
          <input type="radio" checked={mode === 'retry'} onChange={() => setMode('retry')} /> IRIS does not show it — resubmit automatically
        </label>
        <label className="check">
          <input type="radio" checked={mode === 'draft'} onChange={() => setMode('draft')} /> IRIS does not show it — return to draft for correction
        </label>
        <Field label="Note (kept in the audit trail)">
          <input value={note} onChange={(e) => setNote(e.target.value)} />
        </Field>
      </div>
    </Modal>
  )
}
