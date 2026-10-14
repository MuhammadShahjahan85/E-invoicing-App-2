<?php
if (!defined('VP_ROOT')) {
    require __DIR__ . '/includes/bootstrap.php';
}
http_response_code(404);
$page = [
    'title' => 'Page not found',
    'description' => 'The page you were looking for could not be found.',
    'noindex' => true,
    'path' => '404',
];
require VP_ROOT . '/includes/header.php';
?>
<section class="page-hero" style="padding-block:clamp(64px,10vw,140px)">
  <div class="page-hero__bg" aria-hidden="true"><span class="orb orb--1"></span><span class="orb orb--2"></span><span class="gridlines"></span></div>
  <div class="container notfound">
    <p class="notfound__code">404</p>
    <h1 class="page-hero__title">This page has gone <em>off the books</em>.</h1>
    <p class="page-hero__lede" style="margin-inline:auto">The page you were looking for has moved or no longer exists. Try a search, or start from one of these.</p>
    <form action="<?= e(url('search')) ?>" method="get" role="search" style="max-width:520px;margin:2rem auto 0">
      <label class="search-field" style="max-width:none"><span class="sr-only">Search</span><?= icon('search') ?><input type="search" name="q" placeholder="Search the site…"></label>
    </form>
    <div class="page-hero__actions" style="justify-content:center">
      <a class="btn btn--gold" href="<?= e(url()) ?>"><?= icon('home') ?><span>Home</span></a>
      <a class="btn btn--ghost-light" href="<?= e(url('services')) ?>"><span>Services</span></a>
      <a class="btn btn--ghost-light" href="<?= e(url('knowledge-bank')) ?>"><span>Knowledge Bank</span></a>
      <a class="btn btn--ghost-light" href="<?= e(url('contact')) ?>"><span>Contact</span></a>
    </div>
  </div>
</section>
<?php require VP_ROOT . '/includes/footer.php'; ?>
