package system

import (
	"errors"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type visitTestAuthorityMenu struct {
	AuthorityID uint `gorm:"column:sys_authority_authority_id"`
	MenuID      uint `gorm:"column:sys_base_menu_id"`
}

func (visitTestAuthorityMenu) TableName() string { return "sys_authority_menus" }

func TestUserMenuVisitsAreScopedAndSorted(t *testing.T) {
	previousDB := global.GVA_DB
	db, err := gorm.Open(sqlite.Open("file:"+t.TempDir()+"/visits.db"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&system.SysUserMenuVisit{}, &visitTestAuthorityMenu{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE sys_base_menus (id integer primary key, name text, title text, path text, icon text, hidden numeric, parent_id integer, deleted_at datetime)").Error; err != nil {
		t.Fatal(err)
	}
	for _, menu := range []struct {
		id    uint
		name  string
		title string
	}{
		{2, "lower", "Lower"},
		{3, "middle", "Middle"},
		{4, "higher", "Higher"},
	} {
		if err := db.Exec("INSERT INTO sys_base_menus (id, name, title, path, icon, hidden) VALUES (?, ?, ?, ?, ?, ?)", menu.id, menu.name, menu.title, menu.name, "document", false).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&visitTestAuthorityMenu{AuthorityID: 9, MenuID: menu.id}).Error; err != nil {
			t.Fatal(err)
		}
	}
	global.GVA_DB = db
	defer func() { global.GVA_DB = previousDB }()

	service := UserMenuVisitServiceApp
	if err := service.Record(101, 9, 4); err != nil {
		t.Fatal(err)
	}
	if err := service.Record(101, 9, 4); err != nil {
		t.Fatal(err)
	}
	if err := service.Record(101, 9, 3); err != nil {
		t.Fatal(err)
	}
	if err := service.Record(202, 9, 2); err != nil {
		t.Fatal(err)
	}
	if err := service.Record(101, 9, 2); err != nil {
		t.Fatal(err)
	}
	if err := service.Record(303, 9, 99); !errors.Is(err, ErrMenuVisitUnavailable) {
		t.Fatalf("expected unauthorized menu rejection, got %v", err)
	}

	list, err := service.List(101, 9)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[0].MenuID != 4 || list[0].VisitCount != 2 {
		t.Fatalf("unexpected user 101 visits: %#v", list)
	}
	otherUser, err := service.List(202, 9)
	if err != nil {
		t.Fatal(err)
	}
	if len(otherUser) != 1 || otherUser[0].MenuID != 2 || otherUser[0].VisitCount != 1 {
		t.Fatalf("unexpected user 202 visits: %#v", otherUser)
	}
	noVisits, err := service.List(303, 9)
	if err != nil || len(noVisits) != 0 {
		t.Fatalf("expected empty visits for user 303, got %#v, err %v", noVisits, err)
	}

	older := time.Now().Add(-time.Hour)
	newer := time.Now()
	if err := db.Model(&system.SysUserMenuVisit{}).Where("user_id = ? AND menu_id = ?", 101, 3).Update("last_visited_at", older).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&system.SysUserMenuVisit{}).Where("user_id = ? AND menu_id = ?", 101, 2).Update("last_visited_at", newer).Error; err != nil {
		t.Fatal(err)
	}
	list, err = service.List(101, 9)
	if err != nil {
		t.Fatal(err)
	}
	if list[0].MenuID != 4 || list[1].MenuID != 2 || list[1].VisitCount != 1 || list[2].MenuID != 3 {
		t.Fatalf("expected equal-count visits sorted by recency after highest count: %#v", list)
	}
}
