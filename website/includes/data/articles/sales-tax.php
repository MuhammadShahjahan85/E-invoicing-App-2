<?php
/** Knowledge Bank — Sales tax and FBR Digital Invoicing. */
return [

'fbr-digital-invoicing-pakistan-complete-guide' => [
    'slug' => 'fbr-digital-invoicing-pakistan-complete-guide',
    'title' => 'FBR Digital Invoicing in Pakistan: The Complete Guide',
    'seo_title' => 'FBR Digital Invoicing (E-Invoicing) in Pakistan — Complete 2026 Guide',
    'description' => 'Everything about FBR Digital Invoicing: who must comply, deadlines, how to go live through IRIS and PRAL, sandbox scenarios, invoice requirements, the 72-hour rule and Finance Act 2026 penalties.',
    'category' => 'sales-tax',
    'tags' => ['FBR', 'Digital Invoicing', 'E-invoicing', 'PRAL', 'IRIS', 'Chapter XIV'],
    'date' => '2026-10-12',
    'updated' => '2026-10-12',
    'featured' => true,
    'excerpt' => 'Who must comply, how to go live, what each invoice must show, the 72-hour cancellation rule and the penalties introduced by the Finance Act 2026 — in one practical guide.',
    'services' => ['fbr-digital-invoicing', 'sales-tax', 'technology-consulting'],
    'body' => <<<'HTML'
<p>FBR's Digital Invoicing (DI) regime has changed how every sales-tax registered business in Pakistan issues an invoice. Each invoice is now reported to FBR in real time, receives a unique <strong>FBR invoice number</strong>, and is printed with a <strong>QR code</strong> and the FBR Digital Invoicing logo. This guide explains the rules, the go-live process and the controls your system needs — in plain language.</p>

<div class="callout callout--info"><strong>In short:</strong> every sales-tax registered person should now be live. Since the Finance Act 2026, failing to integrate costs Rs 1 million, then up to Rs 5 million if the default continues, and can lead to suspension of registration and sealing of premises.</div>

<h2>The legal framework</h2>
<p>Digital invoicing rests on section 23 of the Sales Tax Act, 1990 (the particulars of a tax invoice and the power to require electronic invoicing) and on <strong>Chapter XIV of the Sales Tax Rules, 2006</strong> (rules 150Q to 150XQ), inserted by SRO 69(I)/2025. Chapter XIV requires registered persons to install an invoicing system integrated with FBR's system and to make supplies only through integrated outlets.</p>
<p>Since then FBR has issued a series of instruments that shape day-to-day compliance:</p>
<ul>
  <li><strong>SRO 1852(I)/2025</strong> set the final dates for registration, testing and issuing e-invoices;</li>
  <li><strong>Sales Tax General Order 01 of 2026</strong> introduced the 72-hour window for cancelling or editing an invoice;</li>
  <li>the <strong>Finance Act 2026</strong> extended invoicing to exempt supplies and advance receipts and introduced new penalties;</li>
  <li><strong>STGO 25 of 2026</strong> prescribed stock transfer notes for goods moved to your own warehouse;</li>
  <li><strong>SRO 1666(I)/2026</strong> extended Chapter XIV to federal excise duty and added FED particulars to the invoice.</li>
</ul>

<h2>Who must comply, and by when</h2>
<p>SRO 1852(I)/2025 made integration mandatory for all registered persons, in phases:</p>
<table class="table">
  <thead><tr><th>Category</th><th>Last date to issue e-invoices</th></tr></thead>
  <tbody>
    <tr><td>Public companies, importers, companies with turnover above Rs 1 billion, individuals and AOPs above Rs 100 million</td><td>1 November 2025</td></tr>
    <tr><td>Companies with turnover of Rs 100 million – 1 billion</td><td>15 November 2025</td></tr>
    <tr><td>Companies with turnover up to Rs 100 million</td><td>1 December 2025</td></tr>
    <tr><td>All other registered persons</td><td>31 December 2025</td></tr>
  </tbody>
</table>
<p>If your business is not yet issuing digital invoices, treat it as urgent. The penalties below apply now.</p>

<h2>How digital invoicing works</h2>
<ol>
  <li>Your system prepares the invoice with every required particular — buyer, HS code, sale type, rate, quantity, value and tax.</li>
  <li>The invoice is sent to FBR's DI API using your business's own <strong>security token</strong>, from a <strong>whitelisted static IP</strong>.</li>
  <li>FBR validates it and returns a unique FBR invoice number — or an error explaining what to correct.</li>
  <li>The invoice is printed or shared with the FBR number, a QR code encoding it, and the DI logo.</li>
  <li>The invoice flows automatically into your <strong>Annex-C</strong> and your buyer's <strong>Annex-A</strong>, which decides the input tax they can claim.</li>
</ol>

<h3>Choosing an integration route</h3>
<p>You can integrate through <strong>PRAL</strong> (FBR's technology company, a licensed integrator free of cost under rule 150XF), through a <strong>licensed integrator</strong>, or — for very small volumes — by entering invoices on the IRIS portal. Most businesses use their own invoicing or ERP system connected through PRAL. We compare the options in <a href="/knowledge-bank/choosing-fbr-e-invoicing-route">Choosing your FBR e-invoicing route</a>.</p>

<h2>Going live, step by step</h2>
<ol>
  <li><strong>Register for Digital Invoicing on IRIS</strong> and choose your integrator. Enter the technical details: the ERP or system provider, software type and version, and a CRM user for PRAL's support portal.</li>
  <li><strong>Whitelist your static IP.</strong> Ask your internet provider for a static public IP; PRAL approves it, usually within a couple of working hours.</li>
  <li><strong>Record your business activity and sector</strong> (manufacturer, importer, distributor, wholesaler, exporter, retailer, service provider). IRIS assigns sandbox scenarios based on these.</li>
  <li><strong>Obtain a sandbox token</strong> and configure it in your system.</li>
  <li><strong>Run your assigned sandbox scenarios</strong> — each one is a realistic invoice type, such as a standard-rate sale, a sale to an unregistered buyer, a Third Schedule item or a reduced-rate supply.</li>
  <li><strong>Receive your production token</strong> and switch to live reporting.</li>
</ol>

<h2>What every digital invoice must show</h2>
<p>Beyond the traditional particulars of section 23 — supplier and buyer names, addresses and NTN/CNIC, date, description, quantity, value excluding tax, rate, tax and value including tax — rule 150R(13) requires particulars such as the HS code, tax period, software registration number, total discount and tax withheld. Since SRO 1666(I)/2026, invoices also carry federal excise duty particulars where FED applies. The printed invoice carries the <strong>FBR invoice number, a QR code and the DI logo</strong>.</p>
<p>A manufacturer or importer supplying an unregistered person must state the buyer's <strong>CNIC or NTN</strong> (section 23(1)(b)). Read more in <a href="/knowledge-bank/sales-tax-invoice-requirements-pakistan">What must a sales tax invoice contain?</a></p>

<h2>What your invoicing system must do</h2>
<p>Rule 150R(4) sets functional requirements for the system itself. It must:</p>
<ul>
  <li>generate, record and store invoice data, and issue invoices in the prescribed format;</li>
  <li>create and record a <strong>digital signature</strong> for each invoice;</li>
  <li>transmit data securely and receive the FBR number, and preserve the data irrevocably;</li>
  <li>print the QR code;</li>
  <li>perform <strong>closings at the end of each day, week and month</strong>;</li>
  <li>log every adjustment, modification, cancellation and system event.</li>
</ul>
<p>Each notified outlet displays an <strong>"Integrated with FBR" signboard</strong> with the software registration number (rule 150R(11)), and electronic records are kept for <strong>six years</strong> (rule 150S).</p>

<h2>Outages, offline invoices and incidents</h2>
<p>If the software, the internet or power fails, invoices issued in the meantime must be identified as issued in <strong>offline mode</strong> and uploaded within <strong>24 hours of restoration</strong> (rule 150XC). Failures, damage, disruption or tampering of the e-invoicing system must be reported to FBR and the Commissioner within <strong>24 hours</strong> (rule 150XA). A good system queues invoices automatically, resends them the moment FBR is reachable, and helps you document incidents.</p>

<h2>Cancelling or correcting an invoice: the 72-hour rule</h2>
<p>Under STGO 01 of 2026, an invoice generated by bona fide mistake may be cancelled, deleted or edited through FBR's system only within <strong>72 hours</strong> of generation. After that, you need the prior approval of the Commissioner Inland Revenue. Build this into your daily routine: review yesterday's invoices every morning.</p>

<h2>Penalties and consequences since the Finance Act 2026</h2>
<ul>
  <li>Failure to integrate or to record sales: <strong>Rs 1 million</strong>, and a second penalty of up to <strong>Rs 5 million</strong> if the default continues a month later; premises may be sealed (section 33(25)).</li>
  <li><strong>Suspension of registration</strong> for failure to integrate (section 21(2)).</li>
  <li>FBR may reduce the <strong>input tax ratio</strong> of persons not complying with digital invoicing (section 8B).</li>
  <li>Issuers of simulated invoices are named on a public <strong>Simulated Invoice Issuers Register</strong>, and buyers lose the related input tax.</li>
  <li>Tax on invoices reported with an FBR number but not accounted for in Annex-C or the return can be recovered (rule 150XD(2)).</li>
</ul>

<h2>A practical compliance checklist</h2>
<ul class="checklist">
  <li>Digital Invoicing registration completed on IRIS, with a whitelisted static IP</li>
  <li>Sandbox scenarios passed and production token installed</li>
  <li>Product master mapped to HS codes, units, sale types, rates and SROs</li>
  <li>Customer master checked against the Active Taxpayer List</li>
  <li>CNIC/NTN captured for unregistered buyers where section 23(1)(b) applies</li>
  <li>Daily review of rejected and unreported invoices; 72-hour correction window respected</li>
  <li>Offline invoices uploaded within 24 hours; incidents logged and reported</li>
  <li>"Integrated with FBR" signboard displayed at each outlet</li>
  <li>Monthly reconciliation of e-invoices, books and Annex-C before filing</li>
  <li>Backups kept, and records retained for six years</li>
</ul>

<h2>How Veridian can help</h2>
<p>We built <a href="/e-invoicing">Veridian E-invoicing Pakistan</a> to meet these requirements end to end — validation against FBR's own reference data, real-time reporting, compliant printing, the 72-hour rule, offline handling, incident letters, digital signatures, daily closings and Annex-C reconciliation. Our team also handles your IRIS registration, sandbox scenarios and go-live. <a href="/book-consultation">Book a free consultation</a>.</p>
HTML,
    'faqs' => [
        ['q' => 'Is FBR digital invoicing free?', 'a' => 'Integration through PRAL is free of cost. You still need a compliant invoicing system — your own software, an ERP with integration, or a licensed integrator\'s service.'],
        ['q' => 'Do I need a static IP for FBR e-invoicing?', 'a' => 'Yes. FBR accepts production calls only from whitelisted static public IP addresses, which you register on IRIS.'],
        ['q' => 'Can I cancel an FBR digital invoice?', 'a' => 'Within 72 hours of generation, an invoice issued by bona fide mistake can be cancelled or edited through FBR\'s system. After 72 hours, the Commissioner\'s prior approval is required.'],
    ],
],

'sales-tax-invoice-requirements-pakistan' => [
    'slug' => 'sales-tax-invoice-requirements-pakistan',
    'title' => 'What Must a Sales Tax Invoice Contain? Section 23 and Rule 150R Explained',
    'seo_title' => 'Sales Tax Invoice Requirements in Pakistan (Section 23, Rule 150R)',
    'description' => 'The particulars every sales tax invoice in Pakistan must show under section 23 of the Sales Tax Act and rule 150R(13), including FBR number, QR code, HS code, CNIC rules and FED particulars.',
    'category' => 'sales-tax',
    'tags' => ['Sales tax invoice', 'Section 23', 'Rule 150R', 'CNIC', 'HS code'],
    'date' => '2026-10-09',
    'updated' => '2026-10-09',
    'featured' => false,
    'excerpt' => 'A field-by-field guide to a compliant sales tax invoice in the digital invoicing era — and the mistakes that cost buyers their input tax.',
    'services' => ['sales-tax', 'fbr-digital-invoicing'],
    'body' => <<<'HTML'
<p>A sales tax invoice is more than a bill. For the buyer it is the evidence that unlocks input tax; for the seller it is a record FBR now sees in real time. An incomplete or incorrect invoice can cost both sides money. Here is what a compliant invoice contains today.</p>

<h2>The traditional particulars (section 23)</h2>
<p>Section 23 of the Sales Tax Act, 1990 requires a registered person making a taxable supply to issue a serially numbered tax invoice showing:</p>
<ul>
  <li>the supplier's name, address and registration number (NTN);</li>
  <li>the buyer's name, address and registration number (NTN or CNIC);</li>
  <li>the date of issue;</li>
  <li>a description and the quantity of the goods or services;</li>
  <li>the value excluding sales tax;</li>
  <li>the rate and amount of sales tax;</li>
  <li>the value including sales tax.</li>
</ul>

<h2>Additional particulars for digital invoices (rule 150R(13))</h2>
<p>Chapter XIV of the Sales Tax Rules adds particulars for invoices issued through an integrated system, including:</p>
<ul>
  <li>the <strong>FBR invoice number</strong> returned when the invoice is reported, and a <strong>QR code</strong> encoding it;</li>
  <li>the FBR Digital Invoicing logo;</li>
  <li>the <strong>HS (PCT) code</strong> of each item and the unit of measure FBR prescribes for it;</li>
  <li>the sale type and, where a reduced rate or exemption applies, the SRO or schedule and serial number;</li>
  <li>the tax period and the software registration number;</li>
  <li>total discount and any tax withheld;</li>
  <li>since SRO 1666(I)/2026, the <strong>federal excise duty particulars</strong> where FED applies: FED type, rate, price per unit, amount payable, and the SRO or schedule reference and serial number.</li>
</ul>

<h2>Buyers who are not registered</h2>
<p>Where a <strong>manufacturer or importer</strong> supplies an unregistered person — for example a distributor — the invoice must show the buyer's <strong>CNIC or NTN</strong> (section 23(1)(b)). Supplies to unregistered persons also generally attract <strong>further tax</strong> (currently 4%) in addition to sales tax, subject to exclusions.</p>

<h2>Invoices beyond ordinary sales</h2>
<ul>
  <li><strong>Exempt supplies</strong> — since the Finance Act 2026, an invoice bearing an FBR number is required for exempt supplies as well.</li>
  <li><strong>Advance receipts</strong> — an "advance receipt invoice" is issued when payment is received before the supply.</li>
  <li><strong>Debit and credit notes</strong> — issued electronically and kept for six years.</li>
  <li><strong>Stock transfers</strong> to your own warehouse under the same STRN are not supplies; they travel with a prescribed stock transfer note instead (STGO 25 of 2026).</li>
</ul>

<h2>Common mistakes that cost input tax</h2>
<ul>
  <li>Using an HS code that is not on FBR's list, or the wrong unit for that code.</li>
  <li>Wrong sale type or rate — for example applying a reduced rate without the SRO reference.</li>
  <li>A buyer NTN that does not match FBR's records, so the invoice never reaches the buyer's Annex-A.</li>
  <li>Invoices left unreported after an outage, beyond the 24-hour upload window.</li>
  <li>Corrections attempted after the 72-hour window without Commissioner approval.</li>
</ul>

<div class="callout callout--info"><strong>Tip:</strong> validate before you report. A system that checks each line against FBR's own reference data — HS codes, units, sale types, rates and SROs — catches most of these errors before the invoice reaches FBR.</div>

<p>Need your invoice format reviewed? <a href="/contact">Talk to our sales tax team</a>.</p>
HTML,
],

'finance-act-2026-sales-tax-changes' => [
    'slug' => 'finance-act-2026-sales-tax-changes',
    'title' => 'Finance Act 2026 and the Autumn SROs: What Changed for Sales Tax',
    'seo_title' => 'Finance Act 2026 Sales Tax Changes in Pakistan — Key Points',
    'description' => 'Key sales tax changes from the Finance Act 2026 and the September 2026 SROs and STGOs: invoices for exempt supplies and advance receipts, penalties, simulated invoices, electronic scrutiny, stock transfer notes and FED particulars.',
    'category' => 'sales-tax',
    'tags' => ['Finance Act 2026', 'Sales tax', 'Penalties', 'SRO 1666(I)/2026', 'STGO 25 of 2026'],
    'date' => '2026-10-06',
    'updated' => '2026-10-06',
    'featured' => false,
    'excerpt' => 'Tougher penalties, invoices for exempt supplies and advance receipts, a register of simulated invoice issuers, electronic scrutiny and stock transfer notes — the changes that matter.',
    'services' => ['sales-tax', 'fbr-digital-invoicing'],
    'body' => <<<'HTML'
<p>The Finance Act 2026, explained in FBR's Circular 01 of 2026, and a cluster of notifications issued in September 2026 have tightened sales tax compliance considerably. Here are the changes every registered business should know.</p>

<h2>1. More documents need an FBR number</h2>
<p>Section 23(1) now requires an invoice bearing a verifiable FBR number for <strong>exempt supplies</strong> and for <strong>advance receipts</strong>, not only for taxable sales. An "advance receipt invoice" is defined in section 2(1AA) as an invoice in the format the Board notifies.</p>

<h2>2. Penalties for not integrating</h2>
<ul>
  <li>Failure to integrate or to record sales: <strong>Rs 1 million</strong>; a second penalty of <strong>up to Rs 5 million</strong> if the default continues one month after the first; and the premises may be sealed (section 33(25)).</li>
  <li>Registration can be <strong>suspended</strong> for failure to integrate (section 21(2)).</li>
  <li>The Board may reduce the <strong>input tax ratio</strong> for persons who do not comply with digital invoicing and other electronic systems (section 8B).</li>
</ul>

<h2>3. Simulated invoices</h2>
<p>Persons issuing invoices without actual supplies are listed on a public <strong>Simulated Invoice Issuers Register</strong>, and buyers lose the input tax on such invoices. Penalties apply to unmatched or unreversed input tax (section 33(29)–(31)). Buyers should verify suppliers and invoices, not just collect paper.</p>

<h2>4. Debit and credit notes</h2>
<p>The Board may prescribe an electronic mechanism for debit and credit notes (section 9). Under the amended Chapter XIV, debit notes, credit notes and advance receipt invoices are issued electronically and kept for six years.</p>

<h2>5. Electronic scrutiny (SRO 1655(I)/2026)</h2>
<p>A new Chapter XII-A of the Sales Tax Rules lets FBR's systems cross-match returns with other data and send <strong>intimations through IRIS</strong>, allowing at least seven days to explain or correct before action. A clean monthly reconciliation of your e-invoices, books and Annex-C is the best defence.</p>

<h2>6. Stock transfer notes (STGO 25 of 2026)</h2>
<p>Goods moved from a factory to your <strong>own warehouse under the same STRN</strong> are not a supply and need no digital invoice — but each consignment must travel with a prescribed <strong>stock transfer note</strong>, be acknowledged by the warehouse, reconciled and kept for six years. A warehouse with a separate STRN receives a taxable supply.</p>

<h2>7. Federal excise duty on the invoice (SRO 1666(I)/2026)</h2>
<p>Chapter XIV now also applies to <strong>federal excise duty</strong> and to Islamabad Capital Territory services. Invoices gain six FED particulars, and tax on invoices reported with an FBR number but not accounted for in Annex-C or the return can be recovered (rule 150XD(2)).</p>

<h2>What to do now</h2>
<ul class="checklist">
  <li>Confirm you issue FBR-numbered invoices for exempt supplies and advance receipts</li>
  <li>Set up stock transfer notes for warehouse movements</li>
  <li>Add FED particulars if you deal in excisable goods or services</li>
  <li>Reconcile e-invoices, books and Annex-C every month before filing</li>
  <li>Check suppliers before claiming input tax</li>
</ul>
<p>The same Finance Act also reduced salaried income tax rates and removed the surcharge — see <a href="/knowledge-bank/salary-income-tax-slabs-tax-year-2027">salary tax slabs for Tax Year 2027</a>.</p>
HTML,
],

'sales-tax-return-annex-c-reconciliation' => [
    'slug' => 'sales-tax-return-annex-c-reconciliation',
    'title' => 'Monthly Sales Tax Return and Annex-C: A Reconciliation Playbook',
    'seo_title' => 'Sales Tax Return & Annex-C Reconciliation in Pakistan — Playbook',
    'description' => 'A step-by-step monthly routine to reconcile digital invoices, books, Annex-C and Annex-A before filing your sales tax return in Pakistan — and avoid FBR intimations.',
    'category' => 'sales-tax',
    'tags' => ['Sales tax return', 'Annex-C', 'Annex-A', 'Input tax', 'Reconciliation'],
    'date' => '2026-09-28',
    'updated' => '2026-10-06',
    'featured' => false,
    'excerpt' => 'With e-invoices feeding Annex-C automatically, your books and FBR\'s data must agree. A ten-step monthly routine that keeps them aligned.',
    'services' => ['sales-tax', 'accounting-bookkeeping'],
    'body' => <<<'HTML'
<p>Digital invoicing changed the monthly sales tax return. Your sales annex, <strong>Annex-C</strong>, is now populated from the invoices you report to FBR (rule 150R(5)), and your customers' purchase annex, <strong>Annex-A</strong>, is populated from it. If your books, your e-invoices and your return disagree, FBR's systems will notice — and since SRO 1655(I)/2026 they can send you an electronic intimation.</p>

<h2>The three-way tie</h2>
<p>Every month, three numbers must agree for each tax rate and sale type:</p>
<ol>
  <li><strong>Books</strong> — sales recorded in your ledger;</li>
  <li><strong>E-invoices</strong> — invoices accepted by FBR with an FBR number;</li>
  <li><strong>Return</strong> — the supplies declared in Annex-C.</li>
</ol>
<p>Differences are normal during the month; unexplained differences at filing time are not.</p>

<h2>A ten-step monthly routine</h2>
<ol>
  <li><strong>Clear the queue.</strong> Make sure no invoices are still pending, rejected or awaiting reconciliation with FBR.</li>
  <li><strong>Upload offline invoices</strong> issued during outages (within 24 hours of restoration).</li>
  <li><strong>Review cancellations</strong> made within the 72-hour window, and any Commissioner approvals for later ones.</li>
  <li><strong>Compare books to e-invoices</strong> by rate and sale type; investigate sales booked without an FBR number, and vice versa.</li>
  <li><strong>Check debit and credit notes</strong> and advance receipt invoices are reflected in both.</li>
  <li><strong>Download Annex-C</strong> from IRIS and tie it to your e-invoice register.</li>
  <li><strong>Review Annex-A</strong> — the input tax suppliers have reported for you — against your purchase ledger; chase suppliers whose invoices are missing.</li>
  <li><strong>Apply input tax rules</strong> — apportionment for exempt supplies, inadmissible input tax, and any reduced input tax ratio.</li>
  <li><strong>Calculate and pay</strong> the tax by the due date (generally the 15th of the following month).</li>
  <li><strong>File the return</strong> (generally by the 18th) and keep the reconciliation with the filed return.</li>
</ol>

<h2>Frequent causes of differences</h2>
<table class="table">
  <thead><tr><th>Difference</th><th>Usual cause</th></tr></thead>
  <tbody>
    <tr><td>Books higher than e-invoices</td><td>Invoices not reported, rejected, or stuck in an outage queue</td></tr>
    <tr><td>E-invoices higher than books</td><td>Duplicates reported after a timeout; invoices cancelled in the books but not with FBR</td></tr>
    <tr><td>Rate mismatch</td><td>Wrong sale type or SRO on the invoice line</td></tr>
    <tr><td>Missing input tax</td><td>Supplier did not report, or reported against a wrong NTN</td></tr>
  </tbody>
</table>

<div class="callout callout--warn"><strong>Watch out:</strong> an invoice reported with an FBR number but not accounted for in Annex-C or the return can be taxed by the officer unless it was cancelled through the approved mechanism (rule 150XD(2)).</div>

<p>Veridian E-invoicing Pakistan shows the <strong>books, FBR and Annex-C tie</strong> on its dashboard and exports an Annex-C reconciliation. <a href="/e-invoicing">See how it works</a>, or let our team run your monthly close.</p>
HTML,
],

'choosing-fbr-e-invoicing-route' => [
    'slug' => 'choosing-fbr-e-invoicing-route',
    'title' => 'PRAL, Licensed Integrator or Your Own System? Choosing Your FBR E-Invoicing Route',
    'seo_title' => 'Choosing an FBR E-Invoicing Integration: PRAL vs Integrator vs Own System',
    'description' => 'Compare the ways to integrate with FBR Digital Invoicing — PRAL, licensed integrators, your own system or the IRIS portal — on cost, control, data privacy and effort.',
    'category' => 'sales-tax',
    'tags' => ['E-invoicing software', 'PRAL', 'Licensed integrator', 'ERP integration'],
    'date' => '2026-09-22',
    'updated' => '2026-10-01',
    'featured' => false,
    'excerpt' => 'Four ways to meet FBR\'s digital invoicing requirement, compared on cost, control, privacy and effort — plus the questions to ask any software vendor.',
    'services' => ['fbr-digital-invoicing', 'technology-consulting'],
    'body' => <<<'HTML'
<p>Every sales-tax registered business must report invoices to FBR in real time. How you do it is a choice — and it affects your costs, your control over data and the effort your team spends each day.</p>

<h2>The four routes</h2>
<table class="table">
  <thead><tr><th>Route</th><th>How it works</th><th>Best for</th></tr></thead>
  <tbody>
    <tr><td><strong>Your own system through PRAL</strong></td><td>Your invoicing software or ERP calls FBR's DI API directly with your own security token. PRAL acts as the licensed integrator free of cost.</td><td>Most businesses that want control and no per-invoice fees</td></tr>
    <tr><td><strong>Licensed integrator</strong></td><td>A firm licensed by FBR operates the integration for you, for a fee within the limits FBR sets.</td><td>Businesses that prefer a managed service</td></tr>
    <tr><td><strong>ERP vendor connector</strong></td><td>Your existing ERP adds a module that reports invoices to FBR.</td><td>Companies already on a major ERP</td></tr>
    <tr><td><strong>IRIS web portal</strong></td><td>Invoices entered manually on FBR's portal.</td><td>Very low invoice volumes only</td></tr>
  </tbody>
</table>

<h2>What to compare</h2>
<ul>
  <li><strong>Total cost</strong> — licence, per-invoice or monthly fees, hardware and staff time.</li>
  <li><strong>Data control</strong> — does your sales data pass through a third party, or go straight from your system to FBR?</li>
  <li><strong>Validation</strong> — does the system check HS codes, units, sale types, rates, SROs and buyer status before reporting?</li>
  <li><strong>Resilience</strong> — what happens during an FBR or internet outage? Are invoices queued, never duplicated, and uploaded within 24 hours?</li>
  <li><strong>Compliance features</strong> — digital signatures, daily/weekly/monthly closings, audit logs, the 72-hour rule, incident reporting, signboards and stock transfer notes.</li>
  <li><strong>Reconciliation</strong> — can it tie your books, FBR and Annex-C?</li>
  <li><strong>Integration</strong> — an API for your ERP or POS, and imports from Excel.</li>
  <li><strong>Support</strong> — who helps with IRIS registration, sandbox scenarios and FBR error codes?</li>
</ul>

<h2>Questions to ask any vendor</h2>
<ol>
  <li>Which version of FBR's DI technical specification do you support, and how quickly do you update when it changes?</li>
  <li>How do you prevent the same invoice being reported twice after a timeout?</li>
  <li>Where is my data stored, and who can see it?</li>
  <li>Show me a printed invoice with every rule 150R(13) particular, including FED particulars.</li>
  <li>How do you handle the 72-hour cancellation window and Commissioner approvals?</li>
  <li>What does a monthly Annex-C reconciliation look like in your system?</li>
</ol>

<h2>Our approach</h2>
<p><a href="/e-invoicing">Veridian E-invoicing Pakistan</a> is installed on your own computer or server and reports directly to FBR with your own token through PRAL — no third party sees your sales. It includes a REST API for ERPs and POS systems, and our team handles registration and go-live.</p>
HTML,
],

];
