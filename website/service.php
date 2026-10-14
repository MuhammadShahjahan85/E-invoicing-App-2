<?php
require __DIR__ . '/includes/bootstrap.php';

$slug = preg_replace('/[^a-z0-9-]/', '', (string) ($_GET['slug'] ?? ''));
$s = service($slug);
if (!$s) {
    not_found();
}
if (!empty($s['url'])) {
    header('Location: ' . url($s['url']), true, 301);
    exit;
}

$crumbs = [['Home', ''], ['Services', 'services'], [$s['title'], 'services/' . $s['slug']]];
$schema = [
    breadcrumb_schema($crumbs),
    [
        '@type' => 'Service',
        'name' => $s['title'],
        'serviceType' => $s['title'],
        'description' => $s['description'],
        'provider' => ['@id' => abs_url('#organization')],
        'areaServed' => ['@type' => 'Country', 'name' => 'Pakistan'],
        'url' => abs_url('services/' . $s['slug']),
        'hasOfferCatalog' => [
            '@type' => 'OfferCatalog',
            'name' => $s['title'],
            'itemListElement' => array_map(fn($o) => ['@type' => 'Offer', 'itemOffered' => ['@type' => 'Service', 'name' => $o[0], 'description' => $o[1]]], $s['offerings']),
        ],
    ],
];
if (!empty($s['faqs'])) {
    $schema[] = faq_schema($s['faqs']);
}
$page = [
    'title' => $s['seo_title'],
    'description' => $s['description'],
    'keywords' => $s['keywords'],
    'path' => 'services/' . $s['slug'],
    'nav' => 'services',
    'schema' => $schema,
];
require __DIR__ . '/includes/header.php';

$wa = whatsapp_link('Hello ' . company_name() . ', I would like to discuss ' . $s['title'] . '.');
echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => $s['nav'],
    'title' => e($s['title']),
    'lede' => e($s['lede']),
    'actions' => '<a class="btn btn--gold btn--lg" href="' . e(url('book-consultation?service=' . $s['slug'])) . '"><span>Book a free consultation</span>' . icon('arrow-right') . '</a>'
        . ($wa ? '<a class="btn btn--ghost-light btn--lg" href="' . e($wa) . '" target="_blank" rel="noopener">' . icon('whatsapp') . '<span>WhatsApp us</span></a>'
               : '<a class="btn btn--ghost-light btn--lg" href="#enquire"><span>Send an enquiry</span></a>'),
]);
?>

<section class="section">
  <div class="container detail">
    <div class="detail__main">
      <div class="detail__block">
        <p class="eyebrow">Overview</p>
        <div class="prose lede-first"><?= content_html($s['intro']) ?></div>
      </div>

      <div class="detail__block">
        <p class="eyebrow">What we deliver</p>
        <h2>How we help</h2>
        <div class="offerings">
          <?php foreach ($s['offerings'] as $o): ?>
          <div class="offering reveal"><h3><?= icon('check-circle') ?><?= e($o[0]) ?></h3><p><?= e($o[1]) ?></p></div>
          <?php endforeach; ?>
        </div>
      </div>

      <div class="detail__block two-lists">
        <div class="list-card reveal">
          <h3><?= icon('users') ?>Who it is for</h3>
          <ul class="ticks"><?php foreach ($s['audience'] as $a): ?><li><?= e($a) ?></li><?php endforeach; ?></ul>
        </div>
        <div class="list-card reveal">
          <h3><?= icon('clipboard-check') ?>What you receive</h3>
          <ul class="ticks"><?php foreach ($s['deliverables'] as $d): ?><li><?= e($d) ?></li><?php endforeach; ?></ul>
        </div>
      </div>

      <div class="detail__block">
        <p class="eyebrow">Our approach</p>
        <h2>How an engagement runs</h2>
        <ol class="steps" style="grid-template-columns:repeat(2,minmax(0,1fr));row-gap:2rem">
          <li class="step reveal"><span class="step__dot">1</span><h3>Free consultation</h3><p>We listen, ask the right questions and explain your options in plain language.</p></li>
          <li class="step reveal"><span class="step__dot">2</span><h3>Fixed proposal</h3><p>A written scope, timeline and fixed fee — before any work begins.</p></li>
          <li class="step reveal"><span class="step__dot">3</span><h3>Delivery</h3><p>Senior-led work, regular updates and documents you can rely on.</p></li>
          <li class="step reveal"><span class="step__dot">4</span><h3>Ongoing care</h3><p>Reminders, updates when the law changes and a team that knows your file.</p></li>
        </ol>
      </div>

      <?php if (!empty($s['faqs'])): ?>
      <div class="detail__block">
        <p class="eyebrow">FAQs</p>
        <h2>Common questions</h2>
        <?= faq_list($s['faqs']) ?>
      </div>
      <?php endif; ?>

      <?php $rel = array_filter(array_map('article', $s['articles'] ?? [])); if ($rel): ?>
      <div class="detail__block">
        <p class="eyebrow">Knowledge Bank</p>
        <h2>Related guides</h2>
        <div class="grid grid--2">
          <?php foreach (array_slice($rel, 0, 4) as $a): echo article_card($a); endforeach; ?>
        </div>
      </div>
      <?php endif; ?>

      <?php $inds = array_filter(array_map('industry', $s['industries'] ?? [])); if ($inds): ?>
      <div class="detail__block">
        <p class="eyebrow">Industries</p>
        <h2>Sectors where this matters most</h2>
        <div class="grid grid--2">
          <?php foreach ($inds as $i): echo industry_card($i); endforeach; ?>
        </div>
      </div>
      <?php endif; ?>

      <div class="detail__block" id="enquire">
        <p class="eyebrow">Enquire</p>
        <h2>Tell us what you need</h2>
        <div class="panel" data-form-wrap><?= contact_form(['id' => 'service-form', 'service' => $s['slug']]) ?></div>
      </div>
    </div>

    <aside class="detail__aside">
      <?= expert_card($s['slug']) ?>
      <div class="side-card">
        <p class="side-card__heading">Other services</p>
        <ul class="side-links">
          <?php foreach (services() as $o): ?>
          <li><a href="<?= e(service_url($o)) ?>"<?= $o['slug'] === $s['slug'] ? ' class="is-active" aria-current="page"' : '' ?>><?= e($o['nav']) ?><?= icon('chevron-right') ?></a></li>
          <?php endforeach; ?>
        </ul>
      </div>
    </aside>
  </div>
</section>

<?= cta_band() ?>
<?php require __DIR__ . '/includes/footer.php'; ?>
