<?php
require __DIR__ . '/includes/bootstrap.php';

$crumbs = [['Home', ''], ['Knowledge Bank', 'knowledge-bank'], ['Free calculators', 'tools']];
$page = [
    'title' => 'Free Tax Calculators — Salary Tax 2026-27, Sales Tax & Amount in Words',
    'description' => 'Free calculators for Pakistan: salary income tax for Tax Year 2027 (2026-27) and 2026, sales tax at 18% with 4% further tax, and rupees in words in lakh and crore.',
    'keywords' => 'salary tax calculator Pakistan 2026-27, income tax calculator Pakistan, tax year 2027 calculator, sales tax calculator Pakistan, 18% sales tax calculator, amount in words rupees, lakh crore converter',
    'path' => 'tools',
    'nav' => 'knowledge',
    'schema' => [
        breadcrumb_schema($crumbs),
        ['@type' => 'WebApplication', 'name' => 'Pakistan salary tax calculator 2026-27', 'applicationCategory' => 'FinanceApplication', 'operatingSystem' => 'Any', 'url' => abs_url('tools') . '#salary-tax', 'offers' => ['@type' => 'Offer', 'price' => '0', 'priceCurrency' => 'PKR'], 'publisher' => ['@id' => abs_url('#organization')]],
        ['@type' => 'WebApplication', 'name' => 'Pakistan sales tax calculator', 'applicationCategory' => 'FinanceApplication', 'operatingSystem' => 'Any', 'url' => abs_url('tools') . '#sales-tax', 'offers' => ['@type' => 'Offer', 'price' => '0', 'priceCurrency' => 'PKR'], 'publisher' => ['@id' => abs_url('#organization')]],
    ],
];
require __DIR__ . '/includes/header.php';

echo page_hero([
    'crumbs' => $crumbs,
    'eyebrow' => 'Free tools',
    'title' => 'Quick answers, <em>calculated</em>.',
    'lede' => 'Free calculators for everyday tax questions in Pakistan. Nothing you type leaves your browser.',
    'actions' => '<a class="btn btn--ghost-light" href="#salary-tax">' . icon('calculator') . '<span>Salary tax</span></a><a class="btn btn--ghost-light" href="#sales-tax">' . icon('percent') . '<span>Sales tax</span></a><a class="btn btn--ghost-light" href="#amount-in-words">' . icon('type') . '<span>Amount in words</span></a>',
]);
?>

