package service

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// MasterExcelState is the rollback boundary used by workbook tests. It models
// business outcomes, not SQL or database locking.
type MasterExcelState struct {
	nextID       uint
	units        map[uint]model.Unit
	brands       map[uint]model.Brand
	categories   map[uint]model.Category
	materials    map[uint]model.Material
	bindings     map[uint]model.MaterialUnit
	images       map[uint]struct{}
	bindingRefs  map[uint]int
	created      int
	updated      int
}

func NewMasterExcelState() *MasterExcelState {
	return &MasterExcelState{
		nextID: 1,
		units: map[uint]model.Unit{}, brands: map[uint]model.Brand{}, categories: map[uint]model.Category{},
		materials: map[uint]model.Material{}, bindings: map[uint]model.MaterialUnit{}, images: map[uint]struct{}{}, bindingRefs: map[uint]int{},
	}
}

func (s *MasterExcelState) clone() *MasterExcelState {
	next := NewMasterExcelState()
	next.nextID = s.nextID
	for id, item := range s.units { next.units[id] = item }
	for id, item := range s.brands { next.brands[id] = item }
	for id, item := range s.categories { next.categories[id] = item }
	for id, item := range s.materials { next.materials[id] = item }
	for id, item := range s.bindings { next.bindings[id] = item }
	for id := range s.images { next.images[id] = struct{}{} }
	for id, count := range s.bindingRefs { next.bindingRefs[id] = count }
	return next
}

func (s *MasterExcelState) AddUnit(name string, actor uint) model.Unit {
	item := model.Unit{Name: name}
	item.ID = s.allocate()
	ApplyCreateAudit(&item.BusinessModel, actor, time.Now())
	s.units[item.ID] = item
	return item
}

func (s *MasterExcelState) AddBrand(code, name, mnemonic string, actor uint) model.Brand {
	item := model.Brand{Code: code, Name: name, Mnemonic: mnemonic}
	item.ID = s.allocate()
	ApplyCreateAudit(&item.BusinessModel, actor, time.Now())
	s.brands[item.ID] = item
	return item
}

func (s *MasterExcelState) AddCategory(code, name, mnemonic string, parent uint, actor uint) model.Category {
	item := model.Category{Code: code, Name: name, Mnemonic: mnemonic, ParentID: parent}
	item.ID = s.allocate()
	ApplyCreateAudit(&item.BusinessModel, actor, time.Now())
	s.categories[item.ID] = item
	return item
}

func (s *MasterExcelState) AddMaterial(item model.Material, bindings []model.MaterialUnit, actor uint) model.Material {
	item.ID = s.allocate()
	ApplyCreateAudit(&item.BusinessModel, actor, time.Now())
	s.materials[item.ID] = item
	for _, binding := range bindings {
		binding.ID = s.allocate()
		binding.MaterialID = item.ID
		ApplyCreateAudit(&binding.BusinessModel, actor, time.Now())
		s.bindings[binding.ID] = binding
	}
	return item
}

func (s *MasterExcelState) RegisterImage(fileID uint) { s.images[fileID] = struct{}{} }
func (s *MasterExcelState) ReferenceBinding(id uint)   { s.bindingRefs[id]++ }

func (s *MasterExcelState) Import(kind string, contents []byte, actor uint) (ImportResult, error) {
	parsed, err := parseMasterWorkbook(kind, contents)
	if err != nil {
		return ImportResult{}, err
	}
	working := s.clone()
	switch kind {
	case "units":
		err = working.applyUnits(parsed.units, actor)
	case "brands":
		err = working.applyBrands(parsed.brands, actor)
	case "categories":
		err = working.applyCategories(parsed.categories, actor)
	case "materials":
		err = working.applyMaterials(parsed, actor)
	default:
		err = domainError(ValidationFailed, "未知资料导入类型", "kind")
	}
	if err != nil {
		return ImportResult{}, err
	}
	created, updated := working.created, working.updated
	working.created, working.updated = s.created, s.updated
	*s = *working
	return ImportResult{CreatedCount: created, UpdatedCount: updated}, nil
}

type ImportResult struct {
	CreatedCount int `json:"createdCount"`
	UpdatedCount int `json:"updatedCount"`
}

