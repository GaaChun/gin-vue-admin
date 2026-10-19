// Package request defines explicit warehouse HTTP input contracts. Server-owned audit,
// snapshot, status, and computed amount fields are intentionally absent.
package request

import "time"

// VersionedID identifies a mutable record. A nil Version intentionally skips the
// optional stale-write check; an explicit zero is rejected by the service.
type VersionedID struct {
	ID      uint    `json:"id" form:"id" binding:"required,gt=0"`
	Version *uint64 `json:"version,omitempty" form:"version"`
}

// DeleteInput deletes a mutable record after optional version validation.
type DeleteInput struct {
	VersionedID
}

// ReviewDeliveryInput requests the draft-to-reviewed delivery action.
type ReviewDeliveryInput struct {
	VersionedID
}

// UnreviewDeliveryInput requests the reviewed-to-draft delivery action.
type UnreviewDeliveryInput struct {
	VersionedID
}

// PageQuery is the common page window accepted by warehouse list endpoints.
type PageQuery struct {
	Page     int `json:"page" form:"page" binding:"omitempty,gte=1"`
	PageSize int `json:"pageSize" form:"pageSize" binding:"omitempty,gte=1,lte=100"`
}

// WithDefaults returns the D6 list defaults while preserving explicitly bound values.
func (query PageQuery) WithDefaults() PageQuery {
	if query.Page == 0 {
		query.Page = 1
	}
	if query.PageSize == 0 {
		query.PageSize = 20
	}
	return query
}

// SortQuery sorts a paginated list by a server-approved field.
type SortQuery struct {
	SortBy    string `json:"sortBy,omitempty" form:"sortBy"`
	SortOrder string `json:"sortOrder,omitempty" form:"sortOrder"`
}

// MasterInput contains the fields shared by simple master-data write commands.
type MasterInput struct {
	Name string `json:"name" binding:"required"`
}

// CreateUnitInput creates a unit from its only writable field.
type CreateUnitInput struct {
	MasterInput
}

// UpdateUnitInput changes a unit after optional version validation.
type UpdateUnitInput struct {
	VersionedID
	MasterInput
}

// CreateBrandInput creates a brand from writable fields.
type CreateBrandInput struct {
	BrandInput
}

// UpdateBrandInput changes a brand after optional version validation.
type UpdateBrandInput struct {
	VersionedID
	BrandInput
}

// CreateCategoryInput creates a category from writable fields.
type CreateCategoryInput struct {
	CategoryInput
}

// UpdateCategoryInput changes a category after optional version validation.
type UpdateCategoryInput struct {
	VersionedID
	CategoryInput
}

// BrandInput contains writable brand fields.
type BrandInput struct {
	Name     string `json:"name" binding:"required"`
	Mnemonic string `json:"mnemonic" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

// CategoryInput contains writable category fields. ParentID zero represents the
// virtual root and is never a persisted category reference.
type CategoryInput struct {
	Name     string `json:"name" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Mnemonic string `json:"mnemonic" binding:"required"`
	ParentID uint   `json:"parentId"`
}

// MaterialUnitInput is a complete writable material-unit binding.
type MaterialUnitInput struct {
	UnitID  uint   `json:"unitId" binding:"required,gt=0"`
	Factor  string `json:"factor" binding:"required"`
	IsBase  bool   `json:"isBase"`
	Enabled bool   `json:"enabled"`
}

// MaterialInput contains writable material fields and its complete unit set.
type MaterialInput struct {
	Name          string              `json:"name" binding:"required"`
	Mnemonic      string              `json:"mnemonic" binding:"required"`
	Code          string              `json:"code" binding:"required"`
	BrandID       uint                `json:"brandId"`
	CategoryID    uint                `json:"categoryId"`
	Specification *string             `json:"specification,omitempty"`
	Alias         *string             `json:"alias,omitempty"`
	ImageFileID   *uint               `json:"imageFileId,omitempty"`
	Units         []MaterialUnitInput `json:"units" binding:"required,min=1"`
}

// CreateMaterialInput creates a material from its explicitly writable fields.
type CreateMaterialInput struct {
	MaterialInput
}

// UpdateMaterialInput replaces a material's writable fields and complete unit set.
type UpdateMaterialInput struct {
	VersionedID
	MaterialInput
}

