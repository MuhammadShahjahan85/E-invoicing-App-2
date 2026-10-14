<?php
require __DIR__ . '/includes/bootstrap.php';

$crumbs = [['Home', ''], ['FBR E-Invoicing Software', 'e-invoicing']];
$faqs = [
    ['q' => 'Is Veridian E-invoicing Pakistan approved for FBR digital invoicing?', 'a' => 'It is built to FBR\'s Digital Invoicing technical specification (DI API v1.12) and Chapter XIV of the Sales Tax Rules, 2006. Each installation is certified for production through the sandbox scenarios FBR assigns to the business on IRIS, and reports through PRAL — FBR\'s own licensed integrator, free of cost.'],
    ['q' => 'Where is my data stored?', 'a' => 'On your own computer or server. The software runs on your premises and sends invoices directly to FBR with your own security token. No third party sees your sales data.'],
    ['q' => 'What happens if the internet or FBR is down?', 'a' => 'Invoices are queued and resent automatically the moment FBR is reachable, identified as issued in offline mode where the rules require, and tracked on the dashboard until accepted. Timeouts are marked for reconciliation and never resent blindly, so a sale is not reported twice.'],
    ['q' => 'Can it connect to our ERP or POS?', 'a' => 'Yes. A REST API lets ERP and POS systems send invoices; the software validates them, reports them to FBR and returns the FBR invoice number and QR code. You can also import customers, products and invoices from Excel, CSV and other files.'],
    ['q' => 'Does it work on phones?', 'a' => 'Yes. It works in any browser on computers, tablets and phones, and installs on Android and iPhone as an app — no app store needed.'],
    ['q' => 'Can we try it first?', 'a' => 'Yes. A 30-day evaluation includes the training simulator and FBR sandbox testing. Production reporting is enabled with a licence for your NTN.'],
    ['q' => 'How long does go-live take?', 'a' => 'With IRIS credentials and a static IP ready, most businesses complete registration, sandbox scenarios and the switch to production within a few working days. Our team handles the process with you.'],
];
$page = [
    'title' => 'FBR E-Invoicing Software in Pakistan — Veridian E-invoicing Pakistan',
    'description' => 'FBR digital invoicing software for Pakistani sales-tax registered businesses: validation with FBR reference data, real-time reporting through PRAL, QR printing, 72-hour rule, offline mode and Annex-C reconciliation.',
    'keywords' => 'FBR e-invoicing software, FBR digital invoicing software, e-invoicing Pakistan, PRAL integration, DI API v1.12, FBR QR code invoice, sales tax invoice software, POS integration FBR, Annex-C reconciliation, Chapter XIV sales tax rules',
    'path' => 'e-invoicing',
    'nav' => 'einvoicing',
    'og_image' => 'assets/img/og-einvoicing.png',
    'schema' => [
        breadcrumb_schema($crumbs),
        [
            '@type' => 'SoftwareApplication',
            'name' => 'Veridian E-invoicing Pakistan',
            'applicationCategory' => 'BusinessApplication',
            'applicationSubCategory' => 'E-invoicing',
            'operatingSystem' => 'Windows, Linux; browser access on Android and iOS',
            'description' => 'On-premise FBR Digital Invoicing software for Pakistani sales-tax registered businesses, built to DI API v1.12 and Chapter XIV of the Sales Tax Rules, 2006.',
            'softwareVersion' => '1.2',
            'author' => ['@id' => abs_url('#organization')],
            'publisher' => ['@id' => abs_url('#organization')],
            'image' => abs_url('assets/img/product/dashboard.webp'),
            'screenshot' => abs_url('assets/img/product/invoice-view.webp'),
            'offers' => ['@type' => 'Offer', 'price' => '0', 'priceCurrency' => 'PKR', 'description' => 'Free 30-day evaluation; licences quoted per business.'],
        ],
        faq_schema($faqs),
    ],
];
require __DIR__ . '/includes/header.php';

$wa = whatsapp_link('Hello ' . company_name() . ', I would like a demo of Veridian E-invoicing Pakistan.');
$aside = '<div class="browser"><div class="browser__bar"><span></span><span></span><span></span><span class="browser__url">Accepted invoice · FBR number & QR</span></div>'
    . '<img src="' . e(asset('img/product/invoice-view.webp')) . '" alt="An invoice accepted by FBR with its FBR invoice number and QR code" width="1440" height="900"></div>';
echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => 'Veridian E-invoicing Pakistan',
    'title' => 'FBR digital invoicing, <em>done properly</em>.',
    'lede' => 'Prepare, validate and report every sales tax invoice to FBR in real time — with the controls the law requires and the reconciliation your return needs. Built by tax professionals, installed on your own server.',
    'actions' => '<a class="btn btn--gold btn--lg" href="' . e(url('book-consultation?service=fbr-digital-invoicing')) . '"><span>Request a demo</span>' . icon('arrow-right') . '</a>'
        . ($wa ? '<a class="btn btn--ghost-light btn--lg" href="' . e($wa) . '" target="_blank" rel="noopener">' . icon('whatsapp') . '<span>WhatsApp us</span></a>' : '<a class="btn btn--ghost-light btn--lg" href="#features"><span>See the features</span></a>')
        ,
    'after' => '<div class="page-hero__meta"><span>' . icon('check') . 'DI API v1.12</span><span>' . icon('check') . 'Chapter XIV compliant</span><span>' . icon('check') . 'On-premise</span><span>' . icon('check') . 'Android &amp; iPhone app</span></div>',
    'aside' => $aside,
    'class' => 'page-hero--product',
]);
?>

<section class="section section--deep" aria-labelledby="law-title">
  <div class="container">
    <div class="section-head">
      <p class="eyebrow eyebrow--light">Why it matters</p>
      <h2 id="law-title">The rules are strict. Your system should be <em>stricter</em>.</h2>
      <p>Since SRO 1852(I)/2025 every sales-tax registered person must issue digital invoices, and the Finance Act 2026 raised the stakes.</p>
    </div>
    <div class="law-grid">
      <div class="law reveal"><strong>Rs 1m</strong><span>Penalty for failing to integrate or record sales — then up to Rs 5 million if the default continues.</span></div>
      <div class="law reveal"><strong>72 hrs</strong><span>Window to cancel or edit an invoice issued by mistake. After that, the Commissioner's approval is needed.</span></div>
      <div class="law reveal"><strong>24 hrs</strong><span>To upload invoices issued offline after restoration — and to report system failures or tampering.</span></div>
      <div class="law reveal"><strong>6 yrs</strong><span>Electronic records must be preserved, unaltered, for six years.</span></div>
    </div>
  </div>
</section>

<section class="section" id="features" aria-labelledby="features-title">
  <div class="container">
    <div class="section-head section-head--center">
      <p class="eyebrow">Features</p>
      <h2 id="features-title">Everything Chapter XIV asks for — and the things it <em>doesn't</em>.</h2>
      <p>One program, installed on your PC or server, used from any browser in your office and on your phone.</p>
    </div>
    <div class="feature-grid">
      <div class="feature reveal"><span class="icon-tile"><?= icon('shield-check') ?></span><h3>Validated before it reaches FBR</h3><p>Every line is checked against FBR's own reference data — HS codes, units, sale types, rates, SROs and buyer status — so rejections become rare.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('zap') ?></span><h3>Real-time, outage-safe reporting</h3><p>Reports through FBR's DI API. During outages invoices queue and resend automatically; timeouts are never resent blindly.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('printer') ?></span><h3>Compliant printing</h3><p>A4 and 80 mm thermal formats with FBR number, QR code, DI logo, every rule 150R(13) particular and FED particulars.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('lock') ?></span><h3>Controls built in</h3><p>72-hour cancellation rule, offline mode, incident register with letters to FBR, digital signatures and daily, weekly and monthly closings.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('layers') ?></span><h3>Books · FBR · Annex-C tie</h3><p>Tax periods with due-date countdowns, a period-close checklist and an Annex-C reconciliation export — every month.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('database') ?></span><h3>The full Customs Tariff</h3><p>All 7,823 HS (PCT) codes built in and searchable; FBR's lists refreshed every day; bulk buyer checks against the Active Taxpayer List.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('upload') ?></span><h3>Import from any file</h3><p>Excel, CSV, PDF, Word and more. Columns recognised automatically; whatever FBR needs that the file lacks is asked for once.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('code') ?></span><h3>ERP & POS API</h3><p>A REST API with API keys and idempotent references connects your existing systems to FBR without double reporting.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('smartphone') ?></span><h3>Desktop, web and phone</h3><p>Installs on Android and iPhone as an app. Share accepted invoices on WhatsApp. Light and dark themes, optional Urdu captions.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('refresh') ?></span><h3>One-click updates</h3><p>New versions install with one click or overnight, digitally verified and with a backup taken first. Restore backups from the app.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('list-checks') ?></span><h3>Sandbox certification</h3><p>All 28 FBR scenarios with FBR's sample data, suggested from your business activity, plus a training simulator that needs no FBR access.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('chart-bar') ?></span><h3>Reports & exports</h3><p>Sales register, tax summaries and buyer-wise reports in PDF, Excel and CSV, and a monthly report pack for your file or tax adviser.</p></div>
    </div>
  </div>
