package service

import (
	"context"
	"reflect"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func persistedMaterialFixture(t *testing.T) (*gorm.DB, *MasterService, *MaterialDetail, []model.DeliveryDetail) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.Unit{}, &model.Brand{}, &model.Category{}, &model.Material{}, &model.MaterialUnit{}, &model.MaterialImage{}, &model.Department{}, &model.DeliveryDetail{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX test_material_one_base ON wh_material_units(material_id) WHERE is_base = 1 AND deleted_at IS NULL").Error; err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"个", "箱", "包"} {
		unit := model.Unit{Name: name}
		if err := db.Create(&unit).Error; err != nil {
			t.Fatal(err)
		}
	}
	s := NewMasterService(db)
	created, err := s.CreateMaterial(context.Background(), 4, req.CreateMaterialInput{MaterialInput: req.MaterialInput{
		Name: "物料", Mnemonic: "wl", Code: "M001",
		Units: []req.MaterialUnitInput{{UnitID: 1, Factor: "1", IsBase: true, Enabled: true}, {UnitID: 2, Factor: "12", Enabled: true}, {UnitID: 3, Factor: "6", Enabled: true}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	for i, binding := range created.Units[:2] {
		quantity := decimal.NewFromInt(int64(i + 2))
		line := model.DeliveryDetail{
			NoteID: 1, LineNo: uint(i + 1), MaterialID: created.ID, MaterialUnitID: binding.ID,
			UnitID: binding.UnitID, BaseUnitID: 1, MaterialCodeSnapshot: "M001", MaterialNameSnapshot: "历史物料",
			UnitNameSnapshot: []string{"个", "箱"}[i], BaseUnitNameSnapshot: "个", Factor: binding.Factor,
			Quantity: quantity, BaseQuantity: quantity.Mul(binding.Factor), UnitPrice: decimal.NewFromInt(5), Amount: quantity.Mul(decimal.NewFromInt(5)),
		}
		if err := db.Create(&line).Error; err != nil {
			t.Fatal(err)
		}
	}
	var history []model.DeliveryDetail
	if err := db.Order("id").Find(&history).Error; err != nil {
		t.Fatal(err)
	}
	created, err = s.MaterialDetail(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	return db, s, created, history
}

func replacementMaterialInput(item *MaterialDetail) req.UpdateMaterialInput {
	version := item.Version
	return req.UpdateMaterialInput{
		VersionedID: req.VersionedID{ID: item.ID, Version: &version},
		MaterialInput: req.MaterialInput{Name: "更新物料", Mnemonic: "gxwl", Code: item.Code, Units: []req.MaterialUnitInput{
			{UnitID: 2, Factor: "1", IsBase: true, Enabled: true},
			{UnitID: 1, Factor: "0.083333", Enabled: false},
		}},
	}
}

func assertMaterialHistoryUnchanged(t *testing.T, db *gorm.DB, before []model.DeliveryDetail) {
	t.Helper()
	var after []model.DeliveryDetail
	if err := db.Order("id").Find(&after).Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("material update polluted delivery history: before=%#v after=%#v", before, after)
	}
}

func assertReplacementMaterial(t *testing.T, s *MasterService, before *MaterialDetail) *MaterialDetail {
	t.Helper()
	after, err := s.MaterialDetail(context.Background(), before.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Version != before.Version+1 || len(after.Units) != 2 {
		t.Fatalf("replacement aggregate = %#v", after)
	}
	for _, binding := range after.Units {
		switch binding.UnitID {
		case 1:
			if binding.ID != before.Units[0].ID || binding.IsBase || binding.Enabled || binding.Factor.String() != "0.083333" {
				t.Fatalf("former base not retained: %#v", binding)
			}
		case 2:
			if binding.ID != before.Units[1].ID || !binding.IsBase || !binding.Enabled || !binding.Factor.Equal(decimal.NewFromInt(1)) {
				t.Fatalf("replacement base = %#v", binding)
			}
		default:
			t.Fatalf("omitted unreferenced binding remains: %#v", binding)
		}
	}
	return after
}

func TestPersistedMaterialCanSwitchReferencedBaseWithoutChangingHistory(t *testing.T) {
	db, s, before, history := persistedMaterialFixture(t)
	if _, err := s.UpdateMaterial(context.Background(), 8, replacementMaterialInput(before)); err != nil {
		t.Fatal(err)
	}
	after := assertReplacementMaterial(t, s, before)
	assertMaterialHistoryUnchanged(t, db, history)
	input := replacementMaterialInput(after)
	input.Units = input.Units[:1]
	if _, err := s.UpdateMaterial(context.Background(), 8, input); err == nil {
		t.Fatal("referenced former-base binding was deleted")
	} else if domain, ok := AsDomainError(err); !ok || domain.Code != ReferenceInUse {
		t.Fatalf("reference error = %v", err)
	}
	persisted, err := s.MaterialDetail(context.Background(), after.ID)
	if err != nil || !reflect.DeepEqual(persisted, after) {
		t.Fatalf("rejected omission changed aggregate: %#v err=%v", persisted, err)
	}
	assertMaterialHistoryUnchanged(t, db, history)
}

func TestPersistedMaterialRejectsInvalidReplacementBases(t *testing.T) {
	for _, tc := range []struct {
		name  string
		units []req.MaterialUnitInput
	}{
		{"missing", []req.MaterialUnitInput{{UnitID: 1, Factor: "1", Enabled: true}, {UnitID: 2, Factor: "12", Enabled: true}}},
		{"multiple", []req.MaterialUnitInput{{UnitID: 1, Factor: "1", IsBase: true, Enabled: true}, {UnitID: 2, Factor: "1", IsBase: true, Enabled: true}}},
		{"disabled", []req.MaterialUnitInput{{UnitID: 1, Factor: "1", IsBase: true, Enabled: false}}},
		{"factor", []req.MaterialUnitInput{{UnitID: 1, Factor: "2", IsBase: true, Enabled: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, s, before, history := persistedMaterialFixture(t)
			input := replacementMaterialInput(before)
			input.Units = tc.units
			if _, err := s.UpdateMaterial(context.Background(), 8, input); err == nil {
				t.Fatal("invalid replacement accepted")
			}
			after, err := s.MaterialDetail(context.Background(), before.ID)
			if err != nil || !reflect.DeepEqual(after, before) {
				t.Fatalf("invalid replacement changed aggregate: %#v err=%v", after, err)
			}
			assertMaterialHistoryUnchanged(t, db, history)
		})
	}
}

func TestPersistedMaterialExcelCanSwitchReferencedBase(t *testing.T) {
	db, s, before, history := persistedMaterialFixture(t)
	contents := workbookWithSheets(t, map[string][][]string{
		"物料":   {{"编号*", "物料名称*", "助记符*", "规格", "别名", "客户编号", "类别编号", "图片文件ID", "版本号"}, {"M001", "更新物料", "gxwl", "", "", "", "", "", "1"}},
		"物料单位": {{"物料编号*", "单位名称*", "换算系数*", "是否基本单位*", "是否启用*"}, {"M001", "箱", "1", "是", "是"}, {"M001", "个", "0.083333", "否", "否"}},
	})
	result, err := s.ImportMasterExcel(context.Background(), 8, "materials", contents)
	if err != nil || result.UpdatedCount != 1 {
		t.Fatalf("import=%#v err=%v", result, err)
	}
	after := assertReplacementMaterial(t, s, before)
	assertMaterialHistoryUnchanged(t, db, history)
	rejected := workbookWithSheets(t, map[string][][]string{
		"物料":   {{"编号*", "物料名称*", "助记符*", "规格", "别名", "客户编号", "类别编号", "图片文件ID", "版本号"}, {"M001", "不能提交", "wl", "", "", "", "", "", "2"}},
		"物料单位": {{"物料编号*", "单位名称*", "换算系数*", "是否基本单位*", "是否启用*"}, {"M001", "箱", "1", "是", "是"}},
	})
	if _, err := s.ImportMasterExcel(context.Background(), 8, "materials", rejected); err == nil {
		t.Fatal("Excel removed referenced former base")
	}
	persisted, err := s.MaterialDetail(context.Background(), after.ID)
	if err != nil || !reflect.DeepEqual(persisted, after) {
		t.Fatalf("rejected Excel changed aggregate: %#v err=%v", persisted, err)
	}
	assertMaterialHistoryUnchanged(t, db, history)
}
