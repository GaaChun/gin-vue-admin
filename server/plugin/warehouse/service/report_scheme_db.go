package service

import (
	"context"
	"strconv"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SharedSchemePermission is an actual publish endpoint, separately grantable in Casbin.
const SharedSchemePermission = "/warehouse/schemes/publish"

type ReportSchemeService struct{ db *gorm.DB }

func NewReportSchemeService(db *gorm.DB) *ReportSchemeService { return &ReportSchemeService{db: db} }
func DefaultReportSchemeService() *ReportSchemeService        { return NewReportSchemeService(global.GVA_DB) }

func schemeAllowed(role uint, path, method string) bool {
	if role == 0 {
		return false
	}
	enforcer := utils.GetCasbin()
	if enforcer == nil {
		return false
	}
	ok, err := enforcer.Enforce(strconv.FormatUint(uint64(role), 10), path, method)
	return err == nil && ok
}
func schemeAccess(role uint, action string) SchemeAccess {
	method := map[string]string{"create": "POST", "update": "PUT", "delete": "DELETE"}[action]
	return SchemeAccess{
		CanQuery:          schemeAllowed(role, "/warehouse/reconciliation/query", "POST"),
		CanManagePersonal: method != "" && schemeAllowed(role, "/warehouse/schemes/"+action, method),
		CanManageShared:   schemeAllowed(role, SharedSchemePermission, "POST"),
	}
}

type SchemePermissions struct {
	Create bool `json:"create"`
	Update bool `json:"update"`
	Delete bool `json:"delete"`
	Shared bool `json:"shared"`
}
type ReportSchemePage struct {
	List        []model.ReportScheme `json:"list"`
	Page        int                  `json:"page"`
	PageSize    int                  `json:"pageSize"`
	Total       int64                `json:"total"`
	Permissions SchemePermissions    `json:"permissions"`
}

func (s *ReportSchemeService) List(ctx context.Context, actor, role uint, query req.ReportSchemeListQuery) (ReportSchemePage, error) {
	if !schemeAccess(role, "").CanQuery {
		return ReportSchemePage{}, domainError(Forbidden, "缺少报表查询权限", "scope")
	}
	page := query.PageQuery.WithDefaults()
	result := ReportSchemePage{List: []model.ReportScheme{}, Page: page.Page, PageSize: page.PageSize, Permissions: SchemePermissions{schemeAccess(role, "create").CanManagePersonal, schemeAccess(role, "update").CanManagePersonal, schemeAccess(role, "delete").CanManagePersonal, schemeAccess(role, "").CanManageShared}}
	db := s.db.WithContext(ctx).Model(&model.ReportScheme{}).Where("scope = 'shared' OR (scope = 'personal' AND owner_id = ?)", actor)
	if query.Scope != "" {
		db = db.Where("scope = ?", query.Scope)
	}
	if query.Name != "" {
		db = db.Where("LOWER(name) LIKE ?", "%"+NormalizeUniqueKey(query.Name)+"%")
	}
	if err := db.Count(&result.Total).Error; err != nil {
		return result, err
	}
	err := db.Order("scope, name, id").Offset((page.Page - 1) * page.PageSize).Limit(page.PageSize).Find(&result.List).Error
	return result, err
}
func (s *ReportSchemeService) Detail(ctx context.Context, actor, role, id uint) (model.ReportScheme, error) {
	var item model.ReportScheme
	if !schemeAccess(role, "").CanQuery {
		return item, domainError(Forbidden, "缺少报表查询权限", "scope")
	}
	err := s.db.WithContext(ctx).Where("id = ? AND (scope = 'shared' OR (scope = 'personal' AND owner_id = ?))", id, actor).First(&item).Error
	if err == gorm.ErrRecordNotFound {
		err = notFound()
	}
	return item, err
}
func (s *ReportSchemeService) Create(ctx context.Context, actor, role uint, in req.CreateReportSchemeInput) (model.ReportScheme, error) {
	access := schemeAccess(role, "create")
	if !access.CanQuery {
		return model.ReportScheme{}, domainError(Forbidden, "缺少报表查询权限", "scope")
	}
	name, columns, err := prepareScheme(in.ReportSchemeInput, access)
	if err != nil {
		return model.ReportScheme{}, err
	}
	item := model.ReportScheme{Name: name, Scope: in.Scope, OwnerID: actor, ColumnsJSON: columns}
	ApplyCreateAudit(&item.BusinessModel, actor, time.Now())
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := checkSchemeName(tx, item, 0); err != nil {
			return err
		}
		return tx.Create(&item).Error
	})
	return item, err
}
func checkSchemeName(tx *gorm.DB, item model.ReportScheme, exclude uint) error {
	db := tx.Model(&model.ReportScheme{}).Where("id <> ? AND scope = ? AND LOWER(TRIM(name)) = ?", exclude, item.Scope, NormalizeUniqueKey(item.Name))
	if item.Scope == "personal" {
		db = db.Where("owner_id = ?", item.OwnerID)
	}
	var count int64
	if err := db.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return domainError(DuplicateValue, "方案名称已存在", "name")
	}
	return nil
}
func checkSchemeMutation(item model.ReportScheme, actor uint, access SchemeAccess) error {
	if !access.CanQuery {
		return domainError(Forbidden, "缺少报表查询权限", "scope")
	}
	if item.Scope == "personal" {
		if item.OwnerID != actor {
			return notFound()
		}
		if !access.CanManagePersonal {
			return domainError(Forbidden, "缺少个人方案维护权限", "scope")
		}
	} else if !access.CanManageShared {
		return domainError(Forbidden, "缺少共享方案管理权限", "scope")
	}
	return nil
}
func (s *ReportSchemeService) Update(ctx context.Context, actor, role uint, in req.UpdateReportSchemeInput) (model.ReportScheme, error) {
	var item model.ReportScheme
	access := schemeAccess(role, "update")
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, in.ID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return notFound()
			}
			return err
		}
		if err := checkSchemeMutation(item, actor, access); err != nil {
			return err
		}
		if err := CheckVersion(item.Version, in.Version); err != nil {
			return err
		}
		if item.Scope == "shared" && in.Scope == "personal" && item.OwnerID != actor {
			return domainError(Forbidden, "仅发布者可将共享方案转为个人，其他用户请另存为", "scope")
		}
		name, columns, err := prepareScheme(in.ReportSchemeInput, access)
		if err != nil {
			return err
		}
		item.Name, item.Scope, item.ColumnsJSON = name, in.Scope, columns
		if err := checkSchemeName(tx, item, item.ID); err != nil {
			return err
		}
		ApplyUpdateAudit(&item.BusinessModel, actor)
		return tx.Model(&item).Updates(map[string]interface{}{"name": item.Name, "scope": item.Scope, "columns_json": item.ColumnsJSON, "updated_by": item.UpdatedBy, "version": item.Version}).Error
	})
	return item, err
}
func (s *ReportSchemeService) Delete(ctx context.Context, actor, role uint, in req.DeleteInput) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item model.ReportScheme
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, in.ID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return notFound()
			}
			return err
		}
		if err := checkSchemeMutation(item, actor, schemeAccess(role, "delete")); err != nil {
			return err
		}
		if err := CheckVersion(item.Version, in.Version); err != nil {
			return err
		}
		ApplyDeleteAudit(&item.BusinessModel, actor)
		if err := tx.Model(&item).Updates(map[string]interface{}{
			"updated_by": item.UpdatedBy, "version": item.Version,
			"deleted_by": item.DeletedBy, "is_deleted": item.IsDeleted,
		}).Error; err != nil {
			return err
		}
		return tx.Delete(&item).Error
	})
}
