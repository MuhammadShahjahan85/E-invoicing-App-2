<?php
require __DIR__ . '/includes/bootstrap.php';

$crumbs = [['Home', ''], ['Knowledge Bank', 'knowledge-bank'], ['Glossary', 'glossary']];
$terms = glossary();
uksort($terms, 'strcasecmp');
$groups = [];
foreach ($terms as $term => $def) {
    $letter = strtoupper($term[0]);
    $groups[$letter][$term] = $def;
}
$page = [
    'title' => 'Pakistan Tax & Business Glossary — NTN, STRN, ATL, Annex-C and More',
    'description' => 'Plain-language definitions of Pakistani tax, corporate and finance terms: NTN, STRN, ATL, IRIS, PRAL, Annex-A, Annex-C, SRO, further tax, withholding tax, SECP, IFRS and more.',
    'keywords' => 'tax glossary Pakistan, what is NTN, what is STRN, what is ATL, Annex-C meaning, SRO meaning, further tax meaning, withholding tax Pakistan, IRIS FBR',
    'path' => 'glossary',
    'nav' => 'knowledge',
    'schema' => [
        breadcrumb_schema($crumbs),
        [
            '@type' => 'DefinedTermSet',
            'name' => 'Pakistan tax and business glossary',
            'url' => abs_url('glossary'),
            'hasDefinedTerm' => array_map(fn($t, $d) => ['@type' => 'DefinedTerm', 'name' => $t, 'description' => strip_tags($d)], array_keys($terms), array_values($terms)),
        ],
    ],
];
require __DIR__ . '/includes/header.php';

echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => 'Glossary',
    'title' => 'Tax and business terms, <em>explained simply</em>.',
    'lede' => count($terms) . ' terms you will meet in Pakistani tax, corporate and finance — from ATL to zero-rating.',
]);
?>

<section class="section">
  <div class="container container--narrow">
    <label class="search-field" style="max-width:none;margin-bottom:1rem"><span class="sr-only">Search the glossary</span><?= icon('search') ?><input type="search" placeholder="Search terms, e.g. Annex-C, ATL, SRO…" data-glossary-search></label>
    <nav class="alpha" aria-label="Jump to letter">
      <?php foreach (range('A', 'Z') as $L): ?>
        <?php if (isset($groups[$L])): ?><a href="#letter-<?= $L ?>"><?= $L ?></a><?php else: ?><span><?= $L ?></span><?php endif; ?>
      <?php endforeach; ?>
    </nav>
    <?php foreach ($groups as $L => $list): ?>
    <div class="glossary-group" id="letter-<?= e($L) ?>">
      <div class="glossary-group__letter" aria-hidden="true"><?= e($L) ?></div>
      <dl>
        <?php foreach ($list as $term => $def): ?>
        <div data-term id="<?= e(trim(preg_replace('/[^a-z0-9]+/', '-', strtolower($term)), '-')) ?>"><dt><?= e($term) ?></dt><dd><?= content_html($def) ?></dd></div>
        <?php endforeach; ?>
      </dl>
    </div>
    <?php endforeach; ?>
    <div class="empty-state" data-glossary-empty hidden><?= icon('search') ?><p>No matching term. <a href="<?= e(url('contact')) ?>">Ask us</a> and we will add it.</p></div>
  </div>
</section>

<?= cta_band() ?>
<?php require __DIR__ . '/includes/footer.php'; ?>
