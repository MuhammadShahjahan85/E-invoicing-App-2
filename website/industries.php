<?php
require __DIR__ . '/includes/bootstrap.php';

$crumbs = [['Home', ''], ['Industries', 'industries']];
$page = [
    'title' => 'Industries We Serve — Sector-Specific Tax & Advisory in Pakistan',
    'description' => 'Tax, audit and advisory for manufacturing, textiles, retail, pharmaceuticals, IT exporters, construction, trading, NGOs, hospitality, agriculture, start-ups and overseas investors in Pakistan.',
    'keywords' => 'industry tax advisory Pakistan, manufacturing tax consultant, textile exporters tax, retail POS FBR, IT company tax Pakistan, NGO audit Pakistan',
    'path' => 'industries',
    'nav' => 'industries',
    'schema' => [breadcrumb_schema($crumbs)],
];
require __DIR__ . '/includes/header.php';

echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => 'Industries',
    'title' => 'Your sector has its own rules. We <em>know</em> them.',
    'lede' => 'Every industry is taxed, regulated and audited differently. Our advice starts with how your business actually works.',
]);
?>

<section class="section">
  <div class="container">
    <div class="grid grid--3">
      <?php foreach (industries() as $i): echo industry_card($i); endforeach; ?>
    </div>
  </div>
</section>

<section class="section section--tint">
  <div class="container split">
    <div class="reveal">
      <p class="eyebrow">Don't see your sector?</p>
      <h2>Every business is <em>unique</em>.</h2>
      <p class="lead">We work with businesses of every kind — from logistics and energy to professional services and family offices. Tell us about yours and we will show you how we can help.</p>
      <div class="hero__actions">
        <a class="btn btn--primary btn--lg" href="<?= e(url('book-consultation')) ?>"><span>Book a free consultation</span><?= icon('arrow-right') ?></a>
      </div>
    </div>
    <div class="reveal">
      <ul class="checks checks--light">
        <li><?= icon('check') ?><span><strong>Sector-specific tax treatment</strong> — schedules, SROs, withholding regimes and exemptions that apply to you.</span></li>
        <li><?= icon('check') ?><span><strong>Regulatory awareness</strong> — from DRAP and PSEB to provincial revenue authorities.</span></li>
        <li><?= icon('check') ?><span><strong>Industry benchmarks</strong> — controls and KPIs that fit how your sector operates.</span></li>
        <li><?= icon('check') ?><span><strong>Technology that fits</strong> — POS, ERP and FBR integration for your workflow.</span></li>
      </ul>
    </div>
  </div>
</section>

<?= cta_band() ?>
<?php require __DIR__ . '/includes/footer.php'; ?>
