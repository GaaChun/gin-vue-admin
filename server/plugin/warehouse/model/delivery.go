package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type DeliveryNote struct {
	BusinessModel
	Number                 string          `json:"number" gorm:"column:number;type:varchar(14);not null;uniqueIndex;comment:不可变送货单号"`
	DeliveredAt            time.Time       `json:"deliveredAt" gorm:"column:delivered_at;type:datetime(6);not null;index:idx_wh_delivery_notes_status_delivered,priority:2;index:idx_wh_delivery_notes_brand_delivered,priority:2;comment:送货时间"`
	Department             string          `json:"department" gorm:"column:department;type:varchar(100);not null;default:'';comment:部门"`
	BrandID                uint            `json:"brandId" gorm:"column:brand_id;not null;index:idx_wh_delivery_notes_brand_delivered,priority:1;comment:品牌ID"`
	BrandCodeSnapshot      string          `json:"brandCodeSnapshot" gorm:"column:brand_code_snapshot;type:varchar(64);not null;comment:品牌编号快照"`
	BrandNameSnapshot      string          `json:"brandNameSnapshot" gorm:"column:brand_name_snapshot;type:varchar(100);not null;comment:品牌名称快照"`
	Status                 string          `json:"status" gorm:"column:status;type:varchar(16);not null;index:idx_wh_delivery_notes_status_delivered,priority:1;comment:状态 draft或reviewed"`
	TotalAmount            decimal.Decimal `json:"totalAmount" gorm:"column:total_amount;type:decimal(38,2);not null;comment:已舍入明细金额合计"`
	CreatedByNameSnapshot  string          `json:"createdByNameSnapshot" gorm:"column:created_by_name_snapshot;type:varchar(100);not null;comment:创建人名称快照"`
	ReviewedByNameSnapshot *string         `json:"reviewedByNameSnapshot,omitempty" gorm:"column:reviewed_by_name_snapshot;type:varchar(100);comment:审核人名称快照"`
}
func (DeliveryNote) TableName() string { return "wh_delivery_notes" }

type DeliveryDetail struct {
	BusinessModel
	NoteID                  uint            `json:"noteId" gorm:"column:note_id;not null;index:idx_wh_delivery_details_note_line,priority:1;comment:送货单ID"`
	LineNo                  uint            `json:"lineNo" gorm:"column:line_no;not null;index:idx_wh_delivery_details_note_line,priority:3;comment:行号"`
	MaterialID              uint            `json:"materialId" gorm:"column:material_id;not null;index;comment:物料ID"`
	MaterialUnitID          uint            `json:"materialUnitId" gorm:"column:material_unit_id;not null;index;comment:物料单位绑定ID"`
	UnitID                  uint            `json:"unitId" gorm:"column:unit_id;not null;index;comment:单位ID"`
	BaseUnitID              uint            `json:"baseUnitId" gorm:"column:base_unit_id;not null;index;comment:基本单位ID"`
	CategoryID              uint            `json:"categoryId" gorm:"column:category_id;not null;index;comment:类别ID"`
	MaterialCodeSnapshot    string          `json:"materialCodeSnapshot" gorm:"column:material_code_snapshot;type:varchar(64);not null;comment:物料编号快照"`
	MaterialNameSnapshot    string          `json:"materialNameSnapshot" gorm:"column:material_name_snapshot;type:varchar(200);not null;comment:物料名称快照"`
	SpecificationSnapshot   *string         `json:"specificationSnapshot,omitempty" gorm:"column:specification_snapshot;type:varchar(255);comment:规格快照"`
	CategoryCodeSnapshot    string          `json:"categoryCodeSnapshot" gorm:"column:category_code_snapshot;type:varchar(64);not null;comment:类别编号快照"`
	CategoryNameSnapshot    string          `json:"categoryNameSnapshot" gorm:"column:category_name_snapshot;type:varchar(100);not null;comment:类别名称快照"`
	UnitNameSnapshot        string          `json:"unitNameSnapshot" gorm:"column:unit_name_snapshot;type:varchar(100);not null;comment:单位名称快照"`
	BaseUnitNameSnapshot    string          `json:"baseUnitNameSnapshot" gorm:"column:base_unit_name_snapshot;type:varchar(100);not null;comment:基本单位名称快照"`
	Factor                  decimal.Decimal `json:"factor" gorm:"column:factor;type:decimal(18,6);not null;comment:单位系数快照"`
	Quantity                decimal.Decimal `json:"quantity" gorm:"column:quantity;type:decimal(18,4);not null;comment:数量"`
	UnitPrice               decimal.Decimal `json:"unitPrice" gorm:"column:unit_price;type:decimal(18,4);not null;comment:单价"`
	BaseQuantity            decimal.Decimal `json:"baseQuantity" gorm:"column:base_quantity;type:decimal(38,10);not null;comment:数量乘系数，不中途舍入"`
	Amount                  decimal.Decimal `json:"amount" gorm:"column:amount;type:decimal(38,2);not null;comment:数量乘单价半入金额"`
}
func (DeliveryDetail) TableName() string { return "wh_delivery_details" }

type NumberCounter struct {
	BusinessModel
	BusinessDate time.Time `json:"businessDate" gorm:"column:business_date;type:date;not null;uniqueIndex;comment:业务日"`
	LastValue    uint      `json:"lastValue" gorm:"column:last_value;not null;comment:当日流水0至999999"`
}
func (NumberCounter) TableName() string { return "wh_number_counters" }

type NumberReservation struct {
	BusinessModel
	Number         string `json:"number" gorm:"column:number;type:varchar(14);not null;uniqueIndex;comment:预占单号"`
	Token          string `json:"token" gorm:"column:token;type:varchar(36);not null;uniqueIndex;comment:随机预占令牌"`
	OwnerID        uint   `json:"ownerId" gorm:"column:owner_id;not null;index;comment:预占人ID"`
	ConsumedNoteID *uint  `json:"consumedNoteId,omitempty" gorm:"column:consumed_note_id;uniqueIndex;comment:已消费送货单ID"`
}
func (NumberReservation) TableName() string { return "wh_number_reservations" }

type ReportScheme struct {
	BusinessModel
	Name        string `json:"name" gorm:"column:name;type:varchar(100);not null;comment:方案名称"`
	Scope       string `json:"scope" gorm:"column:scope;type:varchar(16);not null;comment:personal或shared"`
	OwnerID     uint   `json:"ownerId" gorm:"column:owner_id;not null;index;comment:所有者ID"`
	ColumnsJSON string `json:"columnsJson" gorm:"column:columns_json;type:json;not null;comment:有序列ID数组"`
}
func (ReportScheme) TableName() string { return "wh_report_schemes" }
