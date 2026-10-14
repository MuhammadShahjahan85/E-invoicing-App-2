# Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
# Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.
"""Builds internal/hscodes/pct.tsv.gz, the HS (PCT) code list built into the
software, from FBR's own publications:

  --tariff     the current Pakistan Customs Tariff as text
               (fbr.gov.pk > Customs Tariff > Pakistan Customs Tariff, PDF,
               converted with: pdftotext -layout Tariff-2026-27.pdf tariff.txt)
  --previous   earlier tariffs as text (descriptions of codes that FBR's
               IRIS list still accepts but the current tariff dropped)
  --services   FBR's Customs Tariff chapter 98 (services) as text
  --iris       FBR's IRIS Annex-H template (e.fbr.gov.pk/SOP/IRIS/
               ST_AnnexH_Template.xlsm), whose REFERENCES sheet lists every
               HS code IRIS accepts

Each line of the output is: code, description, context (the heading and the
groups above the code), flags (T = in the current tariff, I = in FBR's IRIS
list, S = from the services chapter, P = described from an earlier tariff).

  python3 -I scripts/hscodes/build.py --tariff tariff.txt \\
      --previous tariff-2025-26.txt --previous tariff-2024-25.txt \\
      --services chapter98.txt --iris ST_AnnexH_Template.xlsm \\
      --version "2026-27" --out internal/hscodes/pct.tsv.gz
"""
import argparse
import gzip
import re
import sys
import zipfile
import xml.etree.ElementTree as ET

# In the tariff, codes and headings start a line; an indented code is a
# cross-reference inside a description. The services chapter is indented.
CODE = re.compile(r'^(\d{4}\.\d{4})\s+(.*)$')
HEADING = re.compile(r'^(\d{2}\.\d{2})\s+(.*)$')
SCODE = re.compile(r'^\s*(\d{4}\.\d{4})\s+(.*)$')
SHEADING = re.compile(r'^\s*(\d{2}\.\d{2})\s+(.*)$')
DASHES = re.compile(r'^((?:-\s*)+)(.*)$')
NOISE = re.compile(r'(PAKISTAN CUSTOMS TARIFF|PCT CODE|^\s*CD \(%\)|^\s*Heading\s+Description|^\s*\d{1,4}\s*$|ST=Sales Tax|MY = Malaysia|^\s*Chapter \d+|^\s*\(SERVICES\))')


def clean(s):
    s = re.sub(r'\s+', ' ', s).strip()
    s = s.replace(' ,', ',').replace('( ', '(').replace(' )', ')')
    s = re.sub(r'(\w)- (?=[a-z])', r'\1-', s)  # "Pure- bred" -> "Pure-bred"
    return s


def split_cols(rest):
    """Splits a line's text from the right-hand rate column (2+ spaces)."""
    parts = re.split(r'\s{2,}', rest.strip())
    return parts[0] if parts else ''


def parse_tariff(path, stop_at_chapter=99):
    """Returns {code: (description, context)} from a tariff in text form."""
    out = {}
    heading = ''
    stack = {}
    last = None  # ('code', code) | ('heading',) | ('group', level)
    with open(path, encoding='utf-8', errors='replace') as f:
        lines = f.read().split('\n')
    for raw in lines:
        raw = raw.lstrip('\f')  # pdftotext starts each page with a form feed
        if not raw.strip() or NOISE.search(raw):
            last = None if not raw.strip() else last
            continue
        m = CODE.match(raw)
        if m:
            code, rest = m.group(1), split_cols(m.group(2))
            if int(code[:2]) >= stop_at_chapter:
                break
            d = DASHES.match(rest)
            level = d.group(1).count('-') if d else 0
            text = clean(d.group(2) if d else rest)
            ctx = [heading] + [stack[k] for k in sorted(stack) if k < level]
            out[code] = [text, ' › '.join(c for c in ctx if c)]
            last = ('code', code)
            continue
        m = HEADING.match(raw)
        if m:
            if int(m.group(1)[:2]) >= stop_at_chapter:
                break
            heading = clean(split_cols(m.group(2))).rstrip('.')
            stack = {}
            last = ('heading',)
            continue
        text = split_cols(raw)
        d = DASHES.match(text.strip())
        if d and d.group(2).strip():
            level = d.group(1).count('-')
            label = clean(d.group(2)).rstrip(':').strip()
            stack = {k: v for k, v in stack.items() if k < level}
            stack[level] = label
            last = ('group', level)
            continue
        # A continuation of the line above.
        more = clean(text)
        if not more or re.fullmatch(r'[\d.%]+', more):
            continue
        if last and last[0] == 'code':
            out[last[1]][0] = clean(out[last[1]][0] + ' ' + more)
        elif last and last[0] == 'heading':
            heading = clean(heading + ' ' + more).rstrip('.')
        elif last and last[0] == 'group':
            lv = last[1]
            stack[lv] = clean(stack.get(lv, '') + ' ' + more).rstrip(':').strip()
    # Context of codes parsed before a heading's continuation was complete
    # can carry a cut heading; that is acceptable for search and display.
    return {k: (v[0], v[1]) for k, v in out.items()}


