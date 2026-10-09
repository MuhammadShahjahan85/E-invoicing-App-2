// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

import { useState } from 'react'
import { ExternalLink } from 'lucide-react'
import { useSession } from '../state'
import { documents, faqs, faqSource, integrators, integratorSource } from './helpContent'

export default function Help() {
  const s = useSession()
  const [q, setQ] = useState('')
  const [tab, setTab] = useState<'errors' | 'rules' | 'faqs' | 'docs' | 'saletypes' | 'support'>('errors')
  const [section, setSection] = useState<'all' | 'sales' | 'purchase'>('all')
  const needle = q.trim().toLowerCase()
  const errors = s.meta.errorCatalogue.filter(
    (e) =>
      (section === 'all' || e.section === section) &&
      (!needle || e.code.includes(needle) || [e.title, e.detail, e.fix].some((t) => t.toLowerCase().includes(needle))),
  )

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
        <button className={'tab' + (tab === 'faqs' ? ' active' : '')} onClick={() => setTab('faqs')}>
          FBR FAQs
        </button>
        <button className={'tab' + (tab === 'docs' ? ' active' : '')} onClick={() => setTab('docs')}>
          Documents & integrators
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
            <div className="row" style={{ alignItems: 'center', gap: 10, flexWrap: 'wrap' }}>
              <input placeholder="Search code or text, e.g. 0052 or HS code" value={q} onChange={(e) => setQ(e.target.value)} style={{ maxWidth: 360 }} />
              <div className="seg" role="group" aria-label="Code list">
                {(
                  [
                    ['all', 'All codes'],
                    ['sales', 'Sales (section 7)'],
                    ['purchase', 'Purchase (section 8)'],
                  ] as const
                ).map(([k, label]) => (
                  <button key={k} className={section === k ? 'on' : ''} aria-pressed={section === k} onClick={() => setSection(k)}>
                    {label}
                  </button>
                ))}
              </div>
            </div>
            <p className="small muted" style={{ marginTop: 8 }}>
              {s.meta.errorCatalogue.length} codes with FBR's own wording, from PRAL's Technical Specification for DI API v1.12 (24 July 2025). Purchase codes
              apply to purchases from cotton ginners (scenario SN009). The message FBR returns on an invoice is always shown as received.
            </p>
          </div>
          <div className="table-wrap">
            <table className="table">
              <thead>
                <tr>
                  <th>Code</th>
                  <th>FBR's message</th>
                  <th>How to fix</th>
                </tr>
              </thead>
              <tbody>
                {errors.map((e) => (
                  <tr key={e.code}>
                    <td className="mono">
                      <b>{e.code}</b>
                      {e.section === 'purchase' && <div className="small muted">purchase</div>}
                    </td>
                    <td>
                      {e.title}
                      {e.detail && e.detail !== e.title && <div className="small muted">{e.detail}</div>}
                    </td>
                    <td className="small">{e.fix}</td>
                  </tr>
                ))}
                {errors.length === 0 && (
                  <tr>
                    <td colSpan={3} className="muted">
                      No code matches.
                    </td>
                  </tr>
                )}
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
                <b>Chapter XIV, Sales Tax Rules 2006</b> (substituted by SRO 69(I)/2025, amended by SRO 1666(I)/2026) — rule 150Q onwards: registered persons
                integrate their invoicing system with FBR through a licensed integrator or PRAL. All deadlines under SRO 1852(I)/2025 ended on 31 December 2025.
              </li>
              <li>
                <b>Rule 150R(4)</b> — the system must issue invoices in the prescribed format, create and record a digital signature, transmit the data securely,
                preserve it irrevocably, print the QR code, perform closings at the close of each day, week and month, and log every adjustment, cancellation and
                system event.
              </li>
              <li>
                <b>Rule 150R(11) and (13)</b> — an “Integrated with FBR” signboard with the software registration number at each outlet; the invoice particulars
                (a)–(z), and since SRO 1666(I)/2026 the federal excise duty particulars (aa)–(ff).
              </li>
              <li>
                <b>Rule 150S</b> — a real-time verifiable electronic invoice for every supply; debit notes, credit notes and advance receipt invoices are also
                issued electronically. Records are kept electronically for six years.
              </li>
              <li>
                <b>Rule 150XA(c)–(d)</b> — operational failures, disruptions, tampering and inoperative hardware or software must be reported to FBR and the Commissioner within 24 hours (see the incident register).
              </li>
              <li>
                <b>Rule 150XC</b> — invoices issued during a failure of the invoicing software, the internet or power must be clearly identified as issued in
                the offline mode and uploaded within 24 hours of restoration.
              </li>
              <li>
                <b>Section 23(1)(b)</b> — a manufacturer or importer supplying an unregistered person must state the buyer's CNIC or NTN on the invoice.
              </li>
              <li>
                <b>Rule 150XD(2)</b> (substituted by SRO 1666(I)/2026) — tax is recovered on any invoice transmitted with an FBR number but not accounted for in
                Annex-C or the return, unless cancelled through the approved mechanism. Use the Annex-C reconciliation before filing.
              </li>
              <li>
                <b>Section 23(1)</b> (Finance Act 2026) — invoices bearing an FBR invoice number are also required for exempt supplies and advance receipts.
              </li>
              <li>
                <b>Sales Tax General Order 01 of 2026</b> — an invoice may be cancelled or edited through FBR's system only within 72 hours of issue; afterwards
                with prior approval of the Commissioner.
              </li>
              <li>
                <b>Sales Tax General Order 25 of 2026</b> — goods moved to your own warehouse under the same STRN are not a supply: no digital invoice, but a
                serially numbered stock transfer note (see Stock transfer notes).
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
                dashboard shows invoices not yet reported, an outage incident is opened and a rule 150XA letter is prepared.
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
              <li>Third Schedule goods: tax is charged on the retail price; the price printed on the pack includes sales tax, so the tax is printed price × rate ÷ (100 + rate).</li>
              <li>Federal excise duty charged separately is part of the value of supply (section 2(46)), so sales tax and further tax are charged on value + FED.</li>
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

      {tab === 'faqs' && (
        <div className="card">
          <div className="card-body">
            <p className="small muted">
              Summary of FBR's Digital Invoicing FAQs. FBR notes that the Sales Tax Act 1990 and the Sales Tax Rules 2006 prevail over the FAQs in case of any
              difference.{' '}
              <a href={faqSource} target="_blank" rel="noopener noreferrer">
                FBR's FAQs <ExternalLink size={11} />
              </a>
            </p>
          </div>
          <div className="faq-list">
            {faqs.map((f, i) => (
              <details key={i} className="faq">
                <summary>{f.q}</summary>
                <p>{f.a}</p>
                {f.app && (
                  <p className="faq-app">
                    <b>In this software:</b> {f.app}
                  </p>
                )}
              </details>
            ))}
          </div>
        </div>
      )}

      {tab === 'docs' && (
        <div className="grid g-2-1">
          <div className="card">
            {documents.map((g) => (
              <div key={g.group}>
                <div className="card-head">
                  <h3>{g.group}</h3>
                </div>
                <div className="table-wrap">
                  <table className="table">
                    <tbody>
                      {g.items.map((d) => (
                        <tr key={d.url}>
                          <td>
                            <a href={d.url} target="_blank" rel="noopener noreferrer">
                              <b>{d.title}</b> <ExternalLink size={11} />
                            </a>
                            <div className="small muted">{d.about}</div>
                          </td>
                          <td className="nowrap small faint">{d.date}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            ))}
          </div>
          <div className="stack">
            <div className="card">
              <div className="card-head">
                <h3>Licensed integrators</h3>
                <a href={integratorSource} target="_blank" rel="noopener noreferrer" className="small">
                  FBR's list <ExternalLink size={11} />
                </a>
              </div>
              <div className="table-wrap">
                <table className="table">
                  <thead>
                    <tr>
                      <th>Integrator</th>
                      <th>Licence no.</th>
                    </tr>
                  </thead>
                  <tbody>
                    {integrators.map((x) => (
                      <tr key={x.name}>
                        <td>{x.name}</td>
                        <td className="mono small">{x.licence || '—'}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <div className="card-foot">
                Any of them can integrate this software with FBR. Contact details and licence certificates are on FBR's list, which FBR keeps up to date.
              </div>
            </div>
            <div className="card card-pad">
              <h3>PRAL support</h3>
              <p className="small">
                Complaints and support requests about Digital Invoicing go to PRAL's customer relationship portal,{' '}
                <a href="https://dicrm.pral.com.pk" target="_blank" rel="noopener noreferrer">
                  dicrm.pral.com.pk <ExternalLink size={11} />
                </a>
                , using the CRM user ID given during registration on IRIS.
              </p>
            </div>
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
            <dd>
              PRAL Digital Invoicing support portal{' '}
              <a href="https://dicrm.pral.com.pk" target="_blank" rel="noopener noreferrer">
                dicrm.pral.com.pk
              </a>
              ; FBR helpline 051-111-772-772
            </dd>
          </dl>
        </div>
      )}
    </>
  )
}
