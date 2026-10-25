package service

import (
	"context"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
)

func TestPersistedDeliveryDetailRemarksRoundTrip(t *testing.T) {
	_, s := persistedDeliveryFixture(t)
	ctx := context.Background()
	reservation, err := s.ReserveNumber(ctx, 4)
	if err != nil { t.Fatal(err) }
	input := deliveryInput(reservation.Token, "2", "3")
	input.Details[0].Remark = "  首行备注  "
	input.Details = append(input.Details, req.DeliveryDetailInput{MaterialID: 1, MaterialUnitID: 1, Quantity: "1", UnitPrice: "4"})
	created, err := s.Create(ctx, 4, "编辑人", input)
	if err != nil { t.Fatal(err) }
	stored, err := s.Detail(ctx, created.Note.ID)
	if err != nil { t.Fatal(err) }
	if len(stored.Details) != 2 || stored.Details[0].Remark != "  首行备注  " || stored.Details[1].Remark != "" { t.Fatalf("created remarks=%#v", stored.Details) }
	firstID, secondID := stored.Details[0].ID, stored.Details[1].ID
	_, err = s.Update(ctx, 4, "编辑人", req.UpdateDeliveryInput{
		VersionedID: req.VersionedID{ID: created.Note.ID}, BrandID: 1, DeliveredAt: created.Note.DeliveredAt,
		Details: []req.DeliveryDetailInput{
			{ID: &secondID, MaterialID: 1, MaterialUnitID: 1, Quantity: "1", UnitPrice: "4", Remark: "更新备注"},
			{ID: &firstID, MaterialID: 1, MaterialUnitID: 1, Quantity: "2", UnitPrice: "3", Remark: ""},
		},
	})
	if err != nil { t.Fatal(err) }
	stored, err = s.Detail(ctx, created.Note.ID)
	if err != nil { t.Fatal(err) }
	if len(stored.Details) != 2 || stored.Details[0].ID != secondID || stored.Details[0].Remark != "更新备注" || stored.Details[1].ID != firstID || stored.Details[1].Remark != "" || stored.Note.TotalAmount.String() != "10" { t.Fatalf("updated remarks/order=%#v", stored) }
}

func TestPersistedDeliveryListFiltersDetailRemarkWithoutDuplicates(t *testing.T) {
	db, s := persistedDeliveryFixture(t)
	ctx := context.Background()
	create := func(remarks ...string) *DeliveryDetail {
		t.Helper()
		reservation, err := s.ReserveNumber(ctx, 4)
		if err != nil { t.Fatal(err) }
		input := deliveryInput(reservation.Token, "1", "2")
		line := input.Details[0]
		input.Details = nil
		for _, remark := range remarks { next := line; next.Remark = remark; input.Details = append(input.Details, next) }
		note, err := s.Create(ctx, 4, "编辑人", input)
		if err != nil { t.Fatal(err) }
		return note
	}
	first := create("前缀needle后缀", "needle第二行")
	second := create("另一个needle备注")
	deletedOnly := create("needle已删除", "不匹配")
	create("普通备注", "")
	if err := db.Delete(&model.DeliveryDetail{}, deletedOnly.Details[0].ID).Error; err != nil { t.Fatal(err) }
	for pageNo, expected := range []uint{second.Note.ID, first.Note.ID} {
		page, err := s.List(ctx, req.DeliveryListQuery{DetailRemark: "needle", PageQuery: req.PageQuery{Page: pageNo + 1, PageSize: 1}})
		if err != nil { t.Fatal(err) }
		if page.Total != 2 || len(page.List) != 1 || page.List[0].ID != expected { t.Fatalf("page %d=%#v", pageNo + 1, page) }
	}
	page, err := s.List(ctx, req.DeliveryListQuery{DetailRemark: "needle", Number: first.Note.Number, MaterialIDs: "1", Status: "draft"})
	if err != nil || page.Total != 1 || len(page.List) != 1 || page.List[0].ID != first.Note.ID { t.Fatalf("combined filters=%#v err=%v", page, err) }
	page, err = s.List(ctx, req.DeliveryListQuery{DetailRemark: "needle' OR 1=1 --"})
	if err != nil || page.Total != 0 || len(page.List) != 0 { t.Fatalf("parameterized filter=%#v err=%v", page, err) }
}
