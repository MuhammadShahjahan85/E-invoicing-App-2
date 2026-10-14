<?php
/** One-click newsletter unsubscribe (link in every newsletter e-mail). */
require __DIR__ . '/includes/bootstrap.php';
require __DIR__ . '/includes/store.php';

$email = trim((string) ($_GET['e'] ?? ''));
$token = (string) ($_GET['t'] ?? '');
$valid = filter_var($email, FILTER_VALIDATE_EMAIL) && hash_equals(unsubscribe_token($email), $token);
$done = false;
if ($valid && ($_SERVER['REQUEST_METHOD'] ?? 'GET') === 'POST') {
    store_update('subscribers', function (array $rows) use ($email) {
        foreach ($rows as &$r) {
            if (strcasecmp($r['email'], $email) === 0) {
                $r['status'] = 'unsubscribed';
                $r['updated'] = date('Y-m-d H:i:s');
            }
        }
        return $rows;
    });
    $done = true;
}
$page = ['title' => 'Unsubscribe', 'noindex' => true, 'path' => 'unsubscribe'];
require __DIR__ . '/includes/header.php';
?>
<section class="page-hero" style="padding-block:clamp(64px,10vw,140px)">
  <div class="page-hero__bg" aria-hidden="true"><span class="orb orb--1"></span></div>
  <div class="container notfound">
    <?php if (!$valid): ?>
      <h1 class="page-hero__title">This link is not valid.</h1>
      <p class="page-hero__lede" style="margin-inline:auto">Please use the link from your most recent e-mail, or write to <?= e(contact_email()) ?> and we will unsubscribe you.</p>
    <?php elseif ($done): ?>
      <h1 class="page-hero__title">You have been unsubscribed.</h1>
      <p class="page-hero__lede" style="margin-inline:auto"><?= e($email) ?> will no longer receive The Veridian Brief. You can subscribe again at any time.</p>
    <?php else: ?>
      <h1 class="page-hero__title">Unsubscribe from The Veridian Brief?</h1>
      <p class="page-hero__lede" style="margin-inline:auto"><?= e($email) ?></p>
      <form method="post" class="page-hero__actions" style="justify-content:center">
        <button class="btn btn--gold" type="submit">Yes, unsubscribe me</button>
        <a class="btn btn--ghost-light" href="<?= e(url()) ?>">Keep me subscribed</a>
      </form>
    <?php endif; ?>
  </div>
</section>
<?php require __DIR__ . '/includes/footer.php'; ?>
