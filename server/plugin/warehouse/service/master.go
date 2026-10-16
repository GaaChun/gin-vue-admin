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

type MasterService struct {
	db     *gorm.DB
	images MaterialImageStorage
}

func NewMasterService(db *gorm.DB) *MasterService { return &MasterService{db: db} }
func NewMasterServiceWithImageStorage(db *gorm.DB, images MaterialImageStorage) *MasterService {
	return &MasterService{db: db, images: images}
}
func DefaultMasterService() *MasterService { return NewMasterService(global.GVA_DB) }

func required(value, field string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", domainError(ValidationFailed, "不能为空", field)
	}
	return value, nil
}
func requiredLimited(value, field string, limit int) (string, error) {
	value, err := required(value, field)
	if err != nil {
		return "", err
	}
	if utf8.RuneCountInString(value) > limit {
		return "", domainError(ValidationFailed, "长度超出限制", field)
	}
	return value, nil
}

func duplicate(field string) error { return domainError(DuplicateValue, "存在重复值", field) }
func notFound() error              { return NewDomainError(NotFound, "记录不存在") }
func dbError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := AsDomainError(err); ok {
		return err
	}
	return NewDomainError(ValidationFailed, "保存失败")
}

func (s *MasterService) CreateUnit(ctx context.Context, actor uint, in req.CreateUnitInput) (*model.Unit, error) {
	name, err := requiredLimited(in.Name, "name", 100)
	if err != nil {
		return nil, err
	}
	item := model.Unit{Name: name}
	ApplyCreateAudit(&item.BusinessModel, actor, time.Now())
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var n int64
		if e := tx.Model(&model.Unit{}).Where("LOWER(TRIM(name)) = ?", NormalizeUniqueKey(name)).Count(&n).Error; e != nil {
			return e
		}
		if n > 0 {
			return duplicate("name")
		}
		return tx.Create(&item).Error
	})
	return &item, dbError(err)
}
func (s *MasterService) UpdateUnit(ctx context.Context, actor uint, in req.UpdateUnitInput) (*model.Unit, error) {
	name, e := requiredLimited(in.Name, "name", 100)
	if e != nil {
		return nil, e
	}
	var item model.Unit
	e = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if x := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, in.ID).Error; x != nil {
			return x
		}
		if x := CheckVersion(item.Version, in.Version); x != nil {
			return x
		}
		var n int64
		if x := tx.Model(&model.Unit{}).Where("LOWER(TRIM(name)) = ? AND id <> ?", NormalizeUniqueKey(name), item.ID).Count(&n).Error; x != nil {
			return x
		}
		if n > 0 {
			return duplicate("name")
		}
		item.Name = name
		ApplyUpdateAudit(&item.BusinessModel, actor)
		return tx.Model(&item).Select("name", "updated_by", "version").Updates(&item).Error
	})
	if e == gorm.ErrRecordNotFound {
		return nil, notFound()
	}
	return &item, dbError(e)
}
func (s *MasterService) DeleteUnit(ctx context.Context, actor uint, in req.DeleteInput) error {
	return s.delete(ctx, in, &model.Unit{}, actor, func(tx *gorm.DB, id uint) error {
		var n int64
		if e := tx.Model(&model.MaterialUnit{}).Where("unit_id = ?", id).Count(&n).Error; e != nil {
			return e
		}
		if n > 0 {
			return NewDomainError(ReferenceInUse, "单位正在被引用")
		}
		return nil
	})
}

