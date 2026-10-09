// Copyright (c) 2026 Veridian Partners Consultancy Private Limited. All rights reserved.
// Veridian E-invoicing Pakistan is proprietary software; see the LICENSE file.

package httpapi

import "net/http"

func (s *Server) routes(m *http.ServeMux) {
	// Public.
	m.HandleFunc("GET /api/v1/system/status", s.handleStatus)
	m.HandleFunc("POST /api/v1/system/setup", s.handleSetup)
	m.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	m.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	m.HandleFunc("GET /api/v1/system/ca.crt", s.handleCACert) // public: phones install it before signing in
	m.HandleFunc("GET /api/v1/system/link-qr.svg", s.perm(PermSelf, s.handleLinkQR))
	m.HandleFunc("GET /api/v1/system/addresses", s.perm(PermSelf, s.handleAddresses))

	// Session.
	m.HandleFunc("POST /api/v1/auth/logout", s.perm(PermSelf, s.handleLogout))
	m.HandleFunc("GET /api/v1/auth/me", s.perm(PermSelf, s.handleMe))
	m.HandleFunc("POST /api/v1/auth/password", s.perm(PermSelf, s.handlePassword))
	m.HandleFunc("GET /api/v1/meta", s.perm(PermSelf, s.handleMeta))

	// Companies.
	m.HandleFunc("GET /api/v1/companies", s.perm(PermSelf, s.handleListCompanies))
	m.HandleFunc("POST /api/v1/companies", s.perm(PermCompanyCreate, s.handleCreateCompany))
	m.HandleFunc("GET /api/v1/companies/{cid}", s.perm(PermRead, s.handleGetCompany))
	m.HandleFunc("PUT /api/v1/companies/{cid}", s.perm(PermCompanyWrite, s.handleUpdateCompany))
	m.HandleFunc("POST /api/v1/companies/{cid}/token", s.perm(PermCompanyWrite, s.handleSetToken))
	m.HandleFunc("POST /api/v1/companies/{cid}/test-connection", s.perm(PermCompanyWrite, s.handleTestConnection))
	m.HandleFunc("POST /api/v1/companies/{cid}/sync-reference", s.perm(PermCompanyWrite, s.handleSyncReference))
	m.HandleFunc("GET /api/v1/companies/{cid}/logo", s.perm(PermRead, s.handleGetLogo))
	m.HandleFunc("PUT /api/v1/companies/{cid}/logo", s.perm(PermCompanyWrite, s.handlePutLogo))
	m.HandleFunc("DELETE /api/v1/companies/{cid}/logo", s.perm(PermCompanyWrite, s.handleDeleteLogo))
	m.HandleFunc("GET /api/v1/companies/{cid}/dashboard", s.perm(PermRead, s.handleDashboard))
	m.HandleFunc("GET /api/v1/companies/{cid}/alerts", s.perm(PermRead, s.handleAlerts))
	m.HandleFunc("GET /api/v1/companies/{cid}/search", s.perm(PermRead, s.handleSearch))
	m.HandleFunc("GET /api/v1/companies/{cid}/compliance", s.perm(PermReports, s.handleCompliance))
	m.HandleFunc("GET /api/v1/companies/{cid}/compliance/pack", s.perm(PermReports, s.handleReportPack))

	// Masters.
	m.HandleFunc("GET /api/v1/companies/{cid}/customers", s.perm(PermRead, s.handleListCustomers))
	m.HandleFunc("POST /api/v1/companies/{cid}/customers", s.perm(PermMastersWrite, s.handleSaveCustomer))
	m.HandleFunc("GET /api/v1/companies/{cid}/customers/{id}", s.perm(PermRead, s.handleGetCustomer))
	m.HandleFunc("PUT /api/v1/companies/{cid}/customers/{id}", s.perm(PermMastersWrite, s.handleSaveCustomer))
	m.HandleFunc("POST /api/v1/companies/{cid}/customers/{id}/check", s.perm(PermMastersWrite, s.handleCheckCustomer))
	m.HandleFunc("POST /api/v1/companies/{cid}/buyer-check", s.perm(PermRead, s.handleBuyerCheck))
	m.HandleFunc("GET /api/v1/companies/{cid}/products", s.perm(PermRead, s.handleListProducts))
	m.HandleFunc("POST /api/v1/companies/{cid}/products", s.perm(PermMastersWrite, s.handleSaveProduct))
	m.HandleFunc("GET /api/v1/companies/{cid}/products/{id}", s.perm(PermRead, s.handleGetProduct))
	m.HandleFunc("PUT /api/v1/companies/{cid}/products/{id}", s.perm(PermMastersWrite, s.handleSaveProduct))

	// Reference data.
	m.HandleFunc("GET /api/v1/companies/{cid}/ref/{kind}", s.perm(PermRead, s.handleRef))

	// Invoices.
	m.HandleFunc("GET /api/v1/companies/{cid}/invoices", s.perm(PermRead, s.handleListInvoices))
	m.HandleFunc("POST /api/v1/companies/{cid}/invoices", s.perm(PermInvoiceWrite, s.handleCreateInvoice))
	m.HandleFunc("POST /api/v1/companies/{cid}/invoices/compute", s.perm(PermRead, s.handleComputeInvoice))
	m.HandleFunc("GET /api/v1/companies/{cid}/invoice-by-ref/{ref}", s.perm(PermRead, s.handleInvoiceByRef))
	m.HandleFunc("GET /api/v1/companies/{cid}/invoices/{id}", s.perm(PermRead, s.handleGetInvoice))
	m.HandleFunc("PUT /api/v1/companies/{cid}/invoices/{id}", s.perm(PermInvoiceWrite, s.handleUpdateInvoice))
	m.HandleFunc("DELETE /api/v1/companies/{cid}/invoices/{id}", s.perm(PermInvoiceWrite, s.handleDeleteInvoice))
	m.HandleFunc("POST /api/v1/companies/{cid}/invoices/{id}/validate", s.perm(PermInvoiceWrite, s.handleValidateInvoice))
	m.HandleFunc("POST /api/v1/companies/{cid}/invoices/{id}/submit", s.perm(PermInvoiceWrite, s.handleSubmitInvoice))
	m.HandleFunc("POST /api/v1/companies/{cid}/invoices/{id}/retry", s.perm(PermInvoiceWrite, s.handleRetryInvoice))
	m.HandleFunc("POST /api/v1/companies/{cid}/invoices/{id}/resolve", s.perm(PermInvoiceManage, s.handleResolveInvoice))
	m.HandleFunc("POST /api/v1/companies/{cid}/invoices/{id}/cancel", s.perm(PermInvoiceManage, s.handleCancelInvoice))
	m.HandleFunc("POST /api/v1/companies/{cid}/invoices/{id}/debit-note", s.perm(PermInvoiceManage, s.handleDebitNote))
	m.HandleFunc("POST /api/v1/companies/{cid}/invoices/{id}/duplicate", s.perm(PermInvoiceWrite, s.handleDuplicateInvoice))
	m.HandleFunc("GET /api/v1/companies/{cid}/invoices/{id}/payload", s.perm(PermRead, s.handleInvoicePayload))
	m.HandleFunc("GET /api/v1/companies/{cid}/invoices/{id}/calls", s.perm(PermRead, s.handleInvoiceCalls))
	m.HandleFunc("GET /api/v1/companies/{cid}/invoices/{id}/print", s.perm(PermRead, s.handlePrint))
	m.HandleFunc("GET /api/v1/companies/{cid}/invoices/{id}/pdf", s.perm(PermRead, s.handleInvoicePDF))
	m.HandleFunc("GET /api/v1/companies/{cid}/invoices/{id}/qr.svg", s.perm(PermRead, s.handleQRSVG))
	m.HandleFunc("GET /api/v1/companies/{cid}/invoices/{id}/qr.png", s.perm(PermRead, s.handleQRPNG))

	// Import.
	m.HandleFunc("GET /api/v1/import/template.csv", s.perm(PermRead, s.handleTemplateCSV))
	m.HandleFunc("GET /api/v1/import/template.xlsx", s.perm(PermRead, s.handleTemplateXLSX))
	m.HandleFunc("POST /api/v1/companies/{cid}/import", s.perm(PermInvoiceWrite, s.handleImport))

	// Scenarios.
	m.HandleFunc("GET /api/v1/companies/{cid}/scenarios", s.perm(PermRead, s.handleScenarios))
	m.HandleFunc("PUT /api/v1/companies/{cid}/scenarios/assigned", s.perm(PermScenarios, s.handleAssignScenarios))
	m.HandleFunc("POST /api/v1/companies/{cid}/scenarios/{sn}/run", s.perm(PermScenarios, s.handleRunScenario))

	// Reports.
	m.HandleFunc("GET /api/v1/companies/{cid}/reports/{kind}", s.perm(PermReports, s.handleReport))
	m.HandleFunc("GET /api/v1/companies/{cid}/calls", s.perm(PermReports, s.handleCalls))

	// Chapter XIV requirements: closings (rule 150R(4)(f)), return filing
	// extensions, Stock Transfer Notes (STGO 25 of 2026), the "Integrated
	// with FBR" signboard (rule 150R(11)) and the invoice signing key.
	m.HandleFunc("GET /api/v1/companies/{cid}/closings", s.perm(PermReports, s.handleClosings))
	m.HandleFunc("GET /api/v1/companies/{cid}/return-extensions", s.perm(PermRead, s.handleListExtensions))
	m.HandleFunc("PUT /api/v1/companies/{cid}/return-extensions/{period}", s.perm(PermCompanyWrite, s.handleSetExtension))
	m.HandleFunc("DELETE /api/v1/companies/{cid}/return-extensions/{period}", s.perm(PermCompanyWrite, s.handleDeleteExtension))
	m.HandleFunc("GET /api/v1/companies/{cid}/stock-transfers", s.perm(PermRead, s.handleListTransfers))
	m.HandleFunc("POST /api/v1/companies/{cid}/stock-transfers", s.perm(PermInvoiceWrite, s.handleCreateTransfer))
	m.HandleFunc("GET /api/v1/companies/{cid}/stock-transfers/{id}", s.perm(PermRead, s.handleGetTransfer))
	m.HandleFunc("POST /api/v1/companies/{cid}/stock-transfers/{id}/receive", s.perm(PermInvoiceWrite, s.handleReceiveTransfer))
	m.HandleFunc("POST /api/v1/companies/{cid}/stock-transfers/{id}/cancel", s.perm(PermInvoiceManage, s.handleCancelTransfer))
	m.HandleFunc("GET /api/v1/companies/{cid}/stock-transfers/{id}/print", s.perm(PermRead, s.handlePrintTransfer))
	m.HandleFunc("GET /api/v1/companies/{cid}/signboard", s.perm(PermRead, s.handleSignboard))
	m.HandleFunc("GET /api/v1/system/signing-key", s.perm(PermSelf, s.handleSigningKey))
	m.HandleFunc("GET /api/v1/system/public-ip", s.perm(PermCompanyWrite, s.handlePublicIP))

	// Incidents.
	m.HandleFunc("GET /api/v1/companies/{cid}/incidents", s.perm(PermRead, s.handleListIncidents))
	m.HandleFunc("POST /api/v1/companies/{cid}/incidents", s.perm(PermIncidents, s.handleSaveIncident))
	m.HandleFunc("PUT /api/v1/companies/{cid}/incidents/{id}", s.perm(PermIncidents, s.handleSaveIncident))
	m.HandleFunc("GET /api/v1/companies/{cid}/incidents/{id}/letter", s.perm(PermRead, s.handleIncidentLetter))

	// Administration.
	m.HandleFunc("GET /api/v1/users", s.perm(PermUsers, s.handleListUsers))
	m.HandleFunc("POST /api/v1/users", s.perm(PermUsers, s.handleSaveUser))
	m.HandleFunc("PUT /api/v1/users/{id}", s.perm(PermUsers, s.handleSaveUser))
	m.HandleFunc("GET /api/v1/companies/{cid}/api-keys", s.perm(PermAPIKeys, s.handleListAPIKeys))
	m.HandleFunc("POST /api/v1/companies/{cid}/api-keys", s.perm(PermAPIKeys, s.handleCreateAPIKey))
	m.HandleFunc("DELETE /api/v1/companies/{cid}/api-keys/{id}", s.perm(PermAPIKeys, s.handleRevokeAPIKey))
	m.HandleFunc("GET /api/v1/audit", s.perm(PermAudit, s.handleAudit))
	m.HandleFunc("GET /api/v1/audit/verify", s.perm(PermAudit, s.handleAuditVerify))
	m.HandleFunc("GET /api/v1/system/backups", s.perm(PermSystem, s.handleListBackups))
	m.HandleFunc("POST /api/v1/system/backups", s.perm(PermSystem, s.handleCreateBackup))
	m.HandleFunc("GET /api/v1/system/backups/{name}", s.perm(PermSystem, s.handleDownloadBackup))
	m.HandleFunc("GET /api/v1/system/license", s.perm(PermSelf, s.handleGetLicense))
	m.HandleFunc("POST /api/v1/system/license", s.perm(PermSystem, s.handleInstallLicense))
	m.HandleFunc("GET /api/v1/system/fbr-logo", s.perm(PermSelf, s.handleGetFBRLogo))
	m.HandleFunc("PUT /api/v1/system/fbr-logo", s.perm(PermSystem, s.handlePutFBRLogo))
	m.HandleFunc("GET /api/v1/system/info", s.perm(PermSystem, s.handleSystemInfo))
	m.HandleFunc("GET /api/v1/system/notifications", s.perm(PermSystem, s.handleGetNotifications))
	m.HandleFunc("PUT /api/v1/system/notifications", s.perm(PermSystem, s.handleSaveNotifications))
	m.HandleFunc("POST /api/v1/system/notifications/test", s.perm(PermSystem, s.handleTestNotification))
}
