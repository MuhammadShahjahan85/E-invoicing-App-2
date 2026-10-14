<?php
require __DIR__ . '/includes/bootstrap.php';

$crumbs = [['Home', ''], ['Knowledge Bank', 'knowledge-bank'], ['Tax & compliance calendar', 'tax-calendar']];
$items = upcoming_deadlines(12);
$byMonth = [];
foreach ($items as $d) {
    $byMonth[date('F Y', strtotime($d['date']))][] = $d;
}
$next = $items[0] ?? null;
$page = [
    'title' => 'Tax Calendar Pakistan ' . date('Y') . '-' . date('y', strtotime('+1 year')) . ' — FBR & SECP Due Dates',
    'description' => 'Pakistan tax and compliance calendar with live countdowns: sales tax payment and return dates, advance tax instalments, income tax return deadlines, AGM and SECP annual return. Add any date to your calendar.',
    'keywords' => 'tax calendar Pakistan, FBR due dates, sales tax return due date, income tax return last date, advance tax due dates Pakistan, SECP annual return due date, AGM deadline Pakistan',
    'path' => 'tax-calendar',
    'nav' => 'knowledge',
    'schema' => [breadcrumb_schema($crumbs)],
];
require __DIR__ . '/includes/header.php';

$aside = '';
if ($next) {
    $aside = '<div class="glass" style="padding:1.5rem">'
        . '<p class="eyebrow eyebrow--light" style="margin-bottom:.5rem">Next deadline</p>'
        . '<p style="font-family:var(--font-display);font-size:1.6rem;color:#fff;line-height:1.2;margin:0 0 .4rem">' . e($next['title']) . '</p>'
        . '<p style="color:var(--on-deep-2);margin:0 0 1rem">' . e(date('l, j F Y', strtotime($next['date']))) . '</p>'
        . '<span class="pill pill--due" data-due="' . e($next['date']) . '"><span class="countdown" style="background:none;border:0;color:inherit;padding:0">' . e(countdown_label(days_until($next['date']))) . '</span></span>'
        . '</div>';
}
echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => 'Tax & compliance calendar',
    'title' => 'Every recurring deadline, <em>counted down</em>.',
    'lede' => 'The main FBR and SECP dates for the next twelve months. Add any of them to Google Calendar, Outlook or your phone with one click.',
    'aside' => $aside,
]);
?>

<section class="section">
  <div class="container detail">
    <div class="detail__main" data-cal-filter>
      <div class="toolbar">
        <div class="filter-chips" role="group" aria-label="Filter deadlines">
          <button class="filter-chip" type="button" data-filter="all" aria-pressed="true">All</button>
          <button class="filter-chip" type="button" data-filter="sales" aria-pressed="false"><?= icon('receipt') ?>Sales tax</button>
          <button class="filter-chip" type="button" data-filter="income" aria-pressed="false"><?= icon('landmark') ?>Income tax</button>
          <button class="filter-chip" type="button" data-filter="corporate" aria-pressed="false"><?= icon('building') ?>Corporate</button>
        </div>
      </div>
      <?php foreach ($byMonth as $month => $list): ?>
      <div class="detail__block" data-month-group style="margin-top:2.5rem">
        <h2 style="font-size:1.6rem;margin-bottom:1rem"><?= e($month) ?></h2>
        <ul class="cal-list">
          <?php foreach ($list as $d): echo deadline_row($d, true); endforeach; ?>
        </ul>
      </div>
      <?php endforeach; ?>
      <div class="callout callout--warn mt-3"><strong>Please note:</strong> these are the general due dates. FBR and SECP extend dates by notification from time to time, a date falling on a public holiday may move, and your own dates depend on your year end and registrations. Withholding statements, provincial sales tax on services and payroll contributions follow their own schedules. <a href="<?= e(url('contact')) ?>">Ask us for a calendar built for your business.</a></div>
    </div>
    <aside class="detail__aside">
      <?= expert_card() ?>
      <div class="side-card">
        <p class="side-card__heading">Related guides</p>
        <ul class="side-links">
          <?php foreach (['sales-tax-return-annex-c-reconciliation', 'how-to-file-income-tax-return-pakistan', 'companies-act-2017-annual-compliance-checklist'] as $slug): $a = article($slug); if (!$a) continue; ?>
          <li><a href="<?= e(article_url($a)) ?>"><?= e($a['title']) ?><?= icon('chevron-right') ?></a></li>
          <?php endforeach; ?>
        </ul>
      </div>
    </aside>
  </div>
</section>

<?= cta_band('Never miss a deadline again.', 'We run the compliance calendar for our clients — reminders, preparation and filing, every month.') ?>
<?php require __DIR__ . '/includes/footer.php'; ?>
