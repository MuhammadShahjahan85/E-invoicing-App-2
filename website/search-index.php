<?php
/** JSON index for the instant search box (served as /search-index.json). */
require __DIR__ . '/includes/bootstrap.php';
require __DIR__ . '/includes/search.php';

header('Content-Type: application/json; charset=utf-8');
header('Cache-Control: public, max-age=3600');
header('X-Robots-Tag: noindex');
echo json_encode(search_index(), JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE);
