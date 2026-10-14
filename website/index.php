<?php
require __DIR__ . '/includes/bootstrap.php';

$homeFaqs = [];
foreach (faqs() as $group) {
    foreach ($group as $f) {
        if (!empty($f['home'])) {
            $homeFaqs[] = $f;
        }
    }
}
$deadlines = upcoming_deadlines(12);
$nextReturn = null;
foreach ($deadlines as $d) {
    if ($d['key'] === 'st-ret') { $nextReturn = $d; break; }
}
$nextIncome = null;
foreach ($deadlines as $d) {
    if ($d['type'] === 'income') { $nextIncome = $d; break; }
}
$articles = articles();
$featured = array_values(array_filter($articles, fn($a) => !empty($a['featured'])));
$insights = array_slice($featured ?: array_values($articles), 0, 3);
$testimonials = cfg('testimonials', []);

$page = [
    'title' => 'Veridian Partners | Audit, Tax, Advisory & FBR E-Invoicing in Pakistan',
    'description' => 'Veridian Partners is a Pakistani audit, tax and advisory firm: income tax and sales tax, SECP company registration, internal audit, bookkeeping, virtual CFO and FBR-integrated e-invoicing software.',
    'keywords' => 'tax consultant in Pakistan, audit and advisory firm Pakistan, FBR e-invoicing software, FBR digital invoicing, sales tax consultant, income tax return filing, company registration SECP, internal audit services, bookkeeping services Pakistan, virtual CFO Pakistan, tax advisors Lahore Karachi Islamabad',
    'path' => '',
    'nav' => 'home',
    'schema' => [faq_schema($homeFaqs)],
];
require __DIR__ . '/includes/header.php';
?>

<!-- ============ HERO ============ -->
<section class="hero" aria-labelledby="hero-title">
  <div class="hero__bg" aria-hidden="true"><span class="orb orb--1"></span><span class="orb orb--2"></span><span class="gridlines"></span></div>
  <div class="container hero__grid">
    <div class="hero__copy">
      <p class="eyebrow eyebrow--light">Audit · Tax · Advisory · Digital Compliance</p>
      <h1 id="hero-title">Clarity that moves your business <em>forward</em>.</h1>
      <p class="hero__lede">Veridian Partners helps Pakistani businesses stay compliant, audit-ready and focused on growth — combining deep expertise in tax, assurance and corporate affairs with technology we build ourselves, including FBR-integrated e-invoicing.</p>
      <div class="hero__actions">
        <a class="btn btn--gold btn--lg" href="<?= e(url('book-consultation')) ?>"><span>Book a free consultation</span><?= icon('arrow-right') ?></a>
        <a class="btn btn--ghost-light btn--lg" href="<?= e(url('services')) ?>"><span>Explore our services</span></a>
      </div>
      <div class="intent">
        <span class="intent__label">Where can we help today?</span>
        <div class="intent__chips">
          <a class="chip-link" href="<?= e(url('e-invoicing')) ?>"><?= icon('qr-code') ?>Go live on FBR e-invoicing</a>
          <a class="chip-link" href="<?= e(url('services/corporate-secp')) ?>"><?= icon('building') ?>Register a company</a>
          <a class="chip-link" href="<?= e(url('services/tax-advisory')) ?>"><?= icon('landmark') ?>File my income tax</a>
          <a class="chip-link" href="<?= e(url('services/sales-tax')) ?>"><?= icon('receipt') ?>Sales tax returns</a>
          <a class="chip-link" href="<?= e(url('services/accounting-bookkeeping')) ?>"><?= icon('book-open') ?>Outsource accounting</a>
          <a class="chip-link" href="<?= e(url('services/internal-audit')) ?>"><?= icon('shield-check') ?>Strengthen controls</a>
          <a class="chip-link" href="<?= e(url('services/business-setup')) ?>"><?= icon('plane') ?>Invest from abroad</a>
        </div>
      </div>
    </div>

    <div class="hero__visual" data-animate aria-hidden="true">
      <div class="glass dash">
        <div class="dash__head">
          <div class="dash__title"><?= brand_mark('', 'dash') ?><span>Compliance overview<small><?= e(date('F Y')) ?></small></span></div>
          <span class="dash__live">Live</span>
        </div>
        <div class="dash__body">
          <div class="ring">
            <svg viewBox="0 0 150 150"><defs><linearGradient id="ringGrad" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#E3B341"/><stop offset="1" stop-color="#6CC6AC"/></linearGradient></defs><circle class="ring__track" cx="75" cy="75" r="64"/><circle class="ring__value" cx="75" cy="75" r="64"/></svg>
            <div class="ring__label"><strong>94%</strong><span>on track</span></div>
          </div>
          <ul class="dash__list">
            <?php if ($nextReturn): ?>
            <li class="dash__item"><span>Sales tax return<small>Due <?= e(date('j M', strtotime($nextReturn['date']))) ?></small></span><span class="pill pill--due"><?= e(countdown_label(days_until($nextReturn['date']))) ?></span></li>
            <?php endif; ?>
            <li class="dash__item"><span>FBR e-invoices today<small>All reported in real time</small></span><span class="pill pill--ok">248 ✓</span></li>
            <?php if ($nextIncome): ?>
            <li class="dash__item"><span><?= e(strtok($nextIncome['title'], '—')) ?><small>Due <?= e(date('j M', strtotime($nextIncome['date']))) ?></small></span><span class="pill pill--info">Planned</span></li>
            <?php endif; ?>
            <li class="dash__item"><span>Annex-C reconciliation<small>Books · FBR · return</small></span><span class="pill pill--ok">Tied</span></li>
          </ul>
        </div>
        <div class="dash__bars"><span style="height:42%"></span><span style="height:58%"></span><span style="height:50%"></span><span style="height:72%"></span><span style="height:64%"></span><span style="height:80%"></span><span style="height:76%"></span><span style="height:92%"></span><span style="height:86%"></span><span style="height:100%"></span></div>
      </div>
      <div class="glass float-card float-card--invoice">
        <span class="mini-qr"><?= decorative_qr() ?></span>
        <span><strong>FBR invoice accepted</strong><small>FBR No. ••••DI••••7823</small></span>
      </div>
      <div class="glass float-card float-card--close">
        <span class="float-card__icon"><?= icon('clipboard-check') ?></span>
        <span><strong>Month-end close</strong><small>Day 3 of 5</small><span class="progress"><span></span></span></span>
      </div>
    </div>
  </div>
  <div class="trust">
    <div class="container">
      <ul class="trust__list">
        <li class="trust__item"><?= icon('qr-code') ?><span><strong>FBR DI API v1.12</strong>Our own e-invoicing software</span></li>
        <li class="trust__item"><?= icon('building') ?><span><strong>Companies Act, 2017</strong>SECP corporate compliance</span></li>
        <li class="trust__item"><?= icon('chart-bar') ?><span><strong>IFRS & IFRS S1/S2</strong>Reporting you can rely on</span></li>
        <li class="trust__item"><?= icon('globe') ?><span><strong>Nationwide</strong>Remote-first, on-site when needed</span></li>
      </ul>
    </div>
  </div>