func (s *MasterExcelState) applyUnits(rows []unitExcelRow, actor uint) error {
	for _, row := range rows {
		name, err := requiredLimited(row.name, "单位名称", 100)
		if err != nil {
			return located(err, unitSheetName, row.row, "单位名称")
		}
		if current, found := s.unitByName(name); found {
			if err := CheckVersion(current.Version, row.version); err != nil {
				return located(err, unitSheetName, row.row, "版本号")
			}
			ApplyUpdateAudit(&current.BusinessModel, actor)
			s.units[current.ID] = current
			s.updated++
			continue
		}
		if row.version != nil {
			return located(domainError(ValidationFailed, "新增记录不能指定版本号", "版本号"), unitSheetName, row.row, "版本号")
		}
		s.AddUnit(name, actor)
		s.created++
	}
	return nil
}

func (s *MasterExcelState) applyBrands(rows []brandExcelRow, actor uint) error {
	seen := map[string]int{}
	for _, row := range rows {
		code, err := requiredLimited(row.code, "编号", 64)
		if err != nil {
			return located(err, brandSheetName, row.row, "编号")
		}
		name, err := requiredLimited(row.name, "客户名称", 100)
		if err != nil {
			return located(err, brandSheetName, row.row, "客户名称")
		}
		mnemonic, err := requiredLimited(row.mnemonic, "助记符", 100)
		if err != nil {
			return located(err, brandSheetName, row.row, "助记符")
		}
		key := NormalizeUniqueKey(code)
		if previous, exists := seen[key]; exists {
			return located(domainError(DuplicateValue, "与第"+strconv.Itoa(previous)+"行重复", "编号"), brandSheetName, row.row, "编号")
		}
		seen[key] = row.row
		if current, found := s.brandByCode(code); found {
			if err := CheckVersion(current.Version, row.version); err != nil {
				return located(err, brandSheetName, row.row, "版本号")
			}
			if _, duplicateName := s.brandByName(name); duplicateName && !strings.EqualFold(strings.TrimSpace(current.Name), name) {
				return located(duplicate("客户名称"), brandSheetName, row.row, "客户名称")
			}
			current.Name, current.Mnemonic = name, mnemonic
			ApplyUpdateAudit(&current.BusinessModel, actor)
			s.brands[current.ID] = current
			s.updated++
			continue
		}
		if row.version != nil {
			return located(domainError(ValidationFailed, "新增记录不能指定版本号", "版本号"), brandSheetName, row.row, "版本号")
		}
		if _, exists := s.brandByName(name); exists {
			return located(duplicate("客户名称"), brandSheetName, row.row, "客户名称")
		}
		s.AddBrand(code, name, mnemonic, actor)
		s.created++
	}
	return nil
}

func (s *MasterExcelState) applyCategories(rows []categoryExcelRow, actor uint) error {
	drafts := make([]CategoryDraft, 0, len(rows))
	byCode := map[string]categoryExcelRow{}
	for _, row := range rows {
		drafts = append(drafts, CategoryDraft{Code: row.code, ParentCode: row.parentCode})
		byCode[NormalizeUniqueKey(row.code)] = row
	}
	order, err := PrepareCategoryParentOrder(drafts)
	if err != nil {
		return located(err, categorySheetName, categoryProblemRow(rows, err), categoryProblemColumn(err))
	}
	resolved := map[string]uint{}
	for id, item := range s.categories {
		resolved[NormalizeUniqueKey(item.Code)] = id
	}
	for _, code := range order {
		row := byCode[code]
		name, err := requiredLimited(row.name, "类别名称", 100)
		if err != nil {
			return located(err, categorySheetName, row.row, "类别名称")
		}
		mnemonic, err := requiredLimited(row.mnemonic, "助记符", 100)
		if err != nil {
			return located(err, categorySheetName, row.row, "助记符")
		}
		parent, err := s.resolveCategoryParent(row, resolved)
		if err != nil {
			return err
		}
		if current, found := s.categoryByCode(row.code); found {
			if err := CheckVersion(current.Version, row.version); err != nil {
				return located(err, categorySheetName, row.row, "版本号")
			}
			if !CategorySiblingNameAvailable(s.categoryNames(current.ID), parent, current.ID, name) {
				return located(duplicate("类别名称"), categorySheetName, row.row, "类别名称")
			}
			current.Name, current.Mnemonic, current.ParentID = name, mnemonic, parent
			ApplyUpdateAudit(&current.BusinessModel, actor)
			s.categories[current.ID] = current
			resolved[code] = current.ID
			s.updated++
			continue
		}
		if row.version != nil {
			return located(domainError(ValidationFailed, "新增记录不能指定版本号", "版本号"), categorySheetName, row.row, "版本号")
		}
		if !CategorySiblingNameAvailable(s.categoryNames(0), parent, 0, name) {
			return located(duplicate("类别名称"), categorySheetName, row.row, "类别名称")
		}
		created := s.AddCategory(strings.TrimSpace(row.code), name, mnemonic, parent, actor)
		resolved[code] = created.ID
		s.created++
	}
	return nil
}

