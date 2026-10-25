package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Use SQLite-native datetime DDL without changing the MySQL model contract.
type sqliteDeliveryNote struct {
	model.DeliveryNote
	DeliveredAt time.Time `gorm:"column:delivered_at;type:datetime;not null"`
}

func (sqliteDeliveryNote) TableName() string { return "wh_delivery_notes" }

func TestDeliveryReviewHTTPCommandsPersistOpinionAndRespectVersion(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil { t.Fatal(err) }
	sqlDB, err := db.DB()
	if err != nil { t.Fatal(err) }
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&sqliteDeliveryNote{}); err != nil { t.Fatal(err) }
	note := model.DeliveryNote{Number: "SK261025000001", Status: "reviewed", DeliveredAt: time.Now()}
	service.ApplyCreateAudit(&note.BusinessModel, 4, time.Now())
	if err := db.Create(&note).Error; err != nil { t.Fatal(err) }
	a := NewDeliveryAPI(service.NewDeliveryService(db))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("claims", &systemReq.CustomClaims{BaseClaims: systemReq.BaseClaims{ID: 4, Username: "操作者"}})
		c.Next()
	})
	router.POST("/warehouse/delivery/reReview", a.ReReview)
	router.POST("/warehouse/delivery/cancelReReview", a.CancelReReview)
	router.POST("/warehouse/delivery/unreview", a.Unreview)
	router.POST("/warehouse/delivery/create", a.Create)
	router.PUT("/warehouse/delivery/update", a.Update)
	call := func(method, path, body string) struct { Code int; Data json.RawMessage } {
		t.Helper()
		request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		var result struct { Code int; Data json.RawMessage }
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil { t.Fatalf("body=%s err=%v", response.Body.String(), err) }
		return result
	}
	result := call(http.MethodPost, "/warehouse/delivery/reReview", fmt.Sprintf(`{"id":%d,"version":1,"opinion":"核对通过"}`, note.ID))
	if result.Code != 0 { t.Fatalf("reReview failed: %s", result.Data) }
	var stored model.DeliveryNote
	if err := db.First(&stored, note.ID).Error; err != nil { t.Fatal(err) }
	if stored.Status != "reReviewed" || stored.ReReviewOpinion == nil || *stored.ReReviewOpinion != "核对通过" || stored.Version != 2 { t.Fatalf("stored=%#v", stored) }
	for _, tc := range []struct{ endpoint, body, errorCode string }{
		{"cancelReReview", fmt.Sprintf(`{"id":%d,"version":1}`, note.ID), "VERSION_CONFLICT"},
		{"unreview", fmt.Sprintf(`{"id":%d,"version":2}`, note.ID), "STATE_CONFLICT"},
	} {
		failure := call(http.MethodPost, "/warehouse/delivery/"+tc.endpoint, tc.body)
		var data struct { ErrorCode string `json:"errorCode"` }
		if err := json.Unmarshal(failure.Data, &data); err != nil { t.Fatal(err) }
		if failure.Code == 0 || data.ErrorCode != tc.errorCode { t.Fatalf("%s response=%s", tc.endpoint, failure.Data) }
	}
	result = call(http.MethodPost, "/warehouse/delivery/cancelReReview", fmt.Sprintf(`{"id":%d,"version":2}`, note.ID))
	if result.Code != 0 { t.Fatalf("cancel failed: %s", result.Data) }
	stored = model.DeliveryNote{}
	if err := db.First(&stored, note.ID).Error; err != nil { t.Fatal(err) }
	if stored.Status != "reviewed" || stored.ReReviewOpinion != nil || stored.Version != 3 { t.Fatalf("cancelled=%#v", stored) }
	result = call(http.MethodPost, "/warehouse/delivery/reReview", fmt.Sprintf(`{"id":%d,"version":3,"opinion":null}`, note.ID))
	if result.Code != 0 { t.Fatalf("null opinion failed: %s", result.Data) }
	var payload map[string]interface{}
	if err := json.Unmarshal(result.Data, &payload); err != nil { t.Fatal(err) }
	if value, exists := payload["reReviewOpinion"]; !exists || value != nil { t.Fatalf("nullable opinion omitted or non-null: %s", result.Data) }
	for _, tc := range []struct { method, path, body string }{
		{http.MethodPost, "/warehouse/delivery/create", `{"reservationToken":"reserved","brandId":1,"deliveredAt":"2026-10-25T00:00:00Z","review":true,"details":[{"materialId":1,"materialUnitId":1,"quantity":"3","amount":"10.00"}]}`},
		{http.MethodPut, "/warehouse/delivery/update", fmt.Sprintf(`{"id":%d,"version":4,"brandId":1,"deliveredAt":"2026-10-25T00:00:00Z","review":true,"details":[{"materialId":1,"materialUnitId":1,"quantity":"3","amount":"10.00"}]}`, note.ID)},
	} {
		failure := call(tc.method, tc.path, tc.body)
		var data struct { ErrorCode string `json:"errorCode"` }
		if err := json.Unmarshal(failure.Data, &data); err != nil { t.Fatal(err) }
		if failure.Code == 0 || data.ErrorCode != "FORBIDDEN" { t.Fatalf("save review bypassed permission: %s", failure.Data) }
	}
}
