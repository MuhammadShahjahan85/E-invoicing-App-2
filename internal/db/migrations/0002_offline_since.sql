-- Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
-- Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

-- When an invoice could first not be reported because FBR's system (or the
-- security token) was unavailable. Invoices issued offline must be uploaded
-- within 24 hours of connectivity being restored, so they stay tracked until
-- FBR accepts them, even if the later upload is rejected.
ALTER TABLE invoices ADD COLUMN offline_since TEXT NOT NULL DEFAULT '';
