// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.
//
// Renders the Windows artwork in ../assets from the brand mark
// (web/public/favicon.svg) and the product fonts (web/node_modules):
//   veridian.ico  program and installer icon, 16 to 256 pixels
//   wizard.bmp    installer welcome and finish pages (164 x 314, drawn at 2x)
//   header.bmp    installer page header (150 x 57, drawn at 2x)
//
// Run after "npm ci" in web/:   NODE_PATH=$(npm root -g) node packaging/windows/art/render.cjs
// (needs Playwright with Chromium).
const { chromium } = require('playwright')
const fs = require('fs')
const os = require('os')
const path = require('path')

const ROOT = path.resolve(__dirname, '../../..')
const OUT = path.resolve(__dirname, '../assets')
const MARK = fs.readFileSync(path.join(ROOT, 'web/public/favicon.svg'), 'utf8')
const font = (p) => 'file://' + path.join(ROOT, 'web/node_modules', p)
const FONTS = `
@font-face { font-family: Jakarta; font-weight: 200 800; src: url(${font('@fontsource-variable/plus-jakarta-sans/files/plus-jakarta-sans-latin-wght-normal.woff2')}); }
@font-face { font-family: Inter; font-weight: 100 900; src: url(${font('@fontsource-variable/inter/files/inter-latin-wght-normal.woff2')}); }
@font-face { font-family: Nastaliq; font-weight: 400; src: url(${font('@fontsource/noto-nastaliq-urdu/files/noto-nastaliq-urdu-arabic-400-normal.woff2')}); }
* { margin: 0; padding: 0; box-sizing: border-box; }
body { width: 100vw; height: 100vh; overflow: hidden; -webkit-font-smoothing: antialiased; }`

// The check-mark "V" of the logo, alone, for watermarks.
const V = (stroke, alpha) =>
  `<svg viewBox="0 0 100 100"><path d="M26 35 L46 70 L72 30" fill="none" stroke="rgba(255,255,255,${alpha})" stroke-width="${stroke}" stroke-linecap="round" stroke-linejoin="round"/></svg>`

const WIZARD = `<!doctype html><meta charset="utf-8"><style>${FONTS}
body { background: linear-gradient(172deg, #2b2a72 0%, #23225e 38%, #16153d 100%); position: relative; font-family: Inter; color: #fff; }
.glow { position: absolute; border-radius: 50%; filter: blur(46px); }
.g1 { width: 300px; height: 300px; left: -150px; bottom: -40px; background: rgba(22,163,74,.42); }
.g2 { width: 220px; height: 220px; right: -130px; top: -100px; background: rgba(227,179,65,.2); }
.wm { position: absolute; width: 560px; height: 560px; left: -40px; top: 210px; transform: rotate(-8deg); }
.brand { position: absolute; top: 86px; left: 0; right: 0; text-align: center; }
.brand .mark { width: 120px; height: 120px; display: block; margin: 0 auto; filter: drop-shadow(0 14px 26px rgba(0,0,0,.35)); }
.name { margin-top: 26px; font: 800 50px/1 Jakarta; letter-spacing: -.02em; }
.tag { margin-top: 12px; font: 600 21px/1 Inter; color: #86efac; letter-spacing: .01em; }
.rule { width: 46px; height: 4px; border-radius: 2px; background: #e3b341; margin: 26px auto 0; }
.ur { margin-top: 18px; font: 400 23px/2.1 Nastaliq; direction: rtl; word-spacing: .18em; color: rgba(255,255,255,.86); }
.card { position: absolute; left: 62px; top: 446px; width: 204px; height: 118px; border-radius: 16px; transform: rotate(-5deg);
  background: linear-gradient(180deg, rgba(255,255,255,.13), rgba(255,255,255,.05)); border: 1.5px solid rgba(255,255,255,.18); }
.card i { position: absolute; left: 18px; height: 8px; border-radius: 4px; background: rgba(255,255,255,.42); }
.qr { position: absolute; right: 16px; top: 18px; width: 48px; height: 48px; display: grid; grid-template-columns: repeat(6, 1fr); gap: 2px; }
.qr b { background: rgba(255,255,255,.55); border-radius: 1px; }
.qr b.o { background: transparent; }
.ok { position: absolute; right: -16px; bottom: -16px; width: 46px; height: 46px; border-radius: 50%; background: #16a34a; border: 3px solid #1f1e55;
  display: grid; place-items: center; box-shadow: 0 8px 18px rgba(0,0,0,.3); }
.foot { position: absolute; bottom: 26px; left: 0; right: 0; text-align: center; font: 700 15px/1 Inter; letter-spacing: .2em; color: rgba(255,255,255,.62); }
</style>
<div class="glow g1"></div><div class="glow g2"></div>
<div class="wm">${V(9, 0.05)}</div>
<div class="brand">
  <div class="mark">${MARK}</div>
  <div class="name">Veridian</div>
  <div class="tag">E-invoicing Pakistan</div>
  <div class="rule"></div>
  <div class="ur">ایف بی آر ڈیجیٹل انوائسنگ</div>
</div>
<div class="card">
  <i style="top:20px;width:82px;background:rgba(134,239,172,.75)"></i>
  <i style="top:42px;width:104px"></i><i style="top:60px;width:92px"></i><i style="top:78px;width:68px"></i>
  <i style="top:96px;width:50px;background:rgba(227,179,65,.85)"></i>
  <div class="qr">${'101101011010110011001101101110011011'.split('').map((c) => `<b class="${c === '1' ? '' : 'o'}"></b>`).join('')}</div>
  <div class="ok"><svg width="22" height="22" viewBox="0 0 24 24"><path d="M5 12.5l4.5 4.5L19 7.5" fill="none" stroke="#fff" stroke-width="3.4" stroke-linecap="round" stroke-linejoin="round"/></svg></div>
</div>
<div class="foot">FBR DIGITAL INVOICING</div>`

