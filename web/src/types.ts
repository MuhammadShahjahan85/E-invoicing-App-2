// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Types mirroring the Go API (internal/store, internal/service).

export type Env = 'simulator' | 'sandbox' | 'production'
export type InvoiceStatus = 'DRAFT' | 'VALIDATED' | 'QUEUED' | 'SUBMITTING' | 'ACCEPTED' | 'REJECTED' | 'UNCERTAIN' | 'CANCELLED'
export type DocType = 'Sale Invoice' | 'Debit Note'
export type RegType = 'Registered' | 'Unregistered'

export interface User {
  id: number
  username: string
  fullName: string
  email: string
  role: string
  allCompanies: boolean
  companyIds: number[]
  active: boolean
  mustChangePassword: boolean
  lastLoginAt: string
  createdAt: string
}

export interface PrintSettings {
  paperSize: string
  wordsStyle: string
  showAmountWords: boolean
  showSignature: boolean
  footerText: string
  termsText: string
  showUnitPrice: boolean
  showHsCode: boolean
  showDiscount: boolean
  copies: number
}

export interface Company {
  id: number
  name: string
  ntnCnic: string
  strn: string
  province: string
  provinceCode: number
  address: string
  city: string
  phone: string
  email: string
  businessActivities: string[]
  sectors: string[]
  assignedScenarios: string[]
  environment: Env
  hasSandboxToken: boolean
  hasProductionToken: boolean
  sandboxTokenExpiry: string
  productionTokenExpiry: string
  invoicePrefix: string
  debitNotePrefix: string
  furtherTaxRate: number
  withholdingFraction: number
  sendInternalRef: boolean
  validateBeforePost: boolean
  returnPaymentDay: number
  returnFilingDay: number
  softwareRegNo: string
  hasLogo: boolean
  printSettings: PrintSettings
  active: boolean
}

export interface Customer {
  id: number
  companyId: number
  code: string
  name: string
  ntnCnic: string
  strn: string
  registrationType: RegType
  province: string
  address: string
  city: string
  phone: string
  email: string
  withholdingMode: string
  statlStatus: string
  fbrRegType: string
  statusCheckedAt: string
  notes: string
  active: boolean
}

export interface Product {
  id: number
  companyId: number
  code: string
  description: string
  hsCode: string
  uom: string
  saleType: string
  rate: string
  sroScheduleNo: string
  sroItemSerialNo: string
  unitPrice: number
  retailPrice: number
  furtherTaxMode: string
  extraTaxRate: number
  fedRate: number
  fedType: string
  fedRateText: string
  fedSro: string
  fedSroSerial: string
  active: boolean
}

export interface Totals {
  gross: number
  discount: number
  valueExclST: number
  retailValue: number
  salesTax: number
  furtherTax: number
  extraTax: number
  fed: number
  stWithheld: number
  totalValue: number
  amountPayable: number
}

export interface Issue {
  line: number
  field: string
  code?: string
  severity: 'error' | 'warning'
  message: string
}

export interface FBRError {
  item: number
  code: string
  message: string
}

export interface InvoiceItem {
  id?: number
  lineNo?: number
  productId?: number | null
  hsCode: string
  description: string
  uom: string
  quantity: number
  unitPrice: number
  discountPercent: number
  discountAmount: number
  valueOverride?: number | null
  saleType: string
  rate: string
  retailPrice: number
  retailValueOverride?: number | null
  furtherTaxMode: string
  furtherTaxOverride?: number | null
  extraTaxRate: number
  extraTaxOverride?: number | null
  fedRate: number
  fedOverride?: number | null
  withholdingOverride?: number | null
  salesTaxOverride?: number | null
  sroScheduleNo: string
  sroItemSerialNo: string
  fedType?: string
  fedRateText?: string
  fedUnitPrice?: number
  fedSro?: string
  fedSroSerial?: string
  gross?: number
  discount?: number
  valueExclST?: number
  retailValue?: number
  salesTax?: number
  furtherTax?: number
  extraTax?: number
  extraTaxEmpty?: boolean
  fed?: number
  stWithheld?: number
  totalValue?: number
  fbrItemInvoiceNo?: string
  fbrStatusCode?: string
  fbrErrorCode?: string
  fbrError?: string
  warnings?: string[]
}

