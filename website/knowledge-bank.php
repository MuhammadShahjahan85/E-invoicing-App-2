<?php
require __DIR__ . '/includes/bootstrap.php';

$crumbs = [['Home', ''], ['Knowledge Bank', 'knowledge-bank']];
$all = array_values(articles());
$cats = article_categories();
$counts = array_count_values(array_column($all, 'category'));
$featured = array_values(array_filter($all, fn($a) => !empty($a['featured'])));
$lead = $featured[0] ?? $all[0];

$page = [
    'title' => 'Knowledge Bank — Pakistan Tax, SECP, Audit & E-Invoicing Guides',
    'description' => 'Practical guides on Pakistani income tax, sales tax, FBR digital invoicing, SECP company law, internal controls and financial reporting — plus a tax calendar, calculators and glossary.',
    'keywords' => 'Pakistan tax guide, FBR guide, sales tax guide Pakistan, SECP guide, income tax return guide, e-invoicing guide, tax calendar Pakistan, tax glossary',
    'path' => 'knowledge-bank',
    'nav' => 'knowledge',
    'schema' => [
        breadcrumb_schema($crumbs),
        [
            '@type' => 'CollectionPage',
            'name' => 'Knowledge Bank',
            'url' => abs_url('knowledge-bank'),
            'hasPart' => array_map(fn($a) => ['@type' => 'Article', 'headline' => $a['title'], 'url' => absolute(article_url($a))], $all),
        ],
    ],
];
require __DIR__ . '/includes/header.php';

echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => 'Knowledge Bank',
    'title' => 'Clear answers to Pakistan\'s <em>trickiest</em> tax and compliance questions.',
    'lede' => 'Practical guides written by practitioners — plus a live tax calendar, free calculators, a glossary and links to every official portal.',
]);
?>

<section class="section section--tight">
  <div class="container">
    <div class="grid grid--3">
      <?php foreach (knowledge_links() as $i => $k): if ($k['href'] === 'knowledge-bank') continue; ?>
      <a class="card card--industry reveal" href="<?= e(url($k['href'])) ?>"><span class="icon-tile icon-tile--sm icon-tile--gold"><?= icon($k['icon']) ?></span><span class="card--industry__body"><span class="card__title"><?= e($k['label']) ?></span><span class="card__text"><?= e($k['text']) ?></span></span><?= icon('arrow-up-right', 'card--industry__arrow') ?></a>
      <?php endforeach; ?>
      <a class="card card--industry reveal" href="#newsletter"><span class="icon-tile icon-tile--sm icon-tile--gold"><?= icon('bell') ?></span><span class="card--industry__body"><span class="card__title">The Veridian Brief</span><span class="card__text">Monthly updates on tax changes and deadlines.</span></span><?= icon('arrow-up-right', 'card--industry__arrow') ?></a>
    </div>
  </div>
</section>

<section class="section section--tint" aria-labelledby="articles-title">
  <div class="container" data-filter-list="#article-list" data-filter-empty="#article-empty">
    <div class="section-head section-head--split" style="margin-bottom:1.5rem">
      <div>
        <p class="eyebrow">Insights & guides</p>
        <h2 id="articles-title">Browse by <em>topic</em></h2>
      </div>
      <p class="muted mb-0" data-filter-count><?= count($all) ?> articles</p>
    </div>
    <div class="toolbar">
      <div class="filter-chips" role="group" aria-label="Filter by topic">
        <button class="filter-chip" type="button" data-filter="all" aria-pressed="true">All topics</button>
        <?php foreach ($cats as $key => $c): if (empty($counts[$key])) continue; ?>
        <button class="filter-chip" type="button" data-filter="<?= e($key) ?>" aria-pressed="false"><?= icon($c['icon']) ?><?= e($c['name']) ?></button>
        <?php endforeach; ?>
      </div>
      <label class="search-field"><span class="sr-only">Search articles</span><?= icon('search') ?><input type="search" placeholder="Search articles…" data-filter-search></label>
    </div>
    <div class="grid grid--3" id="article-list">
      <?php foreach ($all as $a): echo article_card($a, $a['slug'] === $lead['slug'] ? 'is-featured' : ''); endforeach; ?>
    </div>
    <div class="empty-state" id="article-empty" hidden><?= icon('search') ?><p>No articles match your search. <a href="<?= e(url('contact')) ?>">Ask us your question</a> — we may write the next guide about it.</p></div>
  </div>
</section>

<section class="section" id="newsletter" aria-labelledby="nl-title">
  <div class="container container--narrow center">
    <p class="eyebrow">The Veridian Brief</p>
    <h2 id="nl-title">Tax changes, explained — <em>once a month</em>.</h2>
    <p class="lead">New SROs, Finance Act changes, FBR notices and upcoming deadlines, in plain language. Free, and you can unsubscribe any time.</p>
    <div style="max-width:560px;margin:2rem auto 0;text-align:left"><?= newsletter_form('newsletter--light') ?></div>
  </div>
</section>

<?= cta_band('Have a question our guides don\'t answer?', 'Ask us directly. A senior adviser will reply within one business day — the first consultation is free.') ?>
<?php require __DIR__ . '/includes/footer.php'; ?>
