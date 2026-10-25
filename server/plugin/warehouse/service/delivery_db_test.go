package service

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/glebarez/sqlite"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SQLite recognizes datetime but not MySQL's datetime(6) as a time-valued
// column. Keep the production model intact and specialize only fixture DDL.
type sqliteDeliveryNote struct {
	model.DeliveryNote
	DeliveredAt time.Time `gorm:"column:delivered_at;type:datetime;not null"`
}

func (sqliteDeliveryNote) TableName() string { return "wh_delivery_notes" }

func persistedDeliveryFixture(t *testing.T) (*gorm.DB, *DeliveryService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil { t.Fatal(err) }
	sqlDB, err := db.DB()
	if err != nil { t.Fatal(err) }
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.Brand{}, &model.Category{}, &model.Material{}, &model.Unit{}, &model.MaterialUnit{}, &sqliteDeliveryNote{}, &model.DeliveryDetail{}, &model.NumberCounter{}, &model.NumberReservation{}); err != nil { t.Fatal(err) }
	if err := db.Exec("CREATE TABLE sys_users (id integer PRIMARY KEY, nick_name text)").Error; err != nil { t.Fatal(err) }
	if err := db.Exec("INSERT INTO sys_users (id, nick_name) VALUES (4, '修改人')").Error; err != nil { t.Fatal(err) }
	fixture := deliveryFixture(time.Date(2026, 10, 25, 2, 0, 0, 0, time.UTC))
	brand, category, material, unit, binding := fixture.brands[1], fixture.categories[1], fixture.materials[1], fixture.units[1], fixture.bindings[1]
	for _, record := range []interface{}{&brand, &category, &material, &unit, &binding} {
		if err := db.Create(record).Error; err != nil { t.Fatal(err) }
	}
	s := NewDeliveryService(db)
	s.now = func() time.Time { return fixture.now }
	return db, s
}

func createPersistedDelivery(t *testing.T, s *DeliveryService, review bool, quantity, amount string) *DeliveryDetail {
	t.Helper()
	reservation, err := s.ReserveNumber(context.Background(), 4)
	if err != nil { t.Fatal(err) }
	in := deliveryInput(reservation.Token, quantity, "999")
	in.Department, in.Review, in.Details[0].Amount = "销售部", review, &amount
	note, err := s.Create(context.Background(), 4, "审核人", in)
	if err != nil { t.Fatal(err) }
	return note
}

