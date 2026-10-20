package initialize

import (
	"reflect"
	"strings"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
)

func TestWarehouseModelsShareDeletionAuditFields(t *testing.T) {
	models := []interface{}{
		model.Unit{}, model.Brand{}, model.Category{}, model.Material{},
		model.MaterialUnit{}, model.MaterialImage{}, model.DeliveryNote{},
		model.DeliveryDetail{}, model.NumberCounter{}, model.NumberReservation{},
		model.ReportScheme{},
	}
	if len(models) != len(warehouseTables()) {
		t.Fatalf("warehouse model/table coverage mismatch: models=%d tables=%d", len(models), len(warehouseTables()))
	}
	for _, value := range models {
		typ := reflect.TypeOf(value)
		for _, name := range []string{"DeletedAt", "DeletedBy", "IsDeleted"} {
			if _, ok := typ.FieldByName(name); !ok {
				t.Errorf("%s lacks %s", typ.Name(), name)
			}
		}
	}
}
func TestWarehouseDDLUsesNullableGeneratedKeysForActiveUniqueness(t *testing.T) {
	activeCode := findSchemaColumn(t, "wh_materials", "active_code_key")
	if !strings.Contains(activeCode.definition, "deleted_at IS NULL") || !strings.Contains(activeCode.definition, "ELSE NULL") {
		t.Fatalf("active key must become NULL after soft deletion: %s", activeCode.definition)
	}
	if !strings.Contains(activeCode.definition, "LOWER(TRIM(code))") {
		t.Fatalf("active code key must normalize whitespace and case: %s", activeCode.definition)
	}
	activeCodeIndex := findSchemaIndex(t, "uk_wh_materials_active_code")
	if !strings.Contains(activeCodeIndex.sql, "UNIQUE INDEX") || strings.Contains(activeCodeIndex.sql, "deleted_at") {
		t.Fatalf("active uniqueness must target the generated key, not (code, deleted_at): %s", activeCodeIndex.sql)
	}

	baseUnit := findSchemaColumn(t, "wh_material_units", "active_base_material_id")
	if !strings.Contains(baseUnit.definition, "deleted_at IS NULL AND is_base = 1") || !strings.Contains(baseUnit.definition, "ELSE NULL") {
		t.Fatalf("base unit key must only constrain active base bindings: %s", baseUnit.definition)
	}
	if !strings.Contains(findSchemaIndex(t, "uk_wh_material_units_active_base").sql, "UNIQUE INDEX") {
		t.Fatal("an active material must have at most one base-unit binding")
	}
}

func TestWarehouseDDLProtectsHistoryAndReferenceQueries(t *testing.T) {
	for _, name := range []string{
		"uk_wh_delivery_notes_number_history",
		"uk_wh_number_counters_business_date_history",
		"uk_wh_number_reservations_number_history",
		"uk_wh_number_reservations_token_history",
	} {
		if !strings.Contains(findSchemaIndex(t, name).sql, "UNIQUE INDEX") {
			t.Fatalf("%s must remain unique across soft-deleted history", name)
		}
	}
	for _, name := range []string{
		"idx_wh_delivery_details_note_deleted_line_d3",
		"idx_wh_delivery_details_material_id_d3",
		"idx_wh_material_units_unit_id_ref",
	} {
		if findSchemaIndex(t, name).sql == "" {
			t.Fatalf("missing reference/query index %s", name)
		}
	}
}

func TestGeneratedColumnVersionSupport(t *testing.T) {
	if err := validateGeneratedColumnSupport("8.0.36"); err != nil {
		t.Fatalf("supported MySQL version rejected: %v", err)
	}
	if err := validateGeneratedColumnSupport("5.6.51"); err == nil {
		t.Fatal("unsupported MySQL version accepted")
	}
	if err := validateGeneratedColumnSupport("10.1.48-MariaDB"); err == nil {
		t.Fatal("unsupported MariaDB version accepted")
	}
}

func findSchemaColumn(t *testing.T, table, name string) schemaColumn {
	t.Helper()
	for _, column := range warehouseSchemaColumns() {
		if column.table == table && column.name == name {
			return column
		}
	}
	t.Fatalf("missing generated column %s.%s", table, name)
	return schemaColumn{}
}

func findSchemaIndex(t *testing.T, name string) schemaIndex {
	t.Helper()
	for _, index := range warehouseSchemaIndexes() {
		if index.name == name {
			return index
		}
	}
	t.Fatalf("missing index %s", name)
	return schemaIndex{}
}
