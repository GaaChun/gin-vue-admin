package initialize

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type schemaColumn struct {
	table      string
	name       string
	definition string
}

type schemaIndex struct {
	table string
	name  string
	sql   string
}

var mysqlVersionPattern = regexp.MustCompile(`\d+\.\d+`)

func Gorm(ctx context.Context) {
	if global.GVA_DB == nil {
		zap.L().Error("warehouse schema deployment failed", zap.Error(fmt.Errorf("warehouse schema deployment requires a MySQL database connection")))
		return
	}
	if err := migrateWarehouseSchema(global.GVA_DB.WithContext(ctx)); err != nil {
		zap.L().Error("warehouse schema deployment failed", zap.Error(err))
	}
}

func migrateWarehouseSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("warehouse schema deployment requires a MySQL database connection")
	}
	if db.Dialector.Name() != "mysql" {
		return fmt.Errorf("warehouse schema deployment requires MySQL generated columns, got %q", db.Dialector.Name())
	}
	var version string
	if err := db.Raw("SELECT VERSION()").Scan(&version).Error; err != nil {
		return fmt.Errorf("read MySQL version for warehouse generated columns: %w", err)
	}
	if err := validateGeneratedColumnSupport(version); err != nil {
		return err
	}
	if err := db.AutoMigrate(
		new(model.Unit), new(model.Brand), new(model.Category), new(model.Material),
		new(model.MaterialUnit), new(model.MaterialImage), new(model.DeliveryNote),
		new(model.DeliveryDetail), new(model.NumberCounter), new(model.NumberReservation),
		new(model.ReportScheme), new(model.MasterCodeCounter),
	); err != nil {
		return fmt.Errorf("auto-migrate warehouse tables: %w", err)
	}
	for _, table := range warehouseTables() {
		query := fmt.Sprintf("UPDATE `%s` SET `is_deleted` = (`deleted_at` IS NOT NULL), `deleted_by` = CASE WHEN `deleted_at` IS NULL THEN NULL ELSE COALESCE(`deleted_by`, NULLIF(`updated_by`, 0)) END WHERE `is_deleted` <> (`deleted_at` IS NOT NULL) OR (`deleted_at` IS NULL AND `deleted_by` IS NOT NULL) OR (`deleted_at` IS NOT NULL AND `deleted_by` IS NULL AND `updated_by` <> 0)", table)
		if err := db.Exec(query).Error; err != nil {
			return fmt.Errorf("backfill deletion audit fields for %s: %w", table, err)
		}
	}
	for _, column := range warehouseSchemaColumns() {
		if db.Migrator().HasColumn(column.table, column.name) {
			continue
		}
		if err := db.Exec(fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s", column.table, column.name, column.definition)).Error; err != nil {
			return fmt.Errorf("add generated column %s.%s: %w", column.table, column.name, err)
		}
	}
	for _, index := range warehouseSchemaIndexes() {
		if db.Migrator().HasIndex(index.table, index.name) {
			continue
		}
		if err := db.Exec(index.sql).Error; err != nil {
			return fmt.Errorf("add index %s.%s: %w", index.table, index.name, err)
		}
	}
	return nil
}

func validateGeneratedColumnSupport(version string) error {
	lower := strings.ToLower(version)
	match := mysqlVersionPattern.FindString(lower)
	if match == "" {
		return fmt.Errorf("cannot determine MySQL generated-column support from version %q", version)
	}
	parts := strings.Split(match, ".")
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	if strings.Contains(lower, "mariadb") {
		if major > 10 || major == 10 && minor >= 2 {
			return nil
		}
		return fmt.Errorf("MariaDB %q does not support the required generated columns; require MariaDB 10.2+", version)
	}
	if major > 5 || major == 5 && minor >= 7 {
		return nil
	}
	return fmt.Errorf("MySQL %q does not support the required generated columns; require MySQL 5.7+", version)
}
func warehouseTables() []string {
	return []string{
		"wh_units", "wh_brands", "wh_categories", "wh_materials",
		"wh_material_units", "wh_material_images", "wh_delivery_notes",
		"wh_delivery_details", "wh_number_counters", "wh_number_reservations",
		"wh_report_schemes", "wh_master_code_counters",
	}
}