</section>

<!-- ============ INTRO ============ -->
<section class="section" aria-labelledby="intro-title">
  <div class="container statement">
    <div class="reveal">
      <p class="eyebrow">Who we are</p>
      <h2 id="intro-title" class="statement__quote">A modern firm for Pakistan's new era of <em>real-time</em> compliance.</h2>
    </div>
    <div class="reveal">
      <p class="lead">FBR now sees every invoice as it is issued. SECP expects transparent ownership and timely filings. Lenders and investors want numbers they can trust. We help businesses meet all of it — calmly, accurately and on time — so owners can spend their energy on growth.</p>
      <div class="pillars">
        <div class="pillar"><h3><?= icon('users') ?>Senior attention</h3><p>Experienced advisers lead every engagement, from the first call to the final filing.</p></div>
        <div class="pillar"><h3><?= icon('scale') ?>Clear, fixed fees</h3><p>Scope and price agreed upfront. No surprises, no hourly meters running.</p></div>
        <div class="pillar"><h3><?= icon('cpu') ?>Technology-enabled</h3><p>We build software for FBR compliance — and use automation to work faster for you.</p></div>
        <div class="pillar"><h3><?= icon('lock') ?>Confidential by design</h3><p>Strict access controls and confidentiality on every file we touch.</p></div>
      </div>
    </div>
  </div>
  <div class="container mt-3">
    <div class="stats">
      <div class="stat reveal"><span class="stat__value"><span data-count="11">11</span></span><span class="stat__label">Integrated service lines — one point of contact</span></div>
      <div class="stat reveal"><span class="stat__value"><span data-count="7823">7,823</span></span><span class="stat__label">HS codes built into our FBR e-invoicing software</span></div>
      <div class="stat reveal"><span class="stat__value"><span data-count="28">28</span></span><span class="stat__label">FBR sandbox scenarios supported out of the box</span></div>
      <div class="stat reveal"><span class="stat__value">1<sup>day</sup></span><span class="stat__label">Maximum reply time on every enquiry (business days)</span></div>
    </div>
  </div>
