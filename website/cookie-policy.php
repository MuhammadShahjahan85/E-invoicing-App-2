<?php
require __DIR__ . '/includes/bootstrap.php';
require __DIR__ . '/includes/legal.php';

$ga = cfg('site.google_analytics_id') ? '<p>This site currently uses Google Analytics. You can change your choice at any time:</p><p><button class="btn btn--ghost btn--sm" type="button" data-cookie-reset>Change my cookie choice</button></p>' : '<p>Analytics is not currently enabled on this website.</p>';
legal_page('cookie-policy', 'Cookie Policy', 'The small amount of information this website stores in your browser, and your choices.', <<<HTML
<p>We keep tracking to a minimum. This page explains what is stored in your browser when you use our website.</p>

<h2>Essential and preference storage</h2>
<ul>
  <li><strong>Theme preference</strong> (<code>vp-theme</code>) — remembers whether you chose the light or dark theme. Stored in your browser only.</li>
  <li><strong>Cookie choice</strong> (<code>vp-consent</code>) — remembers whether you accepted or declined analytics.</li>
</ul>
<p>These are stored using your browser's local storage, are never sent to us, and do not identify you.</p>

<h2>Fonts</h2>
<p>The website's typefaces are served by Google Fonts. When a page loads, your browser requests the fonts from Google, which receives your IP address as part of that request.</p>

<h2>Analytics (optional)</h2>
<p>If analytics is enabled and you accept it, Google Analytics sets cookies (such as <code>_ga</code>) to help us understand how visitors use the site — which pages are read and how people find us. IP addresses are anonymised. If you decline, analytics is not loaded.</p>
{$ga}

<h2>Managing cookies</h2>
<p>You can also delete cookies and local storage through your browser settings. The website works fully without them.</p>
HTML);
