<?php
/**
 * Recurring tax and corporate deadlines (Tax calendar page and home page widget).
 * Edit the list below when FBR or SECP change a date. Dates are general rules:
 * FBR can extend them by notification, and a date falling on a public holiday
 * may move — the calendar page says so.
 *
 * type: sales | income | corporate
 * rule: ['monthly', day] | ['yearly', month, day]
 */
function deadline_definitions(): array
{
    return [
        ['key' => 'st-pay',  'type' => 'sales',  'rule' => ['monthly', 15],
         'title' => 'Sales tax payment', 'authority' => 'FBR',
         'desc' => 'Deposit sales tax (and FED, where applicable) for the previous month.'],
        ['key' => 'st-ret',  'type' => 'sales',  'rule' => ['monthly', 18],
         'title' => 'Sales tax return (with Annex-C)', 'authority' => 'FBR',
         'desc' => 'File the monthly sales tax return for the previous month. Reconcile e-invoices, books and Annex-C first.'],
        ['key' => 'adv-q1',  'type' => 'income', 'rule' => ['yearly', 9, 25],
         'title' => 'Advance tax — 1st quarterly instalment', 'authority' => 'FBR',
         'desc' => 'Quarterly advance income tax under section 147 for companies and AOPs (quarter ending September).'],
        ['key' => 'adv-q2',  'type' => 'income', 'rule' => ['yearly', 12, 25],
         'title' => 'Advance tax — 2nd quarterly instalment', 'authority' => 'FBR',
         'desc' => 'Quarterly advance income tax under section 147 (quarter ending December).'],
        ['key' => 'adv-q3',  'type' => 'income', 'rule' => ['yearly', 3, 25],
         'title' => 'Advance tax — 3rd quarterly instalment', 'authority' => 'FBR',
         'desc' => 'Quarterly advance income tax under section 147 (quarter ending March).'],
        ['key' => 'adv-q4',  'type' => 'income', 'rule' => ['yearly', 6, 15],
         'title' => 'Advance tax — 4th quarterly instalment', 'authority' => 'FBR',
         'desc' => 'Quarterly advance income tax under section 147 (quarter ending June).'],
        ['key' => 'itr-ind', 'type' => 'income', 'rule' => ['yearly', 9, 30],
         'title' => 'Income tax return — individuals & AOPs', 'authority' => 'FBR',
         'desc' => 'Return of income and wealth statement for the tax year ended 30 June. FBR sometimes extends this date.'],
        ['key' => 'itr-co',  'type' => 'income', 'rule' => ['yearly', 12, 31],
         'title' => 'Income tax return — companies (June year end)', 'authority' => 'FBR',
         'desc' => 'Return of income for companies whose tax year ended on 30 June.'],
        ['key' => 'agm',     'type' => 'corporate', 'rule' => ['yearly', 10, 28],
         'title' => 'AGM — companies with a 30 June year end', 'authority' => 'SECP',
         'desc' => 'Annual General Meeting within 120 days of the financial year end, to lay the audited financial statements.'],
        ['key' => 'annual-return', 'type' => 'corporate', 'rule' => ['yearly', 11, 27],
         'title' => 'Annual return to SECP', 'authority' => 'SECP',
         'desc' => 'Due within 30 days of the AGM — 27 November if the AGM is held on 28 October.'],
    ];
}

function deadline_type_label(string $type): string
{
    return ['sales' => 'Sales tax', 'income' => 'Income tax', 'corporate' => 'Corporate'][$type] ?? ucfirst($type);
}

/**
 * Occurrences from today for the next $months months, sorted by date.
 * Each: date (Y-m-d), title, desc, authority, type, key.
 */
function upcoming_deadlines(int $months = 12): array
{
    $today = new DateTimeImmutable('today', new DateTimeZone('Asia/Karachi'));
    $end = $today->modify('+' . $months . ' months');
    $out = [];
    foreach (deadline_definitions() as $d) {
        $rule = $d['rule'];
        if ($rule[0] === 'monthly') {
            $cursor = $today->modify('first day of this month');
            while ($cursor <= $end) {
                $date = $cursor->setDate((int) $cursor->format('Y'), (int) $cursor->format('n'), $rule[1]);
                if ($date >= $today && $date <= $end) {
                    $out[] = $d + ['date' => $date->format('Y-m-d')];
                }
                $cursor = $cursor->modify('+1 month');
            }
        } else {
            for ($y = (int) $today->format('Y'); $y <= (int) $end->format('Y'); $y++) {
                $date = $today->setDate($y, $rule[1], $rule[2]);
                if ($date >= $today && $date <= $end) {
                    $out[] = $d + ['date' => $date->format('Y-m-d')];
                }
            }
        }
    }
    usort($out, fn($a, $b) => strcmp($a['date'], $b['date']) ?: strcmp($a['key'], $b['key']));
    return $out;
}

function days_until(string $ymd): int
{
    $today = new DateTimeImmutable('today', new DateTimeZone('Asia/Karachi'));
    $due = new DateTimeImmutable($ymd, new DateTimeZone('Asia/Karachi'));
    return (int) $today->diff($due)->format('%r%a');
}

function countdown_label(int $days): string
{
    if ($days <= 0) {
        return 'Due today';
    }
    return $days === 1 ? 'Tomorrow' : 'In ' . $days . ' days';
}

/** One deadline row (used on the home page and the calendar page). */
function deadline_row(array $d, bool $full = false): string
{
    $t = strtotime($d['date']);
    $days = days_until($d['date']);
    $cls = $days <= 0 ? ' is-today' : ($days <= 7 ? ' is-soon' : '');
    $tagCls = ['sales' => 'tag--sales', 'income' => 'tag--income', 'corporate' => 'tag--corporate'][$d['type']] ?? '';
    $date = '<span class="datebox" aria-hidden="true"><strong>' . date('j', $t) . '</strong><span>' . date('M', $t) . '</span></span>';
    if (!$full) {
        return '<li class="deadline" data-deadline-row data-due="' . e($d['date']) . '">' . $date
            . '<span><span class="deadline__title">' . e($d['title']) . '</span><span class="deadline__meta">' . e(date('l, j F Y', $t)) . ' · ' . e($d['authority']) . '</span></span>'
            . '<span class="countdown' . $cls . '">' . e(countdown_label($days)) . '</span></li>';
    }
    return '<li class="cal-item" data-deadline-row data-due="' . e($d['date']) . '" data-type="' . e($d['type']) . '">' . $date
        . '<div><span class="cal-item__title">' . e($d['title']) . '</span><span class="cal-item__desc">' . e($d['desc']) . '</span>'
        . '<span class="cal-item__tags"><span class="tag ' . $tagCls . '">' . e(deadline_type_label($d['type'])) . '</span><span class="tag tag--payroll">' . e($d['authority']) . '</span><span class="tag tag--payroll">' . e(date('l', $t)) . '</span></span></div>'
        . '<div class="cal-item__side"><span class="countdown' . $cls . '">' . e(countdown_label($days)) . '</span>'
        . '<button class="ics-btn" type="button" data-ics="' . e($d['date']) . '" data-ics-title="' . e($d['title']) . '" data-ics-desc="' . e($d['desc']) . '">' . icon('calendar-plus') . 'Add to calendar</button></div></li>';
}
