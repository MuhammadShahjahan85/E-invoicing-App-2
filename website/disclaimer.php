<?php
require __DIR__ . '/includes/bootstrap.php';
require __DIR__ . '/includes/legal.php';

legal_page('disclaimer', 'Disclaimer', 'Important information about the guides, calculators and other content on this website.', <<<HTML
<h2>General information only</h2>
<p>The articles, guides, glossary, FAQs, tax calendar and calculators on this website are provided for general information. They summarise the law and practice as we understand them on the date shown, and they are not a substitute for professional advice on your specific circumstances.</p>

<h2>The law changes</h2>
<p>Pakistani tax and corporate law changes often — through Finance Acts, SROs, circulars, general orders and court decisions. FBR and SECP also extend deadlines by notification. Always confirm the current position, or speak to us, before you act.</p>

<h2>Calculators</h2>
<p>Our calculators give estimates based on the rates and assumptions stated with them. Your actual liability can differ because of other income, exemptions, credits, withholding and changes in the law. They are for guidance only.</p>

<h2>Professional services</h2>
<p>Statutory audits under the Companies Act, 2017 are performed by audit firms registered with the Institute of Chartered Accountants of Pakistan. Our assurance services are internal audit, controls, readiness and advisory engagements. Any professional service we provide is governed by a written engagement agreement.</p>

<h2>Software</h2>
<p>Veridian E-invoicing Pakistan implements FBR's Digital Invoicing requirements as we understand them at the time of each release. Each business remains responsible for its own tax affairs and for the accuracy of the data it enters.</p>

<h2>No liability</h2>
<p>To the extent permitted by law, we accept no responsibility for any loss arising from reliance on the information on this website.</p>
HTML);