func TestPersistedSaveReviewAndReReviewLifecycle(t *testing.T) {
	_, s := persistedDeliveryFixture(t)
	ctx := context.Background()
	created := createPersistedDelivery(t, s, true, "3", "10")
	if created.Note.Status != "reviewed" || created.Note.Version != 1 || created.Note.ReviewedBy == nil || created.Note.ReviewedAt == nil || *created.Note.ReviewedByNameSnapshot != "审核人" {
		t.Fatalf("save review audit = %#v", created.Note)
	}
	persisted, err := s.Detail(ctx, created.Note.ID)
	if err != nil { t.Fatal(err) }
	if persisted.Details[0].UnitPrice.StringFixed(4) != "3.3333" || persisted.Details[0].Amount.StringFixed(2) != "10.00" || persisted.Note.TotalAmount.StringFixed(2) != "10.00" { t.Fatalf("amounts = %#v", persisted) }
	version := created.Note.Version
	id := req.VersionedID{ID: created.Note.ID, Version: &version}
	opinion := "已核对"
	reReviewed, err := s.ReReview(ctx, 8, req.ReReviewDeliveryInput{VersionedID: id, Opinion: &opinion})
	if err != nil { t.Fatal(err) }
	if reReviewed.Status != "reReviewed" || reReviewed.Version != 2 || *reReviewed.ReReviewOpinion != opinion || *reReviewed.ReviewedBy != 4 || !reReviewed.ReviewedAt.Equal(*created.Note.ReviewedAt) { t.Fatalf("reReview = %#v", reReviewed) }
	if _, err := s.CancelReReview(ctx, 8, req.CancelReReviewDeliveryInput{VersionedID: id}); err == nil { t.Fatal("stale cancellation succeeded") }
	version = reReviewed.Version
	if _, err := s.Unreview(ctx, 8, req.UnreviewDeliveryInput{VersionedID: id}); err == nil { t.Fatal("reReviewed note was unreviewed") }
	if _, err := s.Update(ctx, 8, "编辑人", req.UpdateDeliveryInput{VersionedID: id, BrandID: 1, DeliveredAt: created.Note.DeliveredAt, Details: deliveryInput("", "1", "1").Details}); err == nil { t.Fatal("reReviewed note was edited") }
	cancelled, err := s.CancelReReview(ctx, 8, req.CancelReReviewDeliveryInput{VersionedID: id})
	if err != nil { t.Fatal(err) }
	if cancelled.Status != "reviewed" || cancelled.Version != 3 || cancelled.ReReviewOpinion != nil || *cancelled.ReviewedBy != 4 { t.Fatalf("cancel = %#v", cancelled) }
	stored, err := s.Detail(ctx, cancelled.ID)
	if err != nil || stored.Note.ReReviewOpinion != nil || stored.Note.Status != "reviewed" { t.Fatalf("persisted cancellation=%#v err=%v", stored, err) }
	version = cancelled.Version
	draft, err := s.Unreview(ctx, 8, req.UnreviewDeliveryInput{VersionedID: id})
	if err != nil || draft.Status != "draft" || draft.ReviewedAt != nil || draft.ReviewedByNameSnapshot != nil { t.Fatalf("unreview=%#v err=%v", draft, err) }
	if _, err := s.ReReview(ctx, 8, req.ReReviewDeliveryInput{VersionedID: req.VersionedID{ID: draft.ID}}); err == nil { t.Fatal("draft was reReviewed") }
	version = draft.Version
	amount := "9.99"
	line := persisted.Details[0]
	updated, err := s.Update(ctx, 4, "更新审核人", req.UpdateDeliveryInput{VersionedID: id, BrandID: 1, DeliveredAt: draft.DeliveredAt, Department: "新部门", Review: true, Details: []req.DeliveryDetailInput{{ID: &line.ID, MaterialID: 1, MaterialUnitID: 1, Quantity: "10000", Amount: &amount}}})
	if err != nil { t.Fatal(err) }
	if updated.Note.Status != "reviewed" || updated.Note.Version != version+1 || *updated.Note.ReviewedByNameSnapshot != "更新审核人" || updated.Details[0].UnitPrice.StringFixed(4) != "0.0010" || updated.Note.TotalAmount.StringFixed(2) != "9.99" { t.Fatalf("update review=%#v", updated) }
	nullOpinion, err := s.ReReview(ctx, 4, req.ReReviewDeliveryInput{VersionedID: req.VersionedID{ID: draft.ID}})
	if err != nil || nullOpinion.ReReviewOpinion != nil { t.Fatalf("null opinion = %#v err=%v", nullOpinion, err) }
	listed, err := s.List(ctx, req.DeliveryListQuery{})
	if err != nil || len(listed.List) != 1 || listed.List[0].UpdatedByName != "修改人" { t.Fatalf("list=%#v err=%v", listed, err) }
}