</section>

<!-- ============ SERVICES ============ -->
<section class="section section--tint" aria-labelledby="services-title">
  <div class="container">
    <div class="section-head section-head--split">
      <div>
        <p class="eyebrow">What we do</p>
        <h2 id="services-title">End-to-end expertise, <em>one</em> trusted partner.</h2>
        <p>Tax, assurance, corporate, accounting, advisory and technology — joined up, so nothing falls between the cracks.</p>
      </div>
      <a class="btn btn--ghost" href="<?= e(url('services')) ?>"><span>All services</span><?= icon('arrow-right') ?></a>
    </div>
    <div class="grid grid--3">
      <?php $n = 0; foreach (services() as $s): $n++; echo service_card($s, $n); endforeach; ?>
      <a class="card card--service reveal" href="<?= e(url('book-consultation')) ?>" style="background:linear-gradient(160deg,var(--deep-3),var(--deep-1));border-color:transparent;color:var(--on-deep-2)">
        <span class="icon-tile icon-tile--deep"><?= icon('compass') ?></span>
        <h3 class="card__title" style="color:#fff">Not sure where to start?</h3>
        <p class="card__text" style="color:var(--on-deep-2)">Tell us about your business. We will map what you need, what can wait and what it will cost — free.</p>
        <span class="link-arrow link-arrow--light">Book a free consultation <?= icon('arrow-right') ?></span>
      </a>
    </div>
  </div>
</section>

<!-- ============ PRODUCT ============ -->
<section class="section product" aria-labelledby="product-title">
  <span class="gridlines" aria-hidden="true"></span>
  <div class="container split">
    <div class="reveal">
      <p class="eyebrow eyebrow--light">Our software · Veridian E-invoicing Pakistan</p>
      <h2 id="product-title">FBR digital invoicing, <em>built by tax professionals</em>.</h2>
      <p class="product__lede">Prepare, validate and report every sales tax invoice to FBR in real time — with the controls Chapter XIV of the Sales Tax Rules demands, and the reconciliation your tax return needs.</p>
      <div class="deadline-box"><?= icon('alert') ?><p class="mb-0"><strong>Every sales-tax registered business should now be live.</strong> Since the Finance Act 2026, failing to integrate costs Rs 1 million — then up to Rs 5 million — and can lead to suspension of registration.</p></div>
      <ul class="checks checks--2">
        <li><?= icon('check') ?>Validated against FBR's own reference data</li>
        <li><?= icon('check') ?>Real-time reporting, safe during outages</li>
        <li><?= icon('check') ?>QR code, FBR number and DI logo on every print</li>
        <li><?= icon('check') ?>72-hour rule, offline mode and incident letters</li>
        <li><?= icon('check') ?>Books · FBR · Annex-C reconciliation</li>
        <li><?= icon('check') ?>On your own server — your data stays yours</li>
      </ul>
      <div class="hero__actions mb-0">
        <a class="btn btn--gold btn--lg" href="<?= e(url('e-invoicing')) ?>"><span>Explore the software</span><?= icon('arrow-right') ?></a>
        <a class="btn btn--ghost-light btn--lg" href="<?= e(url('book-consultation?service=fbr-digital-invoicing')) ?>"><span>Request a demo</span></a>
      </div>
    </div>
    <div class="product__media reveal">
      <div class="browser">
        <div class="browser__bar"><span></span><span></span><span></span><span class="browser__url">einvoice.yourcompany.local/dashboard</span></div>
        <img src="<?= e(asset('img/product/dashboard.webp')) ?>" alt="Veridian E-invoicing Pakistan dashboard showing invoices reported to FBR, tax due and the books, FBR and Annex-C tie" width="1440" height="900" loading="lazy">
      </div>
      <div class="phone"><img src="<?= e(asset('img/product/phone-invoice.webp')) ?>" alt="An FBR-accepted invoice with QR code on a phone" width="541" height="1101" loading="lazy"></div>
    </div>
  </div>
</section>

