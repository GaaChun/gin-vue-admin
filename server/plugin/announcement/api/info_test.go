package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/announcement/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestCreatedAnnouncementCanBeSavedAgainWithoutCreatingAnotherRecord(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.TempDir()+"/announcement.db"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.Info{}); err != nil {
		t.Fatal(err)
	}
	previousDB, previousLog := global.GVA_DB, global.GVA_LOG
	global.GVA_DB, global.GVA_LOG = db, zap.NewNop()
	t.Cleanup(func() { global.GVA_DB, global.GVA_LOG = previousDB, previousLog })

	invoke := func(method, path, body string, handler gin.HandlerFunc) struct {
		Code int        `json:"code"`
		Data model.Info `json:"data"`
	} {
		t.Helper()
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(method, path, strings.NewReader(body))
		ctx.Request.Header.Set("Content-Type", "application/json")
		handler(ctx)
		var result struct {
			Code int        `json:"code"`
			Data model.Info `json:"data"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	created := invoke(http.MethodPost, "/info/createInfo", `{"title":"first","content":"draft","attachments":[]}`, Info.CreateInfo)
	if created.Code != 0 || created.Data.ID == 0 || created.Data.Title != "first" {
		t.Fatalf("create response must return its persisted identity and fields: %#v", created)
	}
	updated := invoke(http.MethodPut, "/info/updateInfo", fmt.Sprintf(`{"ID":%d,"title":"saved again","content":"edited","attachments":[]}`, created.Data.ID), Info.UpdateInfo)
	if updated.Code != 0 {
		t.Fatalf("update returned code %d", updated.Code)
	}
	var records []model.Info
	if err := db.Find(&records).Error; err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].ID != created.Data.ID || records[0].Title != "saved again" {
		t.Fatalf("repeat save did not update the created record: %#v", records)
	}
}
