<?php
require __DIR__ . '/includes/bootstrap.php';

$slug = preg_replace('/[^a-z0-9-]/', '', (string) ($_GET['slug'] ?? ''));
$a = article($slug);
if (!$a) {
    not_found();
}
$cats = article_categories();
$cat = $cats[$a['category']] ?? ['name' => 'Insights', 'icon' => 'book-open'];
$crumbs = [['Home', ''], ['Knowledge Bank', 'knowledge-bank'], [$a['title'], 'knowledge-bank/' . $a['slug']]];
$minutes = reading_minutes($a['body']);
$absUrl = abs_url('knowledge-bank/' . $a['slug']);

$schema = [
    breadcrumb_schema($crumbs),
    [
        '@type' => 'Article',
        'headline' => $a['title'],
        'description' => $a['description'],
        'datePublished' => $a['date'] . 'T09:00:00+05:00',
        'dateModified' => ($a['updated'] ?? $a['date']) . 'T09:00:00+05:00',
        'author' => ['@type' => 'Organization', 'name' => company_name(), 'url' => abs_url()],
        'publisher' => ['@id' => abs_url('#organization')],
        'mainEntityOfPage' => $absUrl,
        'image' => abs_url('assets/img/og-default.png'),
        'articleSection' => $cat['name'],
        'keywords' => implode(', ', $a['tags'] ?? []),
        'inLanguage' => 'en-PK',
        'wordCount' => str_word_count(strip_tags($a['body'])),
    ],
];
if (!empty($a['faqs'])) {
    $schema[] = faq_schema($a['faqs']);
}
$page = [
    'title' => $a['seo_title'] ?? $a['title'],
    'description' => $a['description'],
    'keywords' => implode(', ', $a['tags'] ?? []),
    'path' => 'knowledge-bank/' . $a['slug'],
    'nav' => 'knowledge',
    'og_type' => 'article',
    'article' => $a,
    'schema' => $schema,
];
require __DIR__ . '/includes/header.php';

$meta = '<div class="page-hero__meta"><span>' . icon($cat['icon']) . e($cat['name']) . '</span><span>' . icon('calendar') . 'Updated ' . e(format_date($a['updated'] ?? $a['date'])) . '</span><span>' . icon('clock') . $minutes . ' min read</span></div>';
echo '<div class="progress-bar" data-progress aria-hidden="true"></div>';
echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => $cat['name'],
    'title' => e($a['title']),
    'lede' => e($a['excerpt']),
    'after' => $meta,
]);

// Related: same category first, then the rest.
$related = array_filter(articles(), fn($o) => $o['slug'] !== $a['slug'] && $o['category'] === $a['category']);
if (count($related) < 3) {
    $related += array_filter(articles(), fn($o) => $o['slug'] !== $a['slug']);
}
$related = array_slice($related, 0, 3);
?>

<section class="section">
  <div class="container article-layout">
    <article>
      <details class="toc toc-mobile" data-toc-wrap>
        <summary class="toc__title" style="cursor:pointer;margin:0">On this page</summary>
        <nav data-toc aria-label="On this page" style="margin-top:.8rem"></nav>
      </details>

      <div class="prose lede-first" data-prose>
        <?= content_html($a['body']) ?>
      </div>

      <?php if (!empty($a['faqs'])): ?>
      <div class="mt-3">
        <h2 style="font-size:clamp(1.6rem,2.6vw,2.05rem)">Frequently asked questions</h2>
        <?= faq_list($a['faqs']) ?>
      </div>
      <?php endif; ?>

      <div class="article-foot">
        <?= share_buttons($a['title'], $absUrl) ?>
        <div class="author">
          <?= brand_mark('author__mark', 'auth') ?>
          <div><strong>Veridian Partners — Tax & Advisory Team</strong><span>Practitioners in Pakistani tax, corporate law, audit and digital compliance. Questions about this guide? <a href="<?= e(url('contact')) ?>">Contact us</a>.</span></div>
        </div>
        <p class="disclaimer-note"><?= icon('info') ?> This guide is general information based on the law as we understand it on the date shown, not advice for your situation. Tax rules change often — please <a href="<?= e(url('contact')) ?>">speak to us</a> before acting on it. See our <a href="<?= e(url('disclaimer')) ?>">disclaimer</a>.</p>
        <?php if (!empty($a['tags'])): ?>
        <div class="cal-item__tags"><?php foreach ($a['tags'] as $t): ?><a class="tag" href="<?= e(url('knowledge-bank?q=' . rawurlencode($t))) ?>">#<?= e($t) ?></a><?php endforeach; ?></div>
        <?php endif; ?>
      </div>
    </article>

    <aside class="article-aside">
      <div class="toc" data-toc-wrap>
        <p class="toc__title">On this page</p>
        <nav data-toc aria-label="Table of contents"></nav>
      </div>
      <?= expert_card($a['services'][0] ?? '') ?>
      <?php $svcs = array_filter(array_map('service', $a['services'] ?? [])); if ($svcs): ?>
      <div class="side-card">
        <p class="side-card__heading">Related services</p>
        <ul class="side-links">
          <?php foreach ($svcs as $s): ?><li><a href="<?= e(service_url($s)) ?>"><?= e($s['title']) ?><?= icon('chevron-right') ?></a></li><?php endforeach; ?>
        </ul>
      </div>
      <?php endif; ?>
    </aside>
  </div>
</section>

<section class="section section--tint" aria-labelledby="related-title">
  <div class="container">
    <div class="section-head section-head--split">
      <div><p class="eyebrow">Keep reading</p><h2 id="related-title">Related <em>guides</em></h2></div>
      <a class="btn btn--ghost" href="<?= e(url('knowledge-bank')) ?>"><span>All insights</span><?= icon('arrow-right') ?></a>
    </div>
    <div class="grid grid--3">
      <?php foreach ($related as $r): echo article_card($r); endforeach; ?>
    </div>
  </div>
</section>

<?= cta_band() ?>
<?php require __DIR__ . '/includes/footer.php'; ?>
