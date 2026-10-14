<?php
/** Human-readable sitemap. */
require __DIR__ . '/includes/bootstrap.php';

$crumbs = [['Home', ''], ['Sitemap', 'sitemap']];
$page = [
    'title' => 'Sitemap',
    'description' => 'Every page on the Veridian Partners website.',
    'path' => 'sitemap',
    'schema' => [breadcrumb_schema($crumbs)],
];
require __DIR__ . '/includes/header.php';
echo page_hero(['crumbs' => $crumbs, 'eyebrow' => 'Sitemap', 'title' => 'Every page, <em>at a glance</em>.']);
?>
<section class="section">
  <div class="container sitemap-cols">
    <div>
      <h2>Main pages</h2>
      <ul>
        <?php foreach ([['Home', ''], ['About us', 'about'], ['Services', 'services'], ['Industries', 'industries'], ['FBR E-Invoicing Software', 'e-invoicing'], ['Knowledge Bank', 'knowledge-bank'], ['Careers', 'careers'], ['Contact', 'contact'], ['Book a consultation', 'book-consultation']] as [$n, $p]): ?>
        <li><a href="<?= e(url($p)) ?>"><?= e($n) ?></a></li>
        <?php endforeach; ?>
      </ul>
      <h2 class="mt-2">Resources</h2>
      <ul>
        <?php foreach (knowledge_links() as $k): ?><li><a href="<?= e(url($k['href'])) ?>"><?= e($k['label']) ?></a></li><?php endforeach; ?>
      </ul>
    </div>
    <div>
      <h2>Services</h2>
      <ul><?php foreach (services() as $s): ?><li><a href="<?= e(service_url($s)) ?>"><?= e($s['title']) ?></a></li><?php endforeach; ?></ul>
      <h2 class="mt-2">Industries</h2>
      <ul><?php foreach (industries() as $i): ?><li><a href="<?= e(url('industries/' . $i['slug'])) ?>"><?= e($i['name']) ?></a></li><?php endforeach; ?></ul>
    </div>
    <div>
      <h2>Guides</h2>
      <ul><?php foreach (articles() as $a): ?><li><a href="<?= e(article_url($a)) ?>"><?= e($a['title']) ?></a></li><?php endforeach; ?></ul>
      <h2 class="mt-2">Legal</h2>
      <ul>
        <?php foreach ([['Privacy policy', 'privacy-policy'], ['Terms of use', 'terms'], ['Cookie policy', 'cookie-policy'], ['Disclaimer', 'disclaimer'], ['Accessibility', 'accessibility']] as [$n, $p]): ?>
        <li><a href="<?= e(url($p)) ?>"><?= e($n) ?></a></li>
        <?php endforeach; ?>
      </ul>
    </div>
  </div>
</section>
<?php require __DIR__ . '/includes/footer.php'; ?>
