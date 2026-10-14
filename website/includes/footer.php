<?php
/** Page footer. */
if (!function_exists('vp_render_footer')) {
function vp_render_footer(): void
{
$services = services();
$industries = array_slice(industries(), 0, 7, true);
$phone = cfg('contact.phone');
$mobile = cfg('contact.mobile');
$address = address_lines();
$socials = social_links();
$wa = whatsapp_link();
$gaId = trim((string) cfg('site.google_analytics_id', ''));
?>
</main>

<footer class="site-footer">
  <div class="site-footer__top container">
    <div class="site-footer__brand">
      <a class="brand brand--light" href="<?= e(url()) ?>" aria-label="<?= e(company_name()) ?> — home">
        <?= brand_mark('brand__mark', 'ftr') ?>
        <span class="brand__text"><span class="brand__name">Veridian</span><span class="brand__sub">Partners</span></span>
      </a>
      <p class="site-footer__about">Audit, tax, corporate and advisory services for businesses across Pakistan — with technology we build ourselves.</p>
      <ul class="site-footer__contact">
        <?php if ($address): ?><li><?= icon('map-pin') ?><span><?= implode('<br>', array_map('e', $address)) ?></span></li><?php endif; ?>
        <?php if ($phone): ?><li><?= icon('phone') ?><a href="<?= e(tel_link($phone)) ?>"><?= e($phone) ?></a></li><?php endif; ?>
        <?php if ($mobile): ?><li><?= icon('smartphone') ?><a href="<?= e(tel_link($mobile)) ?>"><?= e($mobile) ?></a></li><?php endif; ?>
        <li><?= icon('mail') ?><a href="mailto:<?= e(contact_email()) ?>"><?= e(contact_email()) ?></a></li>
        <?php if (cfg('contact.hours')): ?><li><?= icon('clock') ?><span><?= e(cfg('contact.hours')) ?></span></li><?php endif; ?>
      </ul>
      <?php if ($socials): ?>
      <ul class="socials" aria-label="Social media">
        <?php foreach ($socials as $s): ?>
        <li><a href="<?= e($s['href']) ?>" target="_blank" rel="noopener" aria-label="<?= e($s['label']) ?>"><?= icon($s['key'] === 'x' ? 'x-brand' : $s['key']) ?></a></li>
        <?php endforeach; ?>
      </ul>
      <?php endif; ?>
    </div>

    <nav class="site-footer__cols" aria-label="Footer">
      <div>
        <p class="site-footer__heading">Services</p>
        <ul>
          <?php foreach ($services as $s): ?><li><a href="<?= e(service_url($s)) ?>"><?= e($s['nav']) ?></a></li><?php endforeach; ?>
        </ul>
      </div>
      <div>
        <p class="site-footer__heading">Industries</p>
        <ul>
          <?php foreach ($industries as $i): ?><li><a href="<?= e(url('industries/' . $i['slug'])) ?>"><?= e($i['name']) ?></a></li><?php endforeach; ?>
          <li><a href="<?= e(url('industries')) ?>">All industries →</a></li>
        </ul>
      </div>
      <div>
        <p class="site-footer__heading">Knowledge Bank</p>
        <ul>
          <?php foreach (knowledge_links() as $k): ?><li><a href="<?= e(url($k['href'])) ?>"><?= e($k['label']) ?></a></li><?php endforeach; ?>
        </ul>
        <p class="site-footer__heading site-footer__heading--spaced">Company</p>
        <ul>
          <li><a href="<?= e(url('about')) ?>">About us</a></li>
          <li><a href="<?= e(url('careers')) ?>">Careers</a></li>
          <li><a href="<?= e(url('contact')) ?>">Contact</a></li>
          <li><a href="<?= e(url('book-consultation')) ?>">Book a consultation</a></li>
        </ul>
      </div>
    </nav>

    <div class="site-footer__newsletter">
      <p class="site-footer__heading">The Veridian Brief</p>
      <p>Tax changes, FBR notifications and compliance deadlines — explained in plain language, once a month.</p>
      <?= newsletter_form('footer') ?>
    </div>
  </div>

  <div class="site-footer__bottom">
    <div class="container site-footer__bottom-inner">
      <p>© <?= date('Y') ?> <?= e(legal_name()) ?>. All rights reserved.<?php
        $reg = array_filter([cfg('company.secp_cuin') ? 'SECP CUIN ' . cfg('company.secp_cuin') : '', cfg('company.ntn') ? 'NTN ' . cfg('company.ntn') : '']);
        if ($reg) { echo ' <span class="site-footer__reg">' . e(implode(' · ', $reg)) . '</span>'; } ?></p>
      <ul class="site-footer__legal">
        <li><a href="<?= e(url('privacy-policy')) ?>">Privacy</a></li>
        <li><a href="<?= e(url('terms')) ?>">Terms</a></li>
        <li><a href="<?= e(url('cookie-policy')) ?>">Cookies</a></li>
        <li><a href="<?= e(url('disclaimer')) ?>">Disclaimer</a></li>
        <li><a href="<?= e(url('accessibility')) ?>">Accessibility</a></li>
        <li><a href="<?= e(url('sitemap')) ?>">Sitemap</a></li>
      </ul>
    </div>
  </div>
</footer>

<div class="floating" aria-label="Quick actions">
  <?php if ($wa): ?>
  <a class="fab fab--whatsapp" href="<?= e($wa) ?>" target="_blank" rel="noopener" aria-label="Chat with us on WhatsApp"><?= icon('whatsapp') ?><span class="fab__label">Chat with us</span></a>
  <?php endif; ?>
  <button class="fab fab--top" type="button" data-to-top aria-label="Back to top"><?= icon('arrow-up') ?></button>
</div>

<div class="toast" data-toast role="status" aria-live="polite" hidden></div>

<?php if ($gaId): ?>
<div class="cookie" data-cookie hidden>
  <p><?= icon('info') ?> We use cookies to understand how our website is used and to improve it. <a href="<?= e(url('cookie-policy')) ?>">Cookie policy</a></p>
  <div class="cookie__actions">
    <button class="btn btn--ghost btn--sm" type="button" data-cookie-choice="decline">Decline</button>
    <button class="btn btn--primary btn--sm" type="button" data-cookie-choice="accept">Accept</button>
  </div>
</div>
<?php endif; ?>

<script src="<?= e(asset('js/main.js')) ?>" defer></script>
</body>
</html>
<?php
}
}
vp_render_footer();