export interface Invoice {
  id: number
  companyId: number
  environment: Env
  docType: DocType
  internalNo: string
  invoiceDate: string
  status: InvoiceStatus
  customerId: number | null
  sellerNtnCnic: string
  sellerName: string
  sellerProvince: string
  sellerAddress: string
  buyerNtnCnic: string
  buyerName: string
  buyerProvince: string
  buyerAddress: string
  buyerRegistrationType: RegType
  withholdingMode: string
  invoiceRefNo: string
  refInvoiceId: number | null
  scenarioId: string
  externalRef: string
  source: string
  notes: string
  totals: Totals
  fbrInvoiceNumber: string
  fbrDated: string
  fbrStatusCode: string
  fbrErrors: FBRError[] | null
  lastError: string
  validation: Issue[] | null
  submitAttempts: number
  nextAttemptAt: string
  payloadHash: string
  sealHash: string
  prevSealHash: string
  offlineSince: string
  signature: string
  advanceReceipt: boolean
  advanceRef: string
  printCount: number
  cancelledAt: string
  cancelReason: string
  cancelReference: string
  createdAt: string
  updatedAt: string
  submittedAt: string
  acceptedAt: string
  items?: InvoiceItem[]
}

export interface SaleType {
  name: string
  category: string
  defaultRate: string
  basis: 'value' | 'retail_price'
  sroRequired: boolean
  sroTypical: boolean
  extraTaxMustBeEmpty: boolean
  furtherTaxDefault: boolean
  exempt: boolean
  scenario?: string
  note?: string
}

export interface ScenarioItem {
  hsCode: string
  productDescription: string
  rate: string
  uoM: string
  quantity: number
  valueSalesExcludingST: number
  saleType: string
  sroScheduleNo: string
  sroItemSerialNo: string
}

export interface Scenario {
  id: string
  title: string
  description: string
  buyerNTNCNIC: string
  buyerName: string
  buyerRegistrationType: RegType
  item: ScenarioItem
}

export interface ScenarioRun {
  id: number
  scenarioId: string
  invoiceId: number | null
  mode: string
  status: string
  fbrInvoiceNumber: string
  message: string
  runAt: string
}

export interface ScenarioStatus extends Scenario {
  suggested: boolean
  assigned: boolean
  passed: boolean
  lastRun: ScenarioRun | null
}

export interface ScenarioOverview {
  scenarios: ScenarioStatus[]
  assignedCount: number
  passedCount: number
  readyForProduction: boolean
  suggested: string[]
}

export interface Meta {
  saleTypes: SaleType[]
  businessActivities: string[]
  sectors: string[]
  scenarios: Scenario[]
  environments: { value: Env; label: string }[]
  docTypes: DocType[]
  registrationTypes: RegType[]
  provinces: { code: number; name: string }[]
  uoms: string[]
  errorCatalogue: { code: string; section: 'sales' | 'purchase'; title: string; detail: string; fix: string }[]
  roles: string[]
  product: string
  vendor: string
  support: string
  copyright: string
  developedBy: string
  version: string
  cancelWindowHours: number
  cancelApi?: Partial<Record<Env, boolean>>
}

export interface LicenseStatus {
  mode: string
  licensee: string
  licenseId: string
  edition: string
  sellerNtns: string[] | null
  maxCompanies: number
  maxUsers: number
  expiresAt: string
  supportUntil: string
  daysLeft: number
  production: boolean
  message: string
}

export interface ConnectionStatus {
  environment: Env
  healthy: boolean
  lastSuccess: string
  failingSince: string
  failures: number
  lastError: string
  authFailure: boolean
}

export interface DashboardData {
  environment: Env
  stats: {
    statusCounts: { status: string; count: number }[] | null
    todayCount: number
    todayValue: number
    todaySalesTax: number
    monthCount: number
    monthValue: number
    monthSalesTax: number
    needsAttention: number
    topErrors: { code: string; message: string; count: number }[] | null
    pendingUpload: number
    oldestPending: string
  }
  connection: ConnectionStatus
  scenarios: ScenarioOverview
  openIncidents: number
  unreportedIncidents: number
  tokenWarning: string
  license: LicenseStatus
  deadlines: ReturnDeadline[]
  recent?: Invoice[]
  trend?: PeriodRow[]
  topBuyers?: CustomerRow[]
  topItems?: ItemRow[]
  reference?: { lastSync: string; hsSource: string; hsCodes: number }
  tie?: PeriodTie
}

/** PeriodTie compares the books, FBR and Annexure-C for the return due next. */
export interface PeriodTie {
  period: string
  periodLabel: string
  filingDue: string
  daysLeft: number
  books: number
  annexC: number
  annexCTax: number
  accepted: number
  issued: number
  edges: TieEdge[]
  checksOpen: number
  checksTotal: number
}

export interface TieEdge {
  id: 'books-fbr' | 'fbr-annexc' | 'books-annexc'
  title: string
  detail: string
  count: number
  value: number
  link: string
}

export interface ReturnDeadline {
  period: string
  periodLabel: string
  kind: 'payment' | 'filing'
  due: string
  daysLeft: number
  originalDue?: string
  reference?: string
}

