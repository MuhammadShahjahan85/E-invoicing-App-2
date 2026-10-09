// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.
//
// Builds docs/Veridian-E-invoicing-Pakistan-User-Guide.pdf from
// docs/USER-MANUAL.md and its screenshots: a cover, a table of contents with
// page numbers, and the manual set in the product's fonts and colours. The
// Windows installer ships it as the Start menu "User guide".
//
// Needs Node.js with the "marked" and "playwright" packages, the fonts in
// web/node_modules (npm ci in web/) and pdfunite and pdftotext (poppler-utils):
//   NODE_PATH=$(npm root -g) node scripts/user-guide.cjs [version]
const { marked } = require('marked')
const { chromium } = require('playwright')
const { execFileSync } = require('child_process')
const fs = require('fs')
const os = require('os')
const path = require('path')

const ROOT = path.resolve(__dirname, '..')
const DOCS = path.join(ROOT, 'docs')
const OUT = path.join(DOCS, 'Veridian-E-invoicing-Pakistan-User-Guide.pdf')
const VERSION = process.argv[2] || ''
const MONTH = new Date().toLocaleDateString('en-GB', { month: 'long', year: 'numeric', timeZone: 'Asia/Karachi' })
const url = (p) => 'file://' + path.join(ROOT, p)
const MARK = fs.readFileSync(path.join(ROOT, 'web/public/favicon.svg'), 'utf8')

const FONTS = `
@font-face { font-family: Jakarta; font-weight: 200 800; src: url(${url('web/node_modules/@fontsource-variable/plus-jakarta-sans/files/plus-jakarta-sans-latin-wght-normal.woff2')}); }
@font-face { font-family: Inter; font-weight: 100 900; src: url(${url('web/node_modules/@fontsource-variable/inter/files/inter-latin-wght-normal.woff2')}); }
@font-face { font-family: Inter; font-weight: 100 900; unicode-range: U+0100-024F, U+20A0-20CF; src: url(${url('web/node_modules/@fontsource-variable/inter/files/inter-latin-ext-wght-normal.woff2')}); }
@font-face { font-family: Nastaliq; font-weight: 400; src: url(${url('web/node_modules/@fontsource/noto-nastaliq-urdu/files/noto-nastaliq-urdu-arabic-400-normal.woff2')}); }
:root { --navy: #23225e; --navy-2: #16153d; --green: #16a34a; --green-d: #15803d; --gold: #e3b341; --ink: #1f2433; --muted: #5b6275; --line: #e3e6ee; --soft: #f5f7fb; }
* { box-sizing: border-box; }
html { -webkit-print-color-adjust: exact; print-color-adjust: exact; }
body { margin: 0; font-family: Inter, sans-serif; color: var(--ink); }`

const slug = (s) => s.toLowerCase().replace(/<[^>]+>/g, '').replace(/&[a-z]+;/g, '').replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')

