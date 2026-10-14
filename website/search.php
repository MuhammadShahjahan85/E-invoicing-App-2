<?php
require __DIR__ . '/includes/bootstrap.php';
require __DIR__ . '/includes/search.php';

$q = trim(mb_substr((string) ($_GET['q'] ?? ''), 0, 100));
$results = $q !== '' ? site_search($q) : [];
$crumbs = [['Home', ''], ['Search', 'search']];
$page = [
    'title' => $q !== '' ? 'Search results for “' . $q . '”' : 'Search',
    'description' => 'Search Veridian Partners services, guides, glossary and FAQs.',
    'path' => 'search',
    'noindex' => true,
    'schema' => [breadcrumb_schema($crumbs)],
];
require __DIR__ . '/includes/header.php';

echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => 'Search',
    'title' => $q !== '' ? 'Results for “' . e($q) . '”' : 'Search the <em>site</em>',
    'lede' => $q !== '' ? count($results) . ' result' . (count($results) === 1 ? '' : 's') . ' across services, guides, industries, the glossary and FAQs.' : 'Find services, guides, tax terms and answers.',
]);
?>

<section class="section">
  <div class="container container--narrow">
    <form action="<?= e(url('search')) ?>" method="get" role="search" style="margin-bottom:2rem">
      <label class="search-field" style="max-width:none"><span class="sr-only">Search</span><?= icon('search') ?><input type="search" name="q" value="<?= e($q) ?>" placeholder="Search services, guides, tax terms…" autofocus></label>
    </form>
    <?php if ($q !== '' && !$results): ?>
      <div class="empty-state"><?= icon('search') ?><p>Nothing found for “<?= e($q) ?>”. Try a different word, browse the <a href="<?= e(url('knowledge-bank')) ?>">Knowledge Bank</a>, or <a href="<?= e(url('contact')) ?>">ask our team</a>.</p></div>
    <?php endif; ?>
    <div class="help-links">
      <?php foreach ($results as $r): ?>
      <a class="help-link" href="<?= e($r['u']) ?>"><span class="search-result__type"><?= e($r['y']) ?></span><span><strong><?= e($r['t']) ?></strong><small><?= e($r['d']) ?></small></span><?= icon('arrow-right') ?></a>
      <?php endforeach; ?>
    </div>
  </div>
</section>

<?php require __DIR__ . '/includes/footer.php'; ?>
