<?php
/**
 * Reusable page sections.
 */

/** Rewrite root-relative links inside stored content ("/contact") for sites installed in a sub-folder. */
function content_html(string $html): string
{
    if (base_path() === '') {
        return $html;
    }
    return preg_replace_callback('/href="\/(?!\/)([^"]*)"/', fn($m) => 'href="' . e(url($m[1])) . '"', $html);
}

function breadcrumbs(array $crumbs, bool $light = true): string
{
    $out = '<nav class="crumbs' . ($light ? ' crumbs--light' : '') . '" aria-label="Breadcrumb"><ol>';
    $last = count($crumbs) - 1;
    foreach (array_values($crumbs) as $i => [$name, $path]) {
        if ($i === $last) {
            $out .= '<li aria-current="page">' . e($name) . '</li>';
        } else {
            $out .= '<li><a href="' . e(url($path)) . '">' . e($name) . '</a>' . icon('chevron-right', 'crumbs__sep') . '</li>';
        }
    }
    return $out . '</ol></nav>';
}

/**
 * Dark page header used on inner pages.
 * $opts: eyebrow, title (HTML allowed), lede, crumbs, actions (HTML), after (HTML), aside (HTML), class
 */
function page_hero(array $opts): string
{
    $out = '<section class="page-hero ' . e($opts['class'] ?? '') . '">'
        . '<div class="page-hero__bg" aria-hidden="true"><span class="orb orb--1"></span><span class="orb orb--2"></span><span class="gridlines"></span></div>'
        . '<div class="container page-hero__inner' . (!empty($opts['aside']) ? ' page-hero__inner--split' : '') . '"><div class="page-hero__copy">';
    if (!empty($opts['crumbs'])) {
        $out .= breadcrumbs($opts['crumbs']);
    }
    if (!empty($opts['eyebrow'])) {
        $out .= '<p class="eyebrow eyebrow--light">' . e($opts['eyebrow']) . '</p>';
    }
    $out .= '<h1 class="page-hero__title">' . $opts['title'] . '</h1>';
    if (!empty($opts['lede'])) {
        $out .= '<p class="page-hero__lede">' . $opts['lede'] . '</p>';
    }
    if (!empty($opts['actions'])) {
        $out .= '<div class="page-hero__actions">' . $opts['actions'] . '</div>';
    }
    if (!empty($opts['after'])) {
        $out .= $opts['after'];
    }
    $out .= '</div>';
    if (!empty($opts['aside'])) {
        $out .= '<div class="page-hero__aside">' . $opts['aside'] . '</div>';
    }
    return $out . '</div></section>';
}

function service_card(array $s, int $index = 0): string
{
    $num = $index > 0 ? '<span class="card__num">' . sprintf('%02d', $index) . '</span>' : '';
    return '<a class="card card--service reveal" href="' . e(service_url($s)) . '">'
        . $num
        . '<span class="icon-tile">' . icon($s['icon']) . '</span>'
        . '<h3 class="card__title">' . e($s['title']) . '</h3>'
        . '<p class="card__text">' . e($s['short']) . '</p>'
        . '<span class="link-arrow">Learn more ' . icon('arrow-right') . '</span>'
        . '</a>';
}

function industry_card(array $i): string
{
    return '<a class="card card--industry reveal" href="' . e(url('industries/' . $i['slug'])) . '">'
        . '<span class="icon-tile icon-tile--sm">' . icon($i['icon']) . '</span>'
        . '<span class="card--industry__body"><span class="card__title">' . e($i['name']) . '</span>'
        . '<span class="card__text">' . e($i['short']) . '</span></span>'
        . icon('arrow-up-right', 'card--industry__arrow')
        . '</a>';
}

function article_card(array $a, string $variant = ''): string
{
    $cats = article_categories();
    $cat = $cats[$a['category']]['name'] ?? '';
    return '<article class="card card--article ' . e($variant) . ' reveal" data-category="' . e($a['category']) . '" data-search="' . e(mb_strtolower($a['title'] . ' ' . $a['excerpt'] . ' ' . implode(' ', $a['tags'] ?? []))) . '">'
        . '<a class="card--article__link" href="' . e(article_url($a)) . '">'
        . '<div class="card--article__meta"><span class="tag">' . e($cat) . '</span><span>' . e(format_date($a['date'])) . ' · ' . reading_minutes($a['body']) . ' min read</span></div>'
        . '<h3 class="card__title">' . e($a['title']) . '</h3>'
        . '<p class="card__text">' . e($a['excerpt']) . '</p>'
        . '<span class="link-arrow">Read the guide ' . icon('arrow-right') . '</span>'
        . '</a></article>';
}

/** FAQ accordion (native <details>). */
function faq_list(array $items, string $class = ''): string
{
    $out = '<div class="faq ' . e($class) . '">';
    foreach ($items as $item) {
        $out .= '<details class="faq__item"><summary><span>' . e($item['q']) . '</span><span class="faq__toggle" aria-hidden="true"></span></summary>'
            . '<div class="faq__answer"><p>' . content_html($item['a']) . '</p></div></details>';
    }
    return $out . '</div>';
}

