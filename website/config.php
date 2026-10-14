<?php
/*
 * ============================================================================
 *  VERIDIAN PARTNERS — WEBSITE SETTINGS
 * ============================================================================
 *  This is the only file you need to edit. Change the values between the
 *  quotes, save, and upload. Leave a value empty ('') to hide it on the site.
 *
 *  Tip: in Hostinger, open hPanel → Files → File Manager → public_html,
 *  right-click config.php → Edit.
 * ============================================================================
 */

return [

    /* ------------------------------------------------------------------------
     * 1. COMPANY
     * --------------------------------------------------------------------- */
    'company' => [
        'legal_name'  => 'Veridian Partners Consultancy Private Limited',
        'name'        => 'Veridian Partners',
        'tagline'     => 'Audit · Tax · Advisory · Digital Compliance',
        'founded'     => '',          // e.g. '2024'  (shown in the footer and schema)
        'secp_cuin'   => '',          // SECP Corporate Unique Identification No. (shown in the footer)
        'ntn'         => '',          // Company NTN (shown in the footer)
    ],

    /* ------------------------------------------------------------------------
     * 2. CONTACT DETAILS (shown on every page)
     *    Empty values are hidden automatically.
     * --------------------------------------------------------------------- */
    'contact' => [
        // Public e-mail. Leave empty to use info@<your domain> automatically.
        // Create that mailbox in hPanel → Emails (included with Hostinger plans).
        'email'          => '',
        'phone'          => '',       // e.g. '+92 42 1234 5678'
        'mobile'         => '',       // e.g. '+92 300 1234567'
        'whatsapp'       => '',       // digits only, with country code, e.g. '923001234567'
        'address_line_1' => '',       // e.g. 'Office 5, 2nd Floor, Business Centre'
        'address_line_2' => '',       // e.g. 'Main Boulevard, Gulberg III'
        'city'           => '',       // e.g. 'Lahore'
        'province'       => '',       // e.g. 'Punjab'
        'postal_code'    => '',       // e.g. '54660'
        'country'        => 'Pakistan',
        'hours'          => 'Monday – Saturday, 9:00 am – 6:00 pm (PKT)',
        // Google Maps: paste the "Embed a map" src URL from Google Maps (Share → Embed a map → copy only the src="..." link)
        'map_embed_url'  => '',
        // Google Maps link to your office (Share → Copy link)
        'map_link'       => '',
        // Geo coordinates of the office (optional, improves local SEO), e.g. 31.5204 and 74.3587
        'latitude'       => '',
        'longitude'      => '',
    ],

    /* ------------------------------------------------------------------------
     * 3. SOCIAL MEDIA (full links; empty ones are hidden)
     * --------------------------------------------------------------------- */
    'social' => [
        'linkedin'  => '',
        'facebook'  => '',
        'instagram' => '',
        'x'         => '',
        'youtube'   => '',
    ],

    /* ------------------------------------------------------------------------
     * 4. ENQUIRY FORMS
     *    Every enquiry is saved (see the admin area at /admin) AND e-mailed.
     * --------------------------------------------------------------------- */
    'forms' => [
        // Where new enquiries, applications and sign-ups are e-mailed (comma-separate several).
        'notify_to'  => 'muhammadshahjahan.audit@gmail.com',
        // The "From" address of website e-mails. Must be a mailbox on YOUR domain.
        // Leave empty to use info@<your domain>.
        'from_email' => '',
        'from_name'  => 'Veridian Partners Website',
        // Send a polite "we received your message" e-mail to the person who wrote in.
        'auto_reply' => true,
    ],

    /* ------------------------------------------------------------------------
     * 5. SMTP (recommended for reliable delivery)
     *    hPanel → Emails → your mailbox → "Connect apps & devices" shows these.
     *    If 'enabled' is false the site uses the server's built-in mail().
     * --------------------------------------------------------------------- */
    'smtp' => [
        'enabled'  => false,
        'host'     => 'smtp.hostinger.com',
        'port'     => 465,            // 465 (SSL) or 587 (TLS)
        'secure'   => 'ssl',          // 'ssl' for 465, 'tls' for 587
        'username' => '',             // full mailbox address, e.g. info@yourdomain.com
        'password' => '',             // mailbox password
    ],

    /* ------------------------------------------------------------------------
     * 6. ADMIN AREA  (yourdomain.com/admin)
     *    Set a strong password (12+ characters). The admin area stays locked
     *    until you do.
     * --------------------------------------------------------------------- */
    'admin' => [
        'password' => '',
    ],

    /* ------------------------------------------------------------------------
     * 7. WEBSITE ADDRESS, SEARCH ENGINES AND ANALYTICS
     * --------------------------------------------------------------------- */
    'site' => [
        // Your full website address, e.g. 'https://www.veridianpartners.pk'
        // Leave empty to detect it automatically.
        'url'                      => '',
        'google_analytics_id'      => '',   // e.g. 'G-XXXXXXXXXX' (loads only after visitors accept cookies)
        'google_site_verification' => '',   // the content="..." value from Google Search Console
        'bing_site_verification'   => '',   // the content="..." value from Bing Webmaster Tools
        'debug'                    => false, // true shows PHP errors (only while fixing a problem)
    ],

    /* ------------------------------------------------------------------------
     * 8. LEADERSHIP (About page). Add, remove or edit people.
     *    'photo' is optional: upload a square image to assets/img/team/ and
     *    put its file name here, e.g. 'shahjahan.jpg'. Without a photo the
     *    person's initials are shown.
     * --------------------------------------------------------------------- */
    'team' => [
        [
            'name'     => 'Muhammad Shahjahan',
            'role'     => 'Founder & Chief Executive',
            'creds'    => '',   // e.g. 'ACCA, CIA'
            'bio'      => 'Leads Veridian\'s audit, tax and digital compliance practice and the firm\'s technology products, including Veridian E-invoicing Pakistan.',
            'photo'    => '',
            'linkedin' => '',
        ],
    ],

    /* ------------------------------------------------------------------------
     * 9. CLIENT TESTIMONIALS
     *    Only add real quotes you have permission to publish. The section is
     *    hidden while this list is empty.
     * --------------------------------------------------------------------- */
    'testimonials' => [
        // ['quote' => 'Their team took us live on FBR e-invoicing in a week.', 'name' => 'Name Surname', 'role' => 'CFO, Company (Pvt) Ltd'],
    ],

    /* ------------------------------------------------------------------------
     * 10. OPEN POSITIONS (Careers page)
     *     Set 'open' => true to publish a role. With no open roles the page
     *     invites general applications.
     * --------------------------------------------------------------------- */
    'jobs' => [
        [
            'open'     => false,
            'title'    => 'Tax Associate',
            'type'     => 'Full-time',
            'location' => 'On-site / Hybrid',
            'summary'  => 'Prepare income tax and sales tax returns, withholding statements and replies to FBR notices for a portfolio of corporate and individual clients.',
        ],
        [
            'open'     => false,
            'title'    => 'Audit & Assurance Trainee (ACCA / CA / CIA)',
            'type'     => 'Traineeship',
            'location' => 'On-site',
            'summary'  => 'Work on internal audit, controls reviews and audit-readiness engagements while you complete your professional qualification.',
        ],
    ],
];