func (s *MasterExcelState) resolveCategoryParent(row categoryExcelRow, resolved map[string]uint) (uint, error) {
	parentCode := NormalizeUniqueKey(row.parentCode)
	if parentCode == "root" {
		return 0, nil
	}
	parent, ok := resolved[parentCode]
	if !ok {
		return 0, located(domainError(ValidationFailed, "上级类别不存在", "上级类别编号"), categorySheetName, row.row, "上级类别编号")
	}
	if parent == s.categoryByCodeID(row.code) {
		return 0, located(domainError(ValidationFailed, "不能选择自身或后代作为上级", "上级类别编号"), categorySheetName, row.row, "上级类别编号")
	}
	return parent, nil
}

func (s *MasterExcelState) applyMaterials(parsed masterWorkbook, actor uint) error {
	grouped := map[string][]materialUnitExcelRow{}
	for _, row := range parsed.materialUnits {
		grouped[NormalizeUniqueKey(row.material)] = append(grouped[NormalizeUniqueKey(row.material)], row)
	}
	for _, row := range parsed.materials {
		prepared, err := s.prepareNamedUnits(grouped[NormalizeUniqueKey(row.code)])
		if err != nil {
			return err
		}
		item, err := s.prepareMaterialExcel(row)
		if err != nil {
			return err
		}
		if current, found := s.materialByCode(row.code); found {
			if err := CheckVersion(current.Version, row.version); err != nil {
				return located(err, materialSheetName, row.row, "版本号")
			}
			if err := s.replaceMaterialBindings(current.ID, prepared, actor); err != nil {
				return err
			}
			preserve := current.BusinessModel
			current = item
			current.ID, current.BusinessModel = preserve.ID, preserve
			ApplyUpdateAudit(&current.BusinessModel, actor)
			s.materials[current.ID] = current
			s.updated++
			continue
		}
		if row.version != nil {
			return located(domainError(ValidationFailed, "新增记录不能指定版本号", "版本号"), materialSheetName, row.row, "版本号")
		}
		s.AddMaterial(item, materialCommandsToModels(prepared), actor)
		s.created++
	}
	return nil
}

func (s *MasterExcelState) prepareMaterialExcel(row materialExcelRow) (model.Material, error) {
	name, err := requiredLimited(row.name, "物料名称", 200)
	if err != nil {
		return model.Material{}, located(err, materialSheetName, row.row, "物料名称")
	}
	mnemonic, err := requiredLimited(row.mnemonic, "助记符", 100)
	if err != nil {
		return model.Material{}, located(err, materialSheetName, row.row, "助记符")
	}
	code, err := requiredLimited(row.code, "编号", 64)
	if err != nil {
		return model.Material{}, located(err, materialSheetName, row.row, "编号")
	}
	var brandID, categoryID uint
	if row.brandCode != "" {
		brand, found := s.brandByCode(row.brandCode)
		if !found {
			return model.Material{}, located(domainError(ValidationFailed, "客户不存在", "brandId"), materialSheetName, row.row, "客户编号")
		}
		brandID = brand.ID
	}
	if row.categoryCode != "" {
		category, found := s.categoryByCode(row.categoryCode)
		if !found {
			return model.Material{}, located(domainError(ValidationFailed, "类别不存在", "categoryId"), materialSheetName, row.row, "类别编号")
		}
		categoryID = category.ID
	}
	if row.imageFileID != nil {
		if _, ok := s.images[*row.imageFileID]; !ok {
			return model.Material{}, located(domainError(ValidationFailed, "图片未通过物料图片校验", "图片文件ID"), materialSheetName, row.row, "图片文件ID")
		}
	}
	item := model.Material{Name: name, Mnemonic: mnemonic, Code: code, BrandID: brandID, CategoryID: categoryID, ImageFileID: row.imageFileID}
	if row.specification != "" {
		item.Specification = stringPointer(row.specification)
	}
	if row.alias != "" {
		item.Alias = stringPointer(row.alias)
	}
	return item, nil
}

