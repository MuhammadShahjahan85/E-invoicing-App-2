// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing PK is proprietary software; see the LICENSE file.

import { useState } from 'react'
import { useSession } from '../state'

export default function Help() {
  const s = useSession()
  const [q, setQ] = useState('')
  const [tab, setTab] = useState<'errors' | 'rules' | 'saletypes' | 'support'>('errors')
  const needle = q.trim().toLowerCase()
  const errors = s.meta.errorCatalogue.filter((e) => !needle || e.code.includes(needle) || e.title.toLowerCase().includes(needle) || e.fix.toLowerCase().includes(needle))

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Help & error codes</h1>
          <p>Reference for FBR Digital Invoicing rules and the error codes returned by the DI API (technical specification v1.12).</p>
        </div>
      </div>
      <div className="tabs">
        <button className={'tab' + (tab === 'errors' ? ' active' : '')} onClick={() => setTab('errors')}>
          FBR error codes
        </button>
        <button className={'tab' + (tab === 'rules' ? ' active' : '')} onClick={() => setTab('rules')}>
          Compliance rules
        </button>
        <button className={'tab' + (tab === 'saletypes' ? ' active' : '')} onClick={() => setTab('saletypes')}>
          Sale types & rates
        </button>
        <button className={'tab' + (tab === 'support' ? ' active' : '')} onClick={() => setTab('support')}>
          Support
        </button>
      </div>

      {tab === 'errors' && (
        <div className="card">
          <div className="card-body">
            <input placeholder="Search code or text, e.g. 0052 or HS code" value={q} onChange={(e) => setQ(e.target.value)} style={{ maxWidth: 360 }} />
          </div>
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Code</th>
                  <th>Meaning</th>
                  <th>How to fix</th>
                </tr>
              </thead>
              <tbody>
                {errors.map((e) => (
                  <tr key={e.code}>
                    <td className="mono">
                      <b>{e.code}</b>
                    </td>
                    <td>{e.title}</td>
                    <td className="small">{e.fix}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {tab === 'rules' && (
        <div className="grid g2">
          <div className="card card-pad">
            <h3>Legal basis</h3>
            <ul className="small">
              <li>
                <b>Section 23 & 23(5)/(6), Sales Tax Act 1990</b> — tax invoice particulars; invoices to be issued electronically and transmitted to FBR in the
                prescribed manner.
              </li>
              <li>
                <b>Chapter XIV, Sales Tax Rules 2006</b> (substituted by SRO 69(I)/2025) — rule 150Q onwards: every registered person must integrate the
                electronic invoicing system with FBR through a licensed integrator or directly (PRAL).
              </li>
              <li>
                <b>Rule 150S</b> — every supply must be reported in real time and the invoice must carry the FBR invoice number and QR code. Records are kept
                electronically for six years.
              </li>
              <li>
                <b>Rule 150R</b> — failures, disruptions or tampering must be reported to the Commissioner within 24 hours (see the incident register).
              </li>
              <li>
                <b>Offline invoices</b> — invoices issued while FBR's system cannot be reached must be uploaded within 24 hours of the connection being restored.
              </li>
              <li>
                <b>Section 23(1)(b)</b> — a manufacturer or importer supplying an unregistered person must state the buyer's CNIC or NTN on the invoice.
              </li>
              <li>
                <b>Sales Tax General Order 01 of 2026</b> — an invoice may be cancelled or edited through FBR's system only within 72 hours of issue; afterwards
                with prior approval of the Commissioner.
              </li>
              <li>
                <b>Section 33</b> — penalties for failure to integrate or to issue electronic invoices, as enhanced by the Finance Act 2026; registration
                may be suspended under section 21(2), with no input tax adjustment or refund during the suspension.
              </li>
            </ul>
          </div>
          <div className="card card-pad">
            <h3>How this software complies</h3>
            <ul className="small">
              <li>Invoices are reported to FBR's DI API the moment they are saved; the FBR invoice number and QR code (version 2, 25×25, 1 inch) are printed with the FBR Digital Invoicing logo.</li>
              <li>Accepted invoices are locked in the database — they cannot be edited or deleted. Corrections are made with debit notes or an approved cancellation.</li>
              <li>
                If FBR is unreachable the invoice is queued and resent automatically, and every queued invoice is resent the moment the connection returns; the
                dashboard shows invoices not yet reported, an outage incident is opened and a Rule 150R letter is prepared.
              </li>
              <li>Ambiguous outcomes (timeouts after sending) are never resent blindly — they are reconciled against IRIS to avoid double reporting.</li>
              <li>An append-only, hash-chained audit trail records every action; invoices carry a tamper-evident seal.</li>
              <li>The database is backed up daily; copy backups off-site and keep records for six years as rule 150S requires.</li>
            </ul>
          </div>
          <div className="card card-pad">
            <h3>Invoice particulars checked before submission</h3>
            <ul className="small">
              <li>Seller and buyer NTN/CNIC (7-digit NTN or 13-digit CNIC), names, provinces and addresses.</li>
              <li>Buyer registration type — further tax at 4% applies to taxable supplies to unregistered buyers.</li>
              <li>CNIC/NTN of unregistered buyers when the seller is a manufacturer or importer (section 23(1)(b)), and above Rs 100,000 for other sellers.</li>
              <li>HS code (PCT, NNNN.NNNN), description, quantity and FBR unit of measure.</li>
              <li>Sale type and the rate allowed for it on the invoice date; SRO / schedule and serial number where required.</li>
              <li>Value excluding sales tax, sales tax, further tax, extra tax, FED and sales tax withheld at source.</li>
              <li>Third Schedule goods: tax is charged on the printed retail price.</li>
            </ul>
          </div>
          <div className="card card-pad">
            <h3>Going live checklist</h3>
            <ol className="small">
              <li>Register for Digital Invoicing on IRIS and select your integrator (PRAL or a licensed integrator).</li>
              <li>Record your business activity and sector; IRIS assigns the sandbox scenarios.</li>
              <li>Save the sandbox token, run every assigned scenario in the sandbox.</li>
              <li>Have this server's static public IP whitelisted, then generate the production token on IRIS (valid five years).</li>
              <li>Save the production token, test the connection and switch the company to production.</li>
            </ol>
          </div>
        </div>
      )}

      {tab === 'saletypes' && (
        <div className="card">
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Sale type (as FBR expects it)</th>
                  <th>Default rate</th>
                  <th>Tax base</th>
                  <th>SRO needed</th>
                  <th>Scenario</th>
                  <th>Notes</th>
                </tr>
              </thead>
              <tbody>
                {s.meta.saleTypes.map((t) => (
                  <tr key={t.name}>
                    <td>{t.name}</td>
                    <td>{t.defaultRate}</td>
                    <td className="small">{t.basis === 'retail_price' ? 'Retail price' : 'Value'}</td>
                    <td>{t.sroRequired ? 'Yes' : t.sroTypical ? 'Usually' : ''}</td>
                    <td className="mono small">{t.scenario}</td>
                    <td className="small">{t.note}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {tab === 'support' && (
        <div className="card card-pad">
          <h3>{s.meta.product}</h3>
          <dl className="kv">
            <dt>Version</dt>
            <dd>{s.meta.version}</dd>
            <dt>Developer</dt>
            <dd>{s.meta.vendor}</dd>
            <dt>Support</dt>
            <dd>{s.meta.support}</dd>
            <dt>Copyright</dt>
            <dd>
              {s.meta.copyright} This software is licensed, not sold; see the LICENSE file supplied with it. Third-party components are listed in
              THIRD-PARTY-NOTICES.txt.
            </dd>
            <dt>FBR DI help desk</dt>
            <dd>PRAL Digital Invoicing support (IRIS → Digital Invoicing → Help); FBR helpline 051-111-772-772</dd>
          </dl>
        </div>
      )}
    </>
  )
}
