package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestReserveMasterCodeRejectsUncodedOrInvalidResources(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, payload := range []string{`{}`, `{"resource":"units"}`, `{"resource":"delivery"}`, `{"resource":7}`} {
		t.Run(payload, func(t *testing.T) {
			router := gin.New()
			router.POST("/reserveCode", MasterAPI{}.ReserveCode)
			request := httptest.NewRequest(http.MethodPost, "/reserveCode", strings.NewReader(payload))
			request.Header.Set("Content-Type", "application/json")
			writer := httptest.NewRecorder()
			router.ServeHTTP(writer, request)
			if !strings.Contains(writer.Body.String(), `"errorCode":"VALIDATION_FAILED"`) {
				t.Fatalf("invalid resource must fail before allocation: %s", writer.Body.String())
			}
		})
	}
}