type namedMaterialUnit struct {
	unitID  uint
	factor  decimal.Decimal
	isBase  bool
	enabled bool
	row     int
}

func (s *MasterExcelState) prepareNamedUnits(rows []materialUnitExcelRow) ([]namedMaterialUnit, error) {
	result := make([]namedMaterialUnit, 0, len(rows))
	baseCount := 0
	for _, row := range rows {
		unit, found := s.unitByName(row.unitName)
		if !found {
			return nil, located(domainError(ValidationFailed, "单位不存在", "单位名称"), materialUnitSheetName, row.row, "单位名称")
		}
		if row.isBase {
			if !row.enabled || !row.factor.Equal(decimal.NewFromInt(1)) {
				return nil, located(domainError(ValidationFailed, "基本单位必须启用且系数为1", "是否基本单位"), materialUnitSheetName, row.row, "是否基本单位")
			}
			baseCount++
		}
		result = append(result, namedMaterialUnit{unitID: unit.ID, factor: row.factor, isBase: row.isBase, enabled: row.enabled, row: row.row})
	}
	if baseCount != 1 {
		row := 2
		if len(rows) > 0 {
			row = rows[0].row
		}
		return nil, located(domainError(ValidationFailed, "必须且只能有一个启用基本单位", "是否基本单位"), materialUnitSheetName, row, "是否基本单位")
	}
	return result, nil
}

func (s *MasterExcelState) replaceMaterialBindings(materialID uint, commands []namedMaterialUnit, actor uint) error {
	existing := map[uint]model.MaterialUnit{}
	for _, binding := range s.bindings {
		if binding.MaterialID == materialID {
			existing[binding.UnitID] = binding
		}
	}
	wanted := map[uint]namedMaterialUnit{}
	oldBase, newBase := uint(0), uint(0)
	for _, binding := range existing {
		if binding.IsBase && binding.Enabled {
			oldBase = binding.UnitID
		}
	}
	for _, command := range commands {
		wanted[command.unitID] = command
		if command.isBase {
			newBase = command.unitID
		}
	}
	referenced := false
	for _, binding := range existing {
		if s.bindingRefs[binding.ID] > 0 {
			referenced = true
		}
	}
	if referenced && oldBase != newBase {
		return located(NewDomainError(ReferenceInUse, "被送货明细引用的物料不能更换基本单位"), materialUnitSheetName, commands[0].row, "是否基本单位")
	}
	for unitID, binding := range existing {
		if _, keep := wanted[unitID]; keep {
			continue
		}
		if s.bindingRefs[binding.ID] > 0 {
			return located(NewDomainError(ReferenceInUse, "被送货明细引用的单位绑定不能删除"), materialUnitSheetName, commands[0].row, "单位名称")
		}
		delete(s.bindings, binding.ID)
	}
	if oldBase != 0 && oldBase != newBase {
		binding := existing[oldBase]
		binding.IsBase = false
		s.bindings[binding.ID] = binding
	}
	for _, command := range commands {
		binding, found := existing[command.unitID]
		if !found {
			binding = model.MaterialUnit{MaterialID: materialID, UnitID: command.unitID}
			binding.ID = s.allocate()
			ApplyCreateAudit(&binding.BusinessModel, actor, time.Now())
		} else {
			ApplyUpdateAudit(&binding.BusinessModel, actor)
		}
		binding.Factor, binding.IsBase, binding.Enabled = command.factor, command.isBase, command.enabled
		s.bindings[binding.ID] = binding
	}
	return nil
}