/** Service options for form selects. */
function service_options(string $selected = ''): string
{
    $out = '<option value="">Choose a service…</option>';
    foreach (services() as $s) {
        $out .= '<option value="' . e($s['title']) . '"' . ($selected === $s['slug'] ? ' selected' : '') . '>' . e($s['title']) . '</option>';
    }
    return $out . '<option value="Something else">Something else</option>';
}

/** Standard enquiry form. */
function contact_form(array $opts = []): string
{
    $id = $opts['id'] ?? 'contact-form';
    $preset = $opts['service'] ?? '';
    $compact = !empty($opts['compact']);
    $source = $opts['source'] ?? current_path();
    ob_start(); ?>
<form class="form js-form<?= $compact ? ' form--compact' : '' ?>" id="<?= e($id) ?>" action="<?= e(url('submit')) ?>" method="post" novalidate>
  <?= form_guard_fields('contact') ?>
  <input type="hidden" name="source" value="<?= e($source) ?>">
  <div class="form__grid">
    <div class="field">
      <label for="<?= e($id) ?>-name">Full name <span class="req" aria-hidden="true">*</span></label>
      <input id="<?= e($id) ?>-name" name="name" type="text" autocomplete="name" required maxlength="120">
    </div>
    <div class="field">
      <label for="<?= e($id) ?>-email">E-mail <span class="req" aria-hidden="true">*</span></label>
      <input id="<?= e($id) ?>-email" name="email" type="email" autocomplete="email" required maxlength="160">
    </div>
    <div class="field">
      <label for="<?= e($id) ?>-phone">Phone / WhatsApp</label>
      <input id="<?= e($id) ?>-phone" name="phone" type="tel" autocomplete="tel" maxlength="40" placeholder="+92 3xx xxxxxxx">
    </div>
    <?php if (!$compact): ?>
    <div class="field">
      <label for="<?= e($id) ?>-company">Company</label>
      <input id="<?= e($id) ?>-company" name="company" type="text" autocomplete="organization" maxlength="160">
    </div>
    <?php endif; ?>
    <div class="field<?= $compact ? '' : ' field--full' ?>">
      <label for="<?= e($id) ?>-service">How can we help?</label>
      <select id="<?= e($id) ?>-service" name="service"><?= service_options($preset) ?></select>
    </div>
    <div class="field field--full">
      <label for="<?= e($id) ?>-message">Message <span class="req" aria-hidden="true">*</span></label>
      <textarea id="<?= e($id) ?>-message" name="message" rows="<?= $compact ? 3 : 5 ?>" required maxlength="5000" placeholder="Tell us briefly about your business and what you need."></textarea>
    </div>
  </div>
  <label class="check">
    <input type="checkbox" name="consent" value="yes" required>
    <span>I agree to the <a href="<?= e(url('privacy-policy')) ?>">privacy policy</a> and to being contacted about my enquiry.</span>
  </label>
  <div class="form__footer">
    <button class="btn btn--primary btn--lg" type="submit"><span>Send message</span><?= icon('send') ?></button>
    <p class="form__note"><?= icon('lock') ?> Confidential. We reply within one business day.</p>
  </div>
  <div class="form__status" role="status" aria-live="polite"></div>
</form>
<?php
    return ob_get_clean();
}

function newsletter_form(string $variant = ''): string
{
    ob_start(); ?>
<form class="newsletter js-form <?= e($variant) ?>" action="<?= e(url('submit')) ?>" method="post" novalidate>
  <?= form_guard_fields('newsletter') ?>
  <label class="sr-only" for="nl-email-<?= e($variant ?: 'default') ?>">E-mail address</label>
  <div class="newsletter__row">
    <input id="nl-email-<?= e($variant ?: 'default') ?>" type="email" name="email" placeholder="Your e-mail address" autocomplete="email" required maxlength="160">
    <button class="btn btn--gold" type="submit"><span>Subscribe</span><?= icon('arrow-right') ?></button>
  </div>
  <input type="hidden" name="consent" value="yes">
  <p class="newsletter__note">Monthly tax and compliance updates. No spam; unsubscribe any time.</p>
  <div class="form__status" role="status" aria-live="polite"></div>
</form>
<?php
    return ob_get_clean();
}

/** Closing call-to-action band. */
function cta_band(string $title = '', string $text = ''): string
{
    $title = $title ?: 'Let\'s make compliance the <em>easy</em> part of your business.';
    $text = $text ?: 'Tell us what you need. A senior adviser will reply within one business day — the first consultation is free.';
    $wa = whatsapp_link();
    ob_start(); ?>
<section class="cta-band" aria-labelledby="cta-band-title">
  <div class="cta-band__bg" aria-hidden="true"><span class="orb orb--1"></span><span class="orb orb--2"></span></div>
  <div class="container cta-band__inner">
    <div>
      <p class="eyebrow eyebrow--light">Free first consultation</p>
      <h2 id="cta-band-title" class="cta-band__title"><?= $title ?></h2>
      <p class="cta-band__text"><?= e($text) ?></p>
    </div>
    <div class="cta-band__actions">
      <a class="btn btn--gold btn--lg" href="<?= e(url('book-consultation')) ?>"><?= icon('calendar') ?><span>Book a consultation</span></a>
      <?php if ($wa): ?>
      <a class="btn btn--ghost-light btn--lg" href="<?= e($wa) ?>" target="_blank" rel="noopener"><?= icon('whatsapp') ?><span>Chat on WhatsApp</span></a>
      <?php else: ?>
      <a class="btn btn--ghost-light btn--lg" href="<?= e(url('contact')) ?>"><?= icon('mail') ?><span>Send us a message</span></a>
      <?php endif; ?>
    </div>
  </div>
</section>
<?php
    return ob_get_clean();
}

