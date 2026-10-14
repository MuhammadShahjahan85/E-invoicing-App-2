<?php
/** robots.txt (served as /robots.txt). */
require __DIR__ . '/includes/bootstrap.php';

header('Content-Type: text/plain; charset=utf-8');
echo "User-agent: *\n";
echo "Allow: /\n";
echo "Disallow: " . url('admin/') . "\n";
echo "Disallow: " . url('search') . "\n";
echo "Disallow: " . url('submit') . "\n";
echo "Disallow: " . url('thank-you') . "\n";
echo "\nSitemap: " . abs_url('sitemap.xml') . "\n";
