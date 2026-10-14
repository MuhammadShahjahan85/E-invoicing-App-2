<?php
require __DIR__ . '/includes/bootstrap.php';
require __DIR__ . '/includes/legal.php';

$email = e(contact_email());
$name = e(legal_name());
legal_page('terms', 'Terms of Use', 'The terms that apply when you use this website.', <<<HTML
<p>This website is operated by {$name}. By using it you agree to these terms. If you do not agree, please do not use the site.</p>

<h2>Information, not advice</h2>
<p>Content on this website — including articles, guides, calculators, calendars and FAQs — is general information. It is not professional advice for your circumstances, and it does not create a client relationship. Tax and regulatory rules change frequently; please obtain advice before acting. See our <a href="/disclaimer">disclaimer</a>.</p>

<h2>Engagements</h2>
<p>Professional services are provided only under a written engagement letter or agreement, which sets out the scope, fees and terms of that work. Use of Veridian E-invoicing Pakistan is governed by its own licence agreement.</p>

<h2>Intellectual property</h2>
<p>The website, its design, text, graphics, logos and the Veridian name and marks belong to {$name} or its licensors. You may view, print and share pages for your own non-commercial use, with attribution. You may not copy, republish or sell content without our written permission. Veridian E-invoicing Pakistan is proprietary software.</p>

<h2>Acceptable use</h2>
<p>You agree not to misuse the website — for example by attempting to gain unauthorised access, introducing malicious code, sending spam through our forms or scraping content in bulk.</p>

<h2>Links to other websites</h2>
<p>We link to official and third-party websites for convenience. We do not control them and are not responsible for their content or availability.</p>

<h2>Liability</h2>
<p>We take care to keep the website accurate and available, but we do not guarantee that it is error-free, complete or uninterrupted. To the extent permitted by law, we are not liable for any loss arising from reliance on website content or from use of the website.</p>

<h2>Governing law</h2>
<p>These terms are governed by the laws of the Islamic Republic of Pakistan, and the courts of Pakistan have exclusive jurisdiction.</p>

<h2>Contact</h2>
<p>Questions about these terms? E-mail <a href="mailto:{$email}">{$email}</a>.</p>
HTML);
