package model

import (
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// BusinessModel carries server-owned audit and optimistic-concurrency state.
// Clients never supply these fields: audit identity must reflect the authorized actor.
type BusinessModel struct {
	global.GVA_MODEL
	CreatedBy  uint       `json:"createdBy" gorm:"column:created_by;not null;index;comment:创建人ID"`
	UpdatedBy  uint       `json:"updatedBy" gorm:"column:updated_by;not null;index;comment:最后修改人ID"`
	DeletedBy  *uint      `json:"deletedBy,omitempty" gorm:"column:deleted_by;index;comment:删除人ID"`
	IsDeleted  bool       `json:"isDeleted" gorm:"column:is_deleted;not null;default:false;comment:逻辑删除标识"`
	ReviewedBy *uint      `json:"reviewedBy,omitempty" gorm:"column:reviewed_by;index;comment:审核人ID"`
	ReviewedAt *time.Time `json:"reviewedAt,omitempty" gorm:"column:reviewed_at;index;comment:审核时间"`
	Version    uint64     `json:"version" gorm:"column:version;not null;default:1;comment:乐观锁版本"`
}
