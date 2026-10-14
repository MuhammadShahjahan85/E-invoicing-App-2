<?php
require __DIR__ . '/includes/bootstrap.php';

$crumbs = [['Home', ''], ['Contact', 'contact']];
$phone = cfg('contact.phone');
$mobile = cfg('contact.mobile');
$wa = whatsapp_link();
$address = address_lines();
$preset = preg_replace('/[^a-z0-9-]/', '', (string) ($_GET['service'] ?? ''));

$page = [
    'title' => 'Contact Us — Talk to a Tax & Advisory Expert in Pakistan',
    'description' => 'Contact Veridian Partners for tax, audit, SECP, accounting, advisory and FBR e-invoicing. E-mail, phone or WhatsApp — we reply within one business day.',
    'keywords' => 'contact tax consultant Pakistan, tax advisor contact, FBR e-invoicing support, Veridian Partners contact',
    'path' => 'contact',
    'nav' => 'contact',
    'schema' => [breadcrumb_schema($crumbs), ['@type' => 'ContactPage', 'name' => 'Contact ' . company_name(), 'url' => abs_url('contact'), 'mainEntity' => ['@id' => abs_url('#organization')]]],
];
require __DIR__ . '/includes/header.php';

echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => 'Contact',
    'title' => 'Let\'s <em>talk</em>.',
    'lede' => 'Tell us what you need — a question, a deadline, a notice or a new project. A senior adviser will reply within one business day.',
]);
?>

<section class="section">
  <div class="container contact-grid">
    <div class="panel" data-form-wrap>
      <h2 style="font-size:clamp(1.6rem,2.6vw,2.2rem)">Send us a message</h2>
      <p class="muted">Fields marked <span class="req">*</span> are required.</p>
      <?= contact_form(['id' => 'contact-page-form', 'service' => $preset]) ?>
    </div>

    <div class="contact-cards">
      <a class="contact-card" href="mailto:<?= e(contact_email()) ?>"><span class="icon-tile icon-tile--sm"><?= icon('mail') ?></span><span><strong>E-mail</strong><span><?= e(contact_email()) ?></span></span></a>
      <?php if ($phone): ?><a class="contact-card" href="<?= e(tel_link($phone)) ?>"><span class="icon-tile icon-tile--sm"><?= icon('phone') ?></span><span><strong>Phone</strong><span><?= e($phone) ?></span></span></a><?php endif; ?>
      <?php if ($mobile): ?><a class="contact-card" href="<?= e(tel_link($mobile)) ?>"><span class="icon-tile icon-tile--sm"><?= icon('smartphone') ?></span><span><strong>Mobile</strong><span><?= e($mobile) ?></span></span></a><?php endif; ?>
      <?php if ($wa): ?><a class="contact-card" href="<?= e($wa) ?>" target="_blank" rel="noopener"><span class="icon-tile icon-tile--sm" style="color:#1fa855"><?= icon('whatsapp') ?></span><span><strong>WhatsApp</strong><span>Quick questions, documents and updates</span></span></a><?php endif; ?>
      <?php if ($address): ?>
      <<?= cfg('contact.map_link') ? 'a href="' . e(cfg('contact.map_link')) . '" target="_blank" rel="noopener"' : 'div' ?> class="contact-card"><span class="icon-tile icon-tile--sm"><?= icon('map-pin') ?></span><span><strong>Office</strong><span><?= implode('<br>', array_map('e', $address)) ?></span></span></<?= cfg('contact.map_link') ? 'a' : 'div' ?>>
      <?php else: ?>
      <div class="contact-card"><span class="icon-tile icon-tile--sm"><?= icon('globe') ?></span><span><strong>Nationwide service</strong><span>Clients across Pakistan and overseas — online, with on-site visits when needed.</span></span></div>
      <?php endif; ?>
      <?php if (cfg('contact.hours')): ?><div class="contact-card"><span class="icon-tile icon-tile--sm"><?= icon('clock') ?></span><span><strong>Office hours</strong><span><?= e(cfg('contact.hours')) ?></span></span></div><?php endif; ?>

      <div class="side-card mt-2">
        <p class="side-card__heading">What happens next</p>
        <ol class="next-steps">
          <li><span><strong>We read your message</strong>A senior adviser reviews it — usually the same day.</span></li>
          <li><span><strong>We get in touch</strong>Within one business day, by your preferred method.</span></li>
          <li><span><strong>Free consultation</strong>Up to 30 minutes to understand your needs.</span></li>
          <li><span><strong>Clear proposal</strong>Scope, timeline and a fixed fee — in writing.</span></li>
        </ol>
      </div>
    </div>
  </div>
</section>

<?php if (cfg('contact.map_embed_url')): ?>
<section class="section section--tight section--tint">
  <div class="container">
    <div class="map"><iframe src="<?= e(cfg('contact.map_embed_url')) ?>" title="Map showing our office" loading="lazy" referrerpolicy="no-referrer-when-downgrade" allowfullscreen></iframe></div>
  </div>
</section>
<?php endif; ?>

<section class="section">
  <div class="container">
    <div class="grid grid--3">
      <a class="card card--industry reveal" href="<?= e(url('book-consultation')) ?>"><span class="icon-tile icon-tile--sm icon-tile--gold"><?= icon('calendar') ?></span><span class="card--industry__body"><span class="card__title">Book a consultation</span><span class="card__text">Pick the help you need and a time that suits you.</span></span><?= icon('arrow-up-right', 'card--industry__arrow') ?></a>
      <a class="card card--industry reveal" href="<?= e(url('faq')) ?>"><span class="icon-tile icon-tile--sm icon-tile--gold"><?= icon('help-circle') ?></span><span class="card--industry__body"><span class="card__title">Frequently asked questions</span><span class="card__text">Fees, response times, confidentiality and more.</span></span><?= icon('arrow-up-right', 'card--industry__arrow') ?></a>
      <a class="card card--industry reveal" href="<?= e(url('careers')) ?>"><span class="icon-tile icon-tile--sm icon-tile--gold"><?= icon('briefcase') ?></span><span class="card--industry__body"><span class="card__title">Careers</span><span class="card__text">Looking to join us? Apply on our careers page.</span></span><?= icon('arrow-up-right', 'card--industry__arrow') ?></a>
    </div>
  </div>
</section>

<?php require __DIR__ . '/includes/footer.php'; ?>
