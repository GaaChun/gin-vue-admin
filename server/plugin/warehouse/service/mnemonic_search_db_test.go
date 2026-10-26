package service

import (
	"context"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestMasterSearchMatchesEveryMnemonic(t *testing.T) {
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
	if err := db.AutoMigrate(&model.Brand{}, &model.Category{}, &model.Material{}, &model.Unit{}, &model.MaterialUnit{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	s := NewMasterService(db)
	brand, err := s.CreateBrand(ctx, 1, req.CreateBrandInput{BrandInput: req.BrandInput{Name: "客户", Code: "CUSTOMER", Mnemonic: "alpha,beta；gamma delta"}})
	if err != nil {
		t.Fatal(err)
	}
	unit, err := s.CreateUnit(ctx, 1, req.CreateUnitInput{MasterInput: req.MasterInput{Name: "件"}})
	if err != nil {
		t.Fatal(err)
	}
	input := req.MaterialInput{Name: "物料", Code: "MATERIAL", Mnemonic: "alpha,beta；gamma delta", Units: []req.MaterialUnitInput{{UnitID: unit.ID, Factor: "1", IsBase: true, Enabled: true}}}
	material, err := s.CreateMaterial(ctx, 1, req.CreateMaterialInput{MaterialInput: input})
	if err != nil {
		t.Fatal(err)
	}
	for _, keyword := range []string{"alpha", "BETA", "gamma", "delta"} {
		brands, err := s.ListBrands(ctx, req.BrandListQuery{Keyword: keyword})
		if err != nil || brands.Total != 1 || len(brands.List) != 1 || brands.List[0].ID != brand.ID {
			t.Fatalf("brand keyword %q: %#v, %v", keyword, brands, err)
		}
		materials, err := s.ListMaterials(ctx, req.MaterialListQuery{Keyword: keyword})
		if err != nil || materials.Total != 1 || len(materials.List) != 1 || materials.List[0].ID != material.ID {
			t.Fatalf("material keyword %q: %#v, %v", keyword, materials, err)
		}
	}
	brandVersion := brand.Version
	_, err = s.UpdateBrand(ctx, 1, req.UpdateBrandInput{VersionedID: req.VersionedID{ID: brand.ID, Version: &brandVersion}, BrandInput: req.BrandInput{Name: brand.Name, Code: brand.Code, Mnemonic: "epsilon,zeta"}})
	if err != nil {
		t.Fatal(err)
	}
	input.Mnemonic = "epsilon,zeta"
	materialVersion := material.Version
	_, err = s.UpdateMaterial(ctx, 1, req.UpdateMaterialInput{VersionedID: req.VersionedID{ID: material.ID, Version: &materialVersion}, MaterialInput: input})
	if err != nil {
		t.Fatal(err)
	}
	brands, err := s.ListBrands(ctx, req.BrandListQuery{Keyword: "beta"})
	if err != nil || brands.Total != 0 {
		t.Fatalf("obsolete customer mnemonic remains searchable: %#v, %v", brands, err)
	}
	materials, err := s.ListMaterials(ctx, req.MaterialListQuery{Keyword: "beta"})
	if err != nil || materials.Total != 0 {
		t.Fatalf("obsolete material mnemonic remains searchable: %#v, %v", materials, err)
	}
}