def parse_services(path):
    """Chapter 98: {code: (description, heading)}."""
    out = {}
    heading = ''
    last = None
    with open(path, encoding='utf-8', errors='replace') as f:
        for raw in f.read().split('\n'):
            raw = raw.lstrip('\f')
            if not raw.strip() or NOISE.search(raw):
                continue
            m = SCODE.match(raw)
            if m and m.group(1).startswith('98'):
                out[m.group(1)] = [clean(split_cols(m.group(2))).rstrip('.'), heading]
                last = ('code', m.group(1))
                continue
            m = SHEADING.match(raw)
            if m and m.group(1).startswith('98'):
                heading = clean(split_cols(m.group(2))).rstrip('.')
                last = ('heading',)
                continue
            more = clean(split_cols(raw))
            if not more or not last:
                continue
            if last[0] == 'code':
                out[last[1]][0] = clean(out[last[1]][0] + ' ' + more).rstrip('.')
            else:
                heading = clean(heading + ' ' + more).rstrip('.')
    return {k: (v[0], v[1]) for k, v in out.items()}


def parse_iris(path):
    """FBR IRIS Annex-H template: (set of codes, {heading4: description})."""
    ns = {'m': 'http://schemas.openxmlformats.org/spreadsheetml/2006/main',
          'r': 'http://schemas.openxmlformats.org/officeDocument/2006/relationships'}
    z = zipfile.ZipFile(path)
    wb = ET.fromstring(z.read('xl/workbook.xml'))
    rels = {r.get('Id'): r.get('Target') for r in ET.fromstring(z.read('xl/_rels/workbook.xml.rels'))}
    target = None
    for s in wb.find('m:sheets', ns):
        if s.get('name') == 'REFERENCES':
            target = rels[s.get('{%s}id' % ns['r'])]
    if target is None:
        sys.exit('REFERENCES sheet not found')
    shared = [''.join(t.text or '' for t in si.iter('{%s}t' % ns['m']))
              for si in ET.fromstring(z.read('xl/sharedStrings.xml')).findall('m:si', ns)]
    sheet = ET.fromstring(z.read('xl/' + target.replace('/xl/', '').lstrip('/')))
    codes, headings = set(), {}
    for c in sheet.iter('{%s}c' % ns['m']):
        v = c.find('m:v', ns)
        if v is None:
            continue
        val = shared[int(v.text)] if c.get('t') == 's' else v.text
        m = re.match(r'^(\d{4}\.\d{4}):-', val or '')
        if m:
            codes.add(m.group(1))
            continue
        m = re.match(r'^(\d{4}):-(.+)$', val or '')
        if m:
            headings[m.group(1)] = clean(m.group(2))
    return codes, headings


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--tariff', required=True)
    ap.add_argument('--previous', action='append', default=[])
    ap.add_argument('--services')
    ap.add_argument('--iris', required=True)
    ap.add_argument('--version', required=True)
    ap.add_argument('--out', required=True)
    a = ap.parse_args()

    current = parse_tariff(a.tariff)
    previous = [parse_tariff(p) for p in a.previous]
    services = parse_services(a.services) if a.services else {}
    iris, iris_headings = parse_iris(a.iris)

    rows = []
    stats = {'T': 0, 'I': 0, 'S': 0, 'P': 0, 'H': 0}
    for code in sorted(set(current) | iris):
        flags = ''
        if code in current:
            desc, ctx = current[code]
            flags += 'T'
        elif code in services:
            desc, ctx = services[code]
            flags += 'S'
        else:
            desc, ctx = '', ''
            for p in previous:
                if code in p:
                    desc, ctx = p[code]
                    flags += 'P'
                    break
        if code in iris:
            flags += 'I'
        if not desc:
            # Only the four-digit heading is described anywhere.
            desc = iris_headings.get(code[:4], '')
            if code[5:] != '0000':
                desc = desc + ' (subheading ' + code + ')' if desc else 'Subheading ' + code
            stats['H'] += 1
        if not ctx and code[:4] in iris_headings and iris_headings[code[:4]] != desc:
            ctx = iris_headings[code[:4]]
        for f in flags:
            stats[f] = stats.get(f, 0) + 1
        rows.append('\t'.join([code, desc.replace('\t', ' '), ctx.replace('\t', ' '), flags]))

    body = '# Pakistan Customs Tariff ' + a.version + ' and FBR IRIS HS code list\n' + '\n'.join(rows) + '\n'
    with gzip.open(a.out, 'wt', encoding='utf-8', compresslevel=9) as f:
        f.write(body)
    print(f'{len(rows)} codes written to {a.out}: {stats}')


if __name__ == '__main__':
    main()
