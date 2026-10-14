<?php
require __DIR__ . '/includes/bootstrap.php';

$crumbs = [['Home', ''], ['Knowledge Bank', 'knowledge-bank'], ['Official portals', 'official-portals']];
$portals = [
    'Federal tax' => [
        ['FBR', 'Federal Board of Revenue — laws, SROs, circulars, the Active Taxpayer List and notices.', 'https://www.fbr.gov.pk', 'landmark'],
        ['IRIS', 'FBR\'s portal for registration, returns, withholding statements and Digital Invoicing registration.', 'https://iris.fbr.gov.pk', 'monitor'],
        ['PRAL', 'FBR\'s technology company and free licensed integrator for Digital Invoicing.', 'https://www.pral.com.pk', 'qr-code'],
    ],
    'Provincial revenue authorities' => [
        ['Punjab Revenue Authority', 'Sales tax on services in Punjab — registration, returns and POS integration.', 'https://pra.punjab.gov.pk', 'receipt'],
        ['Sindh Revenue Board', 'Sales tax on services in Sindh.', 'https://srb.gos.pk', 'receipt'],
        ['KP Revenue Authority', 'Sales tax on services in Khyber Pakhtunkhwa.', 'https://kpra.kp.gov.pk', 'receipt'],
        ['Balochistan Revenue Authority', 'Sales tax on services in Balochistan.', 'https://bra.gob.pk', 'receipt'],
    ],
    'Companies & investment' => [
        ['SECP', 'Securities and Exchange Commission of Pakistan — company registration, online filings and notifications.', 'https://www.secp.gov.pk', 'building'],
        ['Board of Investment', 'Investment facilitation, branch and liaison office permissions.', 'https://invest.gov.pk', 'globe'],
        ['State Bank of Pakistan', 'Foreign exchange regulations, foreign investment and repatriation.', 'https://www.sbp.org.pk', 'landmark'],
        ['Pakistan Stock Exchange', 'Listed company announcements and listing regulations.', 'https://www.psx.com.pk', 'trending-up'],
    ],
    'Sector & other registrations' => [
        ['PSEB', 'Pakistan Software Export Board — registration for IT and IT-enabled services companies.', 'https://www.pseb.org.pk', 'code'],
        ['Pakistan Single Window', 'Import, export and trade registrations and documentation.', 'https://www.psw.gov.pk', 'truck'],
        ['IPO Pakistan', 'Intellectual Property Organisation — trademarks, patents and copyright.', 'https://ipo.gov.pk', 'award'],
        ['EOBI', 'Employees\' Old-Age Benefits Institution — employer registration and contributions.', 'https://www.eobi.gov.pk', 'users'],
        ['ICAP', 'Institute of Chartered Accountants of Pakistan — audit firm and member directories.', 'https://www.icap.org.pk', 'shield-check'],
    ],
];
$page = [
    'title' => 'Official Portals — FBR, IRIS, SECP, PRA, SRB, KPRA, BRA & More',
    'description' => 'Quick links to Pakistan\'s official tax and corporate portals: FBR, IRIS, PRAL, SECP, Punjab Revenue Authority, Sindh Revenue Board, KPRA, BRA, BOI, SBP, PSEB and more.',
    'keywords' => 'FBR website, IRIS login, SECP website, PRA Punjab, SRB Sindh, KPRA, BRA, PSEB registration, Pakistan Single Window',
    'path' => 'official-portals',
    'nav' => 'knowledge',
    'schema' => [breadcrumb_schema($crumbs)],
];
require __DIR__ . '/includes/header.php';

echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => 'Official portals',
    'title' => 'Every official portal, <em>one page</em>.',
    'lede' => 'The government and regulatory websites Pakistani businesses use most. Links open in a new tab.',
]);
?>

<section class="section">
  <div class="container">
    <?php foreach ($portals as $group => $list): ?>
    <div class="detail__block">
      <h2 style="font-size:clamp(1.5rem,2.4vw,2rem)"><?= e($group) ?></h2>
      <div class="portal-grid">
        <?php foreach ($list as [$name, $text, $href, $ic]): ?>
        <a class="portal reveal" href="<?= e($href) ?>" target="_blank" rel="noopener"><span class="icon-tile icon-tile--sm"><?= icon($ic) ?></span><span><strong><?= e($name) ?></strong><small><?= e($text) ?></small><small class="portal__host"><?= e(parse_url($href, PHP_URL_HOST)) ?><?= icon('external-link') ?></small></span></a>
        <?php endforeach; ?>
      </div>
    </div>
    <?php endforeach; ?>
    <p class="callout mt-3"><?= icon('info') ?> These are independent government and regulatory websites; we are not responsible for their content. Need help with something you found there? <a href="<?= e(url('contact')) ?>">Ask us</a>.</p>
  </div>
</section>

<?= cta_band() ?>
<?php require __DIR__ . '/includes/footer.php'; ?>
