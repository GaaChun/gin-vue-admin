package service

import (
	"strings"

	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
)

func resolveMasterOrder(query req.SortQuery, allowed map[string]string) (string, error) {
	if query.SortBy == "" && query.SortOrder == "" {
		return "id DESC", nil
	}
	column, ok := allowed[query.SortBy]
	if !ok {
		return "", domainError(ValidationFailed, "不支持该排序字段", "sortBy")
	}
	order := strings.ToLower(query.SortOrder)
	if order != "ascending" && order != "descending" && order != "asc" && order != "desc" {
		return "", domainError(ValidationFailed, "排序方向无效", "sortOrder")
	}
	direction := "ASC"
	if order == "descending" || order == "desc" {
		direction = "DESC"
	}
	return column + " " + direction + ", id ASC", nil
}

func auditSortColumns(table string) map[string]string {
	userName := func(field string) string {
		return "COALESCE((SELECT nick_name FROM sys_users AS sort_user WHERE sort_user.id = " + table + "." + field + "), '')"
	}
	return map[string]string{
		"createdByName":  userName("created_by"),
		"createdAt":      table + ".created_at",
		"updatedByName":  userName("updated_by"),
		"updatedAt":      table + ".updated_at",
		"reviewedByName": userName("reviewed_by"),
		"reviewedAt":     table + ".reviewed_at",
		"version":        table + ".version",
	}
}
func deliverySortColumns() map[string]string {
	return map[string]string{
		"number":                 "number",
		"deliveredAt":            "delivered_at",
		"brandNameSnapshot":      "brand_name_snapshot",
		"department":             "department",
		"status":                 "status",
		"totalAmount":            "total_amount",
		"createdByNameSnapshot":  "created_by_name_snapshot",
		"createdAt":              "created_at",
		"reviewedByNameSnapshot": "reviewed_by_name_snapshot",
		"reviewedAt":             "reviewed_at",
	}
}
