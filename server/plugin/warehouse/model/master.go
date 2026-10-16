package model

import "github.com/shopspring/decimal"

type Unit struct {
	BusinessModel
	Name string `json:"name" gorm:"column:name;type:varchar(100);not null;index:idx_wh_units_active_name;comment:单位名称"`
}
func (Unit) TableName() string { return "wh_units" }

type Brand struct {
	BusinessModel
	Name     string `json:"name" gorm:"column:name;type:varchar(100);not null;index:idx_wh_brands_active_name;comment:品牌名称"`
	Mnemonic string `json:"mnemonic" gorm:"column:mnemonic;type:varchar(100);not null;comment:助记符"`
	Code     string `json:"code" gorm:"column:code;type:varchar(64);not null;index:idx_wh_brands_active_code;comment:品牌编号"`
}
func (Brand) TableName() string { return "wh_brands" }

type Category struct {
	BusinessModel
	Name     string `json:"name" gorm:"column:name;type:varchar(100);not null;index:idx_wh_categories_parent_name;comment:类别名称"`
	Code     string `json:"code" gorm:"column:code;type:varchar(64);not null;index:idx_wh_categories_active_code;comment:类别编号"`
	Mnemonic string `json:"mnemonic" gorm:"column:mnemonic;type:varchar(100);not null;comment:助记符"`
	ParentID uint   `json:"parentId" gorm:"column:parent_id;not null;default:0;index:idx_wh_categories_parent_name,priority:1;index:idx_wh_categories_parent_active,priority:1;comment:父类别ID，0为虚拟根"`
}
func (Category) TableName() string { return "wh_categories" }

type Material struct {
	BusinessModel
	Name        string `json:"name" gorm:"column:name;type:varchar(200);not null;comment:物料名称"`
	Mnemonic    string `json:"mnemonic" gorm:"column:mnemonic;type:varchar(100);not null;comment:助记符"`
	Code        string `json:"code" gorm:"column:code;type:varchar(64);not null;index:idx_wh_materials_active_code;comment:物料编号"`
	BrandID     uint   `json:"brandId" gorm:"column:brand_id;not null;index:idx_wh_materials_brand_category_deleted,priority:1;comment:品牌ID"`
	CategoryID  uint   `json:"categoryId" gorm:"column:category_id;not null;index:idx_wh_materials_brand_category_deleted,priority:2;comment:类别ID"`
	Specification *string `json:"specification,omitempty" gorm:"column:specification;type:varchar(255);comment:规格"`
	Alias       *string `json:"alias,omitempty" gorm:"column:alias;type:varchar(200);comment:别名"`
	ImageFileID *uint  `json:"imageFileId,omitempty" gorm:"column:image_file_id;index;comment:图片文件ID"`
}
func (Material) TableName() string { return "wh_materials" }

type MaterialUnit struct {
	BusinessModel
	MaterialID uint            `json:"materialId" gorm:"column:material_id;not null;index:idx_wh_material_units_material_unit,priority:1;index:idx_wh_material_units_base;comment:物料ID"`
	UnitID     uint            `json:"unitId" gorm:"column:unit_id;not null;index:idx_wh_material_units_material_unit,priority:2;comment:单位ID"`
	Factor     decimal.Decimal `json:"factor" gorm:"column:factor;type:decimal(18,6);not null;default:1;comment:相对基本单位系数"`
	IsBase     bool            `json:"isBase" gorm:"column:is_base;not null;default:false;comment:是否基本单位"`
	Enabled    bool            `json:"enabled" gorm:"column:enabled;not null;default:true;comment:是否可选"`
}
func (MaterialUnit) TableName() string { return "wh_material_units" }

type MaterialImage struct {
	BusinessModel
	FileID            uint   `json:"fileId" gorm:"column:file_id;not null;uniqueIndex;comment:系统文件ID"`
	MIME              string `json:"mime" gorm:"column:mime;type:varchar(32);not null;comment:经验证MIME"`
	SizeBytes         int64  `json:"sizeBytes" gorm:"column:size_bytes;not null;comment:文件字节数"`
	ValidationVersion uint   `json:"validationVersion" gorm:"column:validation_version;not null;comment:校验规则版本"`
}
func (MaterialImage) TableName() string { return "wh_material_images" }