export interface PeriodRow {
  period: string
  saleInvoices: number
  debitNotes: number
  valueExclST: number
  salesTax: number
  furtherTax: number
  debitNoteValue: number
  debitNoteSalesTax: number
  stWithheld: number
}

export interface CustomerRow {
  buyerNtnCnic: string
  buyerName: string
  buyerRegistrationType: string
  invoices: number
  valueExclST: number
  salesTax: number
  furtherTax: number
  stWithheld: number
  totalValue: number
}

export interface ItemRow {
  hsCode: string
  description: string
  lines: number
  valueExclST: number
  salesTax: number
}

export interface TaxSummaryRow {
  docType: string
  saleType: string
  rate: string
  invoices: number
  lines: number
  valueExclST: number
  retailValue: number
  salesTax: number
  furtherTax: number
  extraTax: number
  fed: number
  stWithheld: number
}

export interface PeriodCheck {
  id: string
  ok: boolean
  title: string
  detail: string
  link?: string
}

export interface PeriodReview {
  period: string
  periodLabel: string
  from: string
  to: string
  environment: Env
  deadlines: ReturnDeadline[]
  summary: PeriodRow
  bySaleType: TaxSummaryRow[] | null
  statuses: { status: string; docType: string; count: number }[] | null
  incidents: Incident[] | null
  checks: PeriodCheck[]
  counts: Record<string, number>
  topItems: ItemRow[] | null
  topBuyers: CustomerRow[] | null
}

export interface Alert {
  id: string
  severity: 'error' | 'warning' | 'info'
  title: string
  detail: string
  link?: string
}

export interface SearchResults {
  q: string
  invoices: Invoice[]
  customers: Customer[]
  products: Product[]
  hsCodes: HSCode[]
}

export interface RefStatus {
  entries: { key: string; kind: string; source: string; fetchedAt: string }[] | null
  hsCodes: number
  hsSource: string
  lastSync: string
  environment: Env
}

export interface SaleTypeRef {
  id: number
  description: string
  known: boolean
  info: SaleType
}

export interface BuyerStatus {
  regNo: string
  statlActive: boolean
  statlStatus: string
  registrationType: string
  registered: boolean
  checkedAt: string
  error?: string
}

export interface Incident {
  id: number
  companyId: number
  kind: string
  description: string
  startedAt: string
  endedAt: string
  autoDetected: boolean
  reportedAt: string
  reportReference: string
  createdAt: string
}

export interface FBRCall {
  id: number
  invoiceId: number | null
  environment: string
  operation: string
  method: string
  url: string
  requestBody: string
  responseBody: string
  httpStatus: number
  durationMs: number
  errorKind: string
  error: string
  createdAt: string
}

export interface AuditEntry {
  id: number
  ts: string
  userId: number | null
  username: string
  companyId: number | null
  action: string
  entity: string
  entityId: string
  details: string
  ip: string
  hash: string
}

export interface APIKey {
  id: number
  companyId: number
  name: string
  prefix: string
  createdAt: string
  lastUsedAt: string
  revokedAt: string
}

export interface RateRef {
  id: number
  description: string
  value: number
}

export interface HSCode {
  code: string
  description: string
}

export interface Paged<T> {
  items: T[]
  total: number
}

export interface StockTransferItem {
  lineNo: number
  productId?: number
  description: string
  hsCode: string
  quantity: number
  uom: string
  valueAtCost: number
}

export interface StockTransfer {
  id: number
  companyId: number
  seq: number
  number: string
  dispatchedAt: string
  fromName: string
  fromAddress: string
  toName: string
  toAddress: string
  vehicleNo: string
  driverCnic: string
  authorisedBy: string
  receivedBy: string
  receivedAt: string
  notes: string
  status: 'DISPATCHED' | 'RECEIVED' | 'CANCELLED'
  cancelReason: string
  totalValue: number
  createdAt: string
  updatedAt: string
  items?: StockTransferItem[]
  itemCount: number
}

export interface ClosingTotals {
  count: number
  valueExclST: number
  salesTax: number
  furtherTax: number
  extraTax: number
  fed: number
  stWithheld: number
  totalValue: number
}

export interface Closing {
  id: number
  companyId: number
  environment: Env
  kind: 'day' | 'week' | 'month'
  periodKey: string
  periodStart: string
  periodEnd: string
  summary: {
    documents: number
    reported: number
    cancelled: number
    pending: number
    unreconciled: number
    rejected: number
    drafts: number
    sales: ClosingTotals
    debitNotes: ClosingTotals
    firstNo: string
    lastNo: string
    firstFbrNo: string
    lastFbrNo: string
    offlineMode: number
  }
  hash: string
  prevHash: string
  createdAt: string
}

export interface ReturnExtension {
  period: string
  filingDate: string
  reference: string
  createdAt: string
}