/** Small "talk to an expert" side card. */
function expert_card(string $service = ''): string
{
    $wa = whatsapp_link();
    $phone = cfg('contact.phone') ?: cfg('contact.mobile');
    ob_start(); ?>
<div class="side-card side-card--dark">
  <p class="eyebrow eyebrow--light">Talk to an expert</p>
  <h3 class="side-card__title">Get a clear answer — and a fixed quote.</h3>
  <p>Free 30-minute consultation with a senior adviser. Reply within one business day.</p>
  <a class="btn btn--gold btn--block" href="<?= e(url('book-consultation' . ($service ? '?service=' . rawurlencode($service) : ''))) ?>"><?= icon('calendar') ?><span>Book a consultation</span></a>
  <?php if ($wa): ?><a class="btn btn--ghost-light btn--block" href="<?= e($wa) ?>" target="_blank" rel="noopener"><?= icon('whatsapp') ?><span>WhatsApp us</span></a><?php endif; ?>
  <ul class="side-card__contact">
    <?php if ($phone): ?><li><?= icon('phone') ?><a href="<?= e(tel_link($phone)) ?>"><?= e($phone) ?></a></li><?php endif; ?>
    <li><?= icon('mail') ?><a href="mailto:<?= e(contact_email()) ?>"><?= e(contact_email()) ?></a></li>
  </ul>
</div>
<?php
    return ob_get_clean();
}

/** Decorative QR-style graphic (not a real code) for illustrations. */
function decorative_qr(int $size = 21): string
{
    mt_srand(7823);
    $cells = '';
    $finder = function (int $x, int $y) {
        return '<rect x="' . $x . '" y="' . $y . '" width="7" height="7" fill="#0A3A31"/><rect x="' . ($x + 1) . '" y="' . ($y + 1) . '" width="5" height="5" fill="#fff"/><rect x="' . ($x + 2) . '" y="' . ($y + 2) . '" width="3" height="3" fill="#0A3A31"/>';
    };
    for ($y = 0; $y < $size; $y++) {
        for ($x = 0; $x < $size; $x++) {
            $inFinder = ($x < 8 && $y < 8) || ($x > $size - 9 && $y < 8) || ($x < 8 && $y > $size - 9);
            if (!$inFinder && mt_rand(0, 100) < 48) {
                $cells .= '<rect x="' . $x . '" y="' . $y . '" width="1" height="1"/>';
            }
        }
    }
    mt_srand();
    return '<svg viewBox="0 0 ' . $size . ' ' . $size . '" shape-rendering="crispEdges" aria-hidden="true"><g fill="#0A3A31">' . $cells . '</g>'
        . $finder(0, 0) . $finder($size - 7, 0) . $finder(0, $size - 7) . '</svg>';
}

/** Share buttons for an article. */
function share_buttons(string $title, string $absUrl): string
{
    $u = rawurlencode($absUrl);
    $t = rawurlencode($title);
    return '<div class="share" aria-label="Share this page">'
        . '<span class="share__label">Share</span>'
        . '<a class="share__btn" href="https://wa.me/?text=' . $t . '%20' . $u . '" target="_blank" rel="noopener" aria-label="Share on WhatsApp">' . icon('whatsapp') . '</a>'
        . '<a class="share__btn" href="https://www.linkedin.com/sharing/share-offsite/?url=' . $u . '" target="_blank" rel="noopener" aria-label="Share on LinkedIn">' . icon('linkedin') . '</a>'
        . '<a class="share__btn" href="https://www.facebook.com/sharer/sharer.php?u=' . $u . '" target="_blank" rel="noopener" aria-label="Share on Facebook">' . icon('facebook') . '</a>'
        . '<a class="share__btn" href="https://x.com/intent/post?text=' . $t . '&amp;url=' . $u . '" target="_blank" rel="noopener" aria-label="Share on X">' . icon('x-brand') . '</a>'
        . '<a class="share__btn" href="mailto:?subject=' . $t . '&amp;body=' . $u . '" aria-label="Share by e-mail">' . icon('mail') . '</a>'
        . '<button class="share__btn js-copy" type="button" data-copy="' . e($absUrl) . '" aria-label="Copy link">' . icon('link') . '</button>'
        . '<button class="share__btn js-print" type="button" aria-label="Print">' . icon('printer') . '</button>'
        . '</div>';
}