func (s *MasterService) CreateBrand(ctx context.Context, actor uint, in req.CreateBrandInput) (*model.Brand, error) {
	name, e := requiredLimited(in.Name, "name", 100)
	if e != nil {
		return nil, e
	}
	code, e := requiredLimited(in.Code, "code", 64)
	if e != nil {
		return nil, e
	}
	mn, e := requiredLimited(in.Mnemonic, "mnemonic", 100)
	if e != nil {
		return nil, e
	}
	item := model.Brand{Name: name, Code: code, Mnemonic: mn}
	ApplyCreateAudit(&item.BusinessModel, actor, time.Now())
	e = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if x := brandConflict(tx, 0, name, code); x != nil {
			return x
		}
		return tx.Create(&item).Error
	})
	return &item, dbError(e)
}
func (s *MasterService) UpdateBrand(ctx context.Context, actor uint, in req.UpdateBrandInput) (*model.Brand, error) {
	name, e := requiredLimited(in.Name, "name", 100)
	if e != nil {
		return nil, e
	}
	code, e := requiredLimited(in.Code, "code", 64)
	if e != nil {
		return nil, e
	}
	mn, e := requiredLimited(in.Mnemonic, "mnemonic", 100)
	if e != nil {
		return nil, e
	}
	var item model.Brand
	e = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if x := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, in.ID).Error; x != nil {
			return x
		}
		if x := CheckVersion(item.Version, in.Version); x != nil {
			return x
		}
		if x := brandConflict(tx, item.ID, name, code); x != nil {
			return x
		}
		item.Name = name
		item.Code = code
		item.Mnemonic = mn
		ApplyUpdateAudit(&item.BusinessModel, actor)
		return tx.Model(&item).Select("name", "code", "mnemonic", "updated_by", "version").Updates(&item).Error
	})
	if e == gorm.ErrRecordNotFound {
		return nil, notFound()
	}
	return &item, dbError(e)
}
func (s *MasterService) DeleteBrand(ctx context.Context, actor uint, in req.DeleteInput) error {
	return s.delete(ctx, in, &model.Brand{}, actor, func(tx *gorm.DB, id uint) error {
		var n int64
		if e := tx.Model(&model.Material{}).Where("brand_id = ?", id).Count(&n).Error; e != nil {
			return e
		}
		if n > 0 {
			return NewDomainError(ReferenceInUse, "品牌正在被物料引用")
		}
		return nil
	})
}
func brandConflict(tx *gorm.DB, excludeID uint, name, code string) error {
	var n int64
	if err := tx.Model(&model.Brand{}).Where("id <> ? AND LOWER(TRIM(name)) = ?", excludeID, NormalizeUniqueKey(name)).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return duplicate("name")
	}
	if err := tx.Model(&model.Brand{}).Where("id <> ? AND LOWER(TRIM(code)) = ?", excludeID, NormalizeUniqueKey(code)).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return duplicate("code")
	}
	return nil
}

