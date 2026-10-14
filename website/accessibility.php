<?php
require __DIR__ . '/includes/bootstrap.php';
require __DIR__ . '/includes/legal.php';

$email = e(contact_email());
legal_page('accessibility', 'Accessibility', 'Our commitment to a website everyone can use.', <<<HTML
<p>We want everyone to be able to use our website, including people who use assistive technology. We aim to meet the Web Content Accessibility Guidelines (WCAG) 2.2 at level AA.</p>

<h2>What we have done</h2>
<ul>
  <li>A "skip to main content" link and a logical heading structure on every page.</li>
  <li>Full keyboard navigation, including menus, search (<kbd>Ctrl</kbd> + <kbd>K</kbd> or <kbd>/</kbd>), accordions and forms, with visible focus.</li>
  <li>Text and background colours chosen for strong contrast, in both light and dark themes.</li>
  <li>Animations that switch off when your device asks for reduced motion.</li>
  <li>Descriptive labels for form fields, buttons and icons, and clear error messages.</li>
  <li>Pages that adapt to any screen size and zoom level, and print cleanly.</li>
</ul>

<h2>Tell us if something is hard to use</h2>
<p>If you find any part of the website difficult to use, please e-mail <a href="mailto:{$email}">{$email}</a>. We will do our best to fix it and to provide the information you need in another format.</p>
HTML);
