// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Package brand holds product identity values. Change these constants to
// rebrand the product; nothing else in the code base hard-codes the name.
package brand

const (
	// ProductName is shown in the UI, on printed invoices and in logs.
	ProductName = "Veridian E-invoicing Pakistan"
	// ShortName is used for file names, service names and the User-Agent.
	ShortName = "veridian-einvoicing-pakistan"
	// Vendor is the company that sells and supports the product.
	Vendor = "Veridian Partners Consultancy Private Limited"
	// SupportContact is printed in the About screen.
	SupportContact = "muhammadshahjahan.audit@gmail.com"
	// WindowsServiceName is the Windows service identifier.
	WindowsServiceName = "VeridianEInvoicingPakistan"
	// WindowsServiceDisplay is the Windows service display name.
	WindowsServiceDisplay = "Veridian E-invoicing Pakistan (FBR Digital Invoicing)"
	// LegacyWindowsServiceName is the service identifier and ProgramData
	// folder of builds released as "Veridian E-invoicing PK". An upgraded
	// installation keeps using that folder; the installer removes the old
	// service.
	LegacyWindowsServiceName = "VeridianEInvoicingPK"
	// Tagline is shown under the product name.
	Tagline = "FBR Digital Invoicing"
	// Developer is the author and copyright holder of the software.
	Developer = "Veridian Partners Consultancy Private Limited"
	// CopyrightYear is the first year of publication.
	CopyrightYear = "2026"
	// Copyright is the notice shown in the UI, on printouts and in the CLI.
	Copyright = "© " + CopyrightYear + " " + Developer + ". All rights reserved."
	// DevelopedBy is the developer credit line.
	DevelopedBy = "Developed by " + Developer
)

// Version is overridden at build time with
// -ldflags "-X einvoicing/internal/brand.Version=1.2.3".
var Version = "1.0.0-dev"

// BuildDate is overridden at build time.
var BuildDate = "unknown"