func (s *MasterService) CreateCategory(ctx context.Context, actor uint, in req.CreateCategoryInput) (*model.Category, error) {
	item := model.Category{}
	if e := s.applyCategoryInput(ctx, s.db, &item, in.CategoryInput, 0); e != nil {
		return nil, e
	}
	ApplyCreateAudit(&item.BusinessModel, actor, time.Now())
	e := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := s.validateCategory(tx, &item, 0); e != nil {
			return e
		}
		return tx.Create(&item).Error
	})
	return &item, dbError(e)
}
func (s *MasterService) UpdateCategory(ctx context.Context, actor uint, in req.UpdateCategoryInput) (*model.Category, error) {
	var item model.Category
	e := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var nodes []model.Category
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Order("id").Find(&nodes).Error; e != nil {
			return e
		}
		for _, node := range nodes {
			if node.ID == in.ID {
				item = node
				break
			}
		}
		if item.ID == 0 {
			return gorm.ErrRecordNotFound
		}
		if e := CheckVersion(item.Version, in.Version); e != nil {
			return e
		}
		if e := s.applyCategoryInput(ctx, tx, &item, in.CategoryInput, item.ID); e != nil {
			return e
		}
		if e := s.lockAndValidateTree(tx, &item); e != nil {
			return e
		}
		ApplyUpdateAudit(&item.BusinessModel, actor)
		return tx.Model(&item).Select("name", "code", "mnemonic", "parent_id", "updated_by", "version").Updates(&item).Error
	})
	if e == gorm.ErrRecordNotFound {
		return nil, notFound()
	}
	return &item, dbError(e)
}
func (s *MasterService) applyCategoryInput(ctx context.Context, db *gorm.DB, item *model.Category, in req.CategoryInput, self uint) error {
	name, e := requiredLimited(in.Name, "name", 100)
	if e != nil {
		return e
	}
	code, e := requiredLimited(in.Code, "code", 64)
	if e != nil {
		return e
	}
	if strings.EqualFold(code, "ROOT") {
		return domainError(ValidationFailed, "类别编号不能为ROOT", "code")
	}
	mn, e := requiredLimited(in.Mnemonic, "mnemonic", 100)
	if e != nil {
		return e
	}
	item.Name = name
	item.Code = code
	item.Mnemonic = mn
	item.ParentID = in.ParentID
	return nil
}
func (s *MasterService) validateCategory(tx *gorm.DB, item *model.Category, self uint) error {
	if item.ParentID == self && self != 0 {
		return domainError(ValidationFailed, "不能选择自身或后代作为上级", "parentId")
	}
	if item.ParentID != 0 {
		var parent model.Category
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&parent, item.ParentID).Error; e != nil {
			if e == gorm.ErrRecordNotFound {
				return domainError(ValidationFailed, "上级类别不存在或已删除", "parentId")
			}
			return e
		}
	}
	if e := categoryConflict(tx, self, item.ParentID, item.Name, item.Code); e != nil {
		return e
	}
	return nil
}
func categoryConflict(tx *gorm.DB, excludeID, parentID uint, name, code string) error {
	var n int64
	if err := tx.Model(&model.Category{}).Where("id <> ? AND parent_id = ? AND LOWER(TRIM(name)) = ?", excludeID, parentID, NormalizeUniqueKey(name)).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return duplicate("name")
	}
	if err := tx.Model(&model.Category{}).Where("id <> ? AND LOWER(TRIM(code)) = ?", excludeID, NormalizeUniqueKey(code)).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return duplicate("code")
	}
	return nil
}
func (s *MasterService) lockAndValidateTree(tx *gorm.DB, item *model.Category) error {
	var nodes []model.Category
	if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Order("id").Find(&nodes).Error; e != nil {
		return e
	}
	if e := s.validateCategory(tx, item, item.ID); e != nil {
		return e
	}
	parents := map[uint]uint{}
	for _, node := range nodes {
		parents[node.ID] = node.ParentID
	}
	parents[item.ID] = item.ParentID
	for p := item.ParentID; p != 0; p = parents[p] {
		if p == item.ID {
			return domainError(ValidationFailed, "不能选择自身或后代作为上级", "parentId")
		}
	}
	return nil
}
func (s *MasterService) DeleteCategory(ctx context.Context, actor uint, in req.DeleteInput) error {
	return s.delete(ctx, in, &model.Category{}, actor, func(tx *gorm.DB, id uint) error {
		var n int64
		if e := tx.Model(&model.Category{}).Where("parent_id = ?", id).Count(&n).Error; e != nil {
			return e
		}
		if n > 0 {
			return NewDomainError(ReferenceInUse, "类别存在有效子类别")
		}
		if e := tx.Model(&model.Material{}).Where("category_id = ?", id).Count(&n).Error; e != nil {
			return e
		}
		if n > 0 {
			return NewDomainError(ReferenceInUse, "类别正在被物料引用")
		}
		return nil
	})
}
func (s *MasterService) delete(ctx context.Context, in req.DeleteInput, target interface{}, actor uint, refs func(*gorm.DB, uint) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(target, in.ID).Error; e != nil {
			if e == gorm.ErrRecordNotFound {
				return notFound()
			}
			return e
		}
		var version uint64
		switch x := target.(type) {
		case *model.Unit:
			version = x.Version
			ApplyUpdateAudit(&x.BusinessModel, actor)
		case *model.Brand:
			version = x.Version
			ApplyUpdateAudit(&x.BusinessModel, actor)
		case *model.Category:
			version = x.Version
			ApplyUpdateAudit(&x.BusinessModel, actor)
		}
		if e := CheckVersion(version, in.Version); e != nil {
			return e
		}
		if e := refs(tx, in.ID); e != nil {
			return e
		}
		if e := tx.Model(target).Select("updated_by", "version").Updates(target).Error; e != nil {
			return e
		}
		return tx.Delete(target).Error
	})
}