<section class="section">
  <div class="container">

    <!-- Salary tax -->
    <div class="tool" id="salary-tax" data-salary-calc>
      <div class="tool__head">
        <span class="icon-tile icon-tile--gold"><?= icon('calculator') ?></span>
        <div><h2>Salary income tax calculator</h2><p>Tax Year 2027 (July 2026 – June 2027) and Tax Year 2026, for salaried individuals.</p></div>
      </div>
      <div class="tool__body">
        <div class="tool__inputs">
          <div class="field">
            <span class="field__label">Tax year</span>
            <div class="segmented" role="radiogroup" aria-label="Tax year">
              <label><input type="radio" name="year" value="2027" checked><span>2026-27 (TY2027)</span></label>
              <label><input type="radio" name="year" value="2026"><span>2025-26 (TY2026)</span></label>
            </div>
          </div>
          <div class="field">
            <span class="field__label">I am entering my</span>
            <div class="segmented" role="radiogroup" aria-label="Salary period">
              <label><input type="radio" name="period" value="month" checked><span>Monthly salary</span></label>
              <label><input type="radio" name="period" value="year"><span>Annual salary</span></label>
            </div>
          </div>
          <div class="field">
            <label for="salary">Taxable salary</label>
            <div class="money"><span>Rs</span><input id="salary" name="salary" type="text" inputmode="decimal" value="250,000" autocomplete="off"></div>
            <span class="field__hint">Gross taxable salary — exclude exempt allowances.</span>
          </div>
          <div class="table-wrap">
            <table class="table">
              <thead><tr><th>Annual income (Rs)</th><th>Tax</th></tr></thead>
              <tbody data-slab-table></tbody>
            </table>
          </div>
        </div>
        <div class="tool__results" aria-live="polite">
          <div class="result-hero">
            <span class="result-hero__label">Monthly tax</span>
            <strong class="result-hero__value" data-out="monthly">—</strong>
            <span data-out="compare"></span>
          </div>
          <div class="result-grid">
            <div class="result-cell"><span>Annual tax</span><strong data-out="annual">—</strong></div>
            <div class="result-cell"><span>Monthly take-home</span><strong data-out="take">—</strong></div>
            <div class="result-cell"><span>Monthly gross</span><strong data-out="gross">—</strong></div>
            <div class="result-cell"><span>Effective rate</span><strong data-out="effective">—</strong></div>
            <div class="result-cell"><span>Marginal rate</span><strong data-out="marginal">—</strong></div>
            <div class="result-cell"><span>Want it checked?</span><strong><a href="<?= e(url('services/tax-advisory')) ?>">Ask an adviser →</a></strong></div>
          </div>
          <div class="split-bar" aria-hidden="true"><span class="split-bar__take"></span><span class="split-bar__tax"></span></div>
          <div class="legend"><span><i style="background:var(--green-500)"></i>Take-home</span><span><i style="background:var(--gold-300)"></i>Income tax</span></div>
          <p class="tool__note">Based on the salaried slabs of the First Schedule as amended by the Finance Acts 2025 and 2026, for individuals whose salary is more than 75% of taxable income. Excludes other income, credits and exemptions. For guidance only — read our <a href="<?= e(url('knowledge-bank/salary-income-tax-slabs-tax-year-2027')) ?>">Tax Year 2027 slabs guide</a>.</p>
        </div>
      </div>
    </div>

    <!-- Sales tax -->
    <div class="tool" id="sales-tax" data-sales-calc>
      <div class="tool__head">
        <span class="icon-tile icon-tile--gold"><?= icon('percent') ?></span>
        <div><h2>Sales tax calculator</h2><p>Add sales tax to a price, or extract it from a tax-inclusive amount.</p></div>
      </div>
      <div class="tool__body">
        <div class="tool__inputs">
          <div class="field">
            <span class="field__label">The amount I have is</span>
            <div class="segmented" role="radiogroup" aria-label="Amount type">
              <label><input type="radio" name="mode" value="exclusive" checked><span>Excluding tax</span></label>
              <label><input type="radio" name="mode" value="inclusive"><span>Including tax</span></label>
            </div>
          </div>
          <div class="field">
            <label for="st-amount">Amount</label>
            <div class="money"><span>Rs</span><input id="st-amount" name="amount" type="text" inputmode="decimal" value="100,000" autocomplete="off"></div>
          </div>
          <div class="field">
            <label for="st-rate">Sales tax rate</label>
            <select id="st-rate" name="rate">
              <option value="18" selected>18% — standard rate (goods)</option>
              <option value="16">16%</option>
              <option value="15">15%</option>
              <option value="10">10%</option>
              <option value="5">5%</option>
              <option value="0">0% — zero-rated</option>
              <option value="custom">Custom rate…</option>
            </select>
          </div>
          <div class="field" hidden>
            <label for="st-custom">Custom rate (%)</label>
            <input id="st-custom" name="custom_rate" type="text" inputmode="decimal" value="13">
          </div>
          <label class="check mb-0"><input type="checkbox" name="further"><span>Add 4% further tax (supply to an unregistered buyer)</span></label>
        </div>
        <div class="tool__results" aria-live="polite">
          <div class="result-hero">
            <span class="result-hero__label">Total including taxes</span>
            <strong class="result-hero__value" data-out="total">—</strong>
          </div>
          <div class="result-grid">
            <div class="result-cell"><span>Value excluding tax</span><strong data-out="net">—</strong></div>
            <div class="result-cell"><span>Sales tax</span><strong data-out="tax">—</strong></div>
            <div class="result-cell"><span>Further tax</span><strong data-out="further">—</strong></div>
            <div class="result-cell"><span>Issue it with FBR</span><strong><a href="<?= e(url('e-invoicing')) ?>">E-invoicing →</a></strong></div>
          </div>
          <p class="tool__note">Rates vary by product, schedule and province — check the rate that applies to your supply. Further tax has exclusions, for example for exempt and zero-rated supplies.</p>
        </div>
      </div>
    </div>

    <!-- Amount in words -->
    <div class="tool" id="amount-in-words" data-words-calc>
      <div class="tool__head">
        <span class="icon-tile icon-tile--gold"><?= icon('type') ?></span>
        <div><h2>Amount in words</h2><p>Write rupees in words for cheques, invoices and agreements — in lakh and crore, or millions.</p></div>
      </div>
      <div class="tool__body">
        <div class="tool__inputs">
          <div class="field">
            <label for="words-amount">Amount</label>
            <div class="money"><span>Rs</span><input id="words-amount" name="words_amount" type="text" inputmode="decimal" value="1,234,567.50" autocomplete="off"></div>
          </div>
          <div class="field">
            <span class="field__label">Number system</span>
            <div class="segmented" role="radiogroup" aria-label="Number system">
              <label><input type="radio" name="system" value="pk" checked><span>Lakh & crore</span></label>
              <label><input type="radio" name="system" value="intl"><span>Million & billion</span></label>
            </div>
          </div>
        </div>
        <div class="tool__results">
          <p class="words-out" data-out="words" aria-live="polite">—</p>
          <button class="btn btn--ghost btn--sm" type="button" data-copy-words><?= icon('link') ?><span>Copy</span></button>
        </div>
      </div>
    </div>

  </div>
</section>

<section class="section section--tint">
  <div class="container">
    <div class="section-head section-head--center">
      <p class="eyebrow">More free resources</p>
      <h2>Keep your compliance on <em>track</em>.</h2>
    </div>
    <div class="grid grid--3">
      <a class="card card--industry reveal" href="<?= e(url('tax-calendar')) ?>"><span class="icon-tile icon-tile--sm icon-tile--gold"><?= icon('calendar') ?></span><span class="card--industry__body"><span class="card__title">Tax & compliance calendar</span><span class="card__text">Live countdowns, and add any deadline to your own calendar.</span></span><?= icon('arrow-up-right', 'card--industry__arrow') ?></a>
      <a class="card card--industry reveal" href="<?= e(url('glossary')) ?>"><span class="icon-tile icon-tile--sm icon-tile--gold"><?= icon('book-open') ?></span><span class="card--industry__body"><span class="card__title">Glossary</span><span class="card__text">Tax and business terms, explained simply.</span></span><?= icon('arrow-up-right', 'card--industry__arrow') ?></a>
      <a class="card card--industry reveal" href="<?= e(url('official-portals')) ?>"><span class="icon-tile icon-tile--sm icon-tile--gold"><?= icon('external-link') ?></span><span class="card--industry__body"><span class="card__title">Official portals</span><span class="card__text">FBR, IRIS, SECP and provincial authorities in one place.</span></span><?= icon('arrow-up-right', 'card--industry__arrow') ?></a>
    </div>
  </div>
</section>

<?= cta_band() ?>
<?php require __DIR__ . '/includes/footer.php'; ?>
