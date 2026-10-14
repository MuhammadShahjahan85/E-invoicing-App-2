# Veridian Partners Website — Setup Guide for Hostinger

This guide takes you from the zip file to a live website in about 20 minutes. You do not need any programming knowledge.

**What you have**

| File | What it is |
|---|---|
| `website-dist/veridian-partners-website.zip` | The complete website. Upload this to Hostinger. |
| `website/` | The same website as editable files (source). |
| `WEBSITE-SETUP-GUIDE.md` | This guide. |

---

## Part 1 — Put the website online (10 minutes)

1. **Log in to Hostinger** → **hPanel** → **Websites** → click **Manage** next to your domain.
2. Check the **PHP version**: **Advanced → PHP Configuration** → choose **PHP 8.2** or newer (anything from 7.4 works) → **Save**.
3. Open **Files → File Manager**. Open the **public_html** folder.
4. If `public_html` contains Hostinger's placeholder files (for example `default.php` or `index.html`), **delete them**.
5. Click **Upload** (the arrow icon at the top right) → choose `veridian-partners-website.zip` → wait until it finishes.
6. Right-click the zip → **Extract** → when asked for a folder, extract into **public_html itself** (leave the folder name empty or type `.`).
   - The files `index.php`, `config.php`, `.htaccess` and the folders `assets`, `includes`, `admin`, `storage` must sit **directly inside public_html**, not in a sub-folder.
7. Delete the zip file from public_html (it is no longer needed).
8. Visit your domain — the website is live.

> **Tip:** if you do not see `.htaccess` in File Manager, click the gear icon (Settings) → tick **Show hidden files**. It must be there for clean addresses such as `/services`.

---

## Part 2 — Secure it with HTTPS (2 minutes)

1. hPanel → **Security → SSL**. If the certificate is not active, click **Install SSL** (free with Hostinger).
2. Turn on **Force HTTPS** on the same page.
3. Optional (after HTTPS works): open `.htaccess` and remove the `#` in front of the `Strict-Transport-Security` line for extra protection.

---

## Part 3 — Add your company details (5 minutes)

All settings live in **one file: `config.php`**. In File Manager, right-click `config.php` → **Edit**, change the values between the quotes, and **Save**. Empty values are hidden on the website automatically.

**Please fill in these — I could not know them:**

| Setting | Where it shows | Example |
|---|---|---|
| `contact → phone`, `mobile` | Header menu, footer, contact page | `'+92 42 1234 5678'` |
| `contact → whatsapp` | Green WhatsApp button on every page | `'923001234567'` (digits only) |
| `contact → address_line_1/2`, `city`, `province`, `postal_code` | Footer, contact page, Google (local SEO) | `'Lahore'` |
| `contact → map_embed_url`, `map_link` | Map on the contact page | From Google Maps → Share → Embed a map |
| `contact → hours` | Footer and contact page | Already set to Monday – Saturday, 9–6; change if needed |
| `company → founded`, `secp_cuin`, `ntn` | Footer and About page | `'2024'` |
| `social → linkedin`, `facebook`, … | Footer icons | Full links |
| `team` | About page (leadership) | Add credentials (e.g. ACCA, CIA), a LinkedIn link and a photo |
| `testimonials` | Home page (hidden while empty) | Only real quotes, with permission |
| `jobs` | Careers page | Set `'open' => true` to publish a role |
| `admin → password` | Unlocks the admin area | 12+ characters |

The public e-mail address shown on the site is **info@yourdomain** automatically. Create that mailbox in **hPanel → Emails → Email Accounts** (included with Hostinger plans), or set `contact → email` to another address.

---

## Part 4 — Make sure enquiries reach you (5 minutes)

Every form on the website (contact, consultation booking, newsletter, job applications with CVs) is:

1. **saved** in your admin area, and
2. **e-mailed** to the address in `forms → notify_to` (currently your Gmail), with a polite automatic confirmation sent to the person who wrote in.

For the most reliable delivery (so e-mails do not land in spam), switch on SMTP:

1. hPanel → **Emails** → create `info@yourdomain` (if you have not already).
2. In `config.php` → `smtp`:
   ```php
   'enabled'  => true,
   'host'     => 'smtp.hostinger.com',
   'port'     => 465,
   'secure'   => 'ssl',
   'username' => 'info@yourdomain.com',
   'password' => 'the mailbox password',
   ```
3. Visit **yourdomain.com/admin** → log in → **Setup & e-mail** → **Send test e-mail**. Check your inbox (and spam folder).

---

## Part 5 — Your admin area (yourdomain.com/admin)