// The manual without its own title and the introduction above the first
// chapter (the cover replaces them), and with the chapter list.
function manual() {
  const md = fs.readFileSync(path.join(DOCS, 'USER-MANUAL.md'), 'utf8')
  const tokens = marked.lexer(md)
  const first = tokens.findIndex((t) => t.type === 'heading' && t.depth === 2)
  const intro = tokens.slice(0, first).filter((t) => t.type === 'paragraph').map((t) => t.text)
  const body = tokens.slice(first).filter((t) => !(t.type === 'hr'))
  // The closing copyright line is printed on the back of the cover instead.
  while (body.length && body[body.length - 1].type === 'paragraph' && /©/.test(body[body.length - 1].text)) body.pop()
  const chapters = []
  for (const t of body) {
    if (t.type === 'heading' && (t.depth === 2 || t.depth === 3)) {
      const text = t.text.replace(/\*\*|`/g, '')
      chapters.push({ depth: t.depth, text, id: slug(text) })
    }
  }
  const renderer = new marked.Renderer()
  renderer.heading = function ({ tokens, depth, text }) {
    const inner = this.parser.parseInline(tokens)
    const id = slug(text.replace(/\*\*|`/g, ''))
    if (depth === 2) {
      const m = inner.match(/^(\d+)\.\s*(.*)$/)
      return m
        ? `<h2 id="${id}"><span class="num">${m[1]}</span><span>${m[2]}</span></h2>`
        : `<h2 id="${id}"><span>${inner}</span></h2>`
    }
    return `<h${depth} id="${id}">${inner}</h${depth}>`
  }
  renderer.image = ({ href, text }) => `<figure><img src="${href}" alt="${text}"><figcaption>${text}</figcaption></figure>`
  // A picture on its own is a figure, not a paragraph.
  renderer.paragraph = function ({ tokens }) {
    const inner = this.parser.parseInline(tokens)
    const real = tokens.filter((t) => !(t.type === 'text' && !t.text.trim()))
    return real.length === 1 && real[0].type === 'image' ? inner + '\n' : `<p>${inner}</p>\n`
  }
  renderer.link = function ({ href, tokens }) {
    const inner = this.parser.parseInline(tokens)
    // Other guides are not part of this PDF: show their name only.
    if (/\.md(#.*)?$/.test(href) && !/^https?:/.test(href)) return `<span class="ref">${inner}</span>`
    return `<a href="${href}">${inner}</a>`
  }
  // A heading stays on the same page as the picture that follows it.
  const html = marked
    .parser(body, { renderer, gfm: true })
    .replace(/(<h[34][^>]*>[\s\S]*?<\/h[34]>)\s*(<figure>[\s\S]*?<\/figure>)/g, '<div class="keep">$1$2</div>')
  return { intro, chapters, html }
}

const COVER = `<!doctype html><html><head><meta charset="utf-8"><style>${FONTS}
@page { size: A4; margin: 0; }
.cover { width: 210mm; height: 297mm; position: relative; overflow: hidden; color: #fff;
  background: radial-gradient(120mm 120mm at 0% 100%, rgba(22,163,74,.45), transparent 70%),
              radial-gradient(90mm 90mm at 100% 0%, rgba(227,179,65,.22), transparent 70%),
              linear-gradient(170deg, #2b2a72 0%, #23225e 40%, #14133a 100%); }
.wm { position: absolute; right: -40mm; bottom: -30mm; width: 190mm; opacity: .06; }
.top { position: absolute; top: 26mm; left: 24mm; right: 24mm; display: flex; align-items: center; gap: 5mm; }
.top svg { width: 17mm; height: 17mm; }
.brand b { display: block; font: 800 22pt/1 Jakarta; letter-spacing: -.02em; }
.brand span { display: block; margin-top: 2mm; font: 600 10.5pt/1 Inter; color: #86efac; }
.title { position: absolute; top: 96mm; left: 24mm; right: 24mm; }
.eyebrow { font: 700 9.5pt/1 Inter; letter-spacing: .22em; color: #e3b341; }
h1 { margin: 6mm 0 0; font: 800 46pt/1.02 Jakarta; letter-spacing: -.03em; }
.ur { margin-top: 5mm; font: 400 20pt/2 Nastaliq; direction: rtl; text-align: left; color: rgba(255,255,255,.85); }
.lead { margin-top: 8mm; max-width: 140mm; font: 400 12.5pt/1.55 Inter; color: rgba(255,255,255,.86); }
.rule { width: 18mm; height: 1.2mm; border-radius: 1mm; background: #e3b341; margin-top: 10mm; }
.meta { position: absolute; left: 24mm; right: 24mm; bottom: 24mm; display: flex; justify-content: space-between; align-items: flex-end;
  font: 500 9.5pt/1.6 Inter; color: rgba(255,255,255,.78); }
.meta b { color: #fff; font-weight: 700; }
.back { width: 210mm; height: 297mm; padding: 30mm 24mm; break-before: page; display: flex; flex-direction: column; justify-content: flex-end;
  font: 400 9pt/1.6 Inter; color: #5b6275; }
.back p { margin: 0 0 3mm; }
.back b { color: #23225e; }
</style></head><body>
<section class="cover">
  <svg class="wm" viewBox="0 0 100 100"><path d="M26 35 L46 70 L72 30" fill="none" stroke="#fff" stroke-width="9" stroke-linecap="round" stroke-linejoin="round"/></svg>
  <div class="top">${MARK}<div class="brand"><b>Veridian</b><span>E-invoicing Pakistan</span></div></div>
  <div class="title">
    <div class="eyebrow">FBR DIGITAL INVOICING</div>
    <h1>User guide</h1>
    <div class="ur">صارف رہنما</div>
    <div class="lead">{{INTRO}}</div>
    <div class="rule"></div>
  </div>
  <div class="meta">
    <div><b>Veridian Partners Consultancy Private Limited</b><br>Support: muhammadshahjahan.audit@gmail.com</div>
    <div style="text-align:right">${VERSION ? 'Version ' + VERSION + '<br>' : ''}${MONTH}</div>
  </div>
</section>
<section class="back">
  <p><b>Veridian E-invoicing Pakistan</b> — User guide${VERSION ? ', version ' + VERSION : ''}, ${MONTH}.</p>
  <p>© 2026 Veridian Partners Consultancy Private Limited. All rights reserved. Veridian E-invoicing Pakistan is proprietary software, licensed and not sold. No part of this guide may be reproduced without the written permission of the company.</p>
  <p>Tax rates, SROs and FBR's technical requirements change. This guide describes the software; it is not tax advice. Check FBR's latest notifications, or ask your tax adviser, before relying on a rate or rule.</p>
  <p>Screens in this guide use invented sample data in the training simulator.</p>
</section>
</body></html>`

function bodyPage({ chapters, html }, pages) {
  // Chapters with their page numbers; their sections listed underneath.
  const toc = chapters
    .filter((c) => c.depth === 2)
    .map((c) => {
      const subs = chapters.slice(chapters.indexOf(c) + 1)
      const end = subs.findIndex((x) => x.depth === 2)
      const sections = (end < 0 ? subs : subs.slice(0, end)).map((x) => `<a href="#${x.id}">${x.text}</a>`).join(' · ')
      const m = c.text.match(/^(\d+)\.\s*(.*)$/)
      const title = m ? `<em>${m[1]}</em>${m[2]}` : c.text
      return `<li><a class="ch" href="#${c.id}"><span>${title}</span><i></i><b>${pages ? pages[c.id] ?? '' : '00'}</b></a>${sections ? `<div class="secs">${sections}</div>` : ''}</li>`
    })
    .join('')
  return `<!doctype html><html><head><meta charset="utf-8"><base href="${url('docs/')}"><style>${FONTS}
@page { size: A4; margin: 20mm 17mm 20mm 17mm; }
body { font-size: 10pt; line-height: 1.6; }
.toc h2 { margin: 0 0 6mm; font: 800 24pt/1.1 Jakarta; color: var(--navy); letter-spacing: -.02em; border: 0; }
.toc ul { list-style: none; margin: 0; padding: 0; }
.toc li { margin: 0 0 3.4mm; break-inside: avoid; }
.toc a { border: 0; }
.toc .ch { display: flex; align-items: baseline; gap: 2mm; font: 700 10.5pt/1.35 Inter; color: var(--navy); }
.toc .ch em { font-style: normal; display: inline-block; min-width: 7mm; color: var(--green-d); }
.toc .ch i { flex: 1; border-bottom: 1px dotted #b9bfcc; transform: translateY(-1mm); }
.toc .ch b { font-weight: 600; color: var(--muted); font-variant-numeric: tabular-nums; }
.toc .secs { margin: .8mm 0 0 7mm; font-size: 8.6pt; line-height: 1.45; color: var(--muted); }
.toc .secs a { color: var(--muted); }
.chapters { break-before: page; }
h2 { break-before: page; display: flex; align-items: center; gap: 4mm; margin: 0 0 7mm; padding-bottom: 4mm;
  border-bottom: 2px solid var(--line); font: 800 20pt/1.15 Jakarta; color: var(--navy); letter-spacing: -.015em; }
h2 .num { flex: none; display: inline-grid; place-items: center; width: 11mm; height: 11mm; border-radius: 3mm;
  background: linear-gradient(135deg, #1e8e72, #0a3a31); color: #fff; font: 800 14pt/1 Jakarta; }
h3 { margin: 7mm 0 2.5mm; font: 700 12.5pt/1.3 Jakarta; color: var(--green-d); break-after: avoid; }
h4 { margin: 5mm 0 2mm; font: 700 10.5pt/1.3 Inter; color: var(--navy); break-after: avoid; }
p { margin: 0 0 3mm; orphans: 3; widows: 3; }
ul, ol { margin: 0 0 3.5mm; padding-left: 6mm; }
li { margin: .8mm 0; }
li::marker { color: var(--green); }
strong { color: #141a2b; font-weight: 650; }
a { color: var(--navy); text-decoration: none; border-bottom: 1px solid rgba(35,34,94,.25); }
.ref { font-style: italic; }
code { font: 500 8.8pt/1.4 ui-monospace, Menlo, Consolas, monospace; background: var(--soft); border: 1px solid var(--line); border-radius: 1.2mm; padding: .2mm 1.2mm; }
pre { background: var(--soft); border: 1px solid var(--line); border-radius: 2mm; padding: 3mm 4mm; white-space: pre-wrap; break-inside: avoid; }
pre code { background: none; border: 0; padding: 0; }
table { width: 100%; border-collapse: separate; border-spacing: 0; margin: 2mm 0 5mm; font-size: 8.9pt; line-height: 1.45;
  border: 1px solid var(--line); border-radius: 2mm; overflow: hidden; }
thead { display: table-header-group; }
th { background: var(--navy); color: #fff; text-align: left; font-weight: 600; padding: 2mm 3mm; }
td { padding: 1.8mm 3mm; border-top: 1px solid var(--line); vertical-align: top; }
tr { break-inside: avoid; }
tbody tr:nth-child(even) td { background: #f8f9fc; }
blockquote { margin: 3mm 0 4mm; padding: 3mm 4mm; border-left: 1.2mm solid var(--green); background: #f0fdf4; border-radius: 0 2mm 2mm 0; break-inside: avoid; }
blockquote p:last-child { margin: 0; }
.keep { break-inside: avoid; }
figure { margin: 3mm 0 5mm; text-align: center; break-inside: avoid; }
figure img { max-width: 100%; max-height: 125mm; border: 1px solid var(--line); border-radius: 2mm; box-shadow: 0 1.5mm 5mm rgba(20,22,60,.10); }
figcaption { margin-top: 2mm; font-size: 8.3pt; color: var(--muted); }
</style></head><body>
<section class="toc"><h2>Contents</h2><ul>${toc}</ul></section>
<section class="chapters">${html}</section>
</body></html>`
}

const FOOTER = `<div style="width:100%;padding:0 17mm;display:flex;justify-content:space-between;font:500 7.5pt Inter,sans-serif;color:#8a90a2">
<span>Veridian E-invoicing Pakistan · User guide</span><span><span class="pageNumber"></span> / <span class="totalPages"></span></span></div>`

async function pdf(browser, html, file, footer) {
  const page = await browser.newPage()
  const tmp = path.join(path.dirname(file), path.basename(file, '.pdf') + '.html')
  fs.writeFileSync(tmp, html)
  await page.goto('file://' + tmp, { waitUntil: 'load' })
  await page.evaluate(() => document.fonts.ready)
  await page.pdf({
    path: file,
    format: 'A4',
    printBackground: true,
    preferCSSPageSize: true,
    displayHeaderFooter: !!footer,
    headerTemplate: '<span></span>',
    footerTemplate: footer || '<span></span>',
  })
  await page.close()
}

// optimise stores the screenshots as JPEG at print resolution, so that the
// guide stays small enough to ship with the installer.
async function optimise(browser, html, dir) {
  const page = await browser.newPage()
  const srcs = [...new Set([...html.matchAll(/<img src="([^"]+)"/g)].map((m) => m[1]))]
  for (const src of srcs) {
    const file = path.join(DOCS, src)
    if (/^(https?:|data:|file:)/.test(src) || !/\.png$/i.test(src) || !fs.existsSync(file)) continue
    const jpg = await page.evaluate(
      async ({ data, max }) => {
        const img = new Image()
        img.src = data
        await img.decode()
        const k = Math.min(1, max / img.naturalWidth)
        const c = document.createElement('canvas')
        c.width = Math.round(img.naturalWidth * k)
        c.height = Math.round(img.naturalHeight * k)
        const g = c.getContext('2d')
        g.fillStyle = '#fff'
        g.fillRect(0, 0, c.width, c.height)
        g.imageSmoothingQuality = 'high'
        g.drawImage(img, 0, 0, c.width, c.height)
        return c.toDataURL('image/jpeg', 0.86).slice('data:image/jpeg;base64,'.length)
      },
      { data: 'data:image/png;base64,' + fs.readFileSync(file).toString('base64'), max: 1300 },
    )
    const out = path.join(dir, src.replace(/[\\/]/g, '_').replace(/\.png$/i, '.jpg'))
    fs.writeFileSync(out, Buffer.from(jpg, 'base64'))
    html = html.split(`src="${src}"`).join(`src="file://${out}"`)
  }
  await page.close()
  return html
}

// pageOf finds the page on which each chapter starts (the contents take the
// first pages, so a heading's text is looked up after them).
function pageOf(file, chapters) {
  const pages = execFileSync('pdftotext', ['-layout', file, '-'], { encoding: 'utf8', maxBuffer: 64 << 20 }).split('\f')
  const norm = (s) => s.replace(/\s+/g, ' ').trim()
  let from = pages.findIndex((p, i) => i > 0 && !/Contents/.test(p)) // first page after the contents
  if (from < 1) from = 1
  const out = {}
  for (const c of chapters) {
    const want = norm(c.text.replace(/^\d+\.\s*/, ''))
    for (let i = from; i < pages.length; i++) {
      if (norm(pages[i]).includes(want)) {
        out[c.id] = i + 1
        from = i
        break
      }
    }
  }
  return out
}

;(async () => {
  const doc = manual()
  const tmp = fs.mkdtempSync(path.join(os.tmpdir(), 'guide-'))
  const browser = await chromium.launch()
  doc.html = await optimise(browser, doc.html, tmp)
  const intro = doc.intro.map((p) => marked.parseInline(p)).join(' ')
  await pdf(browser, COVER.replace('{{INTRO}}', intro), path.join(tmp, 'cover.pdf'))
  // Two passes: the second fills in the page numbers found in the first.
  await pdf(browser, bodyPage(doc, null), path.join(tmp, 'body.pdf'), FOOTER)
  const pages = pageOf(path.join(tmp, 'body.pdf'), doc.chapters)
  await pdf(browser, bodyPage(doc, pages), path.join(tmp, 'body.pdf'), FOOTER)
  await browser.close()
  execFileSync('pdfunite', [path.join(tmp, 'cover.pdf'), path.join(tmp, 'body.pdf'), OUT])
  const missing = doc.chapters.filter((c) => !pages[c.id]).map((c) => c.text)
  if (missing.length) console.warn('warning: no page found for', missing.join('; '))
  console.log('wrote', path.relative(ROOT, OUT), (fs.statSync(OUT).size / 1048576).toFixed(1) + ' MB')
})().catch((e) => {
  console.error(e)
  process.exit(1)
})
