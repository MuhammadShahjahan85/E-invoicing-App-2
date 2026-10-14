// Local preview screenshots: node website-tools/shots.js <outdir> <path> [<path>...]
// Env: W (width, default 1440), H (height, default 900), FULL=1 full page, THEME=dark, SUFFIX
const { chromium } = require('playwright');
(async () => {
  const [out, ...paths] = process.argv.slice(2);
  const browser = await chromium.launch();
  const ctx = await browser.newContext({ viewport: { width: +(process.env.W || 1440), height: +(process.env.H || 900) }, deviceScaleFactor: +(process.env.DPR || 1), colorScheme: process.env.THEME === 'dark' ? 'dark' : 'light', isMobile: +(process.env.W || 1440) < 600, hasTouch: +(process.env.W || 1440) < 600 });
  const page = await ctx.newPage();
  const errors = [];
  page.on('pageerror', e => errors.push(String(e)));
  page.on('console', m => { if (m.type() === 'error') errors.push(m.text()); });
  for (const p of paths) {
    await page.goto('http://127.0.0.1:8080' + p, { waitUntil: 'networkidle' });
    if (process.env.FULL) {
      await page.evaluate(async () => {
        for (let y = 0; y < document.body.scrollHeight; y += 600) { window.scrollTo(0, y); await new Promise(r => setTimeout(r, 60)); }
        window.scrollTo(0, 0);
      });
    }
    await page.evaluate(() => {
      document.querySelectorAll('.reveal, [data-animate]').forEach(el => el.classList.add('is-visible'));
      document.querySelectorAll('img[loading=lazy]').forEach(i => i.loading = 'eager');
    });
    await page.waitForTimeout(1800);
    const name = (p.replace(/[^a-z0-9]+/gi, '_').replace(/^_|_$/g, '') || 'home') + (process.env.SUFFIX || '') + '.png';
    await page.screenshot({ path: out + '/' + name, fullPage: !!process.env.FULL });
    console.log('saved', name);
  }
  if (errors.length) console.log('ERRORS:\n' + errors.join('\n'));
  await browser.close();
})();