// DeliveryDetailInput is one writable delivery line. A nil ID creates a line;
// non-nil IDs must be verified as belonging to the delivery note.
type DeliveryDetailInput struct {
	ID             *uint  `json:"id,omitempty"`
	MaterialID     uint   `json:"materialId" binding:"required,gt=0"`
	MaterialUnitID uint   `json:"materialUnitId" binding:"required,gt=0"`
	Quantity       string `json:"quantity" binding:"required"`
	UnitPrice      string `json:"unitPrice" binding:"required"`
}

type CreateDeliveryInput struct {
	ReservationToken string                `json:"reservationToken" binding:"required"`
	DeliveredAt      time.Time             `json:"deliveredAt" binding:"required"`
	Department       string                `json:"department"`
	BrandID          uint                  `json:"brandId" binding:"required,gt=0"`
	Details          []DeliveryDetailInput `json:"details" binding:"required,min=1"`
}

// UpdateDeliveryInput replaces editable draft-note content without accepting its
// number, status, snapshots, audit fields, or calculated values.
type UpdateDeliveryInput struct {
	VersionedID
	DeliveredAt time.Time             `json:"deliveredAt" binding:"required"`
	Department  string                `json:"department"`
	BrandID     uint                  `json:"brandId" binding:"required,gt=0"`
	Details     []DeliveryDetailInput `json:"details" binding:"required,min=1"`
}

// ReportSchemeInput contains writable saved-report configuration only.
type ReportSchemeInput struct {
	Name    string   `json:"name" binding:"required"`
	Scope   string   `json:"scope" binding:"required"`
	Columns []string `json:"columns" binding:"required,min=1"`
}

// CreateReportSchemeInput creates a report scheme from writable configuration.
type CreateReportSchemeInput struct {
	ReportSchemeInput
}

// UpdateReportSchemeInput changes a scheme after optional version validation.
type UpdateReportSchemeInput struct {
	VersionedID
	ReportSchemeInput
}

// ReconciliationFilters constrains report data without exposing persistence fields.
type ReconciliationFilters struct {
	DeliveredFrom   *time.Time `json:"deliveredFrom,omitempty"`
	DeliveredTo     *time.Time `json:"deliveredTo,omitempty"`
	ReviewedFrom    *time.Time `json:"reviewedFrom,omitempty"`
	ReviewedTo      *time.Time `json:"reviewedTo,omitempty"`
	Number          string     `json:"number,omitempty"`
	BrandID         *uint      `json:"brandId,omitempty"`
	CategoryID      *uint      `json:"categoryId,omitempty"`
	Department      string     `json:"department,omitempty"`
	MaterialKeyword string     `json:"materialKeyword,omitempty"`
}

// ReconciliationQuery selects report columns and a page of the shared projection.
type ReconciliationQuery struct {
	Filters ReconciliationFilters `json:"filters"`
	Columns []string              `json:"columns" binding:"required,min=1"`
	PageQuery
	SortQuery
}

type UnitListQuery struct {
	PageQuery
	SortQuery
	Name string `json:"name,omitempty" form:"name"`
}

type BrandListQuery struct {
	PageQuery
	SortQuery
	Keyword string `json:"keyword,omitempty" form:"keyword"`
}

// CategoryListQuery filters paginated category data by its optional parent.
type CategoryListQuery struct {
	PageQuery
	ParentID *uint  `json:"parentId,omitempty" form:"parentId"`
	Keyword  string `json:"keyword,omitempty" form:"keyword"`
}

type MaterialListQuery struct {
	PageQuery
	SortQuery
	BrandID    *uint  `json:"brandId,omitempty" form:"brandId"`
	CategoryID *uint  `json:"categoryId,omitempty" form:"categoryId"`
	Keyword    string `json:"keyword,omitempty" form:"keyword"`
}

// DeliveryListQuery filters paginated delivery notes.
type DeliveryListQuery struct {
	PageQuery
	SortQuery
	DeliveredFrom *time.Time `json:"deliveredFrom,omitempty" form:"deliveredFrom"`
	DeliveredTo   *time.Time `json:"deliveredTo,omitempty" form:"deliveredTo"`
	BrandID       *uint      `json:"brandId,omitempty" form:"brandId"`
	Status        string     `json:"status,omitempty" form:"status"`
	Number        string     `json:"number,omitempty" form:"number"`
	Department    string     `json:"department,omitempty" form:"department"`
}

// ReportSchemeListQuery filters paginated report schemes.
type ReportSchemeListQuery struct {
	PageQuery
	Scope string `json:"scope,omitempty" form:"scope"`
	Name  string `json:"name,omitempty" form:"name"`
}

// MnemonicRequest requests a mnemonic suggestion for a display name.
type MnemonicRequest struct {
	Name string `json:"name" binding:"required"`
}