const HEADER = `<!doctype html><meta charset="utf-8"><style>${FONTS}
body { background: #fff; display: flex; align-items: center; justify-content: flex-end; padding-right: 12px; gap: 12px; }
.mark { width: 64px; height: 64px; }
.name { font: 800 31px/1 Jakarta; color: #23225e; letter-spacing: -.02em; }
.tag { margin-top: 7px; font: 600 15px/1 Inter; color: #15803d; }
</style>
<div class="mark">${MARK}</div>
<div><div class="name">Veridian</div><div class="tag">E-invoicing Pakistan</div></div>`

// bmp24 encodes RGBA pixels as a 24-bit Windows bitmap (what NSIS needs).
function bmp24(w, h, rgba) {
  const row = Math.ceil((w * 3) / 4) * 4
  const b = Buffer.alloc(54 + row * h)
  b.write('BM', 0)
  b.writeUInt32LE(b.length, 2)
  b.writeUInt32LE(54, 10)
  b.writeUInt32LE(40, 14)
  b.writeInt32LE(w, 18)
  b.writeInt32LE(h, 22)
  b.writeUInt16LE(1, 26)
  b.writeUInt16LE(24, 28)
  b.writeUInt32LE(row * h, 34)
  b.writeInt32LE(3780, 38)
  b.writeInt32LE(3780, 42)
  for (let y = 0; y < h; y++) {
    const o = 54 + (h - 1 - y) * row
    for (let x = 0; x < w; x++) {
      const i = (y * w + x) * 4
      b[o + x * 3] = rgba[i + 2]
      b[o + x * 3 + 1] = rgba[i + 1]
      b[o + x * 3 + 2] = rgba[i]
    }
  }
  return b
}

