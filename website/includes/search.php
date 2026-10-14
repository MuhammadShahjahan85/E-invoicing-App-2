<?php
/**
 * Site search index (used by /search and by /search-index.json for instant search).
 * Each item: t (title), u (link), d (description), y (type label), k (keywords), w (weight).
 */
function search_index(): array
{
    $items = [];
    $pages = [
        ['Home', '', 'Audit, tax, advisory and FBR e-invoicing in Pakistan.', 'home', 0],
        ['About us', 'about', 'Our story, mission, values and leadership.', 'about company team mission vision values', 0],
        ['FBR E-Invoicing Software', 'e-invoicing', 'Veridian E-invoicing Pakistan — FBR digital invoicing software with QR codes and Annex-C reconciliation.', 'fbr digital invoicing einvoicing e-invoice software pral di api qr code pos erp integration', 8],
        ['Book a free consultation', 'book-consultation', 'Choose the help you need and a time that suits you.', 'appointment meeting call consultation booking demo', 4],
        ['Contact us', 'contact', 'E-mail, phone, WhatsApp and office details.', 'contact phone email whatsapp address office', 3],
        ['Careers', 'careers', 'Jobs and traineeships in tax, audit, accounting and technology.', 'jobs careers hiring trainee internship acca ca vacancy', 2],
        ['Free calculators', 'tools', 'Salary tax 2026-27, sales tax and amount-in-words calculators.', 'calculator salary tax income tax sales tax amount in words lakh crore', 6],
        ['Salary tax calculator 2026-27', 'tools#salary-tax', 'Monthly and annual salary tax under the Tax Year 2027 slabs.', 'salary tax calculator income tax slabs 2027 2026 budget', 7],
        ['Sales tax calculator', 'tools#sales-tax', 'Add or extract 18% sales tax and 4% further tax.', 'sales tax calculator 18% gst further tax', 5],
        ['Amount in words', 'tools#amount-in-words', 'Rupees in words in lakh and crore.', 'amount in words rupees cheque lakh crore', 3],
        ['Tax & compliance calendar', 'tax-calendar', 'Upcoming FBR and SECP deadlines with countdowns.', 'tax calendar due date deadline sales tax return income tax return advance tax agm annual return', 6],
        ['Glossary', 'glossary', 'Pakistani tax and business terms explained.', 'glossary terms definitions meaning', 3],
        ['Official portals', 'official-portals', 'FBR, IRIS, SECP, PRA, SRB, KPRA, BRA and more.', 'fbr iris secp pra srb kpra bra boi sbp pseb psw portal website login', 3],
        ['Frequently asked questions', 'faq', 'Answers about our services, fees and common tax topics.', 'faq questions answers fees', 2],
        ['Knowledge Bank', 'knowledge-bank', 'Guides on tax, sales tax, SECP, audit and reporting.', 'insights guides articles blog', 2],
        ['Privacy policy', 'privacy-policy', 'How we handle personal data.', 'privacy data protection', -2],
        ['Terms of use', 'terms', 'Terms for using this website.', 'terms conditions', -2],
    ];
    foreach ($pages as [$t, $p, $d, $k, $w]) {
        $items[] = ['t' => $t, 'u' => url($p), 'd' => $d, 'y' => 'Page', 'k' => $k, 'w' => $w];
    }
    foreach (services() as $s) {
        $items[] = ['t' => $s['title'], 'u' => service_url($s), 'd' => $s['short'], 'y' => 'Service', 'k' => $s['keywords'] . ' ' . implode(' ', array_column($s['offerings'], 0)), 'w' => 6];
    }
    foreach (industries() as $i) {
        $items[] = ['t' => $i['name'], 'u' => url('industries/' . $i['slug']), 'd' => $i['short'], 'y' => 'Industry', 'k' => $i['keywords'], 'w' => 3];
    }
    foreach (articles() as $a) {
        $items[] = ['t' => $a['title'], 'u' => article_url($a), 'd' => $a['excerpt'], 'y' => 'Guide', 'k' => implode(' ', $a['tags'] ?? []) . ' ' . $a['description'], 'w' => 4];
    }
    foreach (glossary() as $term => $def) {
        $anchor = trim(preg_replace('/[^a-z0-9]+/', '-', strtolower($term)), '-');
        $items[] = ['t' => $term, 'u' => url('glossary') . '#' . $anchor, 'd' => strip_tags($def), 'y' => 'Term', 'k' => '', 'w' => 1];
    }
    foreach (faqs() as $group) {
        foreach ($group as $f) {
            $items[] = ['t' => $f['q'], 'u' => url('faq'), 'd' => strip_tags($f['a']), 'y' => 'FAQ', 'k' => '', 'w' => 0];
        }
    }
    return $items;
}

/** Simple ranked search over the index. */
function site_search(string $q, int $limit = 30): array
{
    $tokens = array_values(array_filter(preg_split('/\s+/', mb_strtolower(preg_replace('/[^\p{L}\p{N}\s()-]/u', ' ', $q))), fn($t) => mb_strlen($t) > 1));
    if (!$tokens) {
        return [];
    }
    $scored = [];
    foreach (search_index() as $item) {
        $t = mb_strtolower($item['t']);
        $k = mb_strtolower($item['k']);
        $d = mb_strtolower($item['d']);
        $score = 0;
        foreach ($tokens as $tok) {
            $s = 0;
            if (strpos($t, $tok) === 0) {
                $s += 14;
            } elseif (strpos(' ' . $t, ' ' . $tok) !== false) {
                $s += 10;
            } elseif (strpos($t, $tok) !== false) {
                $s += 6;
            }
            if (strpos($k, $tok) !== false) {
                $s += 4;
            }
            if (strpos($d, $tok) !== false) {
                $s += 2;
            }
            if ($s === 0) {
                continue 2;
            }
            $score += $s;
        }
        $item['score'] = $score + $item['w'];
        $scored[] = $item;
    }
    usort($scored, fn($a, $b) => $b['score'] <=> $a['score']);
    return array_slice($scored, 0, $limit);
}
