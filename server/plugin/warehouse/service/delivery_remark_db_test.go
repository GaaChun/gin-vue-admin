package service

import (
	"context"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
)

func TestPersistedDeliveryDetailRemarksRoundTrip(t *testing.T) {
	_, s := persistedDeliveryFixture(t)
	ctx := context.Background()
	reservation, err := s.ReserveNumber(ctx, 4)
	if err != nil {
		t.Fatal(err)
	}
	input := deliveryInput(reservation.Token, "2", "3")
	input.Details[0].Remark = "  首行备注  "
	input.Details = append(input.Details, req.DeliveryDetailInput{MaterialID: 1, MaterialUnitID: 1, Quantity: "1", UnitPrice: "4"})
	created, err := s.Create(ctx, 4, "编辑人", input)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := s.Detail(ctx, created.Note.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Details) != 2 || stored.Details[0].Remark != "  首行备注  " || stored.Details[1].Remark != "" {
		t.Fatalf("created remarks=%#v", stored.Details)
	}
	firstID, secondID := stored.Details[0].ID, stored.Details[1].ID
	_, err = s.Update(ctx, 4, "编辑人", req.UpdateDeliveryInput{
		VersionedID: req.VersionedID{ID: created.Note.ID}, BrandID: 1, DeliveredAt: created.Note.DeliveredAt,
		Details: []req.DeliveryDetailInput{
			{ID: &secondID, MaterialID: 1, MaterialUnitID: 1, Quantity: "1", UnitPrice: "4", Remark: "更新备注"},
			{ID: &firstID, MaterialID: 1, MaterialUnitID: 1, Quantity: "2", UnitPrice: "3", Remark: ""},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	stored, err = s.Detail(ctx, created.Note.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Details) != 2 || stored.Details[0].ID != secondID || stored.Details[0].Remark != "更新备注" || stored.Details[1].ID != firstID || stored.Details[1].Remark != "" || stored.Note.TotalAmount.String() != "10" {
		t.Fatalf("updated remarks/order=%#v", stored)
	}
}

func TestPersistedDeliveryListFiltersDetailRemarkWithoutDuplicates(t *testing.T) {
	db, s := persistedDeliveryFixture(t)
	ctx := context.Background()
	create := func(remarks ...string) *DeliveryDetail {
		t.Helper()
		reservation, err := s.ReserveNumber(ctx, 4)
		if err != nil {
			t.Fatal(err)
		}
		input := deliveryInput(reservation.Token, "1", "2")
		line := input.Details[0]
		input.Details = nil
		for _, remark := range remarks {
			next := line
			next.Remark = remark
			input.Details = append(input.Details, next)
		}
		note, err := s.Create(ctx, 4, "编辑人", input)
		if err != nil {
			t.Fatal(err)
		}
		return note
	}
	first := create("前缀needle后缀", "needle第二行")
	second := create("另一个needle备注")
	deletedOnly := create("needle已删除", "不匹配")
	create("普通备注", "")
	if err := db.Delete(&model.DeliveryDetail{}, deletedOnly.Details[0].ID).Error; err != nil {
		t.Fatal(err)
	}
	for pageNo, expected := range []uint{second.Note.ID, first.Note.ID} {
		page, err := s.List(ctx, req.DeliveryListQuery{DetailRemark: "needle", PageQuery: req.PageQuery{Page: pageNo + 1, PageSize: 1}})
		if err != nil {
			t.Fatal(err)
		}
		if page.Total != 2 || len(page.List) != 1 || page.List[0].ID != expected {
			t.Fatalf("page %d=%#v", pageNo+1, page)
		}
	}
	page, err := s.List(ctx, req.DeliveryListQuery{DetailRemark: "needle", Number: first.Note.Number, MaterialIDs: "1", Status: "draft"})
	if err != nil || page.Total != 1 || len(page.List) != 1 || page.List[0].ID != first.Note.ID {
		t.Fatalf("combined filters=%#v err=%v", page, err)
	}
	page, err = s.List(ctx, req.DeliveryListQuery{DetailRemark: "needle' OR 1=1 --"})
	if err != nil || page.Total != 0 || len(page.List) != 0 {
		t.Fatalf("parameterized filter=%#v err=%v", page, err)
	}
}

func TestPersistedDeliveryListFiltersAuditNamesAndTimes(t *testing.T) {
	db, s := persistedDeliveryFixture(t)
	ctx := context.Background()
	first := createPersistedDelivery(t, s, true, "1", "2")
	second := createPersistedDelivery(t, s, false, "1", "2")
	third := createPersistedDelivery(t, s, true, "1", "2")
	if err := db.Exec("INSERT INTO sys_users (id, nick_name) VALUES (40, '当前创建人'), (41, '当前审核人'), (4, '修改人') ON CONFLICT(id) DO UPDATE SET nick_name = excluded.nick_name").Error; err != nil {
		t.Fatal(err)
	}
	parseTime := func(value string) *time.Time {
		t.Helper()
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			t.Fatal(err)
		}
		return &parsed
	}
	start := "2026-10-01T00:00:00+08:00"
	middle := "2026-10-02T00:00:00+08:00"
	end := "2026-10-03T00:00:00+08:00"
	if err := db.Model(&model.DeliveryNote{}).Where("id = ?", first.Note.ID).Updates(map[string]interface{}{
		"created_by": 40, "created_by_name_snapshot": "旧创建人", "updated_by": 4, "reviewed_by": 41,
		"created_at": *parseTime(start), "updated_at": *parseTime(middle), "reviewed_at": *parseTime(end),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.DeliveryNote{}).Where("id = ?", second.Note.ID).Updates(map[string]interface{}{
		"created_by": 42, "created_by_name_snapshot": "历史创建人", "updated_by": 42, "reviewed_by": 42,
		"created_at": *parseTime(middle), "updated_at": *parseTime(end), "reviewed_at": nil,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.DeliveryNote{}).Where("id = ?", third.Note.ID).Updates(map[string]interface{}{
		"created_by": 4, "created_by_name_snapshot": "其他创建人", "updated_by": 42, "reviewed_by": 42, "reviewed_by_name_snapshot": "历史审核人",
		"created_at": *parseTime(end), "updated_at": *parseTime(start), "reviewed_at": *parseTime(middle),
	}).Error; err != nil {
		t.Fatal(err)
	}
	check := func(name string, query req.DeliveryListQuery, expected uint) {
		t.Helper()
		page, err := s.List(ctx, query)
		if err != nil || page.Total != 1 || len(page.List) != 1 || page.List[0].ID != expected {
			t.Fatalf("%s: page=%#v err=%v", name, page, err)
		}
	}
	check("live creator nickname", req.DeliveryListQuery{CreatedBy: "当前创建"}, first.Note.ID)
	check("creator snapshot fallback", req.DeliveryListQuery{CreatedBy: "历史创建"}, second.Note.ID)
	check("modifier nickname", req.DeliveryListQuery{UpdatedBy: "修改"}, first.Note.ID)
	check("reviewer nickname", req.DeliveryListQuery{ReviewedBy: "当前审核"}, first.Note.ID)
	check("reviewer snapshot fallback", req.DeliveryListQuery{ReviewedBy: "历史审核"}, third.Note.ID)
	check("created time range includes start and excludes end", req.DeliveryListQuery{CreatedFrom: parseTime(start), CreatedTo: parseTime(middle)}, first.Note.ID)
	check("updated time range", req.DeliveryListQuery{UpdatedFrom: parseTime(start), UpdatedTo: parseTime(middle)}, third.Note.ID)
	check("review time range", req.DeliveryListQuery{ReviewedFrom: parseTime(middle), ReviewedTo: parseTime(end)}, third.Note.ID)
}
