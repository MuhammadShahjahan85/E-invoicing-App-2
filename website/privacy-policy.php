<?php
require __DIR__ . '/includes/bootstrap.php';
require __DIR__ . '/includes/legal.php';

$email = e(contact_email());
$name = e(legal_name());
legal_page('privacy-policy', 'Privacy Policy', 'How we collect, use and protect personal information on this website and in our work.', <<<HTML
<p>{$name} ("Veridian Partners", "we", "us") respects your privacy. This policy explains what personal information we collect through this website, why we collect it, how we protect it and the choices you have.</p>

<h2>Information we collect</h2>
<ul>
  <li><strong>Information you give us</strong> — when you send an enquiry, book a consultation, subscribe to our newsletter or apply for a job: your name, e-mail address, phone number, company, city, the services you are interested in, your message and, for job applications, your CV and qualifications.</li>
  <li><strong>Technical information</strong> — like most websites, our hosting provider records basic server logs (such as IP address, browser type and pages requested) to keep the site secure and working.</li>
  <li><strong>Analytics</strong> — only if you accept analytics cookies, we use Google Analytics to understand how the site is used, with IP anonymisation.</li>
</ul>
<p>Please do not send sensitive documents (such as tax returns or bank statements) through the website forms. We will arrange a secure way to receive them.</p>

<h2>How we use it</h2>
<ul>
  <li>to reply to your enquiry and provide the consultation or services you ask for;</li>
  <li>to send the newsletter you subscribed to;</li>
  <li>to consider your job application;</li>
  <li>to keep the website secure and improve it;</li>
  <li>to meet our legal and professional obligations.</li>
</ul>
<p>We rely on your consent (which you can withdraw at any time), on taking steps at your request before entering into an engagement, on our legitimate interest in running and improving our business, and on legal obligations.</p>

<h2>Sharing</h2>
<p>We do not sell or rent your personal information. We share it only with service providers who help us run our business — for example our website host and e-mail provider — under confidentiality obligations, or where the law requires it.</p>

<h2>How long we keep it</h2>
<ul>
  <li>Enquiries and consultation requests: up to three years after our last contact, unless you become a client.</li>
  <li>Client records: as long as professional and legal record-keeping rules require.</li>
  <li>Job applications: up to twelve months, unless you ask us to delete them sooner.</li>
  <li>Newsletter: until you unsubscribe.</li>
</ul>

<h2>Security</h2>
<p>We use reasonable technical and organisational measures to protect personal information, including encrypted connections (HTTPS), restricted access and storage that is not publicly reachable. No method of transmission over the internet is completely secure, but we work hard to protect your information.</p>

<h2>Your choices and rights</h2>
<p>You can ask us to tell you what information we hold about you, correct it, delete it or stop using it for marketing. Every newsletter includes an unsubscribe option. To make a request, e-mail <a href="mailto:{$email}">{$email}</a>.</p>

<h2>Cookies</h2>
<p>See our <a href="/cookie-policy">cookie policy</a> for details of the small amount of information stored in your browser.</p>

<h2>Changes</h2>
<p>We may update this policy from time to time. The date at the top shows when it was last changed.</p>

<h2>Contact</h2>
<p>Questions about privacy? E-mail <a href="mailto:{$email}">{$email}</a>.</p>
HTML);
