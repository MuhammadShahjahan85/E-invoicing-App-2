<?php
/**
 * Website admin area: enquiries, consultation requests, subscribers and job
 * applications; CSV export; setup checklist and a test e-mail.
 * Password: config.php → 'admin' → 'password'.
 */
require dirname(__DIR__) . '/includes/bootstrap.php';
require VP_ROOT . '/includes/store.php';
require VP_ROOT . '/includes/mailer.php';

header('X-Robots-Tag: noindex, nofollow');
header('Cache-Control: no-store');
header('X-Frame-Options: DENY');
header('Referrer-Policy: no-referrer');

session_name('vp_admin');
session_set_cookie_params(['lifetime' => 0, 'path' => base_path() . '/admin/', 'secure' => is_https(), 'httponly' => true, 'samesite' => 'Strict']);
session_start();

$password = (string) cfg('admin.password', '');
$locked = mb_strlen($password) < 8;
$authed = !$locked && !empty($_SESSION['vp_admin']) && ($_SESSION['vp_admin_exp'] ?? 0) > time();
if ($authed) {
    $_SESSION['vp_admin_exp'] = time() + 7200;
}
if (empty($_SESSION['csrf'])) {
    $_SESSION['csrf'] = bin2hex(random_bytes(16));
}
$csrf = $_SESSION['csrf'];
$flash = $_SESSION['flash'] ?? '';
unset($_SESSION['flash']);

function admin_url(array $q = []): string
{
    return url('admin/') . ($q ? '?' . http_build_query($q) : '');
}

function check_csrf(): void
{
    $sent = (string) ($_POST['csrf'] ?? $_GET['csrf'] ?? '');
    if (!hash_equals((string) ($_SESSION['csrf'] ?? ''), $sent)) {
        http_response_code(400);
        exit('This request has expired. Please go back, refresh the page and try again.');
    }
}

function go(array $q = [], string $flash = ''): void
{
    if ($flash !== '') {
        $_SESSION['flash'] = $flash;
    }
    header('Location: ' . admin_url($q), true, 303);
    exit;
}

$stores = ['enquiries' => 'Enquiries', 'subscribers' => 'Subscribers', 'applications' => 'Applications'];
$method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
$action = (string) ($_POST['action'] ?? '');

// ---- Login / logout ----------------------------------------------------------------
if (!$locked && !$authed && $method === 'POST' && $action === 'login') {
    if (rate_limited('admin-login|' . client_ip(), 5, 900)) {
        $flash = 'Too many attempts. Please wait 15 minutes and try again.';
    } elseif (hash_equals($password, (string) ($_POST['password'] ?? ''))) {
        session_regenerate_id(true);
        $_SESSION['vp_admin'] = true;
        $_SESSION['vp_admin_exp'] = time() + 7200;
        $_SESSION['csrf'] = bin2hex(random_bytes(16));
        go();
    } else {
        log_line('admin-log', 'Failed login from ' . client_ip());
        $flash = 'Incorrect password.';
    }
}
if ($authed && $method === 'POST' && $action === 'logout') {
    check_csrf();
    $_SESSION = [];
    session_destroy();
    header('Location: ' . admin_url(), true, 303);
    exit;
}

// ---- Actions ---------------------------------------------------------------------------
if ($authed && $method === 'POST' && in_array($action, ['status', 'delete'], true)) {
    check_csrf();
    $store = (string) ($_POST['store'] ?? '');
    $id = (string) ($_POST['id'] ?? '');
    if (isset($stores[$store]) && $id !== '') {
        store_update($store, function (array $rows) use ($action, $id, $store) {
            foreach ($rows as $i => $r) {
                if (($r['id'] ?? '') !== $id) {
                    continue;
                }
                if ($action === 'delete') {
                    if (!empty($r['cv']['file'])) {
                        @unlink(storage_dir('cv') . '/' . basename($r['cv']['file']));
                    }
                    unset($rows[$i]);
                } elseif ($store === 'subscribers') {
                    $rows[$i]['status'] = ($r['status'] ?? '') === 'subscribed' ? 'unsubscribed' : 'subscribed';
                } else {
                    $rows[$i]['status'] = ($r['status'] ?? 'new') === 'new' ? 'done' : 'new';
                }
            }
            return $rows;
        });
    }
    go(['tab' => $store], $action === 'delete' ? 'Deleted.' : 'Updated.');
}
if ($authed && $method === 'POST' && $action === 'test-mail') {
    check_csrf();
    $to = array_map('trim', explode(',', (string) cfg('forms.notify_to', '')));
    $ok = send_mail($to, 'Test e-mail from your website', email_layout('Your website can send e-mail', '<p>This test was sent from the admin area of ' . e(abs_url()) . ' on ' . e(date('j F Y, g:i a')) . '.</p><p>Method: ' . (cfg('smtp.enabled') ? 'SMTP (' . e(cfg('smtp.host')) . ')' : 'PHP mail()') . '</p>'));
    go(['tab' => 'setup'], $ok ? 'Test e-mail sent to ' . implode(', ', $to) . '. Check the inbox (and the spam folder).' : 'The test e-mail could not be sent. See the mail log below, and check the SMTP settings in config.php.');
}

