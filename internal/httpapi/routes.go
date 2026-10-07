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
	m.HandleFunc("GET /api/v1/companies/{cid}/invoices/{id}/payload", s.perm(PermRead, s.handleInvoicePayload))
	m.HandleFunc("GET /api/v1/companies/{cid}/invoices/{id}/calls", s.perm(PermRead, s.handleInvoiceCalls))
	m.HandleFunc("GET /api/v1/companies/{cid}/invoices/{id}/print", s.perm(PermRead, s.handlePrint))
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
}
