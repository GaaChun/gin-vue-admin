package service

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func masterCodeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+filepath.Join(t.TempDir(), "codes.db")+"?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.MasterCodeCounter{}, &model.Brand{}, &model.Category{}, &model.Material{}); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func TestMasterCodeSequencesPersistAndAreIndependent(t *testing.T) {
	db := masterCodeDB(t)
	for resource, prefix := range map[string]string{"brands": "KH", "categories": "LB", "materials": "WL"} {
		first, err := NewMasterService(db).ReserveCode(context.Background(), 7, resource)
		if err != nil || first.Code != prefix+"000001" {
			t.Fatalf("%s first allocation = %+v, %v", resource, first, err)
		}
		// A new service must read persisted progress even when no master was saved.
		next, err := NewMasterService(db).ReserveCode(context.Background(), 8, resource)
		if err != nil || next.Code != prefix+"000002" {
			t.Fatalf("%s next allocation = %+v, %v", resource, next, err)
		}
		var counter model.MasterCodeCounter
		if err := db.Where("resource = ?", resource).First(&counter).Error; err != nil {
			t.Fatal(err)
		}
		if counter.CreatedBy != 7 || counter.UpdatedBy != 8 || counter.LastValue != 2 {
			t.Fatalf("allocation audit/progress = %+v", counter)
		}
	}
}

func TestMasterCodeSkipsExistingAndDeletedManualCodes(t *testing.T) {
	db := masterCodeDB(t)
	first := model.Brand{Name: "现有客户", Code: " kh000001 ", Mnemonic: "XY"}
	deleted := model.Brand{Name: "已删除客户", Code: "KH000002", Mnemonic: "SC"}
	if err := db.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&deleted).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&deleted).Error; err != nil {
		t.Fatal(err)
	}
	value, err := NewMasterService(db).ReserveCode(context.Background(), 1, "brands")
	if err != nil || value.Code != "KH000003" {
		t.Fatalf("allocation must skip normalized historical codes: %+v, %v", value, err)
	}
	// Suggestions do not force the eventual saved code.
	created, err := NewMasterService(db).CreateBrand(context.Background(), 1, req.CreateBrandInput{BrandInput: req.BrandInput{Name: "手动编号", Code: "CUSTOM-42", Mnemonic: "SD"}})
	if err != nil || created.Code != "CUSTOM-42" {
		t.Fatalf("manual override must save unchanged: %+v, %v", created, err)
	}
}

func TestMasterCodeConcurrentFirstAllocationHasNoDuplicates(t *testing.T) {
	db := masterCodeDB(t)
	const allocations = 24
	start := make(chan struct{})
	codes := make(chan string, allocations)
	errors := make(chan error, allocations)
	var workers sync.WaitGroup
	for range allocations {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			value, err := NewMasterService(db).ReserveCode(context.Background(), 1, "materials")
			if err != nil {
				errors <- err
				return
			}
			codes <- value.Code
		}()
	}
	close(start)
	workers.Wait()
	close(errors)
	close(codes)
	for err := range errors {
		t.Errorf("concurrent allocation: %v", err)
	}
	seen := map[string]bool{}
	for code := range codes {
		if seen[code] {
			t.Errorf("duplicate allocation %q", code)
		}
		seen[code] = true
	}
	for value := 1; value <= allocations; value++ {
		if !seen[fmt.Sprintf("WL%06d", value)] {
			t.Errorf("missing sequence value %d", value)
		}
	}
}

func TestMasterCodeRejectsUnsupportedResourceAndExhaustion(t *testing.T) {
	if _, err := NewMasterService(nil).ReserveCode(context.Background(), 1, "units"); err == nil {
		t.Fatal("units have no code field")
	}
	db := masterCodeDB(t)
	if err := db.Create(&model.MasterCodeCounter{Resource: "brands", LastValue: 999999}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewMasterService(db).ReserveCode(context.Background(), 1, "brands"); err == nil {
		t.Fatal("exhausted sequence must not wrap")
	} else if domain, ok := AsDomainError(err); !ok || domain.Code != NumberExhausted {
		t.Fatalf("exhaustion error = %v", err)
	}
}
