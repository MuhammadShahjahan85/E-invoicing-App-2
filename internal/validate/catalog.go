// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

// Package validate checks a DI payload locally before it is sent to FBR, and
// explains FBR's error codes in plain language.
package validate

import (
	"sort"
	"strings"
)

// ErrorInfo explains an FBR DI error code. Title and Detail are FBR's own
// wording (MESSAGE_DESC and BRIEF_MSG_DESC); Fix is this software's guidance.
// The message FBR returns in a response is always shown to the operator
// verbatim as well.
type ErrorInfo struct {
	Code    string `json:"code"`
	Section string `json:"section"` // "sales" or "purchase"
	Title   string `json:"title"`
	Detail  string `json:"detail"`
	Fix     string `json:"fix"`
}

// CatalogueSource names the document the codes are taken from.
const CatalogueSource = "PRAL Technical Specification for DI API v1.12 (24 July 2025), sections 7 and 8"

// catalogue holds the Sales Error Codes (section 7) and Purchase Error Codes
// (section 8) of the specification, with FBR's texts as published.
var catalogue = map[string]ErrorInfo{
	"0001": {Section: "sales", Title: "Seller not registered for sales tax, please provide valid registration/NTN.", Detail: "Seller is not registered for sales tax, please provide valid seller registration/NTN.",
		Fix: "Check the seller NTN (7 digits) or CNIC (13 digits) in Company settings. It must be the registration number the security token was issued to, and the seller must be active for sales tax (check your own NTN under FBR reference library → Buyer check)."},
	"0002": {Section: "sales", Title: "Invalid Buyer Registration No or NTN :", Detail: "Buyer Registration Number or NTN is not in proper format, please provide buyer registration number in 13 digits or NTN in 7 or 9 digits",
		Fix: "Enter the buyer NTN as 7 (or 9) digits or the CNIC as 13 digits, without dashes or spaces. 'Check with FBR' next to the NTN confirms it."},
	"0003": {Section: "sales", Title: "Provide proper invoice type.", Detail: "Invoice type is not valid or empty, please provide valid invoice type",
		Fix: "Use 'Sale Invoice' or 'Debit Note' (FBR's document type list). The software sets this automatically; check imported files."},
	"0005": {Section: "sales", Title: "please provide date in valid format 01-DEC-2021", Detail: "Invoice date is not in proper format, please provide invoice date in \"YYYY-MM-DD\" format. For example: 2025-05-25",
		Fix: "Dates are sent as YYYY-MM-DD automatically; check the date column of imported files."},
	"0006": {Section: "sales", Title: "Sale invoice not exist.", Detail: "Sales invoice does not exist against STWH",
		Fix: "FBR found no sale invoice for the sales tax withholding (STWH) entry. Check the FBR invoice number it refers to."},
	"0007": {Section: "sales", Title: "Wrong Sale type is selected with invoice no (Invoice no)", Detail: "Selected invoice type is not associated with proper registration number, please select actual invoice type",
		Fix: "The sale type does not fit the seller's or buyer's registration. Pick the sale type from FBR's list (FBR reference library → Rates & SROs) that matches the business activity registered on IRIS."},
	"0008": {Section: "sales", Title: "ST withheld at source should either be zero or same as sales tax/fed in st mode.", Detail: "ST withheld at source is not equal to zero or sales tax, please enter ST withheld at source zero or equal to sales tax",
		Fix: "Sales tax withheld at source must be either zero or the full sales tax of the line. Set withholding to 'None' or 'Full' for the line."},
	"0009": {Section: "sales", Title: "Provide Buyer registration No.", Detail: "Buyer Registration Number cannot be empty, please provide proper buyer registration number",
		Fix: "A registered buyer must have an NTN or CNIC. For a buyer who is not registered, choose 'Unregistered' (CNIC needed where section 23(1)(b) applies)."},
	"0010": {Section: "sales", Title: "Provide Buyer Name.", Detail: "Buyer Name cannot be empty, please provide valid buyer name",
		Fix: "Enter the buyer's business name."},
	"0011": {Section: "sales", Title: "Provide invoice type.", Detail: "Invoice type cannot be empty, please provide valid invoice type",
		Fix: "Select the document type (Sale Invoice or Debit Note)."},
	"0012": {Section: "sales", Title: "Provide Buyer Registration Type", Detail: "Buyer Registration type cannot be empty, please provide valid Buyer Registration type",
		Fix: "Select 'Registered' or 'Unregistered' for the buyer; 'Check with FBR' sets it from FBR's records."},
	"0013": {Section: "sales", Title: "Provide valid Sale type.", Detail: "Sale type cannot be empty/null, please provide valid sale type",
		Fix: "Select a sale type from FBR's list. If a new sale type is missing, download the latest FBR data (FBR reference library → Data sync)."},
	"0018": {Section: "sales", Title: "Please provide Sales Tax/FED in ST Mode", Detail: "Sales Tax/FED cannot be empty, please valid provide Sales Tax/FED",
		Fix: "The line needs a sales tax (or FED in sales tax mode) amount. Check the rate and value so the tax is calculated."},
	"0019": {Section: "sales", Title: "Please provide HSCode", Detail: "HS Code cannot be empty, please provide valid HS Code",
		Fix: "Every line needs an HS (PCT) code in the format 0000.0000. Services use Chapter 98 codes such as 9804.0000."},
	"0020": {Section: "sales", Title: "Please provide Rate", Detail: "Rate field cannot be empty, please provide Rate",
		Fix: "Select the rate for the line from FBR's list for the sale type (FBR reference library → Rates & SROs)."},
	"0021": {Section: "sales", Title: "Please provide Value of Sales Excl. ST /Quantity", Detail: "Value of Sales Excl. ST /Quantity cannot be empty, Please provide valid Value of Sales Excl. ST /Quantity",
		Fix: "Enter the quantity and the value excluding sales tax for the line (numbers, not negative)."},
	"0022": {Section: "sales", Title: "Please provide ST withheld at Source or STS Withheld", Detail: "ST withheld at Source or STS Withheld cannot be empty, Please provide valid ST withheld at Source or STS Withheld",
		Fix: "The withholding field cannot be empty. The software sends 0.00 when nothing is withheld; check imported data."},
	"0023": {Section: "sales", Title: "Please provide Sales Tax", Detail: "Sales Tax cannot be empty, Please provide valid Sales Tax",
		Fix: "Provide the sales tax amount; check the rate and value of the line so the tax is calculated."},
	"0024": {Section: "sales", Title: "Please provide ST withheld", Detail: "Sales Tax withheld cannot be empty, Please provide valid Sales Tax withheld",
		Fix: "The sales tax withheld amount cannot be empty; 0.00 is sent when nothing is withheld."},
	"0026": {Section: "sales", Title: "Invoice Reference No. is required.", Detail: "Invoice Reference No. is mandatory requirement for debit/credit note. Please provide valid Invoice Reference No.",
		Fix: "A debit note must quote the FBR invoice number of the original invoice. Create the debit note from the accepted invoice (Debit note button) so the reference is filled in."},
	"0027": {Section: "sales", Title: "Reason is required.", Detail: "Reason is mandatory requirement for debit/credit note. Please provide valid reason for debit/credit note",
		Fix: "A reason is required for a debit/credit note. Enter the reason when creating the note."},
	"0028": {Section: "sales", Title: "Reason Remarks are required.", Detail: "Reason is selected as \"Others\". Please provide valid remarks against this reason",
		Fix: "When the reason for the note is 'Others', describe it in the remarks."},
	"0029": {Section: "sales", Title: "Invoice date must be greater or equal to original invoice no.", Detail: "Debit/Credit note date should be equal or greater from original invoice date",
		Fix: "A debit/credit note cannot be dated before the original invoice."},
	"0030": {Section: "sales", Title: "Unregistered distributer type not allowed before date", Detail: "Unregistered distributer type not allowed before system cut of date",
		Fix: "FBR does not accept this buyer type for invoices dated before its system cut-off date. Check the invoice date and the buyer's registration type."},
	"0031": {Section: "sales", Title: "Provide Sales Tax", Detail: "Sales Tax is not mentioned, please provide Sales Tax",
		Fix: "Sales tax is missing for the line. Check the rate and value."},
	"0032": {Section: "sales", Title: "STWH can only be created for GOV/FTN Holders.", Detail: "User is not FTN holder, STWH can only be created for GOV/FTN Holders without sales invoice.",
		Fix: "Sales tax withholding entries without a sale invoice can only be made by government departments and FTN holders."},
	"0034": {Section: "sales", Title: "{0} only allowed within {1} days of invoice date of the original invoice", Detail: "Debit/Credit note can only be added within 180 days of original invoice date",
		Fix: "A debit/credit note can only be issued within 180 days of the original invoice date. For an older invoice, take advice before adjusting it (section 9 of the Sales Tax Act)."},
	"0035": {Section: "sales", Title: "{0} date must be greater or equal to original invoice date.", Detail: "Note Date must be greater or equal to original invoice date",
		Fix: "The note date cannot be earlier than the original invoice date."},
	"0036": {Section: "sales", Title: "Total {1} value of {0} invoice(s) is greater than {1} of original invoice. Value of Sales", Detail: "Credit Note Value of Sale must be less or equal to the value of Sale in original invoice.",
		Fix: "The total value of notes against an invoice cannot exceed the value of the original invoice. Check earlier notes against the same invoice."},
	"0037": {Section: "sales", Title: "Total {1} value of {0} invoice(s) is greater than {1} of original invoice.ST Withheld as WH Agent", Detail: "Credit Note Value of ST Withheld must be less or equal to the value of ST Withheld in original invoice.",
		Fix: "The total sales tax withheld on notes cannot exceed the withheld tax of the original invoice."},
	"0039": {Section: "sales", Title: "Sale invoice not exist.", Detail: "For registered users, STWH invoice fields must be same as sale invoice",
		Fix: "For registered buyers, the withholding (STWH) entry must match the fields of the sale invoice. Check the referenced invoice."},
	"0041": {Section: "sales", Title: "Provide invoice No.", Detail: "Invoice number cannot be empty, please provide invoice number.",
		Fix: "The invoice number cannot be empty."},
	"0042": {Section: "sales", Title: "Provide invoice date.", Detail: "Invoice date cannot be empty, please provide invoice date.",
		Fix: "Enter the invoice date."},
	"0043": {Section: "sales", Title: "Provide valid Date.", Detail: "Invoice date is not valid, please provide valid invoice date.",
		Fix: "FBR rejected the invoice date. Check that it is not in the future. FBR's clock runs on UTC, so just after midnight Pakistan time an invoice dated 'today' can be refused for a few hours; if so, submit again a little later (the invoice is kept as a draft or in the queue)."},
	"0044": {Section: "sales", Title: "Provide HS Code.", Detail: "HS Code cannot be empty, please provide HS Code",
		Fix: "Every line needs an HS (PCT) code in the format 0000.0000."},
	"0046": {Section: "sales", Title: "Provide rate.", Detail: "Rate cannot be empty, please provide valid rate as per selected Sales Type.",
		Fix: "The rate cannot be empty and must be one FBR accepts for the sale type. Pick it from FBR's list (FBR reference library → Rates & SROs)."},
	"0050": {Section: "sales", Title: "Please provide valid Sales Tax withheld. For sale type 'Cotton ginners', Sales Tax Withheld must be equal to Sales Tax or zero", Detail: "Please provide valid Sales Tax withheld. For sale type 'Cotton ginners', Sales Tax Withheld must be equal to Sales Tax or zero",
		Fix: "For the sale type 'Cotton ginners', the sales tax withheld must equal the sales tax or be zero."},
	"0052": {Section: "sales", Title: "Please provide valid HS Code against invoice no:", Detail: "HS Code that does not match with provided sale type, Please provide valid HS Code against sale type",
		Fix: "The HS code does not match the selected sale type. Check the HS code / sale type pair (FBR reference library → HS codes)."},
	"0053": {Section: "sales", Title: "Provided buyer registration type is invalid", Detail: "Buyer Registration Type is invalid, please provide valid Buyer Registration Type",
		Fix: "Use 'Registered' or 'Unregistered' exactly."},
	"0055": {Section: "sales", Title: "Please Provide ST Withheld as WH Agent", Detail: "Sales tax withheld cannot be empty or invalid format. Please provide valid sales tax withheld.",
		Fix: "Sales tax withheld cannot be empty or in an invalid format; enter 0.00 or the amount withheld."},
	"0056": {Section: "sales", Title: "Buyer not exists in steel sector.", Detail: "Buyer does not exist in steel sector",
		Fix: "For steel sector sale types the buyer must be registered in the steel sector on IRIS. Check the buyer or the sale type."},
	"0057": {Section: "sales", Title: "Reference Invoice does not exist.", Detail: "Reference invoice for debit/ credit note does not exists. Please provide valid Invoice Reference No.",
		Fix: "The FBR invoice number quoted on the debit/credit note was not found. Use the number exactly as FBR issued it; create the note from the original invoice."},
	"0058": {Section: "sales", Title: "Self-invoicing not allowed", Detail: "Buyer and Seller Registration number are same, this type of invoice is not allowed",
		Fix: "Buyer and seller registration numbers are the same. Enter the buyer's NTN/CNIC, not your own."},
	"0064": {Section: "sales", Title: "Reference invoice already exist.", Detail: "Credit note is already added to a invoice",
		Fix: "A credit note has already been recorded against this invoice."},
	"0067": {Section: "sales", Title: "{1} of {0} invoice is greater than {1} of original invoice.", Detail: "Sales Tax value of Debit Note is greater than original invoice's sales tax",
		Fix: "The sales tax on the debit note is greater than the sales tax of the original invoice. Check the values of the note."},
	"0068": {Section: "sales", Title: "{1} of {0} invoice is less than {1} of original invoice.", Detail: "Sales Tax value of Credit Note is less than original invoice's sales tax according to the rate.",
		Fix: "The sales tax on the credit note does not agree with the original invoice's rate. Recalculate the note."},
	"0070": {Section: "sales", Title: "STWH cannot be created for unregistered buyers.", Detail: "User is not registered, STWH is allowed only for registered user",
		Fix: "Sales tax withholding (STWH) is allowed only for registered buyers."},
	"0071": {Section: "sales", Title: "Entry of {0} against the declared invoice is not allowed.", Detail: "Credit note allowed to add only for specific users",
		Fix: "Credit notes against declared invoices are allowed only for specific users. Contact PRAL (DI CRM) or your tax office."},
	"0073": {Section: "sales", Title: "Provide Sale Origination Province of Supplier", Detail: "Sale Origination Province of Supplier cannot be empty, please provide valid Sale Origination Province of Supplier.",
		Fix: "Set the seller's province in Company settings (FBR's province list)."},
	"0074": {Section: "sales", Title: "Provide Destination of Supply", Detail: "Destination of Supply cannot be empty, please provide valid Destination of Supply",
		Fix: "Set the buyer's province (FBR's province list)."},
	"0077": {Section: "sales", Title: "Provide SRO/Schedule No.", Detail: "SRO/Schedule Number cannot be empty, please provide valid SRO/Schedule Number",
		Fix: "This sale type or rate needs an SRO or schedule number. Use 'Pick SRO / schedule from FBR' in the line details."},
	"0078": {Section: "sales", Title: "Provide Item Sr. No.", Detail: "Item serial number cannot be empty, please provide valid item serial number",
		Fix: "This SRO or schedule needs the item serial number. Use 'Pick serial no. from FBR' in the line details."},
	"0079": {Section: "sales", Title: "If Value of Sales Excl. ST greater than {0}. Rate {1} not allowed.", Detail: "If sales value is greater than 20,000 than rate 5% is not allowed",
		Fix: "FBR does not accept a 5% rate when the value of sales excluding tax is above Rs 20,000 (for example electricity supplied to retailers). Use the rate FBR lists for the amount (FBR reference library → Rates & SROs)."},
	"0080": {Section: "sales", Title: "Please provide Further Tax", Detail: "Further Tax' cannot be empty, please provide valid Further Tax",
		Fix: "Further tax cannot be empty; 0.00 is sent when none applies. Check imported data."},
	"0081": {Section: "sales", Title: "Please provide Input Credit not Allowed", Detail: "'Input Credit not Allowed' cannot be empty, please provide ‘Input Credit not Allowed'",
		Fix: "FBR expects an 'Input Credit not Allowed' value for this sale type. The DI v1.12 API has no such field; raise it with PRAL support (DI CRM)."},
	"0082": {Section: "sales", Title: "The Seller is not registered for sales tax. Please provide a valid registration/NTN.", Detail: "The Seller is not registered for sales tax. Please provide a valid registration/NTN.",
		Fix: "FBR's records show the seller is not registered for sales tax. Check the NTN/CNIC in Company settings and the registration status on IRIS."},
	"0083": {Section: "sales", Title: "Mismatch Seller Registration No.", Detail: "Seller Reg No. doesn’t match. Please provide valid Seller Registration Number",
		Fix: "The seller NTN/CNIC on the invoice does not match the registration the security token belongs to. Check Company settings and that the token is for this environment."},
	"0085": {Section: "sales", Title: "Please provide Total Value of Sales (In case of PFAD only)", Detail: "Total Value of Sales is not provided, please provide valid Total Value of Sales (In case of PFAD only)",
		Fix: "For PFAD supplies FBR needs the total value of sales on the line."},
	"0086": {Section: "sales", Title: "You are not an EFS license holder who has imported Compressor Scrap in the last 12 months.", Detail: "You are not an EFS license holder who has imported Compressor Scrap in the last 12 months.",
		Fix: "This sale type is limited to EFS licence holders who imported compressor scrap in the last 12 months."},
	"0087": {Section: "sales", Title: "Petroleum Levy rates not configured properly.", Detail: "Petroleum Levy rates not configured properly. Please update levy rates properly",
		Fix: "FBR's petroleum levy rates are not configured for this case. Contact PRAL support (DI CRM)."},
	"0088": {Section: "sales", Title: "Alphanumeric and (-) contained invoice No. is allowed. (-) should be in between Alphanumeric string.", Detail: "Invoice number is not valid, please provide valid invoice number in alphanumeric format. For example: Inv-001",
		Fix: "Invoice numbers may contain letters and digits with hyphens between them, e.g. Inv-001."},
	"0089": {Section: "sales", Title: "Please provide FED Charged", Detail: "FED Charged cannot be empty, please provide valid FED Charged",
		Fix: "FED in sales tax mode is required for this sale type; enter the FED amount on the line."},
	"0090": {Section: "sales", Title: "Please provide Fixed / notified value or Retail Price", Detail: "Fixed / notified value or Retail Price cannot be empty, please provide valid Fixed / notified value or Retail Price",
		Fix: "Enter the retail price (Third Schedule goods) or the notified value on the line."},
	"0091": {Section: "sales", Title: "Extra tax must be empty.", Detail: "Extra tax must be empty.",
		Fix: "Extra tax must be left empty (not 0) for reduced-rate, exempt, zero-rated and cotton ginner supplies. The software does this automatically; check imported data."},
	"0092": {Section: "sales", Title: "Provide Valid Sale Type.", Detail: "Purchase type cannot be empty, please provide valid purchase type",
		Fix: "Select a valid sale type."},
	"0093": {Section: "sales", Title: "Selected Sale Type are not allowed to Manufacturer.", Detail: "Selected Sale is are not allowed to Manufacturer. Please select proper sale type",
		Fix: "This sale type is not allowed for a manufacturer. Check the business activity registered on IRIS and the sale type."},
	"0095": {Section: "sales", Title: "Please provide Extra Tax", Detail: "Extra Tax cannot be empty, please provide valid extra tax",
		Fix: "Extra tax is required for this sale type or rate; enter it on the line."},
	"0096": {Section: "sales", Title: "For selected HSCode only KWH UOM is allowed.", Detail: "For provided HS Code, only KWH UOM is allowed",
		Fix: "For this HS code (electricity) the unit of measure must be KWH."},
	"0097": {Section: "sales", Title: "Provide UOM KG.", Detail: "Please provide UOM in KG",
		Fix: "For this HS code the unit of measure must be KG."},
	"0098": {Section: "sales", Title: "Please provide Quantity / Electricity Units", Detail: "Quantity / Electricity Unit cannot be empty, please provide valid Quantity / Electricity Unit",
		Fix: "Enter the quantity (or electricity units) for the line."},
	"0099": {Section: "sales", Title: "Provide uom.", Detail: "UOM is not valid. UOM must be according to given HS Code",
		Fix: "Use the unit of measure FBR prescribes for the HS code (shown in the invoice editor; HS_UOM). Units are case-sensitive, e.g. 'KG' not 'kg'."},
	"0100": {Section: "sales", Title: "Cotton Ginners allowed against registered buyers only.", Detail: "Registered user cannot add sale invoice. Only cotton ginner sale type is allowed for registered users.",
		Fix: "Cotton ginner supplies are allowed only to registered buyers."},
	"0101": {Section: "sales", Title: "Please Use Toll Manufacturing Sale Type for Steel Sector.", Detail: "Sale type is not selected properly, please use Toll Manufacturing Sale Type for Steel Sector.",
		Fix: "Use the 'Toll Manufacturing' sale type for steel sector toll manufacturing."},
	"0102": {Section: "sales", Title: "Calculated tax not matched in 3rd schedule", Detail: "The calculated sales tax not calculated as per 3rd schedule calculation formula",
		Fix: "Sales tax on Third Schedule goods must be calculated on the retail price (retail price × quantity × rate). Enter the printed retail price; the software calculates the tax."},
	"0103": {Section: "sales", Title: "The calculated tax for Potassium Chlorate does not match.", Detail: "Calculated tax not matched for potassium chlorate. Calculated value doesn’t match according to potassium chlorate for sales potassium invoices.",
		Fix: "Tax on potassium chlorate must follow FBR's formula (rate plus the per-kg amount). Check the rate, the quantity and that the unit is KG."},
	"0104": {Section: "sales", Title: "The calculated percentage sales tax does not match.", Detail: "Calculated percentage of sales tax not matched. Calculation must be correct with respect to provided rate",
		Fix: "Sales tax must equal value × rate, rounded to 2 decimals. Do not edit tax amounts by hand; check imported values."},
	"0105": {Section: "sales", Title: "The calculated sales tax for the quantity is incorrect.", Detail: "The calculated sales tax for the quantity is incorrect.",
		Fix: "For rates per unit (e.g. rupees per kg) the tax must equal quantity × rate. Check the quantity and unit."},
	"0106": {Section: "sales", Title: "The Buyer is not registered for sales tax. Please provide a valid registration/NTN.", Detail: "The Buyer is not registered for sales tax. Please provide a valid registration/NTN.",
		Fix: "FBR's records show the buyer is not registered for sales tax. Mark the buyer 'Unregistered' (further tax may apply) or verify the NTN with 'Check with FBR'."},
	"0107": {Section: "sales", Title: "Mismatch Buyer Registration No.", Detail: "Buyer Reg No. doesn’t match. Please provide valid Buyer Registration Number",
		Fix: "The buyer registration number does not match FBR's records. Verify the NTN/CNIC with 'Check with FBR'."},
	"0108": {Section: "sales", Title: "Invalid Seller Registration No or NTN", Detail: "Seller Reg No. is not valid. Please provide valid Seller Registration Number/NTN",
		Fix: "The seller NTN/CNIC is not valid. Check Company settings (7-digit NTN or 13-digit CNIC)."},
	"0109": {Section: "sales", Title: "Wrong invoice type is selected in invoice no", Detail: "Invoice type is not selected properly, please select proper invoice type",
		Fix: "Select the correct document type (Sale Invoice or Debit Note)."},
	"0111": {Section: "sales", Title: "Wrong purchase type is selected with invoice no", Detail: "Purchase type is not selected properly, please provide proper purchase type",
		Fix: "The purchase type is not correct (cotton ginner purchases). Check the sale type."},
	"0113": {Section: "sales", Title: "System is unable to parse date. Please provide date in valid format dd-MMM-yy.", Detail: "Date is not in proper format, please provide date in \"YYYY-MM-DD\" format. For example: 2025-05-25",
		Fix: "Dates are sent as YYYY-MM-DD automatically; check the date column of imported files."},
	"0156": {Section: "purchase", Title: "Invalid NTN / Reg No. provided.", Detail: "NTN/Reg. No is invalid/Null, please provide valid NTN/Reg. No.",
		Fix: "Check the NTN / registration number (7-digit NTN or 13-digit CNIC)."},
	"0157": {Section: "purchase", Title: "The Buyer is not registered for sales tax. Please provide a valid registration/NTN.", Detail: "The Buyer is not registered for sales tax. Please provide valid Registration/NTN.",
		Fix: "FBR's records show the buyer is not registered for sales tax. Verify the NTN with 'Check with FBR'."},
	"0158": {Section: "purchase", Title: "Mismatch Buyer Registration No.", Detail: "Buyer Reg No. doesn’t match. Please provide valid Buyer Registration Number",
		Fix: "The buyer registration number does not match FBR's records. Verify the NTN/CNIC."},
	"0159": {Section: "purchase", Title: "FTN holder as seller not allowed for purchases.", Detail: "FTN Holder as Seller is not allowed for purchases",
		Fix: "An FTN holder cannot be the seller on a purchase."},
	"0160": {Section: "purchase", Title: "Provide Buyer Name.", Detail: "Buyer Name cannot be empty, please provide valid buyer name",
		Fix: "Enter the buyer's name."},
	"0161": {Section: "purchase", Title: "Invoice Date must be greater or equal to {0}", Detail: "Invoice Date must be greater or equal to original sale invoice date",
		Fix: "The invoice date cannot be earlier than the original sale invoice date."},
	"0162": {Section: "purchase", Title: "Provide Sale Type.", Detail: "Sale Type cannot be empty/Invalid, please provide valid Sale Type",
		Fix: "Select a valid sale type."},
	"0163": {Section: "purchase", Title: "Selected Sale Type are not allowed to Manufacturer.", Detail: "Provided Sale Type is not allowed for Manufacturer.",
		Fix: "This sale type is not allowed for a manufacturer. Check the business activity on IRIS."},
	"0164": {Section: "purchase", Title: "For selected HSCode only KWH UOM is allowed.", Detail: "For provided HS Code, only KWH UOM is allowed",
		Fix: "For this HS code (electricity) the unit of measure must be KWH."},
	"0165": {Section: "purchase", Title: "Provide UOM KG.", Detail: "Please provide UOM in KG",
		Fix: "For this HS code the unit of measure must be KG."},
	"0166": {Section: "purchase", Title: "Please provide Quantity / Electricity Units", Detail: "Quantity / Electricity Unit cannot be empty, please provide valid Quantity / Electricity Unit",
		Fix: "Enter the quantity (or electricity units)."},
	"0167": {Section: "purchase", Title: "Provide Value of Sales Excl. ST", Detail: "Value of Sales Excl. ST cannot be empty/Invalid, please provide valid Value of Sales Excl. ST",
		Fix: "Enter a valid value of sales excluding sales tax."},
	"0168": {Section: "purchase", Title: "Cotton Ginners allowed against registered buyers only.", Detail: "Only cotton ginner purchase type is allowed for registered users.",
		Fix: "Only the cotton ginner purchase type is allowed for registered users."},
	"0169": {Section: "purchase", Title: "STWH can only be created for GOV/FTN Holders.", Detail: "User is not FTN holder, STWH can only be created for GOV/FTN Holders without purchase invoice.",
		Fix: "Sales tax withholding entries without a purchase invoice can only be made by government departments and FTN holders."},
	"0170": {Section: "purchase", Title: "If Value of Sales Excl. ST greater than {0}. Rate {1} not allowed.", Detail: "If Value of Sales Excl. ST greater than 20000 than rate 5% is not allowed.",
		Fix: "FBR does not accept a 5% rate when the value of sales excluding tax is above Rs 20,000."},
	"0171": {Section: "purchase", Title: "You are not an EFS license holder who has imported Compressor Scrap in the last 12 months.", Detail: "You are not an EFS license holder who has imported Compressor Scrap in the last 12 months.",
		Fix: "This purchase type is limited to EFS licence holders who imported compressor scrap in the last 12 months."},
	"0172": {Section: "purchase", Title: "Petroleum Levy rates not configured properly.", Detail: "Petroleum Levy rates not configured properly. Please update levy rates properly",
		Fix: "FBR's petroleum levy rates are not configured for this case. Contact PRAL support (DI CRM)."},
	"0173": {Section: "purchase", Title: "Alphanumeric and (-) contained invoice No. is allowed. (-) should be in between Alphanumeric string.", Detail: "Invoice number is not valid, please provide valid invoice number in alphanumeric format. For example: Inv-001",
		Fix: "Invoice numbers may contain letters and digits with hyphens between them, e.g. Inv-001."},
	"0174": {Section: "purchase", Title: "Please provide Sales Tax", Detail: "Sales Tax cannot be empty, please provide valid Sales Tax",
		Fix: "Provide the sales tax amount."},
	"0175": {Section: "purchase", Title: "Please provide Fixed / notified value or Retail Price", Detail: "Fixed / notified value or Retail Price cannot be empty, please provide valid Fixed / notified value or Retail Price",
		Fix: "Enter the retail price or notified value."},
	"0176": {Section: "purchase", Title: "Please provide ST withheld at Source", Detail: "ST withheld at Source cannot be empty, please provide valid ST withheld at Source",
		Fix: "Sales tax withheld at source cannot be empty; enter 0.00 or the amount withheld."},
	"0177": {Section: "purchase", Title: "Please provide Further Tax", Detail: "Further Tax cannot be empty, please provide valid further tax",
		Fix: "Further tax cannot be empty; enter 0.00 when none applies."},
	"0300": {Section: "sales", Title: "Provided decimal value is not valid at field", Detail: "Discount Value is not valid at item 1 | Total Value is not valid at item 1 | Fed Payable Value is not valid at item 1 | Extra Tax Value is not valid at item 1 | Further Tax Value is not valid at item 1 | SalesTaxWithheldAtSource Value is not valid at item 1 | Quantity Value is not valid at item 1",
		Fix: "One of the amounts (discount, total, FED, extra tax, further tax, sales tax withheld or quantity) is not a valid number. Amounts are sent with 2 decimals and quantities with up to 4; check imported data for text or commas."},
	"0401": {Section: "sales", Title: "The provided seller NTN/CNIC does not have a valid or authorized access token", Detail: "Unauthorized access: Provided seller registration number is not 13 digits (CNIC) or 7 digits (NTN) or the authorized token does not exist against seller registration number",
		Fix: "The security token is not authorised for this seller NTN/CNIC, or the number is not 7 or 13 digits. Make sure the token belongs to this company and to the environment in use (sandbox and production tokens are different)."},
	"0402": {Section: "sales", Title: "The provided buyer NTN/CNIC does not have a valid or authorized access token", Detail: "Unauthorized access: Provided buyer registration number is not 13 digits (CNIC) or 7 digits (NTN) or the authorized token does not exist against buyer registration number",
		Fix: "The buyer NTN/CNIC must be 7 or 13 digits. If the number is right, contact PRAL support (DI CRM): the token is not authorised for this request."},
}

// Lookup returns FBR's description of an error code and how to fix it.
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

// Catalogue returns all known codes in code order (for the help screen).
func Catalogue() []ErrorInfo {
	out := make([]ErrorInfo, 0, len(catalogue))
	for code, info := range catalogue {
		info.Code = code
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}
