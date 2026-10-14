<?php
require __DIR__ . '/includes/bootstrap.php';

$crumbs = [['Home', ''], ['Careers', 'careers']];
$jobs = array_values(array_filter(cfg('jobs', []), fn($j) => !empty($j['open'])));
$schema = [breadcrumb_schema($crumbs)];
foreach ($jobs as $j) {
    $schema[] = [
        '@type' => 'JobPosting',
        'title' => $j['title'],
        'description' => '<p>' . e($j['summary']) . '</p>',
        'datePosted' => date('Y-m-d'),
        'employmentType' => strtoupper(str_replace(['-', ' '], '_', $j['type'])) === 'FULL_TIME' ? 'FULL_TIME' : 'OTHER',
        'hiringOrganization' => ['@type' => 'Organization', 'name' => legal_name(), 'sameAs' => abs_url()],
        'jobLocation' => ['@type' => 'Place', 'address' => ['@type' => 'PostalAddress', 'addressLocality' => cfg('contact.city') ?: 'Pakistan', 'addressCountry' => 'PK']],
        'directApply' => true,
    ];
}
$page = [
    'title' => 'Careers — Build Your Career in Tax, Audit & Advisory',
    'description' => 'Join Veridian Partners: careers and traineeships in tax, audit and assurance, corporate, accounting and technology in Pakistan. Apply online.',
    'keywords' => 'tax jobs Pakistan, audit trainee Pakistan, ACCA trainee jobs, accounting jobs Pakistan, careers Veridian Partners',
    'path' => 'careers',
    'nav' => 'careers',
    'schema' => $schema,
];
require __DIR__ . '/includes/header.php';

echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => 'Careers',
    'title' => 'Do work that <em>matters</em> — and grow faster.',
    'lede' => 'Join a team where you work on real client problems from day one, learn from senior advisers, and help build the technology that is changing compliance in Pakistan.',
    'actions' => '<a class="btn btn--gold btn--lg" href="#apply"><span>Apply now</span>' . icon('arrow-right') . '</a>',
]);
?>

<section class="section">
  <div class="container">
    <div class="section-head section-head--center">
      <p class="eyebrow">Why Veridian</p>
      <h2>A place to build a <em>real</em> career.</h2>
    </div>
    <div class="feature-grid">
      <div class="feature reveal"><span class="icon-tile"><?= icon('graduation-cap') ?></span><h3>Learning that counts</h3><p>Structured on-the-job training across tax, audit, corporate and reporting — the experience professional qualifications value.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('users') ?></span><h3>Mentors, not managers</h3><p>Work directly with senior advisers who review your work, explain their judgement and help you grow.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('cpu') ?></span><h3>Technology at work</h3><p>Use and help shape our own software and automation, and become fluent in the systems modern finance runs on.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('briefcase') ?></span><h3>Variety</h3><p>Manufacturers, exporters, start-ups, NGOs and investors — no two weeks are the same.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('trending-up') ?></span><h3>Room to grow</h3><p>Clear expectations at each level, regular feedback and responsibility as soon as you are ready for it.</p></div>
      <div class="feature reveal"><span class="icon-tile"><?= icon('heart') ?></span><h3>Respectful culture</h3><p>Integrity, teamwork and balance. We value people who care about doing things properly.</p></div>
    </div>
  </div>
</section>

<section class="section section--tint" aria-labelledby="roles-title">
  <div class="container">
    <div class="section-head">
      <p class="eyebrow">Open positions</p>
      <h2 id="roles-title"><?= $jobs ? 'Current <em>opportunities</em>' : 'Join our talent <em>community</em>' ?></h2>
      <p><?= $jobs ? 'Apply below and tell us which role interests you.' : 'We have no open positions right now, but we are always glad to hear from talented people in tax, audit, accounting, corporate and technology. Send us your CV and we will contact you when a suitable role opens.' ?></p>
    </div>
    <?php if ($jobs): ?>
    <div class="jobs">
      <?php foreach ($jobs as $j): ?>
      <div class="job reveal">
        <div>
          <div class="job__meta"><span class="tag"><?= e($j['type']) ?></span><span class="tag tag--payroll"><?= icon('map-pin') ?><?= e($j['location']) ?></span></div>
          <h3><?= e($j['title']) ?></h3>
          <p><?= e($j['summary']) ?></p>
        </div>
        <a class="btn btn--primary" href="#apply" data-role="<?= e($j['title']) ?>"><span>Apply</span><?= icon('arrow-right') ?></a>
      </div>
      <?php endforeach; ?>
    </div>
    <?php endif; ?>
  </div>