func (s *MasterService) ListUnits(ctx context.Context, q req.UnitListQuery) (resp.Page[model.Unit], error) {
	q.PageQuery = q.PageQuery.WithDefaults()
	var r resp.Page[model.Unit]
	db := s.db.WithContext(ctx).Model(&model.Unit{})
	if q.Name != "" {
		db = db.Where("name LIKE ?", "%"+strings.TrimSpace(q.Name)+"%")
	}
	e := db.Count(&r.Total).Error
	if e == nil {
		e = db.Order("id DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&r.List).Error
	}
	r.Page = q.Page
	r.PageSize = q.PageSize
	return r, dbError(e)
}
func (s *MasterService) ListBrands(ctx context.Context, q req.BrandListQuery) (resp.Page[model.Brand], error) {
	q.PageQuery = q.PageQuery.WithDefaults()
	var r resp.Page[model.Brand]
	db := s.db.WithContext(ctx).Model(&model.Brand{})
	if q.Keyword != "" {
		db = db.Where("name LIKE ? OR code LIKE ? OR mnemonic LIKE ?", "%"+q.Keyword+"%", "%"+q.Keyword+"%", "%"+q.Keyword+"%")
	}
	e := db.Count(&r.Total).Error
	if e == nil {
		e = db.Order("id DESC").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&r.List).Error
	}
	r.Page = q.Page
	r.PageSize = q.PageSize
	return r, dbError(e)
}
func (s *MasterService) ListCategories(ctx context.Context, q req.CategoryListQuery) (resp.Page[model.Category], error) {
	q.PageQuery = q.PageQuery.WithDefaults()
	var r resp.Page[model.Category]
	db := s.db.WithContext(ctx).Model(&model.Category{})
	if q.ParentID != nil {
		db = db.Where("parent_id = ?", *q.ParentID)
	}
	if q.Keyword != "" {
		db = db.Where("name LIKE ? OR code LIKE ? OR mnemonic LIKE ?", "%"+q.Keyword+"%", "%"+q.Keyword+"%", "%"+q.Keyword+"%")
	}
	e := db.Count(&r.Total).Error
	if e == nil {
		e = db.Order("parent_id,id").Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&r.List).Error
	}
	r.Page = q.Page
	r.PageSize = q.PageSize
	return r, dbError(e)
}
func (s *MasterService) UnitDetail(ctx context.Context, id uint) (*model.Unit, error) {
	var v model.Unit
	e := s.db.WithContext(ctx).First(&v, id).Error
	if e == gorm.ErrRecordNotFound {
		return nil, notFound()
	}
	return &v, dbError(e)
}
func (s *MasterService) BrandDetail(ctx context.Context, id uint) (*model.Brand, error) {
	var v model.Brand
	e := s.db.WithContext(ctx).First(&v, id).Error
	if e == gorm.ErrRecordNotFound {
		return nil, notFound()
	}
	return &v, dbError(e)
}
func (s *MasterService) CategoryDetail(ctx context.Context, id uint) (*model.Category, error) {
	var v model.Category
	e := s.db.WithContext(ctx).First(&v, id).Error
	if e == gorm.ErrRecordNotFound {
		return nil, notFound()
	}
	return &v, dbError(e)
}
func (s *MasterService) CategoryTree(ctx context.Context) ([]CategoryTree, error) {
	var items []model.Category
	if e := s.db.WithContext(ctx).Order("parent_id,id").Find(&items).Error; e != nil {
		return nil, dbError(e)
	}
	return assembleCategoryTree(items), nil
}
func (s *MasterService) CategoryDescendants(ctx context.Context, parent uint) ([]uint, error) {
	var items []model.Category
	if e := s.db.WithContext(ctx).Find(&items).Error; e != nil {
		return nil, dbError(e)
	}
	children := map[uint][]uint{}
	for _, v := range items {
		children[v.ParentID] = append(children[v.ParentID], v.ID)
	}
	out := []uint{}
	var walk func(uint)
	walk = func(id uint) {
		out = append(out, id)
		for _, child := range children[id] {
			walk(child)
		}
	}
	walk(parent)
	return out, nil
}

type CategoryTree struct {
	model.Category
	Children []CategoryTree `json:"children"`
}

func assembleCategoryTree(items []model.Category) []CategoryTree {
	children := map[uint][]model.Category{}
	for _, item := range items {
		children[item.ParentID] = append(children[item.ParentID], item)
	}
	var build func(uint) []CategoryTree
	build = func(parent uint) []CategoryTree {
		out := make([]CategoryTree, 0, len(children[parent]))
		for _, item := range children[parent] {
			out = append(out, CategoryTree{Category: item, Children: build(item.ID)})
		}
		return out
	}
	return build(0)
}
