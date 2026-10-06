import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { api, qs } from '../api'
import { Empty, ErrorBox, Pager, Spinner, StatusBadge, useLoad } from '../components/ui'
import { dateFmt, money, statusLabels } from '../format'
import { useCompanyPath, useSession } from '../state'
import type { Invoice, Paged } from '../types'

export default function Invoices() {
  const { company } = useSession()
  const cp = useCompanyPath()
  const nav = useNavigate()
  const [params, setParams] = useSearchParams()
  const [q, setQ] = useState(params.get('q') ?? '')
  const status = params.get('status') ?? ''
  const from = params.get('from') ?? ''
  const to = params.get('to') ?? ''
  const docType = params.get('docType') ?? ''
  const offset = Number(params.get('offset') ?? 0)
  const limit = 50
  const env = company?.environment

  const { data, error, loading } = useLoad(
    () => api.get<Paged<Invoice>>(`${cp}/invoices${qs({ status, from, to, q: params.get('q'), docType, limit, offset, env })}`),
    [cp, status, from, to, params.get('q'), docType, offset, env],
  )

  const setParam = (k: string, v: string) => {
    const p = new URLSearchParams(params)
    if (k !== 'offset') p.delete('offset')
    if (v) p.set(k, v)
    else p.delete(k)
    setParams(p)
  }

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Invoices</h1>
          <p>Sale invoices and debit notes reported to FBR Digital Invoicing.</p>
        </div>
        <div className="actions">
          <Link className="btn btn-primary" to="/invoices/new">
            + New invoice
          </Link>
        </div>
      </div>
      <div className="card">
        <div className="card-body row">
          <form
            className="row"
            style={{ flex: 1 }}
            onSubmit={(e) => {
              e.preventDefault()
              setParam('q', q)
            }}
          >
            <input placeholder="Search invoice no., FBR no., buyer, NTN, ERP ref…" value={q} onChange={(e) => setQ(e.target.value)} style={{ maxWidth: 360 }} />
            <button className="btn">Search</button>
          </form>
          <select value={status} onChange={(e) => setParam('status', e.target.value)} style={{ width: 'auto' }}>
            <option value="">All statuses</option>
            {Object.entries(statusLabels).map(([k, v]) => (
              <option key={k} value={k}>
                {v}
              </option>
            ))}
            <option value="REJECTED,UNCERTAIN,QUEUED">Needs attention</option>
          </select>
          <select value={docType} onChange={(e) => setParam('docType', e.target.value)} style={{ width: 'auto' }}>
            <option value="">All documents</option>
            <option>Sale Invoice</option>
            <option>Debit Note</option>
          </select>
          <input type="date" value={from} onChange={(e) => setParam('from', e.target.value)} style={{ width: 'auto' }} aria-label="From" />
          <input type="date" value={to} onChange={(e) => setParam('to', e.target.value)} style={{ width: 'auto' }} aria-label="To" />
        </div>
        <ErrorBox error={error} />
        {loading && !data ? (
          <div className="card-body">
            <Spinner />
          </div>
        ) : data && data.items.length === 0 ? (
          <Empty>No invoices match. <Link to="/invoices/new">Create one</Link>.</Empty>
        ) : (
          data && (
            <>
              <div className="table-wrap">
                <table className="table">
                  <thead>
                    <tr>
                      <th>Date</th>
                      <th>Invoice No.</th>
                      <th>Type</th>
                      <th>Buyer</th>
                      <th className="num">Value excl. ST</th>
                      <th className="num">Sales tax</th>
                      <th className="num">Total</th>
                      <th>Status</th>
                      <th>FBR invoice no.</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.items.map((inv) => (
                      <tr key={inv.id} className="clickable" onClick={() => nav(`/invoices/${inv.id}`)}>
                        <td className="nowrap">{dateFmt(inv.invoiceDate)}</td>
                        <td className="nowrap">
                          <b>{inv.internalNo}</b>
                          {inv.externalRef && <div className="small faint">ERP {inv.externalRef}</div>}
                        </td>
                        <td className="nowrap small">{inv.docType}</td>
                        <td>
                          {inv.buyerName}
                          <div className="small faint">
                            {inv.buyerNtnCnic || 'No NTN/CNIC'} · {inv.buyerRegistrationType}
                          </div>
                        </td>
                        <td className="num">{money(inv.totals.valueExclST)}</td>
                        <td className="num">{money(inv.totals.salesTax + inv.totals.furtherTax + inv.totals.extraTax)}</td>
                        <td className="num">{money(inv.totals.totalValue)}</td>
                        <td>
                          <StatusBadge status={inv.status} />
                        </td>
                        <td className="mono small">{inv.fbrInvoiceNumber}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <Pager total={data.total} limit={limit} offset={offset} onChange={(o) => setParam('offset', String(o))} />
            </>
          )
        )}
      </div>
    </>
  )
}
