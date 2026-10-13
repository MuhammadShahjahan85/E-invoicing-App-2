-- Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
-- Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

-- The complete Pakistan Customs Tariff is built in (internal/hscodes): each
-- code keeps the heading and groups above it for search and display, and
-- whether it is on the list downloaded from FBR's Digital Invoicing API.
ALTER TABLE ref_hs_codes ADD COLUMN context TEXT NOT NULL DEFAULT '';
ALTER TABLE ref_hs_codes ADD COLUMN in_fbr INTEGER NOT NULL DEFAULT 0;
UPDATE ref_hs_codes SET in_fbr = 1 WHERE source = 'fbr';
DELETE FROM ref_hs_codes WHERE source = 'seed';