func materialCommandsToModels(commands []namedMaterialUnit) []model.MaterialUnit {
	result := make([]model.MaterialUnit, 0, len(commands))
	for _, command := range commands {
		result = append(result, model.MaterialUnit{UnitID: command.unitID, Factor: command.factor, IsBase: command.isBase, Enabled: command.enabled})
	}
	return result
}

func (s *MasterExcelState) categoryNames(exclude uint) map[uint]map[string]uint {
	result := map[uint]map[string]uint{}
	for id, item := range s.categories {
		if id == exclude {
			continue
		}
		if result[item.ParentID] == nil {
			result[item.ParentID] = map[string]uint{}
		}
		result[item.ParentID][NormalizeUniqueKey(item.Name)] = id
	}
	return result
}

func (s *MasterExcelState) unitByName(name string) (model.Unit, bool) {
	for _, item := range s.units {
		if NormalizeUniqueKey(item.Name) == NormalizeUniqueKey(name) {
			return item, true
		}
	}
	return model.Unit{}, false
}
func (s *MasterExcelState) brandByCode(code string) (model.Brand, bool) {
	for _, item := range s.brands {
		if NormalizeUniqueKey(item.Code) == NormalizeUniqueKey(code) {
			return item, true
		}
	}
	return model.Brand{}, false
}
func (s *MasterExcelState) brandByName(name string) (model.Brand, bool) {
	for _, item := range s.brands {
		if NormalizeUniqueKey(item.Name) == NormalizeUniqueKey(name) {
			return item, true
		}
	}
	return model.Brand{}, false
}
func (s *MasterExcelState) categoryByCode(code string) (model.Category, bool) {
	for _, item := range s.categories {
		if NormalizeUniqueKey(item.Code) == NormalizeUniqueKey(code) {
			return item, true
		}
	}
	return model.Category{}, false
}
func (s *MasterExcelState) categoryByCodeID(code string) uint {
	item, found := s.categoryByCode(code)
	if !found {
		return 0
	}
	return item.ID
}
func (s *MasterExcelState) materialByCode(code string) (model.Material, bool) {
	for _, item := range s.materials {
		if NormalizeUniqueKey(item.Code) == NormalizeUniqueKey(code) {
			return item, true
		}
	}
	return model.Material{}, false
}

func (s *MasterExcelState) allocate() uint {
	id := s.nextID
	s.nextID++
	return id
}

func located(err error, sheet string, row int, column string) error {
	domain, ok := AsDomainError(err)
	if !ok {
		domain = NewDomainError(ValidationFailed, "导入失败")
	}
	return domain.WithExcelLocation(sheet, row, column)
}

func categoryProblemRow(rows []categoryExcelRow, err error) int {
	domain, ok := AsDomainError(err)
	if !ok || len(rows) == 0 {
		return 2
	}
	for _, row := range rows {
		if NormalizeUniqueKey(row.code) == "root" && strings.Contains(domain.Message, "ROOT") {
			return row.row
		}
	}
	return rows[0].row
}

func categoryProblemColumn(err error) string {
	domain, ok := AsDomainError(err)
	if ok && domain.Field == "parentCode" {
		return "上级类别编号"
	}
	return "编号"
}

func (s *MasterExcelState) Export(kind string) ([]byte, error) {
	switch kind {
	case "units":
		rows := make([][]string, 0, len(s.units))
		for _, id := range sortedIDs(s.units) {
			item := s.units[id]
			rows = append(rows, []string{item.Name, strconv.FormatUint(item.Version, 10)})
		}
		return writeMasterWorkbook(kind, rows, nil)
	case "brands":
		rows := make([][]string, 0, len(s.brands))
		for _, id := range sortedIDs(s.brands) {
			item := s.brands[id]
			rows = append(rows, []string{item.Code, item.Name, item.Mnemonic, strconv.FormatUint(item.Version, 10)})
		}
		return writeMasterWorkbook(kind, rows, nil)
	case "categories":
		rows := make([][]string, 0, len(s.categories))
		for _, id := range sortedIDs(s.categories) {
			item := s.categories[id]
			parent := "ROOT"
			if item.ParentID != 0 {
				parent = s.categories[item.ParentID].Code
			}
			rows = append(rows, []string{item.Code, item.Name, item.Mnemonic, parent, strconv.FormatUint(item.Version, 10)})
		}
		return writeMasterWorkbook(kind, rows, nil)
	case "materials":
		rows := make([][]string, 0, len(s.materials))
		unitRows := [][]string{}
		for _, id := range sortedIDs(s.materials) {
			item := s.materials[id]
			image := ""
			if item.ImageFileID != nil {
				image = strconv.FormatUint(uint64(*item.ImageFileID), 10)
			}
			rows = append(rows, []string{item.Code, item.Name, item.Mnemonic, stringValue(item.Specification), stringValue(item.Alias), s.brands[item.BrandID].Code, s.categories[item.CategoryID].Code, image, strconv.FormatUint(item.Version, 10)})
			for _, bindingID := range sortedIDs(s.bindings) {
				binding := s.bindings[bindingID]
				if binding.MaterialID != item.ID {
					continue
				}
				unitRows = append(unitRows, []string{item.Code, s.units[binding.UnitID].Name, binding.Factor.StringFixed(6), boolText(binding.IsBase), boolText(binding.Enabled)})
			}
		}
		return writeMasterWorkbook(kind, rows, unitRows)
	default:
		return nil, domainError(ValidationFailed, "未知资料导出类型", "kind")
	}
}

