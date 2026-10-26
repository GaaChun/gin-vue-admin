package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	resp "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/response"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DepartmentService struct{ db *gorm.DB }

func NewDepartmentService(db *gorm.DB) *DepartmentService { return &DepartmentService{db: db} }
func DefaultDepartmentService() *DepartmentService        { return NewDepartmentService(global.GVA_DB) }

func departmentInput(in req.DepartmentInput) (string, string, error) {
	name, err := requiredLimited(in.Name, "name", 100)
	if err != nil {
		return "", "", err
	}
	mnemonic := strings.TrimSpace(in.Mnemonic)
	if utf8.RuneCountInString(mnemonic) > 100 {
		return "", "", domainError(ValidationFailed, "长度超出限制", "mnemonic")
	}
	return name, mnemonic, nil
}

func departmentConflict(tx *gorm.DB, excludeID uint, name string) error {
	var count int64
	if err := tx.Model(&model.Department{}).Where("id <> ? AND LOWER(TRIM(name)) = ?", excludeID, NormalizeUniqueKey(name)).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return duplicate("name")
	}
	return nil
}

func (s *DepartmentService) Create(ctx context.Context, actor uint, in req.CreateDepartmentInput) (*model.Department, error) {
	name, mnemonic, err := departmentInput(in.DepartmentInput)
	if err != nil {
		return nil, err
	}
	item := model.Department{Name: name, Mnemonic: mnemonic}
	ApplyCreateAudit(&item.BusinessModel, actor, time.Now())
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := departmentConflict(tx, 0, name); err != nil {
			return err
		}
		return tx.Create(&item).Error
	})
	return &item, dbError(err)
}

func (s *DepartmentService) Update(ctx context.Context, actor uint, in req.UpdateDepartmentInput) (*model.Department, error) {
	name, mnemonic, err := departmentInput(in.DepartmentInput)
	if err != nil {
		return nil, err
	}
	var item model.Department
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, in.ID).Error; err != nil {
			return err
		}
		if err := CheckVersion(item.Version, in.Version); err != nil {
			return err
		}
		if err := departmentConflict(tx, item.ID, name); err != nil {
			return err
		}
		item.Name, item.Mnemonic = name, mnemonic
		ApplyUpdateAudit(&item.BusinessModel, actor)
		return tx.Model(&item).Select("name", "mnemonic", "updated_by", "version").Updates(&item).Error
	})
	if err == gorm.ErrRecordNotFound {
		return nil, notFound()
	}
	return &item, dbError(err)
}

func (s *DepartmentService) Delete(ctx context.Context, actor uint, in req.DeleteInput) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item model.Department
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, in.ID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return notFound()
			}
			return err
		}
		if err := CheckVersion(item.Version, in.Version); err != nil {
			return err
		}
		ApplyDeleteAudit(&item.BusinessModel, actor)
		if err := tx.Model(&item).Select("updated_by", "version", "deleted_by", "is_deleted").Updates(&item).Error; err != nil {
			return err
		}
		return tx.Delete(&item).Error
	})
}

func (s *DepartmentService) Detail(ctx context.Context, id uint) (*model.Department, error) {
	var item model.Department
	err := s.db.WithContext(ctx).First(&item, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, notFound()
	}
	return &item, dbError(err)
}

func (s *DepartmentService) List(ctx context.Context, q req.DepartmentListQuery) (resp.Page[model.Department], error) {
	q.PageQuery = q.PageQuery.WithDefaults()
	result := resp.Page[model.Department]{List: []model.Department{}, Page: q.Page, PageSize: q.PageSize}
	db := s.db.WithContext(ctx).Model(&model.Department{})
	if keyword := strings.TrimSpace(q.Keyword); keyword != "" {
		// Each mnemonic remains searchable within the existing delimited string.
		db = db.Where("(name LIKE ? OR mnemonic LIKE ?)", "%"+keyword+"%", "%"+keyword+"%")
	}
	columns := auditSortColumns("wh_departments")
	columns["name"], columns["mnemonic"] = "wh_departments.name", "wh_departments.mnemonic"
	order, err := resolveMasterOrder(q.SortQuery, columns)
	if err != nil {
		return result, err
	}
	if err = db.Count(&result.Total).Error; err == nil {
		err = db.Order(order).Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&result.List).Error
	}
	return result, dbError(err)
}

// registerDeliveryDepartment is part of the delivery transaction, not a separately
// authorized master write: free-text input remains supported without extra user actions.
func registerDeliveryDepartment(tx *gorm.DB, actor uint, name string, now time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	name, err := requiredLimited(name, "department", 100)
	if err != nil {
		return err
	}
	var item model.Department
	if err := tx.Where("LOWER(TRIM(name)) = ?", NormalizeUniqueKey(name)).First(&item).Error; err == nil {
		return nil
	} else if err != gorm.ErrRecordNotFound {
		return err
	}
	item.Name = name
	ApplyCreateAudit(&item.BusinessModel, actor, now)
	// The generated active-name key serializes concurrent registrations.
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&item).Error
}

// ImportHistoricalDepartments imports existing live delivery snapshots exactly once.
// Re-running startup must not recreate departments intentionally renamed/deleted later.
func ImportHistoricalDepartments(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		var previous model.DepartmentImport
		if err := tx.Where("migration_key = ?", "delivery-departments-v1").First(&previous).Error; err == nil {
			return nil
		} else if err != gorm.ErrRecordNotFound {
			return err
		}
		marker := model.DepartmentImport{Key: "delivery-departments-v1"}
		claimed := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&marker)
		if claimed.Error != nil {
			return claimed.Error
		}
		if claimed.RowsAffected == 0 {
			return nil
		}
		var names []string
		if err := tx.Model(&model.DeliveryNote{}).Distinct("department").Where("department <> ''").Pluck("department", &names).Error; err != nil {
			return err
		}
		for _, name := range names {
			if err := registerDeliveryDepartment(tx, 0, name, time.Now()); err != nil {
				return err
			}
		}
		return nil
	})
}