</section>

<section class="section section--tint" aria-labelledby="shots-title">
  <div class="container">
    <div class="section-head section-head--split">
      <div>
        <p class="eyebrow">Take a look</p>
        <h2 id="shots-title">Designed for the people who <em>use</em> it every day.</h2>
        <p>Click any screen to enlarge it.</p>
      </div>
      <a class="btn btn--ghost" href="<?= e(url('book-consultation?service=fbr-digital-invoicing')) ?>"><span>Book a live demo</span><?= icon('arrow-right') ?></a>
    </div>
    <div class="gallery">
      <?php
      $shots = [
          ['dashboard', 'Dashboard with the books, FBR and Annex-C tie'],
          ['invoice-editor', 'Invoice entry with live tax calculation'],
          ['invoice-view', 'Accepted invoice with FBR number and QR code'],
          ['library-rates', 'FBR reference library: sale types, rates and SROs'],
          ['library-buyer', 'Buyer check against the Active Taxpayer List'],
          ['compliance', 'Tax periods, returns and the period-close checklist'],
          ['scenarios', 'FBR sandbox scenarios for certification'],
          ['print-a4', 'Printed A4 tax invoice'],
          ['import-match', 'Import from any file with column matching'],
          ['reports', 'Reports with Annex-C reconciliation'],
          ['stock-transfers', 'Stock transfer notes (STGO 25 of 2026)'],
          ['dashboard-dark', 'Dark theme'],
      ];
      foreach ($shots as [$file, $caption]): ?>
      <button class="shot reveal" type="button" data-lightbox="<?= e(asset('img/product/' . $file . '.webp')) ?>" data-caption="<?= e($caption) ?>">
        <img src="<?= e(asset('img/product/' . $file . '.webp')) ?>" alt="<?= e($caption) ?>" width="1440" height="900" loading="lazy">
        <span><?= e($caption) ?></span>
      </button>
      <?php endforeach; ?>
    </div>
  </div>
</section>

<section class="section" aria-labelledby="golive-title">
  <div class="container">
    <div class="section-head">
      <p class="eyebrow">Going live</p>
      <h2 id="golive-title">Live in days, <em>not months</em>.</h2>
      <p>Our team takes you through every step with FBR and PRAL.</p>
    </div>
    <ol class="steps" style="grid-template-columns:repeat(3,minmax(0,1fr));row-gap:2.5rem">
      <li class="step reveal"><span class="step__dot">1</span><h3>Register on IRIS</h3><p>Digital Invoicing registration with PRAL as integrator; the software shows exactly what to enter.</p></li>
      <li class="step reveal"><span class="step__dot">2</span><h3>Whitelist your IP</h3><p>Your static public IP is registered and approved by PRAL, usually within hours.</p></li>
      <li class="step reveal"><span class="step__dot">3</span><h3>Install & set up</h3><p>Windows installer or Linux; the setup wizard captures your business exactly as on IRIS.</p></li>
      <li class="step reveal"><span class="step__dot">4</span><h3>Pass sandbox scenarios</h3><p>Run the scenarios IRIS assigns — practice first in the built-in simulator.</p></li>
      <li class="step reveal"><span class="step__dot">5</span><h3>Production token</h3><p>Install your production token and licence, then switch to live reporting.</p></li>
      <li class="step reveal"><span class="step__dot">6</span><h3>Train & support</h3><p>Your team is trained, and we stay on hand for FBR errors, updates and month-end.</p></li>
    </ol>
  </div>