// Active uniqueness must live in MySQL: concurrent application-side checks cannot serialize writers.
func warehouseSchemaColumns() []schemaColumn {
	return []schemaColumn{
		{"wh_units", "active_name_key", "varchar(100) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN LOWER(TRIM(name)) ELSE NULL END) STORED"},
		{"wh_brands", "active_name_key", "varchar(100) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN LOWER(TRIM(name)) ELSE NULL END) STORED"},
		{"wh_brands", "active_code_key", "varchar(64) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN LOWER(TRIM(code)) ELSE NULL END) STORED"},
		{"wh_categories", "active_code_key", "varchar(64) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN LOWER(TRIM(code)) ELSE NULL END) STORED"},
		{"wh_categories", "active_parent_name_key", "varchar(100) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN LOWER(TRIM(name)) ELSE NULL END) STORED"},
		{"wh_materials", "active_code_key", "varchar(64) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN LOWER(TRIM(code)) ELSE NULL END) STORED"},
		{"wh_material_units", "active_material_unit_key", "varchar(64) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN CONCAT(material_id, ':', unit_id) ELSE NULL END) STORED"},
		{"wh_material_units", "active_base_material_id", "bigint unsigned GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL AND is_base = 1 THEN material_id ELSE NULL END) STORED"},
		{"wh_report_schemes", "active_personal_owner_id", "bigint unsigned GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL AND scope = 'personal' THEN owner_id ELSE NULL END) STORED"},
		{"wh_report_schemes", "active_personal_name_key", "varchar(100) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL AND scope = 'personal' THEN LOWER(TRIM(name)) ELSE NULL END) STORED"},
		{"wh_report_schemes", "active_shared_name_key", "varchar(100) GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL AND scope = 'shared' THEN LOWER(TRIM(name)) ELSE NULL END) STORED"},
	}
}

