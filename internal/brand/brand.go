// Package brand holds product identity values. Change these constants to
// rebrand the product; nothing else in the code base hard-codes the name.
package brand

const (
	// ProductName is shown in the UI, on printed invoices and in logs.
	ProductName = "E-Invoicing Suite PK"
	// ShortName is used for file names, service names and the User-Agent.
	ShortName = "einvoice-pk"
	// Vendor is the company that sells and supports the product.
	Vendor = "Your Company (Pvt) Ltd"
	// SupportContact is printed in the About screen.
	SupportContact = "support@example.com"
	// WindowsServiceName is the Windows service identifier.
	WindowsServiceName = "EInvoicingSuitePK"
	// WindowsServiceDisplay is the Windows service display name.
	WindowsServiceDisplay = "E-Invoicing Suite PK (FBR Digital Invoicing)"
)

// Version is overridden at build time with
// -ldflags "-X einvoicing/internal/brand.Version=1.2.3".
var Version = "1.0.0-dev"

// BuildDate is overridden at build time.
var BuildDate = "unknown"