</section>

<section class="section" id="apply" aria-labelledby="apply-title">
  <div class="container detail">
    <div class="panel" data-form-wrap>
      <h2 id="apply-title" style="font-size:clamp(1.6rem,2.6vw,2.2rem)">Apply online</h2>
      <p class="muted">Your application is confidential and used only for recruitment.</p>
      <form class="form js-form" action="<?= e(url('submit')) ?>" method="post" enctype="multipart/form-data" novalidate>
        <?= form_guard_fields('career') ?>
        <div class="form__grid">
          <div class="field"><label for="cr-name">Full name <span class="req">*</span></label><input id="cr-name" name="name" type="text" autocomplete="name" required maxlength="120"></div>
          <div class="field"><label for="cr-email">E-mail <span class="req">*</span></label><input id="cr-email" name="email" type="email" autocomplete="email" required maxlength="160"></div>
          <div class="field"><label for="cr-phone">Phone <span class="req">*</span></label><input id="cr-phone" name="phone" type="tel" autocomplete="tel" required maxlength="40"></div>
          <div class="field"><label for="cr-city">City</label><input id="cr-city" name="city" type="text" maxlength="80"></div>
          <div class="field">
            <label for="cr-role">Role or area of interest <span class="req">*</span></label>
            <select id="cr-role" name="role" required>
              <option value="">Choose…</option>
              <?php foreach ($jobs as $j): ?><option><?= e($j['title']) ?></option><?php endforeach; ?>
              <option>Tax</option><option>Audit & assurance</option><option>Corporate & secretarial</option><option>Accounting & payroll</option><option>Advisory</option><option>Software & technology</option><option>Internship</option>
            </select>
          </div>
          <div class="field">
            <label for="cr-qual">Qualification / studies</label>
            <input id="cr-qual" name="qualification" type="text" maxlength="160" placeholder="e.g. ACCA (in progress), B.Com">
          </div>
          <div class="field field--full">
            <span class="field__label">CV <span class="req">*</span></span>
            <label class="file-drop">
              <?= icon('upload') ?>
              <span><strong data-file-label>Choose a file or drag it here</strong><br><small>PDF or Word, up to 5 MB</small></span>
              <input type="file" name="cv" accept=".pdf,.doc,.docx,application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document" required>
            </label>
          </div>
          <div class="field field--full"><label for="cr-message">Why Veridian? <span class="muted">(optional)</span></label><textarea id="cr-message" name="message" rows="4" maxlength="5000"></textarea></div>
        </div>
        <label class="check"><input type="checkbox" name="consent" value="yes" required><span>I agree to the <a href="<?= e(url('privacy-policy')) ?>">privacy policy</a> and to my data being kept for recruitment for up to 12 months.</span></label>
        <div class="form__footer">
          <button class="btn btn--primary btn--lg" type="submit"><span>Submit application</span><?= icon('send') ?></button>
        </div>
        <div class="form__status" role="status" aria-live="polite"></div>
      </form>
    </div>
    <aside class="detail__aside">
      <div class="side-card">
        <p class="side-card__heading">Our hiring process</p>
        <ol class="next-steps">
          <li><span><strong>Application</strong>We review every application carefully.</span></li>
          <li><span><strong>Conversation</strong>A short call to get to know you.</span></li>
          <li><span><strong>Practical task</strong>A realistic exercise — no trick questions.</span></li>
          <li><span><strong>Interview & offer</strong>Meet the team and agree the details.</span></li>
        </ol>
      </div>
    </aside>
  </div>
</section>

<script>
document.addEventListener('click', function (e) {
  var a = e.target.closest('[data-role]');
  if (!a) return;
  var sel = document.getElementById('cr-role');
  if (sel) { sel.value = a.getAttribute('data-role'); }
});
</script>
<?php require __DIR__ . '/includes/footer.php'; ?>
