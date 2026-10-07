// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Package validate checks a DI payload locally before it is sent to FBR, and
// explains FBR's error codes in plain language.
package validate

import "strings"

// ErrorInfo explains an FBR DI validation error code. The authoritative
// message is the one FBR returns in the response, which is always shown to
// the operator verbatim; this catalogue adds guidance on how to fix it.
type ErrorInfo struct {
	Code  string `json:"code"`
	Title string `json:"title"`
	Fix   string `json:"fix"`
}

// Catalogue of DI sales error codes (PRAL Technical Specification for DI API,
// "Sales Error Codes"), with remediation guidance.
var catalogue = map[string]ErrorInfo{
	"0001": {Title: "Seller is not registered for sales tax / seller registration number invalid", Fix: "Check the seller NTN (7 digits) or CNIC (13 digits) in Company settings. It must be the registration number the security token was issued to, and the seller must be active on the STATL."},
	"0002": {Title: "Buyer registration number is not in proper format", Fix: "Enter the buyer NTN as 7 (or 9) digits or the CNIC as 13 digits, without dashes or spaces."},
	"0003": {Title: "Invoice type is not valid", Fix: "Use 'Sale Invoice' or 'Debit Note' (doctypecode reference API)."},
	"0005": {Title: "Invoice date format is not valid", Fix: "Dates must be sent as YYYY-MM-DD; the software does this automatically — check imported data."},
	"0006": {Title: "Referenced sale invoice does not exist", Fix: "For a debit note, invoiceRefNo must be the FBR invoice number of an accepted invoice of the same seller."},
	"0007": {Title: "Sale type is not valid / not allowed", Fix: "Pick the sale type from the synced FBR list and make sure it is permitted for the business activity registered on IRIS."},
	"0008": {Title: "Sales tax withheld at source must be zero or equal to sales tax", Fix: "Set withholding to 'None' or 'Full' for this line, or confirm the treatment with your withholding agent."},
	"0009": {Title: "Buyer registration number is required", Fix: "A registered buyer must have an NTN or CNIC."},
	"0010": {Title: "Buyer name is required", Fix: "Enter the buyer's business name."},
	"0011": {Title: "Invoice type is required", Fix: "Select the document type."},
	"0012": {Title: "Buyer registration type is not valid", Fix: "Use 'Registered' or 'Unregistered'. You can confirm a buyer's status with the 'Check FBR status' button (Get_Reg_Type / STATL)."},
	"0013": {Title: "Sale type is missing or not valid", Fix: "Select a sale type from the FBR list."},
	"0018": {Title: "Sales tax / FED in sales tax mode is required", Fix: "Provide the sales tax amount for the line."},
	"0019": {Title: "HS code is required", Fix: "Every line needs an HS (PCT) code — services use Chapter 98 codes such as 9804.0000."},
	"0021": {Title: "Value of sales is not valid", Fix: "Value excluding sales tax must be a non-negative number."},
	"0022": {Title: "Quantity is not valid", Fix: "Enter a quantity (up to 4 decimals)."},
	"0023": {Title: "Unit of measure is not valid", Fix: "Use the UoM FBR prescribes for the HS code (HS_UOM reference). UoM is case sensitive, e.g. 'KG' not 'kg'."},
	"0024": {Title: "Product description is required", Fix: "Enter a description for the line."},
	"0026": {Title: "Fixed / notified value or retail price is not valid", Fix: "For Third Schedule goods enter the printed retail price (including sales tax); the system reports the retail value excluding sales tax."},
	"0027": {Title: "Sales tax applicable is not valid", Fix: "Sales tax must equal value (or retail price) multiplied by the rate."},
	"0028": {Title: "Further tax is not valid", Fix: "Further tax applies only to supplies to unregistered buyers."},
	"0029": {Title: "Extra tax is not valid", Fix: "Check the extra tax amount."},
	"0030": {Title: "FED payable is not valid", Fix: "Check the FED amount."},
	"0031": {Title: "Discount is not valid", Fix: "Discount must be a non-negative amount."},
	"0035": {Title: "Note date cannot be earlier than the original invoice date", Fix: "A debit note must be dated on or after the original invoice."},
	"0036": {Title: "Note value exceeds the original invoice", Fix: "The debit note value cannot exceed the value of the referenced invoice."},
	"0037": {Title: "Note withheld tax exceeds the original invoice", Fix: "Reduce the withheld tax on the note."},
	"0039": {Title: "Reference invoice not found", Fix: "Check the FBR invoice number in invoiceRefNo."},
	"0041": {Title: "Invoice number is required", Fix: "Provide the reference number."},
	"0042": {Title: "Invoice date is required", Fix: "Provide the invoice date."},
	"0043": {Title: "Invoice date is not valid", Fix: "Use a real calendar date in YYYY-MM-DD format; future dates are not accepted."},
	"0044": {Title: "HS code is required", Fix: "Provide the HS code."},
	"0046": {Title: "Provide rate", Fix: "Select the rate from FBR's SaleTypeToRate list for the sale type; the text must match exactly (e.g. '18%', 'Exempt')."},
	"0050": {Title: "Sales tax withheld is not valid for the sale type", Fix: "Remove the withholding amount for this sale type."},
	"0052": {Title: "Provide proper HS code / HS code does not match the sale type", Fix: "Choose an HS code from the FBR list that is valid for the selected sale type."},
	"0053": {Title: "Buyer registration type is not valid", Fix: "Use 'Registered' or 'Unregistered'."},
	"0055": {Title: "Sales tax withheld is not valid", Fix: "Check the withheld amount."},
	"0056": {Title: "Buyer is not registered in the steel sector", Fix: "Steel sale types require a buyer registered in the steel sector."},
	"0057": {Title: "Reference invoice does not exist", Fix: "Check invoiceRefNo."},
	"0058": {Title: "Self-invoicing is not allowed", Fix: "Seller and buyer cannot be the same registration number."},
	"0064": {Title: "A note already exists against the reference invoice", Fix: "Check existing debit notes for this invoice."},
	"0067": {Title: "Debit note sales tax exceeds the original invoice", Fix: "Reduce the sales tax on the note."},
	"0070": {Title: "Sales tax withholding is not allowed for an unregistered buyer", Fix: "Remove withholding or correct the buyer's registration type."},
	"0071": {Title: "Note not permitted for the referenced invoice", Fix: "Check the original invoice."},
	"0073": {Title: "Seller province (origination of supply) is required", Fix: "Set the company's province in Company settings."},
	"0074": {Title: "Buyer province (destination of supply) is required", Fix: "Set the buyer's province."},
	"0077": {Title: "SRO/Schedule number is required", Fix: "For reduced, zero-rated, exempt and other notified rates enter the schedule/SRO (e.g. 'EIGHTH SCHEDULE Table 1')."},
	"0078": {Title: "SRO item serial number is required", Fix: "Enter the serial number of the item in the schedule/SRO."},
	"0091": {Title: "Extra tax must be empty for reduced-rate goods", Fix: "The software sends extraTax as an empty value for 'Goods at Reduced Rate' automatically."},
	"0158": {Title: "Buyer registration number does not match FBR records", Fix: "Verify the buyer NTN/CNIC with Get_Reg_Type."},
	"0160": {Title: "Buyer name is required", Fix: "Enter the buyer's name."},
	"0161": {Title: "Invoice date cannot be earlier than the original invoice date", Fix: "Correct the note date."},
	"0162": {Title: "Sale type is not valid", Fix: "Select a sale type from the FBR list."},
	"0163": {Title: "Sale type not permitted for the registered business activity", Fix: "Update the business activity on IRIS or use a permitted sale type."},
	"0164": {Title: "Only KWH is allowed as UoM for this HS code", Fix: "Set the UoM to KWH."},
	"0165": {Title: "KG is required as UoM", Fix: "Set the UoM to KG."},
	"0166": {Title: "Quantity / electricity units are required", Fix: "Enter the quantity."},
	"0167": {Title: "Value of sales excluding sales tax is required", Fix: "Enter the value."},
	"0168": {Title: "Cotton ginner transactions require a registered buyer", Fix: "Correct the buyer's registration type."},
	"0169": {Title: "Sales tax withholding restricted to government / FTN holders", Fix: "Remove the withholding."},
	"0174": {Title: "Sales tax is required", Fix: "Enter the sales tax amount."},
	"0175": {Title: "Fixed / notified value or retail price is required", Fix: "Enter the printed retail price for Third Schedule goods."},
	"0401": {Title: "Unauthorized seller access", Fix: "The token does not belong to this seller NTN/CNIC or has expired. Check the token for the selected environment."},
	"0402": {Title: "Unauthorized buyer access", Fix: "Check the buyer registration number."},
}

// Lookup returns guidance for an FBR error code.
func Lookup(code string) (ErrorInfo, bool) {
	code = strings.TrimSpace(code)
	info, ok := catalogue[code]
	if ok {
		info.Code = code
	}
	return info, ok
}

// FixFor returns remediation guidance for a code, or a generic hint.
func FixFor(code string) string {
	if info, ok := Lookup(code); ok {
		return info.Fix
	}
	return "See the error text returned by FBR and the DI technical specification."
}

// Catalogue returns all known codes (for the help screen).
func Catalogue() []ErrorInfo {
	out := make([]ErrorInfo, 0, len(catalogue))
	for code, info := range catalogue {
		info.Code = code
		out = append(out, info)
	}
	sortInfos(out)
	return out
}

func sortInfos(a []ErrorInfo) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j].Code < a[j-1].Code; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}