// ico encodes icon images: 32-bit bitmaps up to 128 pixels and PNG at 256.
function ico(images) {
  const data = images.map(({ size, rgba, png }) => {
    if (png) return png
    const mask = Math.ceil(size / 32) * 4 * size
    const b = Buffer.alloc(40 + size * size * 4 + mask)
    b.writeUInt32LE(40, 0)
    b.writeInt32LE(size, 4)
    b.writeInt32LE(size * 2, 8)
    b.writeUInt16LE(1, 12)
    b.writeUInt16LE(32, 14)
    b.writeUInt32LE(size * size * 4 + mask, 20)
    for (let y = 0; y < size; y++) {
      for (let x = 0; x < size; x++) {
        const i = (y * size + x) * 4
        const o = 40 + ((size - 1 - y) * size + x) * 4
        b[o] = rgba[i + 2]
        b[o + 1] = rgba[i + 1]
        b[o + 2] = rgba[i]
        b[o + 3] = rgba[i + 3]
      }
    }
    return b
  })
  const head = Buffer.alloc(6 + 16 * images.length)
  head.writeUInt16LE(1, 2)
  head.writeUInt16LE(images.length, 4)
  let off = head.length
  images.forEach(({ size }, n) => {
    const e = 6 + n * 16
    head[e] = size >= 256 ? 0 : size
    head[e + 1] = size >= 256 ? 0 : size
    head.writeUInt16LE(1, e + 4)
    head.writeUInt16LE(32, e + 6)
    head.writeUInt32LE(data[n].length, e + 8)
    head.writeUInt32LE(off, e + 12)
    off += data[n].length
  })
  return Buffer.concat([head, ...data])
}

// pixels decodes a PNG into RGBA through a canvas.
async function pixels(page, png, w, h) {
  const arr = await page.evaluate(
    async ({ src, w, h }) => {
      const img = new Image()
      img.src = src
      await img.decode()
      const c = document.createElement('canvas')
      c.width = w
      c.height = h
      const g = c.getContext('2d')
      g.drawImage(img, 0, 0)
      return Array.from(g.getImageData(0, 0, w, h).data)
    },
    { src: 'data:image/png;base64,' + png.toString('base64'), w, h },
  )
  return Buffer.from(arr)
}

async function picture(browser, html, w, h) {
  const page = await browser.newPage({ viewport: { width: w, height: h }, deviceScaleFactor: 1 })
  const file = path.join(fs.mkdtempSync(path.join(os.tmpdir(), 'art-')), 'page.html')
  fs.writeFileSync(file, html)
  await page.goto('file://' + file)
  await page.evaluate(() => document.fonts.ready)
  const png = await page.screenshot({ type: 'png' })
  const rgba = await pixels(page, png, w, h)
  await page.close()
  return { png, rgba }
}

;(async () => {
  const browser = await chromium.launch()
  fs.mkdirSync(OUT, { recursive: true })

  // Icon: the mark drawn at each size, so small sizes stay sharp.
  const page = await browser.newPage()
  const icons = []
  for (const size of [16, 20, 24, 32, 40, 48, 64, 96, 128, 256]) {
    const out = await page.evaluate(
      async ({ svg, size }) => {
        const img = new Image()
        img.src = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svg)
        await img.decode()
        const c = document.createElement('canvas')
        c.width = c.height = size
        c.getContext('2d').drawImage(img, 0, 0, size, size)
        return { rgba: Array.from(c.getContext('2d').getImageData(0, 0, size, size).data), png: c.toDataURL('image/png').slice(22) }
      },
      { svg: MARK, size },
    )
    icons.push(size === 256 ? { size, png: Buffer.from(out.png, 'base64') } : { size, rgba: Buffer.from(out.rgba) })
  }
  fs.writeFileSync(path.join(OUT, 'veridian.ico'), ico(icons))

  const wizard = await picture(browser, WIZARD, 328, 628)
  fs.writeFileSync(path.join(OUT, 'wizard.bmp'), bmp24(328, 628, wizard.rgba))
  const header = await picture(browser, HEADER, 300, 114)
  fs.writeFileSync(path.join(OUT, 'header.bmp'), bmp24(300, 114, header.rgba))
  if (process.env.ART_PREVIEW) {
    fs.writeFileSync(path.join(process.env.ART_PREVIEW, 'wizard.png'), wizard.png)
    fs.writeFileSync(path.join(process.env.ART_PREVIEW, 'header.png'), header.png)
  }
  await browser.close()
  console.log('wrote', fs.readdirSync(OUT).join(', '))
})().catch((e) => {
  console.error(e)
  process.exit(1)
})
