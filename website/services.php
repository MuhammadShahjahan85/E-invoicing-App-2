<?php
require __DIR__ . '/includes/bootstrap.php';

$crumbs = [['Home', ''], ['Services', 'services']];
$page = [
    'title' => 'Our Services — Tax, Audit, Corporate, Accounting & Advisory in Pakistan',
    'description' => 'Income tax, sales tax, FBR e-invoicing, SECP corporate compliance, accounting and payroll, internal audit, IFRS reporting, business advisory, virtual CFO, ERP and business setup services in Pakistan.',
    'keywords' => 'tax services Pakistan, audit services Pakistan, corporate services SECP, accounting services Pakistan, advisory services Pakistan, FBR e-invoicing services',
    'path' => 'services',
    'nav' => 'services',
    'schema' => [
        breadcrumb_schema($crumbs),
        [
            '@type' => 'ItemList',
            'name' => 'Services of ' . company_name(),
            'itemListElement' => array_values(array_map(fn($s, $i) => [
                '@type' => 'ListItem', 'position' => $i + 1, 'name' => $s['title'], 'url' => absolute(service_url($s)),
            ], services(), array_keys(array_values(services())))),
        ],
    ],
];
require __DIR__ . '/includes/header.php';

echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => 'Services',
    'title' => 'Everything your business needs to stay <em>compliant</em> and grow.',
    'lede' => 'Eleven integrated service lines — delivered by one team that knows your business, your sector and Pakistan\'s rules.',
    'actions' => '<a class="btn btn--gold btn--lg" href="' . e(url('book-consultation')) . '"><span>Book a free consultation</span>' . icon('arrow-right') . '</a>',
]);
?>

<section class="section">
  <div class="container">
    <div class="grid grid--3">
      <?php $n = 0; foreach (services() as $s): $n++; echo service_card($s, $n); endforeach; ?>
      <a class="card card--service reveal" href="<?= e(url('contact')) ?>">
        <span class="icon-tile icon-tile--gold"><?= icon('message') ?></span>
        <h3 class="card__title">Something else?</h3>
        <p class="card__text">If your question touches tax, compliance, finance or controls, we can almost certainly help — or point you to someone who can.</p>
        <span class="link-arrow">Ask us <?= icon('arrow-right') ?></span>
      </a>
    </div>
  </div>
</section>

<section class="section section--tint">
  <div class="container">
    <div class="section-head section-head--center">
      <p class="eyebrow">Engagement models</p>
      <h2>Work with us the way that <em>suits</em> you.</h2>
      <p>From a single filing to a full outsourced finance function.</p>
    </div>
    <div class="feature-grid">
      <div class="feature reveal"><span class="icon-tile"><?= icon('target') ?></span><h3>One-off projects</h3><p>A company registration, a tax notice, a feasibility study or an e-invoicing go-live — fixed scope, fixed fee, clear deadline.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('refresh') ?></span><h3>Monthly retainers</h3><p>Bookkeeping, payroll, sales tax returns and compliance calendars handled every month for a predictable fee.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('users') ?></span><h3>Co-sourced support</h3><p>Our specialists work alongside your finance or audit team — for the year-end, a peak season or a complex project.</p></div>
    </div>
  </div>
</section>

<?= cta_band() ?>
<?php require __DIR__ . '/includes/footer.php'; ?>
