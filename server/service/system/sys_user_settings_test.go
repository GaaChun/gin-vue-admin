package system

import (
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/model/common"
)

func TestMergeUserSettingsPreservesOtherPreferences(t *testing.T) {
	original := common.JSONMap{"primaryColor": "#123456", "warehouseTableWidths.warehouse.master.units": map[string]any{"name": float64(170)}}
	updates := common.JSONMap{"warehouseTableWidths.warehouse.delivery.list": map[string]any{"number": float64(210)}}
	merged := mergeUserSettings(original, updates)
	if merged["primaryColor"] != "#123456" {
		t.Fatalf("updating table widths dropped other user settings: %#v", merged)
	}
	unitWidths, ok := merged["warehouseTableWidths.warehouse.master.units"].(map[string]any)
	if !ok || unitWidths["name"] != float64(170) {
		t.Fatalf("updating another table dropped existing column widths: %#v", merged)
	}
	deliveryWidths, ok := merged["warehouseTableWidths.warehouse.delivery.list"].(map[string]any)
	if !ok || deliveryWidths["number"] != float64(210) {
		t.Fatalf("table width settings were not saved: %#v", merged)
	}
}
