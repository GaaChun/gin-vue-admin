package request

// MasterExportQuery exports all rows matching the list filters, without pagination.
type MasterExportQuery struct {
	Name       string `form:"name"`
	Keyword    string `form:"keyword"`
	BrandID    *uint  `form:"brandId"`
	CategoryID *uint  `form:"categoryId"`
	ParentID   *uint  `form:"parentId"`
}
