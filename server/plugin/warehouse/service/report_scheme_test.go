package service

import (
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
)

func TestSchemeOwnershipAndSharedPublication(t *testing.T) {
	item := model.ReportScheme{Name: "我的方案", Scope: "personal", OwnerID: 1, ColumnsJSON: `["brand","amount"]`}
	manager := SchemeAccess{CanQuery: true, CanManagePersonal: true, CanManageShared: true}
	if err := checkSchemeMutation(item, 2, manager); err == nil {
		t.Fatal("shared manager modified another user's personal scheme")
	}
	if err := checkSchemeMutation(item, 1, manager); err != nil {
		t.Fatalf("owner cannot modify own scheme: %v", err)
	}
	_, _, err := prepareScheme(req.ReportSchemeInput{Name: "共享", Scope: "shared", Columns: []string{"amount"}}, SchemeAccess{CanQuery: true, CanManagePersonal: true})
	if err == nil {
		t.Fatal("query user published shared scheme")
	}
}
