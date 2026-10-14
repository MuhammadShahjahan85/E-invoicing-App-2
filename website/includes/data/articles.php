<?php
/**
 * Knowledge Bank articles. Each file in data/articles/ returns
 * [slug => article]. To add an article, copy an existing entry and change it.
 *
 * Fields: slug, title, seo_title, description, category (see article_categories()),
 * tags, date (YYYY-MM-DD), updated, featured (bool), excerpt, body (HTML),
 * faqs (optional), services (related service slugs).
 */
$all = [];
foreach (glob(__DIR__ . '/articles/*.php') as $file) {
    $all += require $file;
}
return $all;
