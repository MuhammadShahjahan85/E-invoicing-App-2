-- Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
-- Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

-- Due dates of the monthly sales tax return used by the compliance calendar.
-- Generally tax is paid by the 15th and the return filed by the 18th of the
-- month after the tax period; some sectors have other dates and FBR may
-- extend them, so the days are kept per company.
ALTER TABLE companies ADD COLUMN return_payment_day INTEGER NOT NULL DEFAULT 15;
ALTER TABLE companies ADD COLUMN return_filing_day INTEGER NOT NULL DEFAULT 18;