<!-- ============ WHY ============ -->
<section class="section" aria-labelledby="why-title">
  <div class="container">
    <div class="section-head section-head--center">
      <p class="eyebrow">Why Veridian</p>
      <h2 id="why-title">The rigour of a large firm. The <em>care</em> of a partner.</h2>
      <p>We set out to build the firm we would want to hire ourselves: senior, responsive, transparent — and fluent in both law and technology.</p>
    </div>
    <div class="why">
      <div class="why__item reveal"><span class="why__num">01</span><h3>Partner-level attention</h3><p>Your work is led by experienced advisers who know your file — not passed down a long chain.</p></div>
      <div class="why__item reveal"><span class="why__num">02</span><h3>Joined-up advice</h3><p>Tax, corporate, accounting and audit teams work as one, so a decision in one area never creates a problem in another.</p></div>
      <div class="why__item reveal"><span class="why__num">03</span><h3>We build the tools</h3><p>Our own FBR-integrated software and automation mean fewer manual errors and faster turnaround.</p></div>
      <div class="why__item reveal"><span class="why__num">04</span><h3>Transparent pricing</h3><p>Fixed fees or clear retainers agreed before work starts. You always know what you are paying for.</p></div>
      <div class="why__item reveal"><span class="why__num">05</span><h3>Fast, human responses</h3><p>A reply within one business day, a named contact and WhatsApp when you need a quick answer.</p></div>
      <div class="why__item reveal"><span class="why__num">06</span><h3>Plain-language advice</h3><p>We explain what the law requires and what it means for you — then put it in writing you can rely on.</p></div>
    </div>
  </div>
</section>

<!-- ============ APPROACH ============ -->
<section class="section section--deep" aria-labelledby="approach-title">
  <div class="container">
    <div class="section-head">
      <p class="eyebrow eyebrow--light">How we work</p>
      <h2 id="approach-title">A clear path from first call to <em>peace of mind</em>.</h2>
      <p>Every engagement follows the same disciplined rhythm — so you always know where things stand.</p>
    </div>
    <ol class="steps">
      <li class="step reveal"><span class="step__dot">1</span><h3 style="color:#fff">Listen</h3><p>A free consultation to understand your business, deadlines and concerns.</p></li>
      <li class="step reveal"><span class="step__dot">2</span><h3 style="color:#fff">Diagnose</h3><p>We review your records and FBR/SECP position, and identify risks and opportunities.</p></li>
      <li class="step reveal"><span class="step__dot">3</span><h3 style="color:#fff">Deliver</h3><p>A fixed scope and fee, then the work — filed, reconciled and documented.</p></li>
      <li class="step reveal"><span class="step__dot">4</span><h3 style="color:#fff">Sustain</h3><p>A compliance calendar, proactive reminders and advice as the law changes.</p></li>
    </ol>
  </div>
</section>

<!-- ============ INDUSTRIES ============ -->
<section class="section" aria-labelledby="industries-title">
  <div class="container">
    <div class="section-head section-head--split">
      <div>
        <p class="eyebrow">Industries</p>
        <h2 id="industries-title">Advice that speaks your <em>sector's</em> language.</h2>
        <p>How you are taxed and regulated depends on what you do. Our advice starts there.</p>
      </div>
      <a class="btn btn--ghost" href="<?= e(url('industries')) ?>"><span>All industries</span><?= icon('arrow-right') ?></a>
    </div>
    <div class="grid grid--3">
      <?php foreach (industries() as $i): echo industry_card($i); endforeach; ?>
    </div>
  </div>
</section>

