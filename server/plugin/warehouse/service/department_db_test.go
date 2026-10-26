package service

import (
	"context"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
)

func TestDepartmentCRUDMnemonicVersionAndSnapshot(t *testing.T) {
	db, delivery := persistedDeliveryFixture(t)
	ctx := context.Background()
	note := createPersistedDelivery(t, delivery, false, "1", "2")
	s := NewDepartmentService(db)
	page, err := s.List(ctx, req.DepartmentListQuery{})
	if err != nil || page.Total != 1 || page.List[0].Name != "销售部" {
		t.Fatalf("recorded departments=%#v err=%v", page, err)
	}
	item := page.List[0]
	version := item.Version
	updated, err := s.Update(ctx, 9, req.UpdateDepartmentInput{VersionedID: req.VersionedID{ID: item.ID, Version: &version}, DepartmentInput: req.DepartmentInput{Name: "市场部", Mnemonic: "SC, sales；MARKET 东区"}})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != version+1 || updated.UpdatedBy != 9 {
		t.Fatalf("audit=%#v", updated)
	}
	for _, keyword := range []string{"市场", "SC", "sales", "MARKET", "东区"} {
		page, err := s.List(ctx, req.DepartmentListQuery{Keyword: keyword})
		if err != nil || page.Total != 1 {
			t.Fatalf("keyword %q=%#v err=%v", keyword, page, err)
		}
	}
	if _, err := s.Update(ctx, 9, req.UpdateDepartmentInput{VersionedID: req.VersionedID{ID: item.ID, Version: &version}, DepartmentInput: req.DepartmentInput{Name: "过期修改"}}); err == nil {
		t.Fatal("stale update succeeded")
	}
	if err := s.Delete(ctx, 9, req.DeleteInput{VersionedID: req.VersionedID{ID: item.ID, Version: &version}}); err == nil {
		t.Fatal("stale delete succeeded")
	}
	stored, err := delivery.Detail(ctx, note.Note.ID)
	if err != nil || stored.Note.Department != "销售部" {
		t.Fatalf("history rewritten=%#v err=%v", stored, err)
	}
	history, err := delivery.Departments(ctx)
	if err != nil || len(history) != 1 || history[0] != "销售部" {
		t.Fatalf("historical candidates=%#v err=%v", history, err)
	}
	if _, err := s.Create(ctx, 9, req.CreateDepartmentInput{DepartmentInput: req.DepartmentInput{Name: " 市场部 "}}); err == nil {
		t.Fatal("duplicate name accepted")
	}
	version = updated.Version
	if err := s.Delete(ctx, 9, req.DeleteInput{VersionedID: req.VersionedID{ID: item.ID, Version: &version}}); err != nil {
		t.Fatal(err)
	}
	page, err = s.List(ctx, req.DepartmentListQuery{Keyword: "MARKET"})
	if err != nil || page.Total != 0 {
		t.Fatalf("deleted department listed=%#v err=%v", page, err)
	}
	if _, err := s.Detail(ctx, item.ID); err == nil {
		t.Fatal("deleted detail readable")
	}
	var deleted model.Department
	if err := db.Unscoped().First(&deleted, item.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !deleted.IsDeleted || deleted.DeletedBy == nil || *deleted.DeletedBy != 9 {
		t.Fatalf("delete audit=%#v", deleted)
	}
}

func TestHistoricalDepartmentsImportOnceDoesNotResurrectRenamedOrDeleted(t *testing.T) {
	db, delivery := persistedDeliveryFixture(t)
	ctx := context.Background()
	createPersistedDelivery(t, delivery, false, "1", "1")
	deleted := createPersistedDelivery(t, delivery, false, "1", "1")
	if err := db.Model(&model.DeliveryNote{}).Where("id = ?", deleted.Note.ID).Update("department", "仅删除单据部门").Error; err != nil {
		t.Fatal(err)
	}
	if err := delivery.Delete(ctx, 4, req.DeleteInput{VersionedID: req.VersionedID{ID: deleted.Note.ID}}); err != nil {
		t.Fatal(err)
	}
	// Simulate pre-upgrade history: the delivery table predates department masters.
	if err := db.Unscoped().Where("1 = 1").Delete(&model.Department{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := ImportHistoricalDepartments(db); err != nil {
		t.Fatal(err)
	}
	s := NewDepartmentService(db)
	page, err := s.List(ctx, req.DepartmentListQuery{})
	if err != nil || page.Total != 1 || page.List[0].Name != "销售部" {
		t.Fatalf("import=%#v err=%v", page, err)
	}
	item := page.List[0]
	updated, err := s.Update(ctx, 8, req.UpdateDepartmentInput{VersionedID: req.VersionedID{ID: item.ID}, DepartmentInput: req.DepartmentInput{Name: "改名部门"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := ImportHistoricalDepartments(db); err != nil {
		t.Fatal(err)
	}
	page, err = s.List(ctx, req.DepartmentListQuery{})
	if err != nil || page.Total != 1 || page.List[0].Name != "改名部门" {
		t.Fatalf("restart resurrected old name=%#v err=%v", page, err)
	}
	if err := s.Delete(ctx, 8, req.DeleteInput{VersionedID: req.VersionedID{ID: updated.ID}}); err != nil {
		t.Fatal(err)
	}
	if err := ImportHistoricalDepartments(db); err != nil {
		t.Fatal(err)
	}
	page, err = s.List(ctx, req.DepartmentListQuery{})
	if err != nil || page.Total != 0 {
		t.Fatalf("restart resurrected deleted master=%#v err=%v", page, err)
	}
	// A future free-text delivery may intentionally register that historical name again.
	createPersistedDelivery(t, delivery, false, "1", "1")
	page, err = s.List(ctx, req.DepartmentListQuery{})
	if err != nil || page.Total != 1 || page.List[0].Name != "销售部" {
		t.Fatalf("future registration=%#v err=%v", page, err)
	}
}

func TestDepartmentRegistrationRollsBackWithDelivery(t *testing.T) {
	db, delivery := persistedDeliveryFixture(t)
	ctx := context.Background()
	reservation, err := delivery.ReserveNumber(ctx, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TRIGGER reject_department_delivery BEFORE INSERT ON wh_delivery_details BEGIN SELECT RAISE(ABORT, 'line rejected'); END").Error; err != nil {
		t.Fatal(err)
	}
	in := deliveryInput(reservation.Token, "1", "1")
	in.Department = "不得留下的部门"
	if _, err := delivery.Create(ctx, 4, "录单人", in); err == nil {
		t.Fatal("failed delivery succeeded")
	}
	page, err := NewDepartmentService(db).List(ctx, req.DepartmentListQuery{})
	if err != nil || page.Total != 0 {
		t.Fatalf("partial registration=%#v err=%v", page, err)
	}
}

func TestDeliveryUpdateRegistersFreeTextAndKeepsExistingMnemonic(t *testing.T) {
	db, delivery := persistedDeliveryFixture(t)
	ctx := context.Background()
	created := createPersistedDelivery(t, delivery, false, "1", "2")
	s := NewDepartmentService(db)
	department, err := s.Create(ctx, 4, req.CreateDepartmentInput{DepartmentInput: req.DepartmentInput{Name: "采购部", Mnemonic: "CG；purchase"}})
	if err != nil {
		t.Fatal(err)
	}
	amount := "3"
	input := req.UpdateDeliveryInput{
		VersionedID: req.VersionedID{ID: created.Note.ID}, BrandID: 1, DeliveredAt: created.Note.DeliveredAt, Department: "采购部",
		Details: []req.DeliveryDetailInput{{MaterialID: 1, MaterialUnitID: 1, Quantity: "1", Amount: &amount}},
	}
	if _, err := delivery.Update(ctx, 8, "编辑人", input); err != nil {
		t.Fatal(err)
	}
	retained, err := s.Detail(ctx, department.ID)
	if err != nil || retained.Mnemonic != department.Mnemonic || retained.Version != department.Version {
		t.Fatalf("registration overwrote master=%#v err=%v", retained, err)
	}
	input.Department = "新自由录入部门"
	if _, err := delivery.Update(ctx, 8, "编辑人", input); err != nil {
		t.Fatal(err)
	}
	page, err := s.List(ctx, req.DepartmentListQuery{Keyword: "新自由录入"})
	if err != nil || page.Total != 1 || page.List[0].CreatedBy != 8 {
		t.Fatalf("update registration=%#v err=%v", page, err)
	}
}
