<?php
require __DIR__ . '/includes/bootstrap.php';

$crumbs = [['Home', ''], ['About us', 'about']];
$team = cfg('team', []);
$page = [
    'title' => 'About Us — Veridian Partners Consultancy (Pvt) Ltd',
    'description' => 'Veridian Partners Consultancy Private Limited is a Pakistani audit, tax and advisory firm combining professional judgement with technology — including our own FBR-integrated e-invoicing software.',
    'keywords' => 'about Veridian Partners, Veridian Partners Consultancy Private Limited, tax advisory firm Pakistan, audit and advisory Pakistan',
    'path' => 'about',
    'nav' => 'about',
    'schema' => [breadcrumb_schema($crumbs), ['@type' => 'AboutPage', 'name' => 'About ' . company_name(), 'url' => abs_url('about'), 'mainEntity' => ['@id' => abs_url('#organization')]]],
];
require __DIR__ . '/includes/header.php';

echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => 'About us',
    'title' => 'Built on trust. Driven by <em>clarity</em>.',
    'lede' => 'We are a Pakistani firm of advisers and technologists who believe compliance should be calm, finance should create confidence, and good advice should be easy to understand.',
]);

$facts = array_filter([
    'Legal name' => legal_name(),
    'Founded' => cfg('company.founded'),
    'SECP CUIN' => cfg('company.secp_cuin'),
    'NTN' => cfg('company.ntn'),
    'Head office' => cfg('contact.city') ? cfg('contact.city') . ', Pakistan' : '',
    'Clients served' => 'Across Pakistan and overseas',
]);
?>

<section class="section">
  <div class="container statement">
    <div class="reveal">
      <p class="eyebrow">Our story</p>
      <h2 class="statement__quote">A firm for the way Pakistan does business <em>now</em>.</h2>
    </div>
    <div class="reveal prose">
      <p>Pakistan's tax and regulatory system is going digital fast. Invoices are reported to FBR in real time, returns are cross-matched electronically, and SECP expects transparent ownership and timely filings. Businesses need advisers who understand both the law and the systems that now enforce it.</p>
      <p>Veridian Partners was founded to be that adviser. We bring together expertise in tax, audit and assurance, corporate law, accounting and finance — and we build technology ourselves. Our <a href="<?= e(url('e-invoicing')) ?>">Veridian E-invoicing Pakistan</a> software, built to FBR's technical specification and the Sales Tax Rules, is used by businesses to report every invoice to FBR with confidence.</p>
      <p><strong>Our name says what we stand for.</strong> <em>Veridian</em> comes from viridian — the deep green of growth and renewal. <em>Partners</em> because we work alongside our clients, not above them.</p>
    </div>
  </div>
</section>

<section class="section section--tint">
  <div class="container">
    <div class="mv">
      <div class="mv__card reveal"><h3>Our mission</h3><p>To make compliance effortless and finance a source of confidence for every business we serve.</p></div>
      <div class="mv__card reveal"><h3>Our vision</h3><p>To be Pakistan's most trusted partner for audit, tax and digital compliance — known for integrity, insight and innovation.</p></div>
    </div>
  </div>
</section>

<section class="section">
  <div class="container">
    <div class="section-head section-head--center">
      <p class="eyebrow">Our values</p>
      <h2>What we <em>stand</em> for.</h2>
    </div>
    <div class="values">
      <div class="value reveal"><span class="icon-tile"><?= icon('shield-check') ?></span><h3>Integrity</h3><p>We give honest advice, even when it is not what someone wants to hear, and we never cut corners with the law.</p></div>
      <div class="value reveal"><span class="icon-tile"><?= icon('eye') ?></span><h3>Clarity</h3><p>Plain language, clear fees and written advice you can rely on. No jargon, no surprises.</p></div>
      <div class="value reveal"><span class="icon-tile"><?= icon('award') ?></span><h3>Excellence</h3><p>Every deliverable is prepared carefully and reviewed by a second senior member of the team.</p></div>
      <div class="value reveal"><span class="icon-tile"><?= icon('handshake') ?></span><h3>Partnership</h3><p>We measure success by our clients' success, and we build relationships for the long term.</p></div>
      <div class="value reveal"><span class="icon-tile"><?= icon('lightbulb') ?></span><h3>Innovation</h3><p>We build tools that remove manual work and errors — and use them to serve you better.</p></div>
      <div class="value reveal"><span class="icon-tile"><?= icon('lock') ?></span><h3>Confidentiality</h3><p>Your information is handled with discretion, strict access controls and care.</p></div>
    </div>
  </div>
</section>

