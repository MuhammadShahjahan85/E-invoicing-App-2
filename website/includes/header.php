<?php
/**
 * Page header. Before including, a page may set $page = [
 *   'title' => '', 'description' => '', 'keywords' => '', 'path' => 'services',
 *   'og_image' => 'assets/img/og-default.png', 'og_type' => 'website', 'schema' => [],
 *   'nav' => 'services', 'noindex' => false, 'body_class' => '', 'article' => [...]
 * ];
 */
if (!function_exists('vp_render_header')) {
function vp_render_header(array $page): void
{
$brand = company_name();
$rawTitle = $page['title'] ?? ($brand . ' | Audit, Tax & Advisory Firm in Pakistan');
$fullTitle = (strpos($rawTitle, $brand) === false) ? $rawTitle . ' | ' . $brand : $rawTitle;
$description = $page['description'] ?? 'Veridian Partners is a Pakistani audit, tax, corporate and advisory firm, and the developer of Veridian E-invoicing Pakistan — FBR-integrated digital invoicing software.';
$keywords = $page['keywords'] ?? 'tax consultant Pakistan, audit and advisory Pakistan, FBR e-invoicing, sales tax, income tax return, SECP company registration, internal audit, bookkeeping, virtual CFO';
$canonicalPath = $page['path'] ?? ltrim(current_path(), '/');
$canonical = abs_url($canonicalPath);
$ogImage = abs_url($page['og_image'] ?? 'assets/img/og-default.png');
$ogType = $page['og_type'] ?? 'website';
$navActive = $page['nav'] ?? '';
$gaId = trim((string) cfg('site.google_analytics_id', ''));

$graph = [
    organization_schema(),
    [
        '@type' => 'WebSite',
        '@id' => abs_url('#website'),
        'url' => abs_url(),
        'name' => $brand,
        'inLanguage' => 'en-PK',
        'publisher' => ['@id' => abs_url('#organization')],
        'potentialAction' => [
            '@type' => 'SearchAction',
            'target' => ['@type' => 'EntryPoint', 'urlTemplate' => abs_url('search') . '?q={search_term_string}'],
            'query-input' => 'required name=search_term_string',
        ],
    ],
];
foreach ($page['schema'] ?? [] as $node) {
    $graph[] = $node;
}
$services = services();
$industries = industries();
$latest = array_slice(articles(), 0, 2);
?><!doctype html>
<html lang="en-PK" class="no-js">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<title><?= e($fullTitle) ?></title>
<meta name="description" content="<?= e($description) ?>">
<meta name="keywords" content="<?= e($keywords) ?>">
<meta name="author" content="<?= e(legal_name()) ?>">
<meta name="robots" content="<?= !empty($page['noindex']) ? 'noindex, follow' : 'index, follow, max-image-preview:large, max-snippet:-1' ?>">
<link rel="canonical" href="<?= e($canonical) ?>">
<link rel="alternate" hreflang="en-PK" href="<?= e($canonical) ?>">
<link rel="alternate" hreflang="x-default" href="<?= e($canonical) ?>">
<meta name="geo.region" content="PK">
<?php if (cfg('contact.city')): ?><meta name="geo.placename" content="<?= e(cfg('contact.city')) ?>">
<?php endif; ?>
<?php if (cfg('contact.latitude') && cfg('contact.longitude')): ?><meta name="geo.position" content="<?= e(cfg('contact.latitude') . ';' . cfg('contact.longitude')) ?>">
<?php endif; ?>
<meta property="og:type" content="<?= e($ogType) ?>">
<meta property="og:site_name" content="<?= e($brand) ?>">
<meta property="og:locale" content="en_PK">
<meta property="og:title" content="<?= e($rawTitle) ?>">
<meta property="og:description" content="<?= e($description) ?>">
<meta property="og:url" content="<?= e($canonical) ?>">
<meta property="og:image" content="<?= e($ogImage) ?>">
<meta property="og:image:width" content="1200">
<meta property="og:image:height" content="630">
<meta property="og:image:alt" content="<?= e($brand) ?> — Audit, Tax, Advisory & Digital Compliance">
<?php if (!empty($page['article'])): ?>
<meta property="article:published_time" content="<?= e($page['article']['date']) ?>T09:00:00+05:00">
<meta property="article:modified_time" content="<?= e($page['article']['updated'] ?? $page['article']['date']) ?>T09:00:00+05:00">
<meta property="article:section" content="<?= e(article_categories()[$page['article']['category']]['name'] ?? '') ?>">
<?php endif; ?>
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="<?= e($rawTitle) ?>">
<meta name="twitter:description" content="<?= e($description) ?>">
<meta name="twitter:image" content="<?= e($ogImage) ?>">
<?php if ($v = cfg('site.google_site_verification')): ?><meta name="google-site-verification" content="<?= e($v) ?>">
<?php endif; ?>
<?php if ($v = cfg('site.bing_site_verification')): ?><meta name="msvalidate.01" content="<?= e($v) ?>">
<?php endif; ?>
<meta name="theme-color" content="#0A3A31">
<meta name="format-detection" content="telephone=no">
<link rel="icon" href="<?= e(url('favicon.ico')) ?>" sizes="any">
<link rel="icon" href="<?= e(asset('img/favicon.svg')) ?>" type="image/svg+xml">
<link rel="apple-touch-icon" href="<?= e(asset('img/apple-touch-icon.png')) ?>">
<link rel="manifest" href="<?= e(url('site.webmanifest')) ?>">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Fraunces:ital,opsz,wght@0,9..144,400..700;1,9..144,400..600&amp;family=Manrope:wght@400..800&amp;display=swap">
<link rel="stylesheet" href="<?= e(asset('css/style.css')) ?>">
<script>
(function(){var d=document.documentElement;d.classList.remove('no-js');d.classList.add('js');try{var t=localStorage.getItem('vp-theme');if(t==='dark'||t==='light'){d.setAttribute('data-theme',t);}}catch(e){}})();
</script>
<?= json_ld(['@context' => 'https://schema.org', '@graph' => $graph]) ?>
</head>
<body class="<?= e($page['body_class'] ?? '') ?>" data-base="<?= e(base_path()) ?>"<?= $gaId ? ' data-ga="' . e($gaId) . '"' : '' ?>>
<?= icon_sprite() ?>
<a class="skip-link" href="#main">Skip to main content</a>

<div class="announce" role="region" aria-label="Announcement">
  <div class="container announce__inner">
    <span class="announce__pill">FBR</span>
    <p>Digital invoicing is mandatory for every sales-tax registered business — penalties now reach Rs 5 million.</p>
    <a class="announce__link" href="<?= e(url('e-invoicing')) ?>">Go live with Veridian <?= icon('arrow-right') ?></a>
  </div>
</div>

<header class="site-header" data-header>
  <div class="container site-header__inner">
    <a class="brand" href="<?= e(url()) ?>" aria-label="<?= e($brand) ?> — home">
      <?= brand_mark('brand__mark', 'hdr') ?>
      <span class="brand__text"><span class="brand__name">Veridian</span><span class="brand__sub">Partners</span></span>
    </a>

    <nav class="main-nav" aria-label="Main">
      <ul class="main-nav__list">
        <?php foreach (primary_nav() as $item): $isActive = $navActive === $item['key']; ?>
        <li class="main-nav__item<?= !empty($item['mega']) ? ' has-mega' : '' ?>">
          <?php if (!empty($item['mega'])): ?>
          <a class="main-nav__link<?= $isActive ? ' is-active' : '' ?>" href="<?= e(url($item['href'])) ?>" aria-haspopup="true" aria-expanded="false" data-mega-trigger="<?= e($item['mega']) ?>"<?= $isActive ? ' aria-current="page"' : '' ?>><?= e($item['label']) ?><?= icon('chevron-down', 'main-nav__chev') ?></a>
          <?php else: ?>
          <a class="main-nav__link<?= $isActive ? ' is-active' : '' ?>" href="<?= e(url($item['href'])) ?>"<?= $isActive ? ' aria-current="page"' : '' ?>><?= e($item['label']) ?><?php if (!empty($item['badge'])): ?><span class="nav-badge"><?= e($item['badge']) ?></span><?php endif; ?></a>
          <?php endif; ?>
        </li>
        <?php endforeach; ?>
      </ul>
    </nav>

    <div class="header-actions">
      <button class="icon-btn" type="button" data-search-open aria-label="Search the site (Ctrl+K)">
        <?= icon('search') ?><kbd class="kbd-hint">Ctrl K</kbd>
      </button>
      <button class="icon-btn" type="button" data-theme-toggle aria-label="Switch between light and dark theme">
        <?= icon('moon', 'theme-icon theme-icon--moon') ?><?= icon('sun', 'theme-icon theme-icon--sun') ?>
      </button>
      <a class="btn btn--gold btn--sm header-cta" href="<?= e(url('book-consultation')) ?>">Book a consultation</a>
      <button class="icon-btn menu-toggle" type="button" data-menu-open aria-label="Open menu" aria-controls="mobile-menu" aria-expanded="false"><?= icon('menu') ?></button>
    </div>
  </div>

  <!-- Mega menus -->
  <div class="mega" id="mega-services" data-mega="services" hidden>
    <div class="container mega__inner">
      <div class="mega__main">
        <p class="mega__heading">Our services</p>
        <ul class="mega__grid">
          <?php foreach ($services as $s): ?>
          <li><a class="mega__link" href="<?= e(service_url($s)) ?>"><span class="icon-tile icon-tile--xs"><?= icon($s['icon']) ?></span><span><strong><?= e($s['title']) ?></strong><small><?= e($s['short']) ?></small></span></a></li>
          <?php endforeach; ?>
        </ul>
      </div>
      <aside class="mega__feature">
        <span class="mega__feature-kicker">Our software</span>
        <strong class="mega__feature-title">Veridian E-invoicing Pakistan</strong>
        <p>FBR-integrated invoicing, built by tax professionals. Validation, real-time reporting, QR printing and Annex-C reconciliation.</p>
        <a class="link-arrow link-arrow--light" href="<?= e(url('e-invoicing')) ?>">See the software <?= icon('arrow-right') ?></a>
        <a class="link-arrow link-arrow--light" href="<?= e(url('services')) ?>">All services <?= icon('arrow-right') ?></a>
      </aside>
    </div>
  </div>
  <div class="mega" id="mega-industries" data-mega="industries" hidden>
    <div class="container mega__inner">
      <div class="mega__main">
        <p class="mega__heading">Industries we serve</p>
        <ul class="mega__grid mega__grid--3">
          <?php foreach ($industries as $i): ?>
          <li><a class="mega__link mega__link--compact" href="<?= e(url('industries/' . $i['slug'])) ?>"><span class="icon-tile icon-tile--xs"><?= icon($i['icon']) ?></span><strong><?= e($i['name']) ?></strong></a></li>
          <?php endforeach; ?>
        </ul>
      </div>
      <aside class="mega__feature">
        <span class="mega__feature-kicker">Sector insight</span>
        <strong class="mega__feature-title">Advice shaped by how your sector is taxed and regulated.</strong>
        <p>From Tier-1 retailers' POS integration to exporters' refunds and NPOs' tax exemptions.</p>
        <a class="link-arrow link-arrow--light" href="<?= e(url('industries')) ?>">All industries <?= icon('arrow-right') ?></a>
      </aside>
    </div>
  </div>
  <div class="mega" id="mega-knowledge" data-mega="knowledge" hidden>
    <div class="container mega__inner">
      <div class="mega__main">
        <p class="mega__heading">Knowledge Bank</p>
        <ul class="mega__grid">
          <?php foreach (knowledge_links() as $k): ?>
          <li><a class="mega__link" href="<?= e(url($k['href'])) ?>"><span class="icon-tile icon-tile--xs"><?= icon($k['icon']) ?></span><span><strong><?= e($k['label']) ?></strong><small><?= e($k['text']) ?></small></span></a></li>
          <?php endforeach; ?>
        </ul>
      </div>
      <aside class="mega__feature">
        <span class="mega__feature-kicker">Latest insights</span>
        <?php foreach ($latest as $a): ?>
        <a class="mega__article" href="<?= e(article_url($a)) ?>"><strong><?= e($a['title']) ?></strong><small><?= e(format_date($a['date'])) ?></small></a>
        <?php endforeach; ?>
        <a class="link-arrow link-arrow--light" href="<?= e(url('knowledge-bank')) ?>">All insights <?= icon('arrow-right') ?></a>
      </aside>
    </div>
  </div>
</header>

<!-- Mobile menu -->
<div class="drawer" id="mobile-menu" data-drawer hidden>
  <div class="drawer__backdrop" data-menu-close></div>
  <div class="drawer__panel" role="dialog" aria-modal="true" aria-label="Menu">
    <div class="drawer__head">
      <a class="brand" href="<?= e(url()) ?>"><?= brand_mark('brand__mark', 'drw') ?><span class="brand__text"><span class="brand__name">Veridian</span><span class="brand__sub">Partners</span></span></a>
      <button class="icon-btn" type="button" data-menu-close aria-label="Close menu"><?= icon('x') ?></button>
    </div>
    <button class="drawer__search" type="button" data-search-open><?= icon('search') ?><span>Search services, guides, terms…</span></button>
    <nav class="drawer__nav" aria-label="Mobile">
      <details class="drawer__group">
        <summary>Services<?= icon('chevron-down') ?></summary>
        <ul>
          <?php foreach ($services as $s): ?><li><a href="<?= e(service_url($s)) ?>"><?= icon($s['icon']) ?><?= e($s['title']) ?></a></li><?php endforeach; ?>
          <li><a class="drawer__all" href="<?= e(url('services')) ?>">All services <?= icon('arrow-right') ?></a></li>
        </ul>
      </details>
      <details class="drawer__group">
        <summary>Industries<?= icon('chevron-down') ?></summary>
        <ul>
          <?php foreach ($industries as $i): ?><li><a href="<?= e(url('industries/' . $i['slug'])) ?>"><?= icon($i['icon']) ?><?= e($i['name']) ?></a></li><?php endforeach; ?>
          <li><a class="drawer__all" href="<?= e(url('industries')) ?>">All industries <?= icon('arrow-right') ?></a></li>
        </ul>
      </details>
      <details class="drawer__group">
        <summary>Knowledge Bank<?= icon('chevron-down') ?></summary>
        <ul>
          <?php foreach (knowledge_links() as $k): ?><li><a href="<?= e(url($k['href'])) ?>"><?= icon($k['icon']) ?><?= e($k['label']) ?></a></li><?php endforeach; ?>
        </ul>
      </details>
      <a class="drawer__link" href="<?= e(url('e-invoicing')) ?>">E-Invoicing software <span class="nav-badge">FBR</span></a>
      <a class="drawer__link" href="<?= e(url('about')) ?>">About us</a>
      <a class="drawer__link" href="<?= e(url('careers')) ?>">Careers</a>
      <a class="drawer__link" href="<?= e(url('contact')) ?>">Contact</a>
    </nav>
    <div class="drawer__foot">
      <a class="btn btn--gold btn--block" href="<?= e(url('book-consultation')) ?>"><?= icon('calendar') ?><span>Book a free consultation</span></a>
      <?php if ($wa = whatsapp_link()): ?><a class="btn btn--ghost btn--block" href="<?= e($wa) ?>" target="_blank" rel="noopener"><?= icon('whatsapp') ?><span>WhatsApp us</span></a><?php endif; ?>
      <a class="drawer__contact" href="mailto:<?= e(contact_email()) ?>"><?= icon('mail') ?><?= e(contact_email()) ?></a>
    </div>
  </div>
</div>

<!-- Search -->
<div class="search-modal" data-search hidden>
  <div class="search-modal__backdrop" data-search-close></div>
  <div class="search-modal__panel" role="dialog" aria-modal="true" aria-label="Search">
    <form class="search-modal__form" action="<?= e(url('search')) ?>" method="get" role="search">
      <?= icon('search') ?>
      <input type="search" name="q" placeholder="Search services, guides, tax terms…" autocomplete="off" aria-label="Search" data-search-input>
      <button class="search-modal__esc" type="button" data-search-close aria-label="Close search">Esc</button>
    </form>
    <div class="search-modal__results" data-search-results role="listbox" aria-label="Results">
      <div class="search-modal__suggest">
        <p class="search-modal__label">Popular</p>
        <a href="<?= e(url('e-invoicing')) ?>"><?= icon('qr-code') ?>FBR e-invoicing software</a>
        <a href="<?= e(url('tools#salary-tax')) ?>"><?= icon('calculator') ?>Salary tax calculator 2026-27</a>
        <a href="<?= e(url('tax-calendar')) ?>"><?= icon('calendar') ?>Tax & compliance calendar</a>
        <a href="<?= e(url('knowledge-bank/how-to-register-company-secp')) ?>"><?= icon('building') ?>How to register a company with SECP</a>
      </div>
    </div>
    <p class="search-modal__foot"><span><kbd>↑</kbd><kbd>↓</kbd> to move</span><span><kbd>Enter</kbd> to open</span><span><kbd>Esc</kbd> to close</span></p>
  </div>
</div>

<main id="main" tabindex="-1">
<?php
}
}
vp_render_header($page ?? []);
