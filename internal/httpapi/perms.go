package httpapi

import "einvoicing/internal/store"

// Perm is an API permission.
type Perm string

const (
	PermSelf           Perm = "self"            // any logged-in user (profile, password)
	PermRead           Perm = "read"            // view masters, invoices
	PermCompanyWrite   Perm = "company.write"   // company settings, tokens, reference sync
	PermCompanyCreate  Perm = "company.create"  // add companies
	PermMastersWrite   Perm = "masters.write"   // customers, products
	PermInvoiceWrite   Perm = "invoice.write"   // create/edit/submit invoices
	PermInvoiceManage  Perm = "invoice.manage"  // cancel, reconcile, debit notes
	PermReports        Perm = "reports"         // reports and exports
	PermScenarios      Perm = "scenarios"       // sandbox scenario runs
	PermUsers          Perm = "users"           // user administration
	PermAPIKeys        Perm = "apikeys"         // ERP API keys
	PermAudit          Perm = "audit"           // audit log
	PermSystem         Perm = "system"          // backups, licence, FBR logo
	PermIncidents      Perm = "incidents"       // incident register
)

var rolePerms = map[string]map[Perm]bool{
	store.RoleAdmin: all(),
	store.RoleManager: set(PermSelf, PermRead, PermCompanyWrite, PermMastersWrite, PermInvoiceWrite, PermInvoiceManage, PermReports,
		PermScenarios, PermAudit, PermIncidents),
	store.RoleAccountant: set(PermSelf, PermRead, PermMastersWrite, PermInvoiceWrite, PermInvoiceManage, PermReports, PermIncidents),
	store.RoleOperator:   set(PermSelf, PermRead, PermInvoiceWrite),
	store.RoleAuditor:    set(PermSelf, PermRead, PermReports, PermAudit),
}

// API keys (ERP/POS integrations) may read masters, maintain customers and
// products, create and submit invoices and read reports.
var apiKeyPerms = set(PermRead, PermMastersWrite, PermInvoiceWrite, PermReports)

func set(ps ...Perm) map[Perm]bool {
	m := map[Perm]bool{}
	for _, p := range ps {
		m[p] = true
	}
	return m
}

func all() map[Perm]bool {
	return set(PermSelf, PermRead, PermCompanyWrite, PermCompanyCreate, PermMastersWrite, PermInvoiceWrite, PermInvoiceManage,
		PermReports, PermScenarios, PermUsers, PermAPIKeys, PermAudit, PermSystem, PermIncidents)
}

func allowed(rc *reqCtx, p Perm) bool {
	if rc.APIKey != nil {
		return apiKeyPerms[p]
	}
	if rc.User == nil {
		return false
	}
	return rolePerms[rc.User.Role][p]
}

// PermissionsFor lists a role's permissions (for the UI).
func PermissionsFor(role string) []Perm {
	var out []Perm
	for p := range rolePerms[role] {
		out = append(out, p)
	}
	return out
}
