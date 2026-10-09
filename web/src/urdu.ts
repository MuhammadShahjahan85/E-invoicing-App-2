// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Urdu captions shown beside the English names of menus and pages when
// "Urdu captions" is on.

export const urdu: Record<string, string> = {
  Overview: 'جائزہ',
  Dashboard: 'ڈیش بورڈ',
  'Sales tax overview': 'سیلز ٹیکس کا جائزہ',
  Sales: 'فروخت',
  'New invoice': 'نئی انوائس',
  Invoices: 'انوائسز',
  Invoice: 'انوائس',
  'Edit invoice': 'انوائس میں ترمیم',
  'Import data': 'ڈیٹا درآمد',
  'Stock transfer notes': 'اسٹاک منتقلی نوٹس',
  Masters: 'بنیادی ریکارڈ',
  'Customers (buyers)': 'گاہک',
  'Products & services': 'مصنوعات اور خدمات',
  'FBR & compliance': 'ایف بی آر اور تعمیل',
  'Tax periods & returns': 'ٹیکس مدت اور گوشوارے',
  'FBR reference library': 'ایف بی آر حوالہ جات',
  'Sandbox scenarios': 'سینڈ باکس منظرنامے',
  Reports: 'رپورٹس',
  'Incident register': 'واقعات کا رجسٹر',
  'Audit trail': 'آڈٹ ٹریل',
  Settings: 'ترتیبات',
  Company: 'کمپنی',
  'FBR integration': 'ایف بی آر سے رابطہ',
  'Invoice printing': 'انوائس کی پرنٹنگ',
  'Users & roles': 'صارفین اور اختیارات',
  'ERP API keys': 'ای آر پی کیز',
  'System & licence': 'سسٹم اور لائسنس',
  Support: 'مدد',
  'Mobile app & access': 'موبائل ایپ',
  'Help & error codes': 'رہنمائی اور ایرر کوڈز',
  Account: 'اکاؤنٹ',
  'Change password': 'پاس ورڈ کی تبدیلی',
}

/** Tip is a short compliance reminder in English and Urdu. */
export interface Tip {
  en: string
  ur: string
  to?: string
  link?: string
}

export const tips: Tip[] = [
  {
    en: 'Report every invoice at the time of supply: Digital Invoicing is real-time, and each copy you hand over should carry the FBR invoice number and QR code.',
    ur: 'ہر انوائس فراہمی کے وقت ہی ایف بی آر کو رپورٹ کریں، اور خریدار کو دی جانے والی کاپی پر ایف بی آر انوائس نمبر اور کیو آر کوڈ ہونا چاہیے۔',
  },
  {
    en: 'Invoices issued while FBR or the internet was down must be uploaded within 24 hours of the connection coming back (rule 150XC).',
    ur: 'ایف بی آر یا انٹرنیٹ بند ہونے کے دوران جاری انوائسز رابطہ بحال ہونے کے 24 گھنٹوں کے اندر اپ لوڈ کریں۔',
    to: '/invoices?status=QUEUED',
    link: 'Pending invoices',
  },
  {
    en: 'Report system failures and disruptions to FBR and the Commissioner within 24 hours (rule 150XA(c)).',
    ur: 'سسٹم کی خرابی یا تعطل کی اطلاع 24 گھنٹوں کے اندر ایف بی آر اور کمشنر کو دیں۔',
    to: '/incidents',
    link: 'Incident register',
  },
  {
    en: 'Check that a buyer is on the Active Taxpayers List before treating them as registered — further tax applies to unregistered buyers.',
    ur: 'خریدار کو رجسٹرڈ ماننے سے پہلے ایکٹو ٹیکس پیئر لسٹ میں اس کی موجودگی چیک کریں؛ غیر رجسٹرڈ خریدار پر مزید ٹیکس لگتا ہے۔',
    to: '/library?tab=buyer',
    link: 'Verify a buyer',
  },
  {
    en: 'Sales tax is generally paid by the 15th and the return filed by the 18th of the following month; FBR sometimes extends the filing date.',
    ur: 'سیلز ٹیکس عموماً اگلے مہینے کی 15 تاریخ تک جمع اور گوشوارہ 18 تاریخ تک داخل کیا جاتا ہے؛ کبھی کبھی ایف بی آر تاریخ بڑھا دیتا ہے۔',
    to: '/compliance',
    link: 'Tax periods',
  },
  {
    en: 'Before filing, match Annexure-C on IRIS with the documents reported to FBR (rule 150XD(2)). The report pack lists them all.',
    ur: 'گوشوارہ داخل کرنے سے پہلے IRIS پر اینیکسچر سی کا ایف بی آر کو رپورٹ شدہ دستاویزات سے موازنہ کریں۔',
    to: '/compliance',
    link: 'Report pack',
  },
  {
    en: 'Manufacturers and importers must show the CNIC or NTN of unregistered buyers on their invoices (section 23(1)(b) of the Sales Tax Act).',
    ur: 'مینوفیکچررز اور امپورٹرز غیر رجسٹرڈ خریداروں کی انوائس پر ان کا شناختی کارڈ یا این ٹی این درج کریں۔',
  },
  {
    en: 'Display the “Integrated with FBR” signboard where you make sales, with your software registration number (rule 150R(11)).',
    ur: 'فروخت کی جگہ پر سافٹ ویئر رجسٹریشن نمبر کے ساتھ "ایف بی آر سے منسلک" کا بورڈ آویزاں کریں۔',
    to: '/settings/fbr',
    link: 'Print the signboard',
  },
  {
    en: 'Wrong invoice? Cancel it through FBR within the permitted window, or issue a debit note against it — never edit an accepted invoice.',
    ur: 'غلط انوائس؟ اجازت شدہ مدت میں ایف بی آر کے ذریعے منسوخ کریں یا اس کے خلاف ڈیبٹ نوٹ جاری کریں۔',
  },
  {
    en: 'Keep regular backups: electronic records must be kept for six years (rule 150S), and the System page shows when the last backup ran.',
    ur: 'باقاعدگی سے بیک اپ رکھیں؛ ریکارڈ چھ سال تک محفوظ رکھنا ضروری ہے۔',
    to: '/settings/system',
    link: 'Backups',
  },
]
