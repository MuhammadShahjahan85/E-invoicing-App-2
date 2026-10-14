<?php
/** XML sitemap for search engines (served as /sitemap.xml). */
require __DIR__ . '/includes/bootstrap.php';

$today = date('Y-m-d');
$urls = [
    ['', '1.0', 'weekly'], ['services', '0.9', 'monthly'], ['e-invoicing', '0.9', 'monthly'],
    ['industries', '0.8', 'monthly'], ['knowledge-bank', '0.8', 'weekly'], ['about', '0.7', 'monthly'],
    ['contact', '0.7', 'yearly'], ['book-consultation', '0.7', 'yearly'], ['careers', '0.6', 'monthly'],
    ['tools', '0.8', 'monthly'], ['tax-calendar', '0.8', 'weekly'], ['glossary', '0.6', 'monthly'],
    ['official-portals', '0.5', 'yearly'], ['faq', '0.6', 'monthly'], ['sitemap', '0.3', 'monthly'],
    ['privacy-policy', '0.2', 'yearly'], ['terms', '0.2', 'yearly'], ['cookie-policy', '0.2', 'yearly'],
    ['disclaimer', '0.2', 'yearly'], ['accessibility', '0.2', 'yearly'],
];
$entries = [];
foreach ($urls as [$p, $prio, $freq]) {
    $entries[] = [abs_url($p), $today, $prio, $freq];
}
foreach (services() as $s) {
    if (empty($s['url'])) {
        $entries[] = [abs_url('services/' . $s['slug']), $today, '0.8', 'monthly'];
    }
}
foreach (industries() as $i) {
    $entries[] = [abs_url('industries/' . $i['slug']), $today, '0.7', 'monthly'];
}
foreach (articles() as $a) {
    $entries[] = [abs_url('knowledge-bank/' . $a['slug']), $a['updated'] ?? $a['date'], '0.7', 'monthly'];
}

header('Content-Type: application/xml; charset=utf-8');
echo '<?xml version="1.0" encoding="UTF-8"?>' . "\n";
echo '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">' . "\n";
foreach ($entries as [$loc, $mod, $prio, $freq]) {
    echo '  <url><loc>' . e($loc) . '</loc><lastmod>' . e($mod) . '</lastmod><changefreq>' . $freq . '</changefreq><priority>' . $prio . "</priority></url>\n";
}
echo '</urlset>' . "\n";
