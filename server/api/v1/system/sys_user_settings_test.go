package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/common"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestSetSelfSettingUsesAuthenticatedAccountAndMergesPreferences(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.TempDir()+"/settings.db"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE sys_users (id integer primary key, origin_setting text, deleted_at datetime, updated_at datetime)").Error; err != nil {
		t.Fatal(err)
	}
	for id, settings := range map[uint]string{
		101: `{"primaryColor":"#123456","warehouseTableWidths.warehouse.delivery.list":{"number":210}}`,
		202: `{"primaryColor":"#abcdef","warehouseDocumentDisplay.delivery":{"dateDefault":"today"}}`,
	} {
		if err := db.Exec("INSERT INTO sys_users (id, origin_setting) VALUES (?, ?)", id, settings).Error; err != nil {
			t.Fatal(err)
		}
	}
	previousDB, previousLog := global.GVA_DB, global.GVA_LOG
	global.GVA_DB, global.GVA_LOG = db, zap.NewNop()
	t.Cleanup(func() { global.GVA_DB, global.GVA_LOG = previousDB, previousLog })

	readSettings := func(id uint) common.JSONMap {
		t.Helper()
		var user model.SysUser
		if err := db.First(&user, id).Error; err != nil {
			t.Fatal(err)
		}
		return user.OriginSetting
	}
	invoke := func(actor uint, body string) int {
		t.Helper()
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPut, "/user/setSelfSetting", strings.NewReader(body))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Set("claims", &request.CustomClaims{BaseClaims: request.BaseClaims{ID: actor}})
		new(BaseApi).SetSelfSetting(ctx)
		var result struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result.Code
	}

	otherBefore := readSettings(202)
	if code := invoke(101, `{"id":202,"uid":202,"warehouseDocumentDisplay.delivery":{"dateDefault":"last7","list":{"columns":["number"]}}}`); code != 0 {
		t.Fatalf("save failed with code %d", code)
	}
	current := readSettings(101)
	if current["primaryColor"] != "#123456" || current["warehouseTableWidths.warehouse.delivery.list"] == nil {
		t.Fatalf("display save removed unrelated preferences: %#v", current)
	}
	display, ok := current["warehouseDocumentDisplay.delivery"].(map[string]any)
	if !ok || display["dateDefault"] != "last7" {
		t.Fatalf("authenticated user's display preference not saved: %#v", current)
	}
	if !reflect.DeepEqual(readSettings(202), otherBefore) {
		t.Fatal("request body selected another account")
	}
	firstBefore := readSettings(101)
	if code := invoke(202, `{"warehouseDocumentDisplay.delivery":{"dateDefault":"yesterday"}}`); code != 0 {
		t.Fatalf("second account save failed with code %d", code)
	}
	if !reflect.DeepEqual(readSettings(101), firstBefore) {
		t.Fatal("second account save modified first account")
	}
	secondBefore := readSettings(202)
	for _, actor := range []uint{0, 999} {
		if code := invoke(actor, `{"warehouseDocumentDisplay.delivery":{"dateDefault":"thisYear"}}`); code == 0 {
			t.Fatalf("nonexistent authenticated actor %d was accepted", actor)
		}
	}
	if !reflect.DeepEqual(readSettings(101), firstBefore) || !reflect.DeepEqual(readSettings(202), secondBefore) {
		t.Fatal("rejected actor changed persisted user preferences")
	}
}
