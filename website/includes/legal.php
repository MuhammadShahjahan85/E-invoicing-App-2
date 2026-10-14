<?php
/**
 * Shared layout for legal pages. Call legal_page($slug, $title, $description, $bodyHtml).
 */
function legal_page(string $slug, string $title, string $description, string $body, string $updated = '2026-10-14'): void
{
    global $page;
    $crumbs = [['Home', ''], [$title, $slug]];
    $page = [
        'title' => $title,
        'description' => $description,
        'path' => $slug,
        'schema' => [breadcrumb_schema($crumbs)],
    ];
    require VP_ROOT . '/includes/header.php';
    echo page_hero([
        'crumbs' => $crumbs,
        'eyebrow' => 'Legal',
        'title' => e($title),
        'lede' => e($description),
    ]);
    echo '<section class="section"><div class="container"><div class="legal prose">'
        . '<p class="legal-meta"><span>Last updated: ' . e(format_date($updated)) . '</span><span>' . e(legal_name()) . '</span></p>'
        . content_html($body)
        . '</div></div></section>';
    require VP_ROOT . '/includes/footer.php';
}
