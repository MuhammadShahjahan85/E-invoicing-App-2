<?php
require __DIR__ . '/includes/bootstrap.php';

$crumbs = [['Home', ''], ['FAQs', 'faq']];
$groups = faqs();
$all = array_merge(...array_values($groups));
$page = [
    'title' => 'Frequently Asked Questions — Working with Veridian Partners',
    'description' => 'Answers about our services, fees, response times and confidentiality, tax filing, the Active Taxpayer List, FBR e-invoicing and company registration in Pakistan.',
    'keywords' => 'tax consultant FAQ Pakistan, FBR e-invoicing FAQ, company registration FAQ Pakistan, ATL FAQ',
    'path' => 'faq',
    'nav' => 'knowledge',
    'schema' => [breadcrumb_schema($crumbs), faq_schema($all)],
];
require __DIR__ . '/includes/header.php';

echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => 'FAQs',
    'title' => 'Frequently asked <em>questions</em>.',
    'lede' => 'Straight answers about working with us and the topics clients ask about most.',
]);
?>

<section class="section">
  <div class="container detail">
    <div class="detail__main">
      <?php foreach ($groups as $title => $items): ?>
      <div class="detail__block">
        <h2 style="font-size:clamp(1.5rem,2.4vw,2rem)"><?= e($title) ?></h2>
        <?= faq_list($items) ?>
      </div>
      <?php endforeach; ?>
    </div>
    <aside class="detail__aside">
      <?= expert_card() ?>
      <div class="side-card">
        <p class="side-card__heading">Jump to</p>
        <ul class="side-links">
          <li><a href="<?= e(url('glossary')) ?>">Glossary<?= icon('chevron-right') ?></a></li>
          <li><a href="<?= e(url('tax-calendar')) ?>">Tax calendar<?= icon('chevron-right') ?></a></li>
          <li><a href="<?= e(url('tools')) ?>">Free calculators<?= icon('chevron-right') ?></a></li>
          <li><a href="<?= e(url('knowledge-bank')) ?>">All guides<?= icon('chevron-right') ?></a></li>
        </ul>
      </div>
    </aside>
  </div>
</section>

<?= cta_band() ?>
<?php require __DIR__ . '/includes/footer.php'; ?>
