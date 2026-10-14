<?php
require __DIR__ . '/includes/bootstrap.php';

$slug = preg_replace('/[^a-z0-9-]/', '', (string) ($_GET['slug'] ?? ''));
$ind = industry($slug);
if (!$ind) {
    not_found();
}

$crumbs = [['Home', ''], ['Industries', 'industries'], [$ind['name'], 'industries/' . $ind['slug']]];
$page = [
    'title' => $ind['name'] . ' — Tax, Audit & Advisory in Pakistan',
    'description' => $ind['description'],
    'keywords' => $ind['keywords'],
    'path' => 'industries/' . $ind['slug'],
    'nav' => 'industries',
    'schema' => [breadcrumb_schema($crumbs), [
        '@type' => 'WebPage',
        'name' => $ind['name'] . ' advisory',
        'description' => $ind['description'],
        'about' => $ind['name'],
        'url' => abs_url('industries/' . $ind['slug']),
        'publisher' => ['@id' => abs_url('#organization')],
    ]],
];
require __DIR__ . '/includes/header.php';

echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => 'Industries',
    'title' => e($ind['name']),
    'lede' => e($ind['lede']),
    'actions' => '<a class="btn btn--gold btn--lg" href="' . e(url('book-consultation')) . '"><span>Talk to a sector adviser</span>' . icon('arrow-right') . '</a>',
]);

$related = [];
foreach ($ind['help'] as [$svcSlug]) {
    foreach (services() as $svc) {
        foreach ($svc['articles'] ?? [] as $aSlug) {
            if ($svc['slug'] === $svcSlug && ($a = article($aSlug))) {
                $related[$aSlug] = $a;
            }
        }
    }
}
?>

<section class="section">
  <div class="container detail">
    <div class="detail__main">
      <div class="detail__block">
        <p class="eyebrow">The challenges</p>
        <h2>What keeps <?= e(mb_strtolower($ind['name'])) ?> businesses up at night</h2>
        <ol class="challenge-list">
          <?php foreach ($ind['challenges'] as $c): ?><li class="reveal"><?= e($c) ?></li><?php endforeach; ?>
        </ol>
      </div>

      <div class="detail__block">
        <p class="eyebrow">How we help</p>
        <h2>Services for <?= e(mb_strtolower($ind['name'])) ?></h2>
        <div class="help-links">
          <?php foreach ($ind['help'] as [$svcSlug, $text]): $svc = service($svcSlug); if (!$svc) continue; ?>
          <a class="help-link reveal" href="<?= e(service_url($svc)) ?>"><span class="icon-tile icon-tile--sm"><?= icon($svc['icon']) ?></span><span><strong><?= e($svc['title']) ?></strong><small><?= e($text) ?></small></span><?= icon('arrow-right') ?></a>
          <?php endforeach; ?>
        </div>
      </div>

      <?php if ($related): ?>
      <div class="detail__block">
        <p class="eyebrow">Knowledge Bank</p>
        <h2>Useful guides</h2>
        <div class="grid grid--2">
          <?php foreach (array_slice($related, 0, 4) as $a): echo article_card($a); endforeach; ?>
        </div>
      </div>
      <?php endif; ?>

      <div class="detail__block" id="enquire">
        <p class="eyebrow">Enquire</p>
        <h2>Discuss your business with us</h2>
        <div class="panel" data-form-wrap><?= contact_form(['id' => 'industry-form']) ?></div>
      </div>
    </div>

    <aside class="detail__aside">
      <?= expert_card() ?>
      <div class="side-card">
        <p class="side-card__heading">Other industries</p>
        <ul class="side-links">
          <?php foreach (industries() as $o): ?>
          <li><a href="<?= e(url('industries/' . $o['slug'])) ?>"<?= $o['slug'] === $ind['slug'] ? ' class="is-active" aria-current="page"' : '' ?>><?= e($o['name']) ?><?= icon('chevron-right') ?></a></li>
          <?php endforeach; ?>
        </ul>
      </div>
    </aside>
  </div>
</section>

<?= cta_band() ?>
<?php require __DIR__ . '/includes/footer.php'; ?>