func warehouseSchemaIndexes() []schemaIndex {
	return []schemaIndex{
		{"wh_units", "uk_wh_units_active_name", "CREATE UNIQUE INDEX `uk_wh_units_active_name` ON `wh_units` (`active_name_key`)"},
		{"wh_brands", "uk_wh_brands_active_name", "CREATE UNIQUE INDEX `uk_wh_brands_active_name` ON `wh_brands` (`active_name_key`)"},
		{"wh_brands", "uk_wh_brands_active_code", "CREATE UNIQUE INDEX `uk_wh_brands_active_code` ON `wh_brands` (`active_code_key`)"},
		{"wh_categories", "uk_wh_categories_active_code", "CREATE UNIQUE INDEX `uk_wh_categories_active_code` ON `wh_categories` (`active_code_key`)"},
		{"wh_categories", "uk_wh_categories_active_parent_name", "CREATE UNIQUE INDEX `uk_wh_categories_active_parent_name` ON `wh_categories` (`parent_id`, `active_parent_name_key`)"},
		{"wh_materials", "uk_wh_materials_active_code", "CREATE UNIQUE INDEX `uk_wh_materials_active_code` ON `wh_materials` (`active_code_key`)"},
		{"wh_material_units", "uk_wh_material_units_active_material_unit", "CREATE UNIQUE INDEX `uk_wh_material_units_active_material_unit` ON `wh_material_units` (`active_material_unit_key`)"},
		{"wh_material_units", "uk_wh_material_units_active_base", "CREATE UNIQUE INDEX `uk_wh_material_units_active_base` ON `wh_material_units` (`active_base_material_id`)"},
		{"wh_report_schemes", "uk_wh_report_schemes_active_personal_name", "CREATE UNIQUE INDEX `uk_wh_report_schemes_active_personal_name` ON `wh_report_schemes` (`active_personal_owner_id`, `active_personal_name_key`)"},
		{"wh_report_schemes", "uk_wh_report_schemes_active_shared_name", "CREATE UNIQUE INDEX `uk_wh_report_schemes_active_shared_name` ON `wh_report_schemes` (`active_shared_name_key`)"},
		{"wh_delivery_notes", "uk_wh_delivery_notes_number_history", "CREATE UNIQUE INDEX `uk_wh_delivery_notes_number_history` ON `wh_delivery_notes` (`number`)"},
		{"wh_number_counters", "uk_wh_number_counters_business_date_history", "CREATE UNIQUE INDEX `uk_wh_number_counters_business_date_history` ON `wh_number_counters` (`business_date`)"},
		{"wh_number_reservations", "uk_wh_number_reservations_number_history", "CREATE UNIQUE INDEX `uk_wh_number_reservations_number_history` ON `wh_number_reservations` (`number`)"},
		{"wh_number_reservations", "uk_wh_number_reservations_token_history", "CREATE UNIQUE INDEX `uk_wh_number_reservations_token_history` ON `wh_number_reservations` (`token`)"},
		{"wh_materials", "idx_wh_materials_brand_category_deleted_d3", "CREATE INDEX `idx_wh_materials_brand_category_deleted_d3` ON `wh_materials` (`brand_id`, `category_id`, `deleted_at`)"},
		{"wh_materials", "idx_wh_materials_category_id_ref", "CREATE INDEX `idx_wh_materials_category_id_ref` ON `wh_materials` (`category_id`)"},
		{"wh_categories", "idx_wh_categories_parent_deleted_d3", "CREATE INDEX `idx_wh_categories_parent_deleted_d3` ON `wh_categories` (`parent_id`, `deleted_at`)"},
		{"wh_delivery_notes", "idx_wh_delivery_notes_status_delivered_deleted_d3", "CREATE INDEX `idx_wh_delivery_notes_status_delivered_deleted_d3` ON `wh_delivery_notes` (`status`, `delivered_at`, `deleted_at`)"},
		{"wh_delivery_notes", "idx_wh_delivery_notes_brand_delivered_d3", "CREATE INDEX `idx_wh_delivery_notes_brand_delivered_d3` ON `wh_delivery_notes` (`brand_id`, `delivered_at`)"},
		{"wh_delivery_notes", "idx_wh_delivery_notes_reviewed_at_d3", "CREATE INDEX `idx_wh_delivery_notes_reviewed_at_d3` ON `wh_delivery_notes` (`reviewed_at`)"},
		{"wh_delivery_details", "idx_wh_delivery_details_note_deleted_line_d3", "CREATE INDEX `idx_wh_delivery_details_note_deleted_line_d3` ON `wh_delivery_details` (`note_id`, `deleted_at`, `line_no`)"},
		{"wh_delivery_details", "idx_wh_delivery_details_material_id_d3", "CREATE INDEX `idx_wh_delivery_details_material_id_d3` ON `wh_delivery_details` (`material_id`)"},
		{"wh_delivery_details", "idx_wh_delivery_details_material_unit_id_d3", "CREATE INDEX `idx_wh_delivery_details_material_unit_id_d3` ON `wh_delivery_details` (`material_unit_id`)"},
		{"wh_delivery_details", "idx_wh_delivery_details_category_id_d3", "CREATE INDEX `idx_wh_delivery_details_category_id_d3` ON `wh_delivery_details` (`category_id`)"},
		{"wh_delivery_details", "idx_wh_delivery_details_unit_id_d3", "CREATE INDEX `idx_wh_delivery_details_unit_id_d3` ON `wh_delivery_details` (`unit_id`)"},
		{"wh_delivery_details", "idx_wh_delivery_details_base_unit_id_d3", "CREATE INDEX `idx_wh_delivery_details_base_unit_id_d3` ON `wh_delivery_details` (`base_unit_id`)"},
		{"wh_material_units", "idx_wh_material_units_unit_id_ref", "CREATE INDEX `idx_wh_material_units_unit_id_ref` ON `wh_material_units` (`unit_id`)"},
	}
}