func TestPersistedSaveReviewRollsBackAllWrites(t *testing.T) {
	db, s := persistedDeliveryFixture(t)
	ctx := context.Background()
	reservation, err := s.ReserveNumber(ctx, 4)
	if err != nil { t.Fatal(err) }
	if err := db.Exec("CREATE TRIGGER reject_delivery_line BEFORE INSERT ON wh_delivery_details BEGIN SELECT RAISE(ABORT, 'line rejected'); END").Error; err != nil { t.Fatal(err) }
	in := deliveryInput(reservation.Token, "3", "2")
	in.Review = true
	if _, err := s.Create(ctx, 4, "审核人", in); err == nil { t.Fatal("failed detail write succeeded") }
	var count int64
	if err := db.Model(&model.DeliveryNote{}).Count(&count).Error; err != nil || count != 0 { t.Fatalf("partial notes=%d err=%v", count, err) }
	var reserved model.NumberReservation
	if err := db.First(&reserved, reservation.ID).Error; err != nil || reserved.ConsumedNoteID != nil { t.Fatalf("consumed on rollback=%#v err=%v", reserved, err) }
	if err := db.Exec("DROP TRIGGER reject_delivery_line").Error; err != nil { t.Fatal(err) }
	created, err := s.Create(ctx, 4, "审核人", in)
	if err != nil { t.Fatal(err) }
	if _, err := s.Unreview(ctx, 4, req.UnreviewDeliveryInput{VersionedID: req.VersionedID{ID: created.Note.ID}}); err != nil { t.Fatal(err) }
	before, err := s.Detail(ctx, created.Note.ID)
	if err != nil { t.Fatal(err) }
	if err := db.Exec("CREATE TRIGGER reject_delivery_edit BEFORE UPDATE ON wh_delivery_details BEGIN SELECT RAISE(ABORT, 'line rejected'); END").Error; err != nil { t.Fatal(err) }
	line := before.Details[0]
	version := before.Note.Version
	_, err = s.Update(ctx, 4, "审核人", req.UpdateDeliveryInput{VersionedID: req.VersionedID{ID: before.Note.ID, Version: &version}, BrandID: 1, DeliveredAt: before.Note.DeliveredAt, Review: true, Details: []req.DeliveryDetailInput{{ID: &line.ID, MaterialID: 1, MaterialUnitID: 1, Quantity: "8", UnitPrice: "9"}}})
	if err == nil { t.Fatal("failed update succeeded") }
	after, err := s.Detail(ctx, before.Note.ID)
	if err != nil || after.Note.Status != "draft" || after.Note.Version != before.Note.Version || after.Note.ReviewedAt != nil || !after.Details[0].Amount.Equal(before.Details[0].Amount) { t.Fatalf("partial update=%#v err=%v", after, err) }
}

func TestPersistedReconciliationStatusQueryExportAndPrintProjection(t *testing.T) {
	db, s := persistedDeliveryFixture(t)
	createPersistedDelivery(t, s, false, "3", "10")
	createPersistedDelivery(t, s, true, "3", "20")
	re := createPersistedDelivery(t, s, true, "3", "30")
	if _, err := s.ReReview(context.Background(), 4, req.ReReviewDeliveryInput{VersionedID: req.VersionedID{ID: re.Note.ID}}); err != nil { t.Fatal(err) }
	report := NewReconciliationService(db)
	for _, tc := range []struct { statuses []string; count int; amount string }{
		{nil, 2, "50.00"}, {[]string{}, 2, "50.00"}, {[]string{"draft"}, 1, "10.00"}, {[]string{"reReviewed"}, 1, "30.00"}, {[]string{"draft", "reviewed", "reReviewed"}, 3, "60.00"},
	} {
		t.Run(fmt.Sprint(tc.statuses), func(t *testing.T) {
			query := req.ReconciliationQuery{Filters: req.ReconciliationFilters{Statuses: tc.statuses}, Columns: []string{"noteNumber", "amount"}, PageQuery: req.PageQuery{Page: 1, PageSize: 1}}
			page, err := report.Query(context.Background(), query, true)
			if err != nil || page.Total != int64(tc.count) || page.Summary.Amount == nil || *page.Summary.Amount != tc.amount { t.Fatalf("query=%#v err=%v", page, err) }
			print, err := report.Query(context.Background(), query, false)
			if err != nil || len(print.List) != tc.count || *print.Summary.Amount != tc.amount { t.Fatalf("print=%#v err=%v", print, err) }
			data, err := report.Export(context.Background(), query)
			if err != nil { t.Fatal(err) }
			book, err := excelize.OpenReader(bytes.NewReader(data))
			if err != nil { t.Fatal(err) }
			defer book.Close()
			for i, row := range print.List {
				number, err := book.GetCellValue("对账表", fmt.Sprintf("A%d", 5+i))
				if err != nil || number != *row.Values["noteNumber"] { t.Fatalf("export number=%s row=%#v err=%v", number, row, err) }
				amount, err := book.GetCellValue("对账表", fmt.Sprintf("B%d", 5+i))
				if err != nil || amount != *row.Values["amount"] { t.Fatalf("export amount=%s row=%#v err=%v", amount, row, err) }
			}
		})
	}
	query := req.ReconciliationQuery{Filters: req.ReconciliationFilters{Statuses: []string{"void"}}, Columns: []string{"amount"}}
	if _, err := report.Query(context.Background(), query, true); err == nil { t.Fatal("invalid status accepted") }
	if _, err := report.Export(context.Background(), query); err == nil { t.Fatal("invalid export status accepted") }
	options, err := report.Options(context.Background())
	if err != nil || len(options.Departments) != 1 || options.Departments[0] != "销售部" { t.Fatalf("options=%#v err=%v", options, err) }
}