// ---- Downloads ---------------------------------------------------------------------------
if ($authed && isset($_GET['export'])) {
    check_csrf();
    $store = (string) $_GET['export'];
    if (!isset($stores[$store])) {
        go();
    }
    $rows = store_all($store);
    $cols = [];
    foreach ($rows as $r) {
        foreach ($r as $k => $v) {
            if ($k !== 'cv') {
                $cols[$k] = true;
            }
        }
    }
    $cols = array_keys($cols);
    header('Content-Type: text/csv; charset=utf-8');
    header('Content-Disposition: attachment; filename="' . $store . '-' . date('Y-m-d') . '.csv"');
    $out = fopen('php://output', 'w');
    fwrite($out, "\xEF\xBB\xBF");
    fputcsv($out, $cols, ',', '"', '');
    foreach ($rows as $r) {
        $line = [];
        foreach ($cols as $c) {
            $v = is_array($r[$c] ?? '') ? json_encode($r[$c]) : (string) ($r[$c] ?? '');
            // Neutralise spreadsheet formulas (a leading tab keeps e.g. +92 phone numbers as text).
            $line[] = preg_match('/^[=+\-@\t\r]/', $v) ? "\t" . $v : $v;
        }
        fputcsv($out, $line, ',', '"', '');
    }
    fclose($out);
    exit;
}
if ($authed && isset($_GET['cv'])) {
    check_csrf();
    $id = (string) $_GET['cv'];
    foreach (store_all('applications') as $r) {
        if (($r['id'] ?? '') === $id && !empty($r['cv']['file'])) {
            $path = storage_dir('cv') . '/' . basename($r['cv']['file']);
            if (is_file($path)) {
                header('Content-Type: application/octet-stream');
                header('Content-Disposition: attachment; filename="' . preg_replace('/[^A-Za-z0-9._ -]/', '_', $r['cv']['name']) . '"');
                header('Content-Length: ' . filesize($path));
                readfile($path);
                exit;
            }
        }
    }
    go(['tab' => 'applications'], 'File not found.');
}

$tab = (string) ($_GET['tab'] ?? 'enquiries');
if (!isset($stores[$tab]) && $tab !== 'setup') {
    $tab = 'enquiries';
}
$data = [];
foreach (array_keys($stores) as $s) {
    $data[$s] = $authed ? store_all($s) : [];
}
$newCount = count(array_filter($data['enquiries'], fn($r) => ($r['status'] ?? '') === 'new'));
$subCount = count(array_filter($data['subscribers'], fn($r) => ($r['status'] ?? '') === 'subscribed'));
$appNew = count(array_filter($data['applications'], fn($r) => ($r['status'] ?? '') === 'new'));
?><!doctype html>
<html lang="en" class="no-js">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex, nofollow">
<title>Admin — <?= e(company_name()) ?></title>
<link rel="icon" href="<?= e(asset('img/favicon.svg')) ?>" type="image/svg+xml">
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Fraunces:opsz,wght@9..144,400..600&amp;family=Manrope:wght@400..800&amp;display=swap">
<link rel="stylesheet" href="<?= e(asset('css/style.css')) ?>">
<script>(function(){var d=document.documentElement;d.classList.remove('no-js');d.classList.add('js');try{var t=localStorage.getItem('vp-theme');if(t)d.setAttribute('data-theme',t);}catch(e){}})();</script>
</head>
<body class="admin-body" data-base="<?= e(base_path()) ?>">
<?= icon_sprite() ?>
<div class="admin-wrap">
  <div class="admin-top">
    <a class="brand" href="<?= e(admin_url()) ?>"><?= brand_mark('brand__mark', 'adm') ?><span class="brand__text"><span class="brand__name">Veridian</span><span class="brand__sub">Admin</span></span></a>
    <?php if ($authed): ?>
    <div class="admin-actions">
      <a class="btn btn--ghost btn--sm" href="<?= e(url()) ?>" target="_blank" rel="noopener"><?= icon('external-link') ?><span>View website</span></a>
      <form method="post"><input type="hidden" name="action" value="logout"><input type="hidden" name="csrf" value="<?= e($csrf) ?>"><button class="btn btn--primary btn--sm" type="submit">Log out</button></form>
    </div>
    <?php endif; ?>
  </div>

