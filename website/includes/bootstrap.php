<?php
/**
 * Veridian Partners website — shared setup and helpers.
 * Every page starts with: require __DIR__ . '/includes/bootstrap.php';
 */
declare(strict_types=1);

define('VP_ROOT', dirname(__DIR__));
define('VP_VERSION', '1.0.0');

date_default_timezone_set('Asia/Karachi');
mb_internal_encoding('UTF-8');

$GLOBALS['VP_CONFIG'] = require VP_ROOT . '/config.php';

if (cfg('site.debug')) {
    error_reporting(E_ALL);
    ini_set('display_errors', '1');
} else {
    error_reporting(E_ALL & ~E_DEPRECATED & ~E_NOTICE);
    ini_set('display_errors', '0');
}

require __DIR__ . '/icons.php';
require __DIR__ . '/partials.php';
require __DIR__ . '/deadlines.php';

/* --------------------------------------------------------------------------
 * Configuration
 * ----------------------------------------------------------------------- */

/** Read a setting with a dotted path, e.g. cfg('contact.phone'). */
function cfg(string $path, $default = null)
{
    $value = $GLOBALS['VP_CONFIG'];
    foreach (explode('.', $path) as $key) {
        if (!is_array($value) || !array_key_exists($key, $value)) {
            return $default;
        }
        $value = $value[$key];
    }
    return $value;
}