func TestDeliveryOpinionMigrationPreservesExistingNotes(t *testing.T) {
	db, s := persistedDeliveryFixture(t)
	note := createPersistedDelivery(t, s, true, "3", "10")
	if err := db.Migrator().DropColumn(&model.DeliveryNote{}, "re_review_opinion"); err != nil { t.Fatal(err) }
	if err := db.AutoMigrate(&sqliteDeliveryNote{}); err != nil { t.Fatal(err) }
	var migrated model.DeliveryNote
	if err := db.First(&migrated, note.Note.ID).Error; err != nil { t.Fatal(err) }
	if migrated.ReReviewOpinion != nil || migrated.Status != "reviewed" || migrated.Version != note.Note.Version || !migrated.TotalAmount.Equal(note.Note.TotalAmount) { t.Fatalf("migration changed note=%#v", migrated) }
}

func TestPersistedDeliveryListSortsLastEditorNameAndTime(t *testing.T) {
	db, s := persistedDeliveryFixture(t)
	first := createPersistedDelivery(t, s, false, "1", "1")
	second := createPersistedDelivery(t, s, false, "1", "2")
	if err := db.Exec("INSERT INTO sys_users (id, nick_name) VALUES (8, 'A editor'), (9, 'Z editor')").Error; err != nil { t.Fatal(err) }
	early := time.Date(2026, 10, 24, 1, 0, 0, 0, time.UTC)
	if err := db.Model(&model.DeliveryNote{}).Where("id = ?", first.Note.ID).Updates(map[string]interface{}{"updated_by": 9, "updated_at": early}).Error; err != nil { t.Fatal(err) }
	if err := db.Model(&model.DeliveryNote{}).Where("id = ?", second.Note.ID).Updates(map[string]interface{}{"updated_by": 8, "updated_at": early.Add(time.Hour)}).Error; err != nil { t.Fatal(err) }
	for _, tc := range []struct { field, direction string; firstID uint }{
		{"updatedByName", "ascending", second.Note.ID},
		{"updatedByName", "descending", first.Note.ID},
		{"updatedAt", "ascending", first.Note.ID},
		{"updatedAt", "descending", second.Note.ID},
	} {
		page, err := s.List(context.Background(), req.DeliveryListQuery{SortQuery: req.SortQuery{SortBy: tc.field, SortOrder: tc.direction}})
		if err != nil || len(page.List) != 2 || page.List[0].ID != tc.firstID {
			t.Fatalf("%s %s list=%#v err=%v", tc.field, tc.direction, page, err)
		}
	}
}
