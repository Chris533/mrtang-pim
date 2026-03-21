package admin

import "mrtang-pim/internal/pim"

type SourceLogFilter struct {
	ActionType string
	Status     string
	TargetType string
	Actor      string
	Query      string
	Page       int
	PageSize   int
}

type SourceLogsPageData struct {
	Items        []pim.SourceActionLog
	Filter       SourceLogFilter
	Total        int
	Page         int
	Pages        int
	PageSize     int
	SuccessCount int
	FailedCount  int
}
