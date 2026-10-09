// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Import data: a wizard that reads a sales register in any format, matches
// its columns to invoice particulars, asks for whatever FBR needs that the
// file does not hold, checks every invoice and then imports them.

import { useRef, useState, type DragEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  ArrowLeft,
  ArrowRight,
  Check,
  CircleAlert,
  CircleCheck,
  FileSpreadsheet,
  Info,
  ListChecks,
  RotateCcw,
  Sparkles,
  TriangleAlert,
  UploadCloud,
  Wand2,
} from 'lucide-react'
import { api } from '../api'
import { ErrorBox, HSCodeInput, Modal, StatusBadge } from '../components/ui'
import { money } from '../format'
import { useCompanyPath, useSession, useToast } from '../state'
import type { ImportAnalysis, ImportField, ImportMissing, ImportOptions, ImportSummary } from '../types'

type Step = 'file' | 'match' | 'missing' | 'check' | 'done'

const stepList: { key: Step; label: string }[] = [
  { key: 'file', label: 'Choose file' },
  { key: 'match', label: 'Match columns' },
  { key: 'missing', label: 'Missing details' },
  { key: 'check', label: 'Check & import' },
]

const formats = ['Excel XLSX', 'Excel XLS', 'CSV', 'TSV / TXT', 'OpenDocument', 'JSON', 'FBR DI JSON', 'XML', 'Word DOCX', 'HTML', 'PDF']
const accept = '.xlsx,.xlsm,.xls,.csv,.tsv,.tab,.txt,.prn,.dat,.ods,.json,.xml,.docx,.html,.htm,.pdf'

type Choices = Omit<ImportOptions, 'sheet' | 'headerRow' | 'mapping'>
const freshChoices = (): Choices => ({ defaults: {}, grouping: '', provinceFromAddress: true, regRule: 'ntn', valueMap: {} })

const groupLabels: Record<string, string> = {
  row: 'Each line is a separate invoice',
  'buyer-date': 'Lines with the same buyer and date form one invoice',
  single: 'All lines form one invoice',
}