- **Enquiries** — every contact message and consultation request, newest first. Reply, mark as done, delete, or **Export to Excel**.
- **Subscribers** — newsletter sign-ups (export them to send your monthly "Veridian Brief" from Gmail, Mailchimp or similar). Every welcome e-mail carries an unsubscribe link.
- **Applications** — job applications with **CV download**.
- **Setup & e-mail** — a checklist of what is configured, the test e-mail button and recent e-mail problems.

The admin area stays locked until you set a password in `config.php`.

---

## Part 6 — Get found on Google (SEO)

The website is already built for search engines: page titles and descriptions, keywords, structured data (Organization, Services, Articles, FAQs, Breadcrumbs, Software), Open Graph images for WhatsApp/LinkedIn/Facebook sharing, an automatic **sitemap** (`/sitemap.xml`) and `robots.txt`, fast loading, mobile-friendly design and 15 in-depth Knowledge Bank guides.

To start ranking:

1. **Google Search Console** — go to search.google.com/search-console → **Add property** → URL prefix → your domain → choose **HTML tag** → copy only the `content="…"` value into `config.php` → `site → google_site_verification` → Save → click **Verify**. Then open **Sitemaps** and submit `sitemap.xml`.
2. **Bing Webmaster Tools** — sign in and **import from Google Search Console** (one click).
3. **Google Business Profile** — create a free profile for "Veridian Partners Consultancy (Pvt) Ltd" with the same address, phone and website. This is the most important step for appearing in local searches such as "tax consultant in Lahore".
4. Optional: **Google Analytics** — create a GA4 property and put the `G-XXXXXXX` ID into `site → google_analytics_id`. A cookie banner appears automatically and analytics loads only for visitors who accept.
5. Keep publishing — add a Knowledge Bank article every month (see Part 7). Fresh, useful content is what lifts a professional firm in search results.

---

## Part 7 — Updating content later

| To change… | Edit this file |
|---|---|
| Contact details, team, jobs, testimonials, settings | `config.php` |
| Services (text, FAQs, keywords) | `includes/data/services.php` |
| Industries | `includes/data/industries.php` |
| Knowledge Bank articles | `includes/data/articles/*.php` — copy an existing article block, change the slug, title, date and text |
| FAQs | `includes/data/faqs.php` |
| Glossary | `includes/data/glossary.php` |
| Tax calendar deadlines | `includes/deadlines.php` |
| Salary tax slabs (each June after the budget) | `assets/js/main.js` → search for `SLABS` |
| Team photos | Upload square images to `assets/img/team/` and put the file name in `config.php` → `team → photo` |

New articles appear automatically in the Knowledge Bank, search, sitemap and related-article lists.

> **Before editing a `.php` file, download a copy** (right-click → Download). If a page shows an error after an edit, re-upload the copy. Setting `site → debug` to `true` in `config.php` shows the exact error; set it back to `false` afterwards.

---

## Troubleshooting

| Problem | Solution |
|---|---|
| Home page works but `/services`, `/about` show "404" | `.htaccess` is missing from public_html — re-extract the zip with hidden files shown. |
| "403 Forbidden" on the whole site | The files were extracted into a sub-folder. Move them directly into public_html. |
| Forms say "page has been open for a long time" | Refresh the page and send again (spam protection expires after three days). |
| Enquiries are saved but no e-mail arrives | Check the spam folder, then switch on SMTP (Part 4) and use **Send test e-mail**. |
| The admin area says "locked" | Set `admin → password` in `config.php` (at least 8, ideally 12+ characters). |
| Changes to CSS/JS do not appear | Files carry a version number automatically; press Ctrl + F5 once. |

---

## What is included

- **Pages:** Home, About, Services (overview + 10 detail pages), Industries (overview + 12 detail pages), FBR E-Invoicing software, Knowledge Bank (15 guides), Free calculators (salary tax 2026-27, sales tax, amount in words), Tax & compliance calendar (live countdowns, add to calendar), Glossary (70+ terms), Official portals, FAQs, Careers (with CV upload), Contact, 3-step consultation booking, Search, Privacy, Terms, Cookies, Disclaimer, Accessibility, Sitemap, 404, Thank-you and Unsubscribe pages.
- **Features:** mega menus, mobile menu, instant search (Ctrl + K), light/dark theme, WhatsApp button, newsletter, scroll animations, reading progress and table of contents on guides, share buttons, print-friendly pages, accessibility (keyboard navigation, screen-reader labels, reduced motion), security headers, spam protection without annoying CAPTCHAs, and an admin area.
- **Content accuracy:** tax figures reflect the law as published up to October 2026 (Finance Act 2026, SRO 1852(I)/2025, SRO 1666(I)/2026, STGO 01 and 25 of 2026, the Tax Year 2027 salary slabs). Please have a qualified colleague review the guides before launch and after each budget, and adjust anything that differs from your professional view.

© Veridian Partners Consultancy Private Limited. All rights reserved.
