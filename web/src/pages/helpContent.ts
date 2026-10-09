// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Reference material from FBR's Digital Invoicing pages (fbr.gov.pk), as
// published in October 2026. FBR's own text prevails; links open the
// official documents.

export interface FAQ {
  q: string
  a: string
  app?: string // how this software handles it
}

export const faqSource = 'https://fbr.gov.pk/faqs/173967/173969'

export const faqs: FAQ[] = [
  {
    q: 'What is an electronic invoice?',
    a: 'A tax invoice created digitally in the specified structured format. A paper invoice that is scanned, copied or otherwise converted is not an electronic invoice.',
  },
  {
    q: 'Is electronic invoicing mandatory, and from when?',
    a: 'Yes, for all corporate and non-corporate registered persons (SRO 709(I)/2025). The final dates were set by SRO 1852(I)/2025, which superseded SRO 1413(I)/2025: e-invoices from 1 November 2025 for public companies, importers, companies with turnover above Rs 1 billion and individuals/AOPs above Rs 100 million; 15 November 2025 for companies between Rs 100 million and Rs 1 billion; 1 December 2025 for other companies; and 31 December 2025 for all other registered persons.',
  },
  {
    q: 'Who is a licensed integrator, and must I integrate through one?',
    a: 'A person licensed by FBR under Chapter XIV of the Sales Tax Rules to integrate registered persons (section 2(15A)). Integration of a POS, ERP or invoicing system must be through a licensed integrator; under rule 150XF PRAL acts as a licensed integrator and provides integration free of cost on demand.',
    app: 'Choose PRAL (free) or another licensed integrator when you register for Digital Invoicing on IRIS; the security token you receive is entered under FBR integration.',
  },
  {
    q: 'Is any fee payable to FBR?',
    a: 'No fee is payable to FBR by the registered person or the integrator. A licensed integrator may charge for configuration and integration within the limit FBR specifies.',
  },
  {
    q: 'Do I need to visit FBR, or download software from FBR?',
    a: 'No visit is needed and FBR provides no downloadable software. FBR publishes the technical documentation; any commercial invoicing software can be integrated through a licensed integrator.',
  },
  {
    q: 'What if I integrate after the notified date?',
    a: 'An extension may be sought under the rules (rule 150V). A person who does not integrate by the (extended) date, or contravenes the provisions, is liable to penalty under section 33.',
  },
  {
    q: 'What happens during business interruptions, such as the internet being down?',
    a: 'The registered person and the integrator must follow the Sales Tax Act and Rules in all such cases.',
    app: 'Invoices issued while FBR cannot be reached are marked as issued in the offline mode and uploaded automatically as soon as the connection returns, well within the 24 hours rule 150XC allows. Failures lasting long enough are recorded in the incident register for reporting under rule 150XA.',
  },
  {
    q: 'How is an e-invoice cancelled or corrected?',
    a: 'Under section 9 of the Sales Tax Act, a debit or credit note is issued when a supply is cancelled, goods are returned or the value or nature of the supply changes.',
    app: 'Debit notes are issued against the original invoice. Under Sales Tax General Order 01 of 2026 an invoice can be cancelled or edited in FBR’s system within 72 hours of issue; after that the Commissioner’s approval is needed.',
  },
  {
    q: 'What must an invoice to an unregistered buyer contain?',
    a: 'Under section 23(1)(b), when a manufacturer or importer supplies an unregistered distributor, the invoice must show the CNIC or NTN of that distributor.',
    app: 'The buyer’s CNIC or NTN is required before submission when the seller is a manufacturer or importer.',
  },
  {
    q: 'When must the digital invoice be issued?',
    a: 'At the time of supply — receipt of payment or delivery of goods, whichever is earlier (section 2(44)). Since the Finance Act 2026, section 23(1) also requires invoices for exempt supplies and for advance receipts, bearing a verifiable FBR invoice number.',
  },
  {
    q: 'Several products share one HS code but have different sale types. How are they invoiced?',
    a: 'Add them as separate items with their own descriptions; each line carries its own sale type and rate.',
  },
  {
    q: 'Does e-invoicing apply only to local sales?',
    a: 'No. A digital invoice is issued for all sales to be reported in Annexure-C of the sales tax return, including exports.',
  },
  {
    q: 'How do I register and integrate step by step?',
    a: 'Follow PRAL’s Digital Invoicing User Manual: register on IRIS, choose the integrator, enter technical details and the IP addresses to whitelist, complete the sandbox scenarios and obtain the production token.',
    app: 'Settings → FBR integration shows the exact details to enter on IRIS and can find this server’s public IP.',
  },
]

export interface Integrator {
  name: string
  licence: string
}

export const integratorSource = 'https://fbr.gov.pk/list-of-license-interprator/173967/173971'