/** HTML-escape. */
function e($value): string
{
    return htmlspecialchars((string) $value, ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
}

/* --------------------------------------------------------------------------
 * Addresses
 * ----------------------------------------------------------------------- */

/** Folder the site lives in, relative to the web root ('' at the domain root). */
function base_path(): string
{
    static $base = null;
    if ($base !== null) {
        return $base;
    }
    $base = '';
    $doc = realpath((string) ($_SERVER['DOCUMENT_ROOT'] ?? '')) ?: '';
    $root = realpath(VP_ROOT) ?: VP_ROOT;
    if ($doc !== '' && strpos($root, $doc) === 0) {
        $base = rtrim(str_replace('\\', '/', substr($root, strlen($doc))), '/');
    }
    return $base;
}

function request_host(): string
{
    $host = strtolower((string) ($_SERVER['HTTP_HOST'] ?? $_SERVER['SERVER_NAME'] ?? 'localhost'));
    return preg_replace('/[^a-z0-9.\-:\[\]]/', '', $host) ?: 'localhost';
}

function is_https(): bool
{
    return (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off')
        || (($_SERVER['SERVER_PORT'] ?? '') == 443)
        || (strtolower((string) ($_SERVER['HTTP_X_FORWARDED_PROTO'] ?? '')) === 'https');
}

/** Scheme and host, e.g. https://www.example.com (no trailing slash). */
function site_origin(): string
{
    $configured = rtrim((string) cfg('site.url', ''), '/');
    if ($configured !== '') {
        $parts = parse_url($configured);
        if (!empty($parts['scheme']) && !empty($parts['host'])) {
            return $parts['scheme'] . '://' . $parts['host'] . (isset($parts['port']) ? ':' . $parts['port'] : '');
        }
    }
    return (is_https() ? 'https' : 'http') . '://' . request_host();
}

/** The bare domain, e.g. example.com (used for default e-mail addresses). */
function site_domain(): string
{
    $host = parse_url(site_origin(), PHP_URL_HOST) ?: 'localhost';
    $host = preg_replace('/^www\./', '', $host);
    if ($host === 'localhost' || filter_var($host, FILTER_VALIDATE_IP)) {
        return 'example.com';
    }
    return $host;
}

/** Site-relative link, e.g. url('services/tax-advisory'). */
function url(string $path = ''): string
{
    $path = ltrim($path, '/');
    return base_path() . '/' . $path;
}

/** Absolute link, e.g. https://example.com/services. */
function abs_url(string $path = ''): string
{
    return site_origin() . url($path);
}

/** Absolute form of a link that url() already produced, e.g. absolute(service_url($s)). */
function absolute(string $href): string
{
    return site_origin() . $href;
}

/** Link to a file in /assets with a cache-busting version. */
function asset(string $path): string
{
    $file = VP_ROOT . '/assets/' . ltrim($path, '/');
    $v = is_file($file) ? (string) filemtime($file) : VP_VERSION;
    return url('assets/' . ltrim($path, '/')) . '?v=' . $v;
}

/** Path of the current request, without the base folder or query, e.g. '/services'. */
function current_path(): string
{
    $path = parse_url((string) ($_SERVER['REQUEST_URI'] ?? '/'), PHP_URL_PATH) ?: '/';
    $base = base_path();
    if ($base !== '' && strpos($path, $base) === 0) {
        $path = substr($path, strlen($base));
    }
    return '/' . ltrim($path, '/');
}

/* --------------------------------------------------------------------------
 * Company details
 * ----------------------------------------------------------------------- */

function company_name(): string
{
    return (string) cfg('company.name', 'Veridian Partners');
}

function legal_name(): string
{
    return (string) cfg('company.legal_name', 'Veridian Partners Consultancy Private Limited');
}

function contact_email(): string
{
    $email = trim((string) cfg('contact.email', ''));
    return $email !== '' ? $email : 'info@' . site_domain();
}

function tel_link(string $number): string
{
    return 'tel:' . preg_replace('/[^0-9+]/', '', $number);
}

function whatsapp_number(): string
{
    return preg_replace('/\D/', '', (string) cfg('contact.whatsapp', ''));
}

function whatsapp_link(string $message = ''): string
{
    $n = whatsapp_number();
    if ($n === '') {
        return '';
    }
    if ($message === '') {
        $message = 'Hello ' . company_name() . ', I would like to discuss ';
    }
    return 'https://wa.me/' . $n . '?text=' . rawurlencode($message);
}

/** Office address as lines (empty array when no address is set). */
function address_lines(): array
{
    $lines = [];
    foreach (['address_line_1', 'address_line_2'] as $k) {
        $v = trim((string) cfg('contact.' . $k, ''));
        if ($v !== '') {
            $lines[] = $v;
        }
    }
    $city = trim(implode(' ', array_filter([cfg('contact.city', ''), cfg('contact.postal_code', '')])));
    $region = implode(', ', array_filter([$city, cfg('contact.province', ''), cfg('contact.country', '')]));
    if ($lines || cfg('contact.city')) {
        $lines[] = $region;
    }
    return $lines;
}

function social_links(): array
{
    $labels = ['linkedin' => 'LinkedIn', 'facebook' => 'Facebook', 'instagram' => 'Instagram', 'x' => 'X (Twitter)', 'youtube' => 'YouTube'];
    $out = [];
    foreach ($labels as $key => $label) {
        $href = trim((string) cfg('social.' . $key, ''));
        if ($href !== '') {
            $out[] = ['key' => $key, 'label' => $label, 'href' => $href];
        }
    }
    return $out;
}

/* --------------------------------------------------------------------------
 * Content
 * ----------------------------------------------------------------------- */

function vp_data(string $name): array
{
    static $cache = [];
    if (!isset($cache[$name])) {
        $cache[$name] = require VP_ROOT . '/includes/data/' . $name . '.php';
    }
    return $cache[$name];
}

function services(): array
{
    return vp_data('services');
}

function service(string $slug): ?array
{
    return services()[$slug] ?? null;
}

function service_url(array $s): string
{
    return url($s['url'] ?? ('services/' . $s['slug']));
}

function industries(): array
{
    return vp_data('industries');
}

function industry(string $slug): ?array
{
    return industries()[$slug] ?? null;
}

function article_categories(): array
{
    return [
        'tax'        => ['name' => 'Income Tax', 'icon' => 'landmark'],
        'sales-tax'  => ['name' => 'Sales Tax & E-invoicing', 'icon' => 'receipt'],
        'corporate'  => ['name' => 'Corporate & SECP', 'icon' => 'building'],
        'assurance'  => ['name' => 'Audit & Controls', 'icon' => 'shield-check'],
        'reporting'  => ['name' => 'Financial Reporting', 'icon' => 'chart-bar'],
        'business'   => ['name' => 'Business & Investment', 'icon' => 'trending-up'],
    ];
}

/** All articles, newest first. */
function articles(): array
{
    static $sorted = null;
    if ($sorted === null) {
        $sorted = vp_data('articles');
        uasort($sorted, fn($a, $b) => strcmp($b['date'], $a['date']));
    }
    return $sorted;
}

function article(string $slug): ?array
{
    return articles()[$slug] ?? null;
}

function article_url(array $a): string
{
    return url('knowledge-bank/' . $a['slug']);
}

function reading_minutes(string $html): int
{
    $words = str_word_count(strip_tags($html));
    return max(2, (int) ceil($words / 200));
}

function faqs(): array
{
    return vp_data('faqs');
}

function glossary(): array
{
    return vp_data('glossary');
}

function format_date(string $ymd): string
{
    $t = strtotime($ymd);
    return $t ? date('j F Y', $t) : $ymd;
}

/* --------------------------------------------------------------------------
 * Navigation
 * ----------------------------------------------------------------------- */

function primary_nav(): array
{
    return [
        ['key' => 'services',   'label' => 'Services',       'href' => 'services',       'mega' => 'services'],
        ['key' => 'industries', 'label' => 'Industries',     'href' => 'industries',     'mega' => 'industries'],
        ['key' => 'einvoicing', 'label' => 'E-Invoicing',    'href' => 'e-invoicing',    'badge' => 'FBR'],
        ['key' => 'knowledge',  'label' => 'Knowledge Bank', 'href' => 'knowledge-bank', 'mega' => 'knowledge'],
        ['key' => 'about',      'label' => 'About',          'href' => 'about'],
        ['key' => 'careers',    'label' => 'Careers',        'href' => 'careers'],
        ['key' => 'contact',    'label' => 'Contact',        'href' => 'contact'],
    ];
}

function knowledge_links(): array
{
    return [
        ['href' => 'knowledge-bank', 'icon' => 'book-open',  'label' => 'Insights & guides',      'text' => 'Practical articles on tax, SECP, audit and reporting'],
        ['href' => 'tax-calendar',   'icon' => 'calendar',   'label' => 'Tax & compliance calendar', 'text' => 'Every recurring deadline, with live countdowns'],
        ['href' => 'tools',          'icon' => 'calculator', 'label' => 'Free calculators',       'text' => 'Salary tax, sales tax and amount-in-words'],
        ['href' => 'glossary',       'icon' => 'type',       'label' => 'Tax & business glossary', 'text' => 'NTN, STRN, ATL, Annex-C and 50+ terms explained'],
        ['href' => 'faq',            'icon' => 'help-circle', 'label' => 'Frequently asked questions', 'text' => 'Straight answers about working with us'],
        ['href' => 'official-portals', 'icon' => 'external-link', 'label' => 'Official portals',  'text' => 'FBR, IRIS, SECP, PRA, SRB and more in one place'],
    ];
}

/* --------------------------------------------------------------------------
 * Storage (enquiries, subscribers, applications) — kept in /storage,
 * which is closed to the web.
 * ----------------------------------------------------------------------- */

function storage_dir(string $sub = ''): string
{
    $dir = VP_ROOT . '/storage' . ($sub !== '' ? '/' . $sub : '');
    if (!is_dir($dir)) {
        @mkdir($dir, 0750, true);
    }
    return $dir;
}

/** A secret for signing form tokens, created on first use. */
function vp_secret(): string
{
    static $secret = null;
    if ($secret !== null) {
        return $secret;
    }
    $file = storage_dir() . '/secret.php';
    if (is_file($file)) {
        $value = include $file;
        if (is_string($value) && strlen($value) >= 32) {
            return $secret = $value;
        }
    }
    $secret = bin2hex(random_bytes(32));
    @file_put_contents($file, "<?php\nreturn '" . $secret . "';\n", LOCK_EX);
    return $secret;
}

/** Signed time token placed in every form (spam protection, no cookies needed). */
function form_token(): string
{
    $t = (string) time();
    return $t . '.' . hash_hmac('sha256', $t, vp_secret());
}

/** Returns '' when valid, otherwise a short reason. */
function check_form_token(string $token, int $minSeconds = 3, int $maxSeconds = 259200): string
{
    $parts = explode('.', $token, 2);
    if (count($parts) !== 2 || !ctype_digit($parts[0])) {
        return 'invalid';
    }
    if (!hash_equals(hash_hmac('sha256', $parts[0], vp_secret()), $parts[1])) {
        return 'invalid';
    }
    $age = time() - (int) $parts[0];
    if ($age < $minSeconds) {
        return 'too-fast';
    }
    if ($age > $maxSeconds) {
        return 'expired';
    }
    return '';
}

/** Hidden fields every public form carries. */
function form_guard_fields(string $formType): string
{
    return '<input type="hidden" name="form" value="' . e($formType) . '">'
        . '<input type="hidden" name="_t" value="' . e(form_token()) . '">'
        . '<div class="hp" aria-hidden="true"><label>Leave this field empty<input type="text" name="website" tabindex="-1" autocomplete="off"></label></div>';
}

/* --------------------------------------------------------------------------
 * Page output helpers
 * ----------------------------------------------------------------------- */

function json_ld(array $data): string
{
    return '<script type="application/ld+json">'
        . json_encode($data, JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE | JSON_HEX_TAG | JSON_HEX_AMP)
        . '</script>' . "\n";
}

/** schema.org description of the firm, reused on every page. */
function organization_schema(): array
{
    $org = [
        '@type' => ['ProfessionalService', 'AccountingService'],
        '@id' => abs_url('#organization'),
        'name' => company_name(),
        'legalName' => legal_name(),
        'url' => abs_url(),
        'logo' => abs_url('assets/img/logo-mark-512.png'),
        'image' => abs_url('assets/img/og-default.png'),
        'description' => 'Audit, tax, corporate and advisory firm in Pakistan, and developer of Veridian E-invoicing Pakistan, FBR-integrated digital invoicing software.',
        'email' => contact_email(),
        'areaServed' => ['@type' => 'Country', 'name' => 'Pakistan'],
        'knowsAbout' => [
            'Income tax', 'Sales tax', 'FBR Digital Invoicing', 'SECP corporate compliance', 'Internal audit',
            'IFRS financial reporting', 'Bookkeeping', 'Business advisory', 'Company registration in Pakistan',
        ],
        'priceRange' => '$$',
    ];
    if ($p = cfg('contact.phone') ?: cfg('contact.mobile')) {
        $org['telephone'] = $p;
    }
    if (cfg('company.founded')) {
        $org['foundingDate'] = (string) cfg('company.founded');
    }
    if (cfg('contact.city')) {
        $org['address'] = array_filter([
            '@type' => 'PostalAddress',
            'streetAddress' => trim(cfg('contact.address_line_1', '') . ' ' . cfg('contact.address_line_2', '')),
            'addressLocality' => cfg('contact.city'),
            'addressRegion' => cfg('contact.province'),
            'postalCode' => cfg('contact.postal_code'),
            'addressCountry' => 'PK',
        ]);
    }
    if (cfg('contact.latitude') && cfg('contact.longitude')) {
        $org['geo'] = ['@type' => 'GeoCoordinates', 'latitude' => cfg('contact.latitude'), 'longitude' => cfg('contact.longitude')];
    }
    $sameAs = array_column(social_links(), 'href');
    if ($sameAs) {
        $org['sameAs'] = $sameAs;
    }
    return $org;
}

function breadcrumb_schema(array $crumbs): array
{
    $items = [];
    foreach (array_values($crumbs) as $i => [$name, $path]) {
        $items[] = ['@type' => 'ListItem', 'position' => $i + 1, 'name' => $name, 'item' => abs_url($path)];
    }
    return ['@type' => 'BreadcrumbList', 'itemListElement' => $items];
}

function faq_schema(array $qa): array
{
    $list = [];
    foreach ($qa as $item) {
        $list[] = [
            '@type' => 'Question',
            'name' => $item['q'],
            'acceptedAnswer' => ['@type' => 'Answer', 'text' => strip_tags($item['a'], '<a><strong><em><br><ul><li>')],
        ];
    }
    return ['@type' => 'FAQPage', 'mainEntity' => $list];
}

/** Initials for avatars, e.g. "Muhammad Shahjahan" → "MS". */
function initials(string $name): string
{
    $out = '';
    foreach (preg_split('/\s+/', trim($name)) as $w) {
        if ($w !== '') {
            $out .= mb_strtoupper(mb_substr($w, 0, 1));
        }
        if (mb_strlen($out) >= 2) {
            break;
        }
    }
    return $out;
}

/** Send a 404 page and stop. */
function not_found(): void
{
    http_response_code(404);
    require VP_ROOT . '/404.php';
    exit;
}