</section>

<section class="section section--tint" aria-labelledby="editions-title">
  <div class="container">
    <div class="section-head section-head--center">
      <p class="eyebrow">Editions</p>
      <h2 id="editions-title">Every edition is <em>fully</em> compliant.</h2>
      <p>Editions differ by the number of companies and users, integrations and support. Licences are quoted for your business — start with a free 30-day evaluation.</p>
    </div>
    <div class="editions">
      <div class="edition reveal">
        <h3>Standard</h3>
        <p class="edition__for">For a single business with one NTN.</p>
        <p class="edition__price">Quoted per business</p>
        <ul class="ticks"><li>All FBR compliance features</li><li>One company (NTN)</li><li>Small team of users</li><li>Desktop, web and phone access</li><li>E-mail support and updates</li></ul>
        <a class="btn btn--ghost btn--block" href="<?= e(url('book-consultation?service=fbr-digital-invoicing')) ?>">Request a quote</a>
      </div>
      <div class="edition edition--featured reveal">
        <span class="edition__badge">Recommended</span>
        <h3>Professional</h3>
        <p class="edition__for">For growing businesses and groups.</p>
        <p class="edition__price">Quoted per business</p>
        <ul class="ticks"><li>Everything in Standard</li><li>Several companies (NTNs)</li><li>Larger teams and roles</li><li>ERP / POS API integration</li><li>Go-live service and priority support</li></ul>
        <a class="btn btn--gold btn--block" href="<?= e(url('book-consultation?service=fbr-digital-invoicing')) ?>">Request a quote</a>
      </div>
      <div class="edition reveal">
        <h3>Enterprise</h3>
        <p class="edition__for">For multi-branch and high-volume operations.</p>
        <p class="edition__price">Custom</p>
        <ul class="ticks"><li>Everything in Professional</li><li>Unlimited companies and users</li><li>Multi-outlet and custom integrations</li><li>On-site training</li><li>Service levels agreed with you</li></ul>
        <a class="btn btn--ghost btn--block" href="<?= e(url('contact')) ?>">Talk to us</a>
      </div>
    </div>
  </div>
</section>

<section class="section" aria-labelledby="ein-faq-title">
  <div class="container faq-layout">
    <div class="faq-layout__intro">
      <p class="eyebrow">FAQs</p>
      <h2 id="ein-faq-title">Questions about the <em>software</em>.</h2>
      <p class="lead">Want the full picture of the regime? Read our <a href="<?= e(url('knowledge-bank/fbr-digital-invoicing-pakistan-complete-guide')) ?>">complete guide to FBR digital invoicing</a>.</p>
    </div>
    <?= faq_list($faqs) ?>
  </div>
</section>

<section class="section section--tint" id="demo" aria-labelledby="demo-title">
  <div class="container split split--wide-left" style="align-items:start">
    <div class="panel" data-form-wrap>
      <h2 id="demo-title" style="font-size:clamp(1.6rem,2.6vw,2.2rem)">Request a demo</h2>
      <p class="muted">See the software with your own products and customers. We reply within one business day.</p>
      <?= contact_form(['id' => 'demo-form', 'service' => 'fbr-digital-invoicing']) ?>
    </div>
    <div>
      <?= expert_card('fbr-digital-invoicing') ?>
      <div class="side-card">
        <p class="side-card__heading">Related reading</p>
        <ul class="side-links">
          <?php foreach (['fbr-digital-invoicing-pakistan-complete-guide', 'choosing-fbr-e-invoicing-route', 'sales-tax-invoice-requirements-pakistan', 'finance-act-2026-sales-tax-changes'] as $slug): $a = article($slug); if (!$a) continue; ?>
          <li><a href="<?= e(article_url($a)) ?>"><?= e($a['title']) ?><?= icon('chevron-right') ?></a></li>
          <?php endforeach; ?>
        </ul>
      </div>
    </div>
  </div>
</section>

<?php require __DIR__ . '/includes/footer.php'; ?>