// FBR's list of licensed integrators (as published in October 2026).
export const integrators: Integrator[] = [
  { name: 'Haball (Pvt) Ltd', licence: '398586644' },
  { name: 'WebDNAworks (Private) Limited', licence: '394537651' },
  { name: 'EY Ford Rhodes', licence: '649954048' },
  { name: 'PRAL (free of cost — rule 150XF)', licence: '' },
  { name: 'OpenPort Pakistan (Pvt.) Limited', licence: '715193372' },
  { name: 'TMR Consulting (Private) Limited', licence: '620903892' },
  { name: 'NatureTech (Private) Limited', licence: '555378169' },
  { name: 'Dynamic Resources (Private) Limited', licence: '90332116' },
]

export interface OfficialDoc {
  title: string
  date: string
  about: string
  url: string
}

export const documents: { group: string; items: OfficialDoc[] }[] = [
  {
    group: 'Technical',
    items: [
      {
        title: 'Technical Specification for DI API, version 1.12',
        date: '24 Jul 2025',
        about: 'PRAL’s API: endpoints, JSON fields, reference APIs, QR code and logo, error codes and sandbox scenarios. This software implements it.',
        url: 'https://download1.fbr.gov.pk/Docs/20257301172130815TechnicalDocumentationforDIAPIV1.12.pdf',
      },
      {
        title: 'Digital Invoicing User Manual, version 1.5',
        date: '2025',
        about: 'PRAL’s step-by-step guide to registration on IRIS, technical details, IP whitelisting, sandbox and production.',
        url: 'https://download1.fbr.gov.pk/Docs/20257301171649798DIUserManualV1.5.pdf',
      },
      {
        title: 'Digital Invoicing technical assistance',
        date: 'FBR web page',
        about: 'FBR’s page with the current technical documents.',
        url: 'https://fbr.gov.pk/di-technical-assistance/173967/173970',
      },
    ],
  },
  {
    group: 'Law and FBR orders',
    items: [
      {
        title: 'SRO 69(I)/2025 — Chapter XIV, Sales Tax Rules 2006',
        date: '29 Jan 2025',
        about: 'Rules 150Q–150XQ: integration, system functions (digital signature, closings, logs), invoice particulars, signboard, offline invoices, licensed integrators.',
        url: 'https://download1.fbr.gov.pk/SROs/2025129141598258SRO69(I)2025.pdf',
      },
      {
        title: 'SRO 1666(I)/2026 — amendments to Chapter XIV',
        date: '29 Sep 2026',
        about: 'Extends e-invoicing to federal excise and ICT services, adds the FED particulars (aa)–(ff), advance receipt invoices and Annex-C accountability (rule 150XD(2)).',
        url: 'https://download1.fbr.gov.pk/SROs/202693089244653SRO1666dated29-09-2026.pdf',
      },
      {
        title: 'SRO 1852(I)/2025 — integration deadlines',
        date: '24 Sep 2025',
        about: 'Final dates for registration, testing and issuing electronic invoices by category of registered person.',
        url: 'https://download1.fbr.gov.pk/SROs/2025924149054920SRO1852.pdf',
      },
      {
        title: 'Sales Tax General Order 01 of 2026',
        date: '30 Mar 2026',
        about: 'Issuance of electronic invoices and integration; cancellation or editing within 72 hours of issue.',
        url: 'https://download1.fbr.gov.pk/Docs/2026331133557466STGO01of2026.pdf',
      },
      {
        title: 'Sales Tax General Order 25 of 2026',
        date: '28 Sep 2026',
        about: 'Goods moved to the registered person’s own warehouse under the same STRN: no digital invoice; a stock transfer note in the prescribed format instead.',
        url: 'https://download1.fbr.gov.pk/Docs/20269281593856844STGO25of2026.pdf',
      },
      {
        title: 'Circular 01 of 2026 — Finance Act 2026 explained',
        date: '11 Sep 2026',
        about: 'Invoices for exempt supplies and advance receipts (section 23(1)), enhanced penalties (section 33), suspension (section 21(2)) and more.',
        url: 'https://download1.fbr.gov.pk/Docs/20269111791418742Circular01of2026.pdf',
      },
      {
        title: 'SRO 1655(I)/2026 — electronic scrutiny (Chapter XII-A)',
        date: '25 Sep 2026',
        about: 'Discrepancies found by FBR’s system are intimated through IRIS with at least seven days to explain or correct; the Annex-C reconciliation helps answer them.',
        url: 'https://download1.fbr.gov.pk/SROs/20269251892143851SRO1655.pdf',
      },
      {
        title: 'Digital Invoicing legal provisions',
        date: 'FBR web page',
        about: 'FBR’s page listing the current notifications and orders.',
        url: 'https://fbr.gov.pk/di-legal-provisions/173967/173968',
      },
    ],
  },
]
