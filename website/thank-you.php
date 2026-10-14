<?php
/** Shown after a form is sent without JavaScript. */
require __DIR__ . '/includes/bootstrap.php';

$type = preg_replace('/[^a-z]/', '', (string) ($_GET['type'] ?? 'contact'));
$messages = [
    'contact' => ['Thank you — message received.', 'A senior adviser will reply within one business day.'],
    'consultation' => ['Your consultation request is in.', 'We will confirm the time within one business day.'],
    'newsletter' => ['You are subscribed.', 'The Veridian Brief will arrive once a month. You can unsubscribe at any time.'],
    'career' => ['Application received.', 'Thank you for your interest in Veridian. We review every application and will be in touch if there is a suitable role.'],
    'error' => ['We could not send your message.', 'Please go back and check the form, or e-mail us at ' . contact_email() . '.'],
];
[$title, $text] = $messages[$type] ?? $messages['contact'];
$page = ['title' => $title, 'noindex' => true, 'path' => 'thank-you'];
require __DIR__ . '/includes/header.php';
?>
<section class="page-hero" style="padding-block:clamp(64px,10vw,140px)">
  <div class="page-hero__bg" aria-hidden="true"><span class="orb orb--1"></span><span class="orb orb--2"></span></div>
  <div class="container notfound">
    <div class="form-success__icon" style="background:rgba(108,198,172,.15);color:var(--green-200)"><?= icon($type === 'error' ? 'alert' : 'check') ?></div>
    <h1 class="page-hero__title"><?= e($title) ?></h1>
    <p class="page-hero__lede" style="margin-inline:auto"><?= e($text) ?></p>
    <div class="page-hero__actions" style="justify-content:center">
      <a class="btn btn--gold" href="<?= e(url()) ?>"><span>Back to home</span></a>
      <a class="btn btn--ghost-light" href="<?= e(url('knowledge-bank')) ?>"><span>Browse the Knowledge Bank</span></a>
    </div>
  </div>
</section>
<?php require __DIR__ . '/includes/footer.php'; ?>