export default function ImportPage() {
  const s = useSession()
  const cp = useCompanyPath()
  const toast = useToast()
  const navigate = useNavigate()
  const [step, setStep] = useState<Step>('file')
  const [file, setFile] = useState<File | null>(null)
  const [busy, setBusy] = useState('')
  const [error, setError] = useState<unknown>()
  const [a, setA] = useState<ImportAnalysis | null>(null)
  const [mapping, setMapping] = useState<Record<string, number>>({})
  const [choices, setChoices] = useState<Choices>(freshChoices)
  const [summary, setSummary] = useState<ImportSummary | null>(null)
  const [result, setResult] = useState<ImportSummary | null>(null)
  const [submit, setSubmit] = useState(false)
  const [prompt, setPrompt] = useState(false)
  const [over, setOver] = useState(false)
  const picker = useRef<HTMLInputElement>(null)

  const missing = a?.missing ?? []
  const env = s.company?.environment ?? 'simulator'

  /** analyze reads the file on the server and works out the matching and what is missing. */
  const analyze = async (f: File, extra: { sheet?: number; headerRow?: number; mapping?: Record<string, number> } = {}) => {
    setBusy('analyze')
    setError(undefined)
    try {
      const fd = new FormData()
      fd.append('file', f)
      if (extra.sheet !== undefined) fd.append('sheet', String(extra.sheet))
      if (extra.headerRow !== undefined) fd.append('headerRow', String(extra.headerRow))
      if (extra.mapping) fd.append('mapping', JSON.stringify(extra.mapping))
      const res = await api.upload<ImportAnalysis>(`${cp}/import/analyze`, 'POST', fd, '')
      setA(res)
      if (!extra.mapping) setMapping(res.mapping ?? {})
      // Suggested answers for what is missing; the user's own answers win.
      const suggested: Record<string, string> = {}
      let grouping = ''
      for (const m of res.missing ?? []) {
        if (m.input === 'grouping') grouping = m.suggest
        else if (m.input && m.input !== 'valuemap' && m.input !== 'regrule' && m.suggest) suggested[m.field] = m.suggest
      }
      setChoices((c) => ({ ...c, defaults: { ...suggested, ...c.defaults }, grouping: c.grouping || grouping }))
      return res
    } catch (e) {
      setError(e)
      return null
    } finally {
      setBusy('')
    }
  }

  const pick = async (f: File | undefined | null) => {
    if (!f) return
    setFile(f)
    setSummary(null)
    setResult(null)
    setChoices(freshChoices())
    setA(null)
    if (await analyze(f)) setStep('match')
  }

  const run = async (preview: boolean) => {
    if (!file || !a) return null
    setBusy(preview ? 'check' : 'import')
    setError(undefined)
    try {
      const fd = new FormData()
      fd.append('file', file)
      fd.append('options', JSON.stringify({ sheet: a.sheet, headerRow: a.headerRow, mapping, ...choices }))
      const q = preview ? '?preview=1' : submit ? '?submit=1' : ''
      return await api.upload<ImportSummary>(`${cp}/import${q}`, 'POST', fd, '')
    } catch (e) {
      setError(e)
      return null
    } finally {
      setBusy('')
    }
  }

  /** afterMatching re-checks what is missing for the user's matching and asks for it. */
  const afterMatching = async () => {
    if (!file || !a) return
    const res = await analyze(file, { sheet: a.sheet, headerRow: a.headerRow, mapping })
    if (!res) return
    if ((res.missing ?? []).length > 0) {
      setStep('missing')
      setPrompt(true)
    } else {
      await check()
    }
  }

  const check = async () => {
    const res = await run(true)
    if (res) {
      setSummary(res)
      setStep('check')
    }
  }

  const doImport = async () => {
    const res = await run(false)
    if (res) {
      setResult(res)
      setStep('done')
      toast(res.failed ? 'info' : 'ok', `${res.ok} of ${res.invoices} invoice${res.invoices === 1 ? '' : 's'} imported`)
    }
  }

  const restart = () => {
    setStep('file')
    setFile(null)
    setA(null)
    setSummary(null)
    setResult(null)
    setChoices(freshChoices())
    setError(undefined)
  }

  const blocking = missing.filter((m) => m.level === 'fix' && m.input === '' && m.field === 'unit_price')
  const stepIndex = stepList.findIndex((x) => x.key === step)

  return (
    <>
      <div className="page-head">
        <div>
          <div className="eyebrow">
            <Sparkles size={14} /> Smart import
          </div>
          <h1>Import invoices from any file</h1>
          <p>
            Upload your sales register as it is — Excel, CSV, text, JSON, XML, Word, web page or PDF. The columns are recognised for you, anything FBR needs
            that the file does not hold is asked for, and every invoice is checked before it is saved.
          </p>
        </div>
        <div className="actions">
          <a className="btn" href="/api/v1/import/template.xlsx" title="An Excel sheet with every column the import understands">
            <FileSpreadsheet size={16} /> Excel template
          </a>
        </div>
      </div>

      <div className="wizard-steps" aria-label="Import steps">
        {stepList.map((st, i) => (
          <span key={st.key} className={'ws' + (i === stepIndex || (step === 'done' && i === 3) ? ' on' : '') + (i < stepIndex || step === 'done' ? ' done' : '')}>
            <i>{i < stepIndex || step === 'done' ? <Check size={13} strokeWidth={3} /> : i + 1}</i>
            {st.label}
          </span>
        ))}
      </div>

      <ErrorBox error={error} />

      {step === 'file' && (
        <div className="grid g-2-1">
          <div className="card card-pad">
            <div
              className={'dropzone' + (over ? ' over' : '')}
              role="button"
              tabIndex={0}
              onClick={() => picker.current?.click()}
              onKeyDown={(e) => (e.key === 'Enter' || e.key === ' ') && picker.current?.click()}
              onDragOver={(e: DragEvent) => {
                e.preventDefault()
                setOver(true)
              }}
              onDragLeave={() => setOver(false)}
              onDrop={(e: DragEvent) => {
                e.preventDefault()
                setOver(false)
                pick(e.dataTransfer.files?.[0])
              }}
            >
              <span className="dz-ic">{busy === 'analyze' ? <span className="spinner" /> : <UploadCloud size={28} />}</span>
              <b>{busy === 'analyze' ? `Reading ${file?.name ?? 'your file'}…` : 'Drop your file here, or click to choose'}</b>
              <span className="small">Sales registers, invoice lists and ERP exports — up to 20 MB and 20,000 lines</span>
              <div className="format-chips">
                {formats.map((f) => (
                  <span key={f}>{f}</span>
                ))}
              </div>
              <input
                ref={picker}
                type="file"
                accept={accept}
                hidden
                aria-label="Choose a file to import"
                onChange={(e) => {
                  pick(e.target.files?.[0])
                  e.target.value = ''
                }}
              />
            </div>
          </div>
          <div className="card">
            <div className="card-head">
              <h3>
                <Wand2 size={17} /> How it works
              </h3>
            </div>
            <div className="card-body">
              <ol className="steps">
                <li>Choose the file exactly as your software exports it — titles, blank lines and totals rows are skipped.</li>
                <li>Check how its columns were matched; change any you like.</li>
                <li>Fill in what FBR needs that the file does not hold, such as the province or the sale type.</li>
                <li>Every invoice is checked against FBR's rules before anything is saved; then import them as drafts or submit them.</li>
              </ol>
              <p className="small muted" style={{ marginTop: 10 }}>
                Invoices already imported (same invoice number) are not created twice, so a corrected file can be uploaded again.
              </p>
            </div>
          </div>
        </div>
      )}

      {step === 'match' && a && (
        <MatchStep
          a={a}
          mapping={mapping}
          setMapping={setMapping}
          busy={busy}
          onReread={(extra) => file && analyze(file, extra).then((r) => r && setMapping(r.mapping ?? {}))}
          onBack={restart}
          onNext={afterMatching}
        />
      )}

      {step === 'missing' && a && (
        <>
          <div className="card">
            <div className="card-head">
              <h3>
                <ListChecks size={17} /> Details your file does not hold
              </h3>
              <span className="sub">{a.fileName}</span>
            </div>
            <div className="card-body stack" style={{ gap: 14 }}>
              {missing.map((m, i) => (
                <MissingCard key={m.field + m.input + i} m={m} a={a} choices={choices} setChoices={setChoices} cp={cp} />
              ))}
            </div>
          </div>
          <div className="wizard-bar">
            <button className="btn" onClick={() => setStep('match')}>
              <ArrowLeft size={16} /> Back to matching
            </button>
            <span className="spacer" />
            {blocking.length > 0 && <span className="small" style={{ color: 'var(--danger)' }}>Match a price or value column to continue.</span>}
            <button className="btn btn-primary" disabled={!!busy || blocking.length > 0} onClick={check}>
              {busy === 'check' ? 'Checking every invoice…' : 'Check the invoices'} <ArrowRight size={16} />
            </button>
          </div>
        </>
      )}

      {step === 'check' && summary && (
        <>
          <ResultTiles summary={summary} />
          <ResultTable summary={summary} />
          <div className="wizard-bar">
            <button className="btn" onClick={() => setStep(missing.length ? 'missing' : 'match')}>
              <ArrowLeft size={16} /> Back
            </button>
            <span className="spacer" />
            {s.can('invoice.write') && (
              <label className="check">
                <input type="checkbox" checked={submit} onChange={(e) => setSubmit(e.target.checked)} /> Submit each invoice to FBR right away
                {env !== 'production' ? ` (${env === 'sandbox' ? 'FBR sandbox' : 'training simulator'})` : ''}
              </label>
            )}
            <button className="btn btn-primary" disabled={!!busy || summary.invoices === 0} onClick={doImport}>
              {busy === 'import' ? 'Importing…' : `${submit ? 'Import & submit' : 'Import'} ${summary.invoices} invoice${summary.invoices === 1 ? '' : 's'}`}
            </button>
          </div>
          {submit && env === 'production' && (
            <div className="alert alert-warn mt">
              <TriangleAlert size={18} />
              <div className="alert-body">These invoices will be reported to FBR production as real tax invoices. Invoices with errors stay as drafts for you to correct.</div>
            </div>
          )}
        </>
      )}

      {step === 'done' && result && (
        <>
          <div className="card card-pad done-card">
            <span className="done-ic">
              <CircleCheck size={30} />
            </span>
            <div>
              <h2>
                {result.ok} of {result.invoices} invoice{result.invoices === 1 ? '' : 's'} imported
              </h2>
              <p className="muted">
                {submit ? 'Accepted invoices carry their FBR invoice numbers; any that FBR rejected are listed below.' : 'They are saved as drafts: open each one to check it and submit it to FBR.'}
                {result.failed > 0 && ` ${result.failed} could not be imported — see the reasons below.`}
              </p>
            </div>
            <div className="actions" style={{ marginLeft: 'auto' }}>
              <button className="btn" onClick={restart}>
                <RotateCcw size={16} /> Import another file
              </button>
              <button className="btn btn-primary" onClick={() => navigate(submit ? '/invoices' : '/invoices?status=DRAFT,VALIDATED')}>
                Open the invoices <ArrowRight size={16} />
              </button>
            </div>
          </div>
          <ResultTable summary={result} />
        </>
      )}

      {prompt && a && (
        <Modal
          title="Some details FBR needs are not in your file"
          onClose={() => setPrompt(false)}
          footer={
            <>
              <button
                className="btn"
                onClick={() => {
                  setPrompt(false)
                  setStep('match')
                }}
              >
                Change the matching
              </button>
              <button className="btn btn-primary" onClick={() => setPrompt(false)}>
                Fill them in
              </button>
            </>
          }
        >
          <p className="muted" style={{ marginTop: 0 }}>
            {a.fileName} was read ({a.format}, {a.rows} line{a.rows === 1 ? '' : 's'}), but FBR also needs the following. You can supply them on the next screen
            — once for the whole file.
          </p>
          <ul className="miss-list">
            {missing.map((m, i) => (
              <li key={i} className={m.level}>
                {m.level === 'fix' ? <CircleAlert size={17} /> : m.level === 'required' ? <TriangleAlert size={17} /> : <Info size={17} />}
                <span>
                  <b>{m.label}</b>
                  {m.blank > 0 && m.blank < a.rows ? ` · ${m.blank} of ${a.rows} lines` : ''}
                  <small>{m.message}</small>
                </span>
              </li>
            ))}
          </ul>
        </Modal>
      )}
    </>
  )
}

