<?php
require __DIR__ . '/includes/bootstrap.php';

$crumbs = [['Home', ''], ['Book a consultation', 'book-consultation']];
$preset = preg_replace('/[^a-z0-9-]/', '', (string) ($_GET['service'] ?? ''));
$page = [
    'title' => 'Book a Free Consultation with a Tax & Advisory Expert',
    'description' => 'Book a free 30-minute consultation with Veridian Partners about tax, FBR e-invoicing, SECP company matters, accounting, audit or advisory. We reply within one business day.',
    'keywords' => 'free tax consultation Pakistan, book consultation tax advisor, FBR e-invoicing demo, company registration consultation',
    'path' => 'book-consultation',
    'nav' => 'contact',
    'schema' => [breadcrumb_schema($crumbs)],
];
require __DIR__ . '/includes/header.php';

echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => 'Free consultation',
    'title' => 'Book a free <em>consultation</em>.',
    'lede' => 'Three short steps. Tell us what you need and when suits you — we will confirm within one business day.',
]);
?>

<section class="section">
  <div class="container detail">
    <div class="detail__main" data-form-wrap>
      <form class="wizard js-form" data-wizard action="<?= e(url('submit')) ?>" method="post" novalidate>
        <?= form_guard_fields('consultation') ?>
        <input type="hidden" name="source" value="/book-consultation">
        <div class="wizard__progress" aria-hidden="true">
          <div class="wizard__tab is-active"><b>1</b><span>Your needs</span></div>
          <div class="wizard__tab"><b>2</b><span>Your business</span></div>
          <div class="wizard__tab"><b>3</b><span>Your details</span></div>
        </div>
        <div class="wizard__body">
          <fieldset class="wizard__step is-current" style="border:0;padding:0;margin:0">
            <legend class="sr-only">Step 1: your needs</legend>
            <h2>What can we help with?</h2>
            <p>Choose everything that applies.</p>
            <div class="choices choices--cards" data-preselect data-required-group="Please choose at least one area.">
              <?php foreach (services() as $s): ?>
              <label class="choice"><input type="checkbox" name="services[]" value="<?= e($s['title']) ?>" data-slug="<?= e($s['slug']) ?>"<?= $preset === $s['slug'] ? ' checked' : '' ?>><span><?= icon($s['icon']) ?><?= e($s['nav']) ?></span></label>
              <?php endforeach; ?>
              <label class="choice"><input type="checkbox" name="services[]" value="Not sure yet"><span><?= icon('help-circle') ?>Not sure yet</span></label>
            </div>
            <div class="field mt-2">
              <label for="bc-urgency">How urgent is it?</label>
              <select id="bc-urgency" name="urgency">
                <option value="Planning ahead">Planning ahead</option>
                <option value="Within a month">Within a month</option>
                <option value="This week">This week</option>
                <option value="Urgent — deadline or notice">Urgent — a deadline or FBR notice</option>
              </select>
            </div>
            <div class="wizard__nav"><span></span><button class="btn btn--primary" type="button" data-wizard-next><span>Next</span><?= icon('arrow-right') ?></button></div>
          </fieldset>

          <fieldset class="wizard__step" style="border:0;padding:0;margin:0">
            <legend class="sr-only">Step 2: your business</legend>
            <h2>About your business</h2>
            <p>This helps us bring the right expert to the call.</p>
            <div class="wizard__group">
              <span class="field__label">You are</span>
              <div class="choices">
                <?php foreach (['Private limited company', 'Single-member company', 'Public / listed company', 'Partnership or AOP', 'Sole proprietor', 'Individual / salaried', 'NGO / non-profit', 'Foreign company / investor', 'Not yet registered'] as $i => $t): ?>
                <label class="choice"><input type="radio" name="entity" value="<?= e($t) ?>"<?= $i === 0 ? ' checked' : '' ?>><span><?= e($t) ?></span></label>
                <?php endforeach; ?>
              </div>
            </div>
            <div class="form__grid">
              <div class="field">
                <label for="bc-industry">Industry</label>
                <select id="bc-industry" name="industry">
                  <option value="">Choose…</option>
                  <?php foreach (industries() as $i): ?><option><?= e($i['name']) ?></option><?php endforeach; ?>
                  <option>Other</option>
                </select>
              </div>
              <div class="field">
                <label for="bc-turnover">Annual turnover</label>
                <select id="bc-turnover" name="turnover">
                  <option value="">Prefer not to say</option>
                  <option>Up to Rs 10 million</option>
                  <option>Rs 10 – 100 million</option>
                  <option>Rs 100 million – 1 billion</option>
                  <option>Above Rs 1 billion</option>
                </select>
              </div>
              <div class="field">
                <label for="bc-company">Company name</label>
                <input id="bc-company" name="company" type="text" autocomplete="organization" maxlength="160">
              </div>
              <div class="field">
                <label for="bc-city">City</label>
                <input id="bc-city" name="city" type="text" autocomplete="address-level2" maxlength="80" placeholder="e.g. Lahore">
              </div>
            </div>
            <div class="wizard__nav"><button class="btn btn--ghost" type="button" data-wizard-prev><?= icon('arrow-left') ?><span>Back</span></button><button class="btn btn--primary" type="button" data-wizard-next><span>Next</span><?= icon('arrow-right') ?></button></div>
          </fieldset>

          <fieldset class="wizard__step" style="border:0;padding:0;margin:0">
            <legend class="sr-only">Step 3: your details</legend>
            <h2>How can we reach you?</h2>
            <p>We will confirm the time within one business day.</p>
            <div class="form__grid">
              <div class="field"><label for="bc-name">Full name <span class="req">*</span></label><input id="bc-name" name="name" type="text" autocomplete="name" required maxlength="120"></div>
              <div class="field"><label for="bc-email">E-mail <span class="req">*</span></label><input id="bc-email" name="email" type="email" autocomplete="email" required maxlength="160"></div>
              <div class="field"><label for="bc-phone">Phone / WhatsApp <span class="req">*</span></label><input id="bc-phone" name="phone" type="tel" autocomplete="tel" required maxlength="40" placeholder="+92 3xx xxxxxxx"></div>
              <div class="field"><label for="bc-date">Preferred date</label><input id="bc-date" name="preferred_date" type="date" min="<?= e(date('Y-m-d')) ?>"></div>
            </div>
            <div class="wizard__group">
              <span class="field__label">Preferred way to meet</span>
              <div class="choices">
                <label class="choice"><input type="radio" name="mode" value="Phone call" checked><span><?= icon('phone') ?>Phone call</span></label>
                <label class="choice"><input type="radio" name="mode" value="Video call"><span><?= icon('monitor') ?>Video call</span></label>
                <label class="choice"><input type="radio" name="mode" value="WhatsApp"><span><?= icon('message') ?>WhatsApp</span></label>
                <label class="choice"><input type="radio" name="mode" value="In person"><span><?= icon('users') ?>In person</span></label>
              </div>
            </div>
            <div class="wizard__group">
              <span class="field__label">Preferred time</span>
              <div class="choices">
                <label class="choice"><input type="radio" name="time" value="Morning (9–12)" checked><span>Morning</span></label>
                <label class="choice"><input type="radio" name="time" value="Afternoon (12–3)"><span>Afternoon</span></label>
                <label class="choice"><input type="radio" name="time" value="Late afternoon (3–6)"><span>Late afternoon</span></label>
              </div>
            </div>
            <div class="field field--full" style="margin-bottom:1.1rem"><label for="bc-message">Anything we should know?</label><textarea id="bc-message" name="message" rows="4" maxlength="5000" placeholder="Deadlines, notices received, systems you use…"></textarea></div>
            <label class="check"><input type="checkbox" name="consent" value="yes" required><span>I agree to the <a href="<?= e(url('privacy-policy')) ?>">privacy policy</a> and to being contacted about my request.</span></label>
            <div class="wizard__nav"><button class="btn btn--ghost" type="button" data-wizard-prev><?= icon('arrow-left') ?><span>Back</span></button><button class="btn btn--gold btn--lg" type="submit"><span>Request my consultation</span><?= icon('send') ?></button></div>
            <div class="form__status" role="status" aria-live="polite"></div>
          </fieldset>
        </div>
      </form>
    </div>
    <aside class="detail__aside">
      <div class="side-card side-card--dark">
        <p class="eyebrow eyebrow--light">What to expect</p>
        <h3 class="side-card__title">30 minutes. No obligation. Real answers.</h3>
        <ul class="ticks" style="color:var(--on-deep-2)">
          <li style="color:var(--on-deep-2)">A senior adviser who understands your sector</li>
          <li style="color:var(--on-deep-2)">Your options explained in plain language</li>
          <li style="color:var(--on-deep-2)">A written, fixed-fee proposal if you want one</li>
          <li style="color:var(--on-deep-2)">Complete confidentiality</li>
        </ul>
      </div>
      <div class="side-card">
        <p class="side-card__heading">Prefer to talk now?</p>
        <ul class="side-links">
          <?php if ($wa = whatsapp_link()): ?><li><a href="<?= e($wa) ?>" target="_blank" rel="noopener">WhatsApp us<?= icon('chevron-right') ?></a></li><?php endif; ?>
          <?php if ($p = cfg('contact.phone') ?: cfg('contact.mobile')): ?><li><a href="<?= e(tel_link($p)) ?>">Call <?= e($p) ?><?= icon('chevron-right') ?></a></li><?php endif; ?>
          <li><a href="mailto:<?= e(contact_email()) ?>">E-mail us<?= icon('chevron-right') ?></a></li>
          <li><a href="<?= e(url('contact')) ?>">Send a message<?= icon('chevron-right') ?></a></li>
        </ul>
      </div>
    </aside>
  </div>
</section>

<?php require __DIR__ . '/includes/footer.php'; ?>