<!-- ============ COMPLIANCE HUB ============ -->
<section class="section section--tint" aria-labelledby="hub-title">
  <div class="container">
    <div class="section-head section-head--center">
      <p class="eyebrow">Free compliance hub</p>
      <h2 id="hub-title">Stay ahead of every <em>deadline</em>.</h2>
      <p>Live countdowns to the next filing dates, free calculators and a plain-language glossary — use them any time.</p>
    </div>
    <div class="split split--wide-left" style="align-items:start">
      <div class="widget reveal">
        <div class="widget__head">
          <h3 class="widget__title"><?= icon('calendar') ?>Upcoming deadlines</h3>
          <a class="link-arrow" href="<?= e(url('tax-calendar')) ?>">Full calendar <?= icon('arrow-right') ?></a>
        </div>
        <ul class="deadlines">
          <?php foreach (array_slice($deadlines, 0, 5) as $d): echo deadline_row($d); endforeach; ?>
        </ul>
        <p class="tool__note">General due dates. FBR or SECP may extend a date by notification.</p>
      </div>
      <div class="grid" style="gap:1rem">
        <a class="card card--industry reveal" href="<?= e(url('tools#salary-tax')) ?>"><span class="icon-tile icon-tile--sm icon-tile--gold"><?= icon('calculator') ?></span><span class="card--industry__body"><span class="card__title">Salary tax calculator 2026-27</span><span class="card__text">Monthly and annual tax under the new Tax Year 2027 slabs.</span></span><?= icon('arrow-up-right', 'card--industry__arrow') ?></a>
        <a class="card card--industry reveal" href="<?= e(url('tools#sales-tax')) ?>"><span class="icon-tile icon-tile--sm icon-tile--gold"><?= icon('percent') ?></span><span class="card--industry__body"><span class="card__title">Sales tax calculator</span><span class="card__text">Add or extract 18% sales tax and 4% further tax.</span></span><?= icon('arrow-up-right', 'card--industry__arrow') ?></a>
        <a class="card card--industry reveal" href="<?= e(url('tools#amount-in-words')) ?>"><span class="icon-tile icon-tile--sm icon-tile--gold"><?= icon('type') ?></span><span class="card--industry__body"><span class="card__title">Amount in words</span><span class="card__text">Rupees in lakh and crore — for cheques and invoices.</span></span><?= icon('arrow-up-right', 'card--industry__arrow') ?></a>
        <a class="card card--industry reveal" href="<?= e(url('glossary')) ?>"><span class="icon-tile icon-tile--sm icon-tile--gold"><?= icon('book-open') ?></span><span class="card--industry__body"><span class="card__title">Tax & business glossary</span><span class="card__text">NTN, STRN, ATL, Annex-C and more, explained simply.</span></span><?= icon('arrow-up-right', 'card--industry__arrow') ?></a>
      </div>
    </div>
  </div>
</section>

<!-- ============ INSIGHTS ============ -->
<section class="section" aria-labelledby="insights-title">
  <div class="container">
    <div class="section-head section-head--split">
      <div>
        <p class="eyebrow">Knowledge Bank</p>
        <h2 id="insights-title">Insight you can <em>act on</em>.</h2>
        <p>Practical guides to Pakistani tax, corporate law, audit and reporting — written by practitioners, updated as the law changes.</p>
      </div>
      <a class="btn btn--ghost" href="<?= e(url('knowledge-bank')) ?>"><span>All insights</span><?= icon('arrow-right') ?></a>
    </div>
    <div class="grid grid--3">
      <?php foreach ($insights as $i => $a): echo article_card($a, $i === 0 ? 'is-featured' : ''); endforeach; ?>
    </div>
  </div>
</section>

<?php if ($testimonials): ?>
<!-- ============ TESTIMONIALS ============ -->
<section class="section section--tint" aria-labelledby="quotes-title">
  <div class="container">
    <div class="section-head section-head--center">
      <p class="eyebrow">Client voices</p>
      <h2 id="quotes-title">Trusted by businesses that <em>value</em> getting it right.</h2>
    </div>
    <div class="quotes">
      <?php foreach ($testimonials as $t): ?>
      <figure class="quote-card reveal"><?= icon('quote') ?><blockquote><?= e($t['quote']) ?></blockquote><figcaption><strong><?= e($t['name']) ?></strong><span><?= e($t['role']) ?></span></figcaption></figure>
      <?php endforeach; ?>
    </div>
  </div>
</section>
<?php endif; ?>

<!-- ============ FAQ ============ -->
<section class="section<?= $testimonials ? '' : ' section--tint' ?>" aria-labelledby="faq-title">
  <div class="container faq-layout">
    <div class="faq-layout__intro">
      <p class="eyebrow">Questions</p>
      <h2 id="faq-title">Good questions deserve <em>straight</em> answers.</h2>
      <p class="lead">Can't find what you are looking for? Our team is a message away.</p>
      <div class="hero__actions">
        <a class="btn btn--primary" href="<?= e(url('faq')) ?>"><span>All FAQs</span><?= icon('arrow-right') ?></a>
        <a class="btn btn--ghost" href="<?= e(url('contact')) ?>"><span>Ask a question</span></a>
      </div>
      <p class="eyebrow mt-2">Serving clients across Pakistan</p>
      <div class="cities">
        <?php foreach (['Lahore', 'Karachi', 'Islamabad', 'Rawalpindi', 'Faisalabad', 'Multan', 'Peshawar', 'Sialkot', 'Gujranwala', 'Quetta', 'Hyderabad', 'Overseas'] as $c): ?>
        <span class="city"><?= icon('map-pin') ?><?= e($c) ?></span>
        <?php endforeach; ?>
      </div>
    </div>
    <?= faq_list($homeFaqs) ?>
  </div>
</section>

<?= cta_band() ?>

<?php require __DIR__ . '/includes/footer.php'; ?>
