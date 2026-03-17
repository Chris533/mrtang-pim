package admin

import "strings"

func sourceReviewStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "imported":
		return "待审核"
	case "approved":
		return "待桥接"
	case "promoted":
		return "已桥接"
	case "rejected":
		return "已拒绝"
	default:
		return blankFallback(strings.TrimSpace(status), "-")
	}
}

func sourceAssetStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending":
		return "待处理"
	case "processing":
		return "处理中"
	case "processed":
		return "已处理"
	case "failed":
		return "处理失败"
	default:
		return blankFallback(strings.TrimSpace(status), "-")
	}
}

func sourceSyncStatusLabel(status string, linked bool) string {
	if !linked {
		return "未桥接"
	}
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "synced":
		return "已同步"
	case "error":
		return "同步失败"
	case "approved", "ready":
		return "待同步"
	case "":
		return "已桥接"
	default:
		return status
	}
}

func sourceActionTypeLabel(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "update_review_status":
		return "审核状态变更"
	case "promote_product":
		return "桥接商品"
	case "promote_and_sync":
		return "桥接并同步"
	case "retry_sync":
		return "重试同步"
	case "process_asset":
		return "处理图片"
	default:
		return blankFallback(strings.TrimSpace(action), "-")
	}
}

func blankFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
