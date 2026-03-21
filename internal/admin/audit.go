package admin

type AuditFilter struct {
	Domain   string
	Status   string
	Query    string
	Page     int
	PageSize int
}

type AuditPageData struct {
	Items        []mrtangAdminRecentAction
	Filter       AuditFilter
	Total        int
	Page         int
	Pages        int
	SuccessCount int
	FailedCount  int
}