<?php if ($locked): ?>
  <div class="panel login-box">
    <h1 style="font-size:1.8rem">Admin area is locked</h1>
    <p>To use the admin area, set a password of at least 12 characters in <code>config.php</code>:</p>
    <pre style="white-space:pre-wrap;background:var(--surface-2);padding:1rem;border-radius:12px;border:1px solid var(--line)">'admin' => [
    'password' => 'your-strong-password',
],</pre>
    <p class="muted small">In Hostinger: hPanel → Files → File Manager → public_html → config.php → Edit. Save, then reload this page.</p>
  </div>

<?php elseif (!$authed): ?>
  <div class="panel login-box">
    <h1 style="font-size:1.8rem">Sign in</h1>
    <p class="muted">Website admin for <?= e(company_name()) ?>.</p>
    <?php if ($flash): ?><div class="notice"><?= e($flash) ?></div><?php endif; ?>
    <form method="post" class="form">
      <input type="hidden" name="action" value="login">
      <div class="field" style="margin-bottom:1rem"><label for="pw">Password</label><input id="pw" type="password" name="password" autocomplete="current-password" required autofocus></div>
      <button class="btn btn--primary btn--block" type="submit">Sign in</button>
    </form>
  </div>

<?php else: ?>
  <?php if ($flash): ?><div class="notice notice--ok"><?= e($flash) ?></div><?php endif; ?>
  <?php if (mb_strlen($password) < 12): ?><div class="notice">Your admin password is short. Please use 12 or more characters in config.php.</div><?php endif; ?>

  <div class="result-grid" style="grid-template-columns:repeat(4,minmax(0,1fr));margin-bottom:1.5rem">
    <div class="result-cell"><span>New enquiries</span><strong><?= $newCount ?></strong></div>
    <div class="result-cell"><span>All enquiries</span><strong><?= count($data['enquiries']) ?></strong></div>
    <div class="result-cell"><span>Active subscribers</span><strong><?= $subCount ?></strong></div>
    <div class="result-cell"><span>New applications</span><strong><?= $appNew ?></strong></div>
  </div>

  <nav class="admin-tabs" aria-label="Sections">
    <?php foreach ($stores as $k => $label): ?>
    <a href="<?= e(admin_url(['tab' => $k])) ?>"<?= $tab === $k ? ' class="is-active"' : '' ?>><?= e($label) ?><span class="count"><?= count($data[$k]) ?></span></a>
    <?php endforeach; ?>
    <a href="<?= e(admin_url(['tab' => 'setup'])) ?>"<?= $tab === 'setup' ? ' class="is-active"' : '' ?>>Setup & e-mail</a>
  </nav>

  <?php if ($tab === 'setup'): ?>
  <?php
    $checks = [
        ['Public e-mail address', contact_email(), true],
        ['Phone number', cfg('contact.phone') ?: cfg('contact.mobile') ?: 'Not set — add it in config.php', (bool) (cfg('contact.phone') ?: cfg('contact.mobile'))],
        ['WhatsApp number', cfg('contact.whatsapp') ?: 'Not set — the WhatsApp button is hidden', (bool) cfg('contact.whatsapp')],
        ['Office address', address_lines() ? implode(', ', address_lines()) : 'Not set — the site says "Nationwide service"', (bool) address_lines()],
        ['Social media links', social_links() ? implode(', ', array_column(social_links(), 'label')) : 'None set', (bool) social_links()],
        ['Website address', cfg('site.url') ?: site_origin() . ' (detected automatically)', true],
        ['HTTPS', is_https() ? 'Secure connection' : 'Not secure — turn on SSL and Force HTTPS in hPanel', is_https()],
        ['Enquiries e-mailed to', (string) cfg('forms.notify_to'), (bool) cfg('forms.notify_to')],
        ['E-mail method', cfg('smtp.enabled') ? 'SMTP via ' . cfg('smtp.host') : 'PHP mail() — SMTP is recommended for reliable delivery', (bool) cfg('smtp.enabled')],
        ['Google Analytics', cfg('site.google_analytics_id') ?: 'Not set (optional)', (bool) cfg('site.google_analytics_id')],
        ['Google Search Console', cfg('site.google_site_verification') ? 'Verification tag added' : 'Not set — recommended for SEO', (bool) cfg('site.google_site_verification')],
        ['Storage folder writable', is_writable(storage_dir()) ? 'Yes' : 'No — set folder permissions of /storage to 755', is_writable(storage_dir())],
        ['PHP version', PHP_VERSION, version_compare(PHP_VERSION, '7.4', '>=')],
    ];
    $logFile = storage_dir() . '/mail-log.php';
    $log = is_file($logFile) ? array_slice(array_filter(explode("\n", (string) file_get_contents($logFile)), fn($l) => $l !== '' && strpos($l, '<?php') !== 0), -20) : [];
  ?>
  <div class="split" style="align-items:start;gap:1.5rem">
    <div class="admin-card">
      <h2 style="font-size:1.4rem">Setup checklist</h2>
      <div class="table-wrap" style="border:0"><table class="table admin-table"><tbody>
        <?php foreach ($checks as [$label, $value, $ok]): ?>
        <tr><td style="width:38%"><?= $ok ? '<span class="yes">' . icon('check-circle', '', 'OK') . '</span>' : '<span class="partial">' . icon('alert', '', 'Needs attention') . '</span>' ?> <?= e($label) ?></td><td style="font-weight:500;color:var(--ink-2)"><?= e($value) ?></td></tr>
        <?php endforeach; ?>
      </tbody></table></div>
      <p class="muted small">Change any of these in <code>config.php</code>.</p>
    </div>
    <div class="admin-card">
      <h2 style="font-size:1.4rem">Test e-mail</h2>
      <p>Sends a test message to <strong><?= e((string) cfg('forms.notify_to')) ?></strong>.</p>
      <form method="post"><input type="hidden" name="action" value="test-mail"><input type="hidden" name="csrf" value="<?= e($csrf) ?>"><button class="btn btn--primary" type="submit"><?= icon('send') ?><span>Send test e-mail</span></button></form>
      <h3 class="mt-2" style="font-size:1.05rem">Recent e-mail problems</h3>
      <?php if ($log): ?><pre style="white-space:pre-wrap;font-size:.8rem;background:var(--surface-2);padding:1rem;border-radius:12px;border:1px solid var(--line);max-height:300px;overflow:auto"><?= e(implode("\n", $log)) ?></pre>
      <?php else: ?><p class="muted">None recorded.</p><?php endif; ?>
    </div>
  </div>

  <?php else: $rows = $data[$tab]; ?>
  <div class="admin-card">
    <div class="toolbar" style="margin-bottom:1rem">
      <label class="search-field"><span class="sr-only">Filter</span><?= icon('search') ?><input type="search" placeholder="Filter by name, e-mail, service…" data-admin-filter></label>
      <a class="btn btn--ghost btn--sm" href="<?= e(admin_url(['export' => $tab, 'csrf' => $csrf])) ?>"><?= icon('download') ?><span>Export to Excel (CSV)</span></a>
    </div>
    <?php if (!$rows): ?>
      <div class="empty-state"><?= icon('mail') ?><p>Nothing here yet. New <?= e(strtolower($stores[$tab])) ?> will appear here, newest first.</p></div>
    <?php else: ?>
    <div class="table-wrap">
      <table class="table admin-table">
        <thead><tr>
          <?php if ($tab === 'subscribers'): ?><th>Date</th><th>E-mail</th><th>Status</th><th></th>
          <?php elseif ($tab === 'applications'): ?><th>Received</th><th>Applicant</th><th>Role</th><th>CV</th><th>Message</th><th>Status</th><th></th>
          <?php else: ?><th>Received</th><th>From</th><th>Request</th><th>Message</th><th>Status</th><th></th><?php endif; ?>
        </tr></thead>
        <tbody>
        <?php foreach ($rows as $r): $isNew = in_array($r['status'] ?? '', ['new'], true); ?>
          <tr class="<?= $isNew ? 'row-new' : '' ?>" data-row>
            <td style="white-space:nowrap"><?= e(date('j M Y', strtotime($r['created']))) ?><br><small class="muted"><?= e(date('g:i a', strtotime($r['created']))) ?></small></td>
            <?php if ($tab === 'subscribers'): ?>
              <td><a href="mailto:<?= e($r['email']) ?>"><?= e($r['email']) ?></a></td>
              <td><span class="tag<?= $r['status'] === 'subscribed' ? '' : ' tag--payroll' ?>"><?= e($r['status']) ?></span></td>
            <?php elseif ($tab === 'applications'): ?>
              <td><strong><?= e($r['name']) ?></strong><br><a href="mailto:<?= e($r['email']) ?>"><?= e($r['email']) ?></a><br><a href="<?= e(tel_link($r['phone'])) ?>"><?= e($r['phone']) ?></a><?= $r['city'] ? '<br><small class="muted">' . e($r['city']) . '</small>' : '' ?></td>
              <td><?= e($r['role'] ?? '') ?><br><small class="muted"><?= e($r['qualification'] ?? '') ?></small></td>
              <td><?php if (!empty($r['cv'])): ?><a class="btn-xs" href="<?= e(admin_url(['cv' => $r['id'], 'csrf' => $csrf])) ?>">Download</a><?php endif; ?></td>
              <td class="msg"><?= e($r['message']) ?></td>
              <td><span class="tag<?= $isNew ? '' : ' tag--payroll' ?>"><?= e($r['status']) ?></span></td>
            <?php else: ?>
              <td><strong><?= e($r['name']) ?></strong><?= $r['company'] ? '<br>' . e($r['company']) : '' ?><br><a href="mailto:<?= e($r['email']) ?>"><?= e($r['email']) ?></a><?= $r['phone'] ? '<br><a href="' . e(tel_link($r['phone'])) . '">' . e($r['phone']) . '</a>' : '' ?></td>
              <td><span class="tag"><?= e($r['type'] === 'consultation' ? 'Consultation' : 'Enquiry') ?></span><br><?= e($r['service'] ?? '') ?>
                <?php if ($r['type'] === 'consultation'): ?><br><small class="muted"><?= e(implode(' · ', array_filter([$r['urgency'] ?? '', $r['entity'] ?? '', $r['industry'] ?? '', $r['turnover'] ?? '', $r['city'] ?? '', ($r['preferred_date'] ?? '') ? 'Prefers ' . $r['preferred_date'] : '', trim(($r['mode'] ?? '') . ' ' . ($r['time'] ?? ''))]))) ?></small><?php endif; ?>
              </td>
              <td class="msg"><?= e($r['message']) ?></td>
              <td><span class="tag<?= $isNew ? '' : ' tag--payroll' ?>"><?= e($r['status']) ?></span></td>
            <?php endif; ?>
            <td>
              <div class="admin-actions">
                <?php if ($tab !== 'subscribers'): ?><a class="btn-xs" href="mailto:<?= e($r['email']) ?>?subject=<?= rawurlencode('Re: your enquiry — ' . company_name()) ?>">Reply</a><?php endif; ?>
                <form method="post"><input type="hidden" name="action" value="status"><input type="hidden" name="store" value="<?= e($tab) ?>"><input type="hidden" name="id" value="<?= e($r['id']) ?>"><input type="hidden" name="csrf" value="<?= e($csrf) ?>"><button class="btn-xs" type="submit"><?= $tab === 'subscribers' ? ($r['status'] === 'subscribed' ? 'Unsubscribe' : 'Resubscribe') : ($isNew ? 'Mark done' : 'Mark new') ?></button></form>
                <form method="post" onsubmit="return confirm('Delete this record permanently?')"><input type="hidden" name="action" value="delete"><input type="hidden" name="store" value="<?= e($tab) ?>"><input type="hidden" name="id" value="<?= e($r['id']) ?>"><input type="hidden" name="csrf" value="<?= e($csrf) ?>"><button class="btn-xs btn-xs--danger" type="submit">Delete</button></form>
              </div>
            </td>
          </tr>
        <?php endforeach; ?>
        </tbody>
      </table>
    </div>
    <?php endif; ?>
  </div>
  <?php endif; ?>
<?php endif; ?>
  <p class="muted small center mt-2"><?= e(legal_name()) ?> · Website v<?= e(VP_VERSION) ?></p>
</div>
<script>
(function () {
  var f = document.querySelector('[data-admin-filter]');
  if (!f) return;
  f.addEventListener('input', function () {
    var q = f.value.toLowerCase();
    document.querySelectorAll('[data-row]').forEach(function (r) { r.hidden = q && r.textContent.toLowerCase().indexOf(q) === -1; });
  });
})();
</script>
</body>
</html>
