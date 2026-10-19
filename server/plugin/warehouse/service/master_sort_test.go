package service

import (
	"testing"

	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
)

func TestResolveMasterOrderUsesOnlyApprovedFields(t *testing.T) {
	allowed := map[string]string{"name": "wh_units.name"}
	order, err := resolveMasterOrder(req.SortQuery{SortBy: "name", SortOrder: "descending"}, allowed)
	if err != nil || order != "wh_units.name DESC, id ASC" {
		t.Fatalf("approved sort was not resolved safely: %q, %v", order, err)
	}
	if _, err := resolveMasterOrder(req.SortQuery{SortBy: "name; DROP TABLE wh_units", SortOrder: "asc"}, allowed); err == nil {
		t.Fatal("unapproved sort expression was accepted")
	}
	if _, err := resolveMasterOrder(req.SortQuery{SortBy: "name", SortOrder: "sideways"}, allowed); err == nil {
		t.Fatal("invalid sort direction was accepted")
	}
}
func TestDeliverySortUsesApprovedColumns(t *testing.T) {
	order, err := resolveMasterOrder(req.SortQuery{SortBy: "totalAmount", SortOrder: "desc"}, deliverySortColumns())
	if err != nil || order != "total_amount DESC, id ASC" {
		t.Fatalf("delivery amount sort was not resolved safely: %q, %v", order, err)
	}
	if _, err := resolveMasterOrder(req.SortQuery{SortBy: "number; DROP TABLE wh_delivery_notes", SortOrder: "asc"}, deliverySortColumns()); err == nil {
		t.Fatal("unapproved delivery sort expression was accepted")
	}
}
