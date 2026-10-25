package model

// MasterCodeCounter keeps independent, non-resetting sequences for coded master resources.
// Resource stays unique across history, just as delivery number counters do.
type MasterCodeCounter struct {
	BusinessModel
	Resource  string `json:"resource" gorm:"column:resource;type:varchar(32);not null;uniqueIndex:uk_wh_master_code_counters_resource"`
	LastValue uint   `json:"lastValue" gorm:"column:last_value;not null;default:0"`
}

func (MasterCodeCounter) TableName() string { return "wh_master_code_counters" }
