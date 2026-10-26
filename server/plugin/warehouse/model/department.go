package model

// Department is independent master data. Delivery notes retain their name snapshots.
type Department struct {
	BusinessModel
	Name     string `json:"name" gorm:"column:name;type:varchar(100);not null;comment:部门名称"`
	Mnemonic string `json:"mnemonic" gorm:"column:mnemonic;type:varchar(100);not null;default:'';comment:多个助记符"`
}

func (Department) TableName() string { return "wh_departments" }

// DepartmentImport records the one-time historical import, including an empty history.
// It deliberately survives master-data renames and deletions.
type DepartmentImport struct {
	Key string `gorm:"column:migration_key;type:varchar(100);primaryKey"`
}

func (DepartmentImport) TableName() string { return "wh_department_imports" }
