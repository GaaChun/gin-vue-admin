package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateUnitInvalidPayloadReturnsFieldError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := MasterAPI{}
	router.POST("/units", handler.CreateUnit)
	request := httptest.NewRequest(http.MethodPost, "/units", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	writer := httptest.NewRecorder()
	router.ServeHTTP(writer, request)
	if !strings.Contains(writer.Body.String(), `"errorCode":"VALIDATION_FAILED"`) {
		t.Fatalf("expected structured validation failure, got %s", writer.Body.String())
	}
}
