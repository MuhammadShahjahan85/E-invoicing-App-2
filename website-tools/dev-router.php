<?php
/**
 * Local preview only (not uploaded): mimics website/.htaccess for PHP's built-in server.
 *   php -S 127.0.0.1:8080 -t website website-tools/dev-router.php
 */
$root = $_SERVER['DOCUMENT_ROOT'];
$path = rawurldecode(parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH));

if (preg_match('#^/(includes|storage)(/|$)#', $path) || $path === '/config.php') {
    http_response_code(403);
    echo 'Forbidden';
    return true;
}
$map = ['/sitemap.xml' => 'sitemap-xml.php', '/robots.txt' => 'robots.php', '/search-index.json' => 'search-index.php'];
if (isset($map[$path])) {
    $_SERVER['SCRIPT_NAME'] = '/' . $map[$path];
    require $root . '/' . $map[$path];
    return true;
}
if ($path !== '/' && is_file($root . $path)) {
    return false; // static file
}
$routes = [
    '#^/services/([a-z0-9-]+)$#' => 'service.php',
    '#^/industries/([a-z0-9-]+)$#' => 'industry.php',
    '#^/knowledge-bank/([a-z0-9-]+)$#' => 'article.php',
];
foreach ($routes as $re => $file) {
    if (preg_match($re, $path, $m)) {
        $_GET['slug'] = $m[1];
        require $root . '/' . $file;
        return true;
    }
}
if ($path === '/') {
    require $root . '/index.php';
    return true;
}
if (is_dir($root . $path) && is_file($root . rtrim($path, '/') . '/index.php')) {
    chdir($root . rtrim($path, '/'));
    require $root . rtrim($path, '/') . '/index.php';
    return true;
}
$file = $root . rtrim($path, '/') . '.php';
if (is_file($file)) {
    require $file;
    return true;
}
require $root . '/404.php';
return true;