func writeMasterWorkbook(kind string, rows, secondary [][]string) ([]byte, error) {
	book := excelize.NewFile()
	defer book.Close()
	var err error
	switch kind {
	case "units":
		err = writeSheet(book, unitSheetName, []excelColumn{{"单位名称", true}, {"版本号", false}}, rows)
	case "brands":
		err = writeSheet(book, brandSheetName, []excelColumn{{"编号", true}, {"客户名称", true}, {"助记符", true}, {"版本号", false}}, rows)
	case "categories":
		err = writeSheet(book, categorySheetName, []excelColumn{{"编号", true}, {"类别名称", true}, {"助记符", true}, {"上级类别编号", true}, {"版本号", false}}, rows)
	case "materials":
		err = writeSheet(book, materialSheetName, materialColumns(), rows)
		if err == nil {
			err = writeSheet(book, materialUnitSheetName, materialUnitColumns(), secondary)
		}
	}
	if err != nil {
		return nil, err
	}
	if book.GetSheetName(0) == "Sheet1" {
		_ = book.DeleteSheet("Sheet1")
	}
	return workbookBytes(book)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolText(value bool) string {
	if value {
		return "是"
	}
	return "否"
}

func (s *MasterExcelState) load(db *gorm.DB) error {
	var units []model.Unit
	var brands []model.Brand
	var categories []model.Category
	var materials []model.Material
	var bindings []model.MaterialUnit
	var images []model.MaterialImage
	if err := db.Find(&units).Error; err != nil { return err }
	if err := db.Find(&brands).Error; err != nil { return err }
	if err := db.Find(&categories).Error; err != nil { return err }
	if err := db.Find(&materials).Error; err != nil { return err }
	if err := db.Find(&bindings).Error; err != nil { return err }
	if err := db.Find(&images).Error; err != nil { return err }
	for _, item := range units { s.units[item.ID] = item; s.noteID(item.ID) }
	for _, item := range brands { s.brands[item.ID] = item; s.noteID(item.ID) }
	for _, item := range categories { s.categories[item.ID] = item; s.noteID(item.ID) }
	for _, item := range materials { s.materials[item.ID] = item; s.noteID(item.ID) }
	for _, item := range bindings { s.bindings[item.ID] = item; s.noteID(item.ID) }
	for _, item := range images { s.images[item.FileID] = struct{}{} }
	return nil
}

func (s *MasterExcelState) save(db *gorm.DB) error {
	for _, item := range s.units { if err := db.Save(&item).Error; err != nil { return err } }
	for _, item := range s.brands { if err := db.Save(&item).Error; err != nil { return err } }
	for _, item := range s.categories { if err := db.Save(&item).Error; err != nil { return err } }
	for _, item := range s.materials { if err := db.Save(&item).Error; err != nil { return err } }
	for _, item := range s.bindings { if err := db.Save(&item).Error; err != nil { return err } }
	return nil
}

func (s *MasterExcelState) noteID(id uint) {
	if id >= s.nextID { s.nextID = id + 1 }
}

func sortedIDs[T any](values map[uint]T) []uint {
	ids := make([]uint, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