<?php if ($team): ?>
<section class="section section--tint" aria-labelledby="team-title">
  <div class="container">
    <div class="section-head">
      <p class="eyebrow">Leadership</p>
      <h2 id="team-title">The people behind <em>Veridian</em>.</h2>
    </div>
    <div class="team">
      <?php foreach ($team as $p): ?>
      <div class="person reveal">
        <?php if (!empty($p['photo']) && is_file(VP_ROOT . '/assets/img/team/' . $p['photo'])): ?>
        <img class="person__photo" src="<?= e(asset('img/team/' . $p['photo'])) ?>" alt="<?= e($p['name']) ?>" width="96" height="96" loading="lazy">
        <?php else: ?>
        <span class="person__initials" aria-hidden="true"><?= e(initials($p['name'])) ?></span>
        <?php endif; ?>
        <h3><?= e($p['name']) ?><?= !empty($p['creds']) ? ' <small class="muted" style="font-weight:600;font-size:.85rem">' . e($p['creds']) . '</small>' : '' ?></h3>
        <span class="person__role"><?= e($p['role']) ?></span>
        <?php if (!empty($p['bio'])): ?><p><?= e($p['bio']) ?></p><?php endif; ?>
        <?php if (!empty($p['linkedin'])): ?><a class="link-arrow" href="<?= e($p['linkedin']) ?>" target="_blank" rel="noopener"><?= icon('linkedin') ?> LinkedIn</a><?php endif; ?>
      </div>
      <?php endforeach; ?>
    </div>
  </div>
</section>
<?php endif; ?>

<section class="section" aria-labelledby="diff-title">
  <div class="container">
    <div class="section-head">
      <p class="eyebrow">How we are different</p>
      <h2 id="diff-title">The best of both <em>worlds</em>.</h2>
      <p>The breadth and rigour clients expect from a large firm, with the attention and speed of a dedicated partner.</p>
    </div>
    <div class="table-wrap">
      <table class="table compare">
        <thead><tr><th></th><th class="is-us">Veridian Partners</th><th>Typical large firm</th><th>Typical sole practitioner</th></tr></thead>
        <tbody>
          <?php
          $rows = [
              ['Senior adviser on every engagement', 'yes', 'partial', 'yes'],
              ['Tax, corporate, audit and advisory under one roof', 'yes', 'yes', 'partial'],
              ['Our own FBR-integrated software', 'yes', 'partial', 'no'],
              ['Fixed, transparent fees agreed upfront', 'yes', 'partial', 'partial'],
              ['Reply within one business day', 'yes', 'partial', 'partial'],
              ['WhatsApp access for quick questions', 'yes', 'no', 'yes'],
          ];
          $iconFor = ['yes' => ['check-circle', 'yes', 'Yes'], 'partial' => ['minus-circle', 'partial', 'Sometimes'], 'no' => ['x', 'no', 'Rarely']];
          foreach ($rows as $r): ?>
          <tr>
            <td><?= e($r[0]) ?></td>
            <?php foreach ([1, 2, 3] as $i): [$ic, $cls, $label] = $iconFor[$r[$i]]; ?>
            <td class="<?= $i === 1 ? 'is-us ' : '' ?><?= $cls ?>"><?= $r[$i] === 'partial' ? '<span class="small" style="font-weight:700">Sometimes</span>' : icon($ic === 'minus-circle' ? 'info' : $ic, '', $label) ?></td>
            <?php endforeach; ?>
          </tr>
          <?php endforeach; ?>
        </tbody>
      </table>
    </div>
  </div>
</section>

<section class="section section--tint">
  <div class="container split" style="align-items:start">
    <div>
      <p class="eyebrow">Our commitments</p>
      <h2>Professional standards you can <em>count on</em>.</h2>
      <ul class="checks checks--light">
        <li><?= icon('check') ?><span><strong>Independence and ethics.</strong> We disclose conflicts of interest and decline work that would compromise our objectivity.</span></li>
        <li><?= icon('check') ?><span><strong>Confidentiality.</strong> Client information is accessed only by the people working on it, and never shared without consent or legal requirement.</span></li>
        <li><?= icon('check') ?><span><strong>Quality review.</strong> Filings and advice are reviewed before they leave our office.</span></li>
        <li><?= icon('check') ?><span><strong>Continuous learning.</strong> We track every Finance Act, SRO and circular, so our advice reflects the law as it is today.</span></li>
      </ul>
    </div>
    <div class="side-card">
      <p class="side-card__heading">Company facts</p>
      <div class="table-wrap" style="border:0">
        <table class="table">
          <tbody>
            <?php foreach ($facts as $k => $v): ?><tr><td style="width:40%"><?= e($k) ?></td><td style="font-weight:500;color:var(--ink-2)"><?= e($v) ?></td></tr><?php endforeach; ?>
          </tbody>
        </table>
      </div>
      <div class="hero__actions" style="margin:1.5rem 0 0">
        <a class="btn btn--primary" href="<?= e(url('careers')) ?>"><span>Join our team</span><?= icon('arrow-right') ?></a>
        <a class="btn btn--ghost" href="<?= e(url('contact')) ?>"><span>Contact us</span></a>
      </div>
    </div>
  </div>
</section>

<?= cta_band() ?>
<?php require __DIR__ . '/includes/footer.php'; ?>
