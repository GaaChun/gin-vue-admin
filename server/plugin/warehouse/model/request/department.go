package request

type DepartmentInput struct {
	Name     string `json:"name" binding:"required"`
	Mnemonic string `json:"mnemonic"`
}

type CreateDepartmentInput struct{ DepartmentInput }

type UpdateDepartmentInput struct {
	VersionedID
	DepartmentInput
}

type DepartmentListQuery struct {
	PageQuery
	SortQuery
	Keyword string `json:"keyword" form:"keyword"`
}