/** MatchStep shows each column of the file and what it is used for. */
function MatchStep({
  a,
  mapping,
  setMapping,
  busy,
  onReread,
  onBack,
  onNext,
}: {
  a: ImportAnalysis
  mapping: Record<string, number>
  setMapping: (m: Record<string, number>) => void
  busy: string
  onReread: (extra: { sheet?: number; headerRow?: number }) => void
  onBack: () => void
  onNext: () => void
}) {
  const fieldOf = (col: number) => Object.keys(mapping).find((f) => mapping[f] === col) ?? ''
  const setField = (col: number, field: string) => {
    const next: Record<string, number> = {}
    for (const [f, c] of Object.entries(mapping)) {
      if (c !== col && f !== field) next[f] = c
    }
    if (field) next[field] = col
    setMapping(next)
  }
  const groups = ['Invoice', 'Buyer', 'Item', 'Tax'] as const
  const needed = a.fields.filter((f) => f.need !== 'optional')
  const colName = (i: number) => {
    let n = i + 1
    let out = ''
    while (n > 0) {
      out = String.fromCharCode(65 + ((n - 1) % 26)) + out
      n = Math.floor((n - 1) / 26)
    }
    return out
  }
  return (
    <>
      <div className="card mb">
        <div className="card-body row" style={{ gap: 14 }}>
          <span className="chip">
            <FileSpreadsheet size={14} /> {a.fileName}
          </span>
          <span className="small muted">
            Read as {a.format} · {a.rows} line{a.rows === 1 ? '' : 's'}
          </span>
          <span className="spacer" />
          {a.sheets.length > 1 && (
            <label className="row small" style={{ gap: 8 }}>
              Table
              <select value={a.sheet} onChange={(e) => onReread({ sheet: Number(e.target.value) })} style={{ width: 'auto' }}>
                {a.sheets.map((sh, i) => (
                  <option key={i} value={i}>
                    {sh.name} ({sh.rows} rows)
                  </option>
                ))}
              </select>
            </label>
          )}
          <label className="row small" style={{ gap: 8 }}>
            Headings are on row
            <select value={a.headerRow} onChange={(e) => onReread({ sheet: a.sheet, headerRow: Number(e.target.value) })} style={{ width: 'auto' }}>
              {Array.from({ length: Math.max(a.headerRow + 1, 15) }, (_, i) => (
                <option key={i} value={i}>
                  {i + 1}
                </option>
              ))}
            </select>
          </label>
        </div>
        {(a.notes ?? []).map((n, i) => (
          <div key={i} className="card-foot row" style={{ gap: 8 }}>
            <Info size={14} /> {n}
          </div>
        ))}
      </div>

      <div className="grid g-2-1">
        <div className="card">
          <div className="card-head">
            <h3>
              <Wand2 size={17} /> Columns in your file
            </h3>
            <span className="sub">
              {Object.keys(mapping).length} of {a.columns.length} used
            </span>
          </div>
          <div className="table-wrap">
            <table className="table map-table">
              <thead>
                <tr>
                  <th>Column</th>
                  <th>Examples</th>
                  <th style={{ width: '38%' }}>Use as</th>
                </tr>
              </thead>
              <tbody>
                {a.columns.map((c) => {
                  const f = fieldOf(c.index)
                  const auto = f !== '' && f === c.field
                  return (
                    <tr key={c.index} className={'map-row' + (f ? '' : ' unused')}>
                      <td>
                        <div className="cell-title">{c.header}</div>
                        <div className="cell-sub">Column {colName(c.index)}</div>
                      </td>
                      <td>
                        <div className="map-sample" title={(c.samples ?? []).join(' · ')}>
                          {(c.samples ?? []).slice(0, 3).join(' · ') || '—'}
                        </div>
                      </td>
                      <td>
                        <select value={f} onChange={(e) => setField(c.index, e.target.value)} aria-label={`Use column ${c.header} as`}>
                          <option value="">Don't import</option>
                          {groups.map((g) => (
                            <optgroup key={g} label={g}>
                              {a.fields
                                .filter((x) => x.group === g)
                                .map((x) => (
                                  <option key={x.key} value={x.key}>
                                    {x.label}
                                    {x.need === 'required' ? ' *' : ''}
                                  </option>
                                ))}
                            </optgroup>
                          ))}
                        </select>
                        {f && (
                          <div className="cell-sub" style={{ marginTop: 4 }}>
                            {auto ? (c.by === 'values' ? 'Recognised from its values' : 'Recognised from the heading') : 'Your choice'}
                          </div>
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
        <div className="card">
          <div className="card-head">
            <h3>
              <ListChecks size={17} /> What FBR needs
            </h3>
          </div>
          <ul className="need-list">
            {needed.map((f: ImportField) => {
              const col = mapping[f.key]
              const has = col !== undefined
              return (
                <li key={f.key} className={has ? 'ok' : f.need}>
                  {has ? <CircleCheck size={16} /> : f.need === 'required' ? <TriangleAlert size={16} /> : <Info size={16} />}
                  <span>
                    <b>{f.label}</b>
                    <small>{has ? `from “${a.columns[col]?.header}”` : f.need === 'required' ? 'not in the file — you will be asked' : 'not in the file'}</small>
                  </span>
                </li>
              )
            })}
          </ul>
        </div>
      </div>

      <div className="wizard-bar">
        <button className="btn" onClick={onBack}>
          <ArrowLeft size={16} /> Choose another file
        </button>
        <span className="spacer" />
        <button className="btn btn-primary" disabled={!!busy} onClick={onNext}>
          {busy ? 'Working…' : 'Continue'} <ArrowRight size={16} />
        </button>
      </div>
    </>
  )
}

/** MissingCard asks for one missing detail. */
function MissingCard({
  m,
  a,
  choices,
  setChoices,
  cp,
}: {
  m: ImportMissing
  a: ImportAnalysis
  choices: Choices
  setChoices: (fn: (c: Choices) => Choices) => void
  cp: string
}) {
  const value = choices.defaults[m.field] ?? ''
  const setDefault = (v: string) => setChoices((c) => ({ ...c, defaults: { ...c.defaults, [m.field]: v } }))
  const list = (xs: string[] | null | undefined, v: string, set: (v: string) => void, label: string) => (
    <select value={v} onChange={(e) => set(e.target.value)} aria-label={label}>
      <option value="">— choose —</option>
      {(xs ?? []).map((x) => (
        <option key={x} value={x}>
          {x}
        </option>
      ))}
    </select>
  )
  const valueList = m.field === 'uom' ? a.lists.uoms : m.field === 'sale_type' ? a.lists.saleTypes : a.lists.provinces
  return (
    <div className={'miss-card ' + m.level}>
      <span className="miss-ic">{m.level === 'fix' ? <CircleAlert size={18} /> : m.level === 'required' ? <TriangleAlert size={18} /> : <Info size={18} />}</span>
      <div className="miss-main">
        <div className="miss-title">
          {m.label}
          {m.level === 'required' && <span className="badge b-amber">Needed by FBR</span>}
          {m.level === 'fix' && <span className="badge b-red">Please fix</span>}
        </div>
        <p>{m.message}</p>
        {m.input === 'grouping' && (
          <div className="seg-cards">
            {Object.entries(groupLabels).map(([k, label]) => (
              <label key={k} className={'seg-card' + (choices.grouping === k ? ' on' : '')}>
                <input type="radio" name="grouping" checked={choices.grouping === k} onChange={() => setChoices((c) => ({ ...c, grouping: k }))} />
                {label}
              </label>
            ))}
          </div>
        )}
        {m.input === 'date' && <input type="date" value={value} onChange={(e) => setDefault(e.target.value)} style={{ maxWidth: 220 }} aria-label={m.label} />}
        {m.input === 'text' && <input value={value} onChange={(e) => setDefault(e.target.value)} placeholder={m.suggest || 'Enter a value'} style={{ maxWidth: 360 }} aria-label={m.label} />}
        {m.input === 'hs' && (
          <div style={{ maxWidth: 360 }}>
            <HSCodeInput value={value} onChange={setDefault} companyPath={cp} />
          </div>
        )}
        {m.input === 'saletype' && <div style={{ maxWidth: 420 }}>{list(a.lists.saleTypes, value, setDefault, m.label)}</div>}
        {m.input === 'uom' && <div style={{ maxWidth: 320 }}>{list(a.lists.uoms, value, setDefault, m.label)}</div>}
        {m.input === 'rate' && <div style={{ maxWidth: 200 }}>{list(a.lists.rates, value, setDefault, m.label)}</div>}
        {m.input === 'province' && (
          <div className="stack" style={{ gap: 8 }}>
            {m.message.includes('address') && (
              <label className="check">
                <input type="checkbox" checked={choices.provinceFromAddress} onChange={(e) => setChoices((c) => ({ ...c, provinceFromAddress: e.target.checked }))} />{' '}
                Work it out from the city in the buyer's address
              </label>
            )}
            <div style={{ maxWidth: 320 }}>{list(a.lists.provinces, value, setDefault, m.label)}</div>
          </div>
        )}
        {m.input === 'regrule' && (
          <select value={choices.regRule} onChange={(e) => setChoices((c) => ({ ...c, regRule: e.target.value }))} style={{ maxWidth: 520 }} aria-label={m.label}>
            <option value="ntn">Registered when an NTN/CNIC is given (recommended)</option>
            <option value="Registered">All buyers are registered</option>
            <option value="Unregistered">All buyers are unregistered</option>
          </select>
        )}
        {m.input === 'valuemap' && (
          <table className="table value-map">
            <thead>
              <tr>
                <th>In your file</th>
                <th className="num">Lines</th>
                <th>Use FBR's</th>
              </tr>
            </thead>
            <tbody>
              {(m.values ?? []).map((v) => (
                <tr key={v.value}>
                  <td className="cell-title">{v.value}</td>
                  <td className="num">{v.lines}</td>
                  <td>
                    {list(valueList, choices.valueMap[m.field]?.[v.value] ?? '', (to) =>
                      setChoices((c) => ({ ...c, valueMap: { ...c.valueMap, [m.field]: { ...(c.valueMap[m.field] ?? {}), [v.value]: to } } })),
                    `FBR value for ${v.value}`)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

function ResultTiles({ summary }: { summary: ImportSummary }) {
  const total = summary.results.reduce((t, r) => t + Number(r.total || 0), 0)
  const warned = summary.results.filter((r) => (r.warnings ?? []).length > 0 || (r.issues ?? []).some((i) => i.severity === 'warning')).length
  return (
    <div className="grid g4 mb">
      <div className="card stat hero">
        <div className="stat-top">
          <span className="label">Invoices found</span>
        </div>
        <div className="value">{summary.invoices}</div>
        <div className="sub">{summary.results.reduce((t, r) => t + (r.lines || 0), 0)} lines</div>
      </div>
      <div className="card stat">
        <div className="stat-top">
          <span className="label">Ready to import</span>
          <span className="stat-icon ok">
            <CircleCheck size={18} />
          </span>
        </div>
        <div className="value">{summary.ok}</div>
        <div className="sub">passed FBR's checks</div>
      </div>
      <div className="card stat">
        <div className="stat-top">
          <span className="label">Need attention</span>
          <span className={'stat-icon ' + (summary.failed ? 'danger' : 'ok')}>{summary.failed ? <CircleAlert size={18} /> : <CircleCheck size={18} />}</span>
        </div>
        <div className="value">{summary.failed}</div>
        <div className="sub">{warned ? `${warned} ${summary.failed ? 'more ' : ''}with warnings` : summary.failed ? 'errors are shown below' : 'nothing to correct'}</div>
      </div>
      <div className="card stat">
        <div className="stat-top">
          <span className="label">Total value</span>
          <span className="stat-icon gold">
            <FileSpreadsheet size={18} />
          </span>
        </div>
        <div className="value">
          <small>Rs</small>
          {money(total)}
        </div>
        <div className="sub">including taxes</div>
      </div>
    </div>
  )
}

function ResultTable({ summary }: { summary: ImportSummary }) {
  return (
    <div className="card">
      <div className="card-head">
        <h3>{summary.preview ? 'Checked invoices' : 'Imported invoices'}</h3>
        <span className="sub">
          {summary.ok} ready · {summary.failed} with errors
        </span>
      </div>
      <div className="table-wrap">
        <table className="table">
          <thead>
            <tr>
              <th>Invoice</th>
              <th>Date</th>
              <th>Buyer</th>
              <th className="num">Lines</th>
              <th className="num">Total</th>
              <th>Status</th>
              <th style={{ width: '36%' }}>Messages</th>
            </tr>
          </thead>
          <tbody>
            {summary.results.map((r, i) => (
              <tr key={i}>
                <td className="nowrap">
                  <div className="cell-title">{r.invoiceId ? <Link to={`/invoices/${r.invoiceId}`}>{r.ref}</Link> : r.ref}</div>
                  <div className="cell-sub">
                    Row{r.rows.length > 1 ? 's' : ''} {r.rows.length > 3 ? `${r.rows[0]}–${r.rows[r.rows.length - 1]}` : r.rows.join(', ')}
                  </div>
                </td>
                <td className="nowrap">{r.date}</td>
                <td>{r.buyer || '—'}</td>
                <td className="num">{r.lines}</td>
                <td className="num">{money(r.total)}</td>
                <td>
                  {/^[A-Z]+$/.test(r.status) ? (
                    <StatusBadge status={r.status} />
                  ) : (
                    <span className={'badge ' + (r.status === 'ok' ? 'b-green' : 'b-red')}>{r.status === 'ok' ? 'Ready' : r.status === 'invalid' ? 'Has errors' : 'Not imported'}</span>
                  )}
                </td>
                <td className="small">
                  {(r.errors ?? []).map((e, k) => (
                    <div key={k} className="msg err">
                      {e}
                    </div>
                  ))}
                  {(r.issues ?? []).map((e, k) => (
                    <div key={'i' + k} className={'msg ' + (e.severity === 'error' ? 'err' : 'warn')}>
                      {e.line > 0 && `Line ${e.line}: `}
                      {e.message}
                    </div>
                  ))}
                  {(r.warnings ?? []).map((e, k) => (
                    <div key={'w' + k} className="msg warn">
                      {e}
                    </div>
                  ))}
                  {!(r.errors ?? []).length && !(r.issues ?? []).length && !(r.warnings ?? []).length && <span className="faint">No remarks</span>}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
