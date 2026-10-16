package service

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

func MasterTemplate(kind string) ([]byte, error) { return masterTemplate(kind) }

func (s *MasterService) ImportMasterExcel(ctx context.Context, actor uint, kind string, contents []byte) (ImportResult, error) {
	parsed, err := parseMasterWorkbook(kind, contents)
	if err != nil { return ImportResult{}, err }
	result := ImportResult{}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		working := NewMasterExcelState()
		if err := working.load(tx); err != nil { return err }
		var applyErr error
		switch kind {
		case "units": applyErr = working.applyUnits(parsed.units, actor)
		case "brands": applyErr = working.applyBrands(parsed.brands, actor)
		case "categories": applyErr = working.applyCategories(parsed.categories, actor)
		case "materials": applyErr = working.applyMaterials(parsed, actor)
		default: applyErr = domainError(ValidationFailed, "未知资料导入类型", "kind")
		}
		if applyErr != nil { return applyErr }
		result = ImportResult{CreatedCount: working.created, UpdatedCount: working.updated}
		return working.save(tx)
	})
	return result, dbError(err)
}


const (
	unitSheetName         = "单位"
	brandSheetName        = "品牌"
	categorySheetName     = "类别"
	materialSheetName     = "物料"
	materialUnitSheetName = "物料单位"
	excelTextFormat       = "@"
)

type excelColumn struct {
	Name     string
	Required bool
}

type excelCellError struct {
	sheet  string
	row    int
	column string
	reason string
}

func (e excelCellError) domain() *DomainError {
	return domainError(ValidationFailed, e.reason, e.column).WithExcelLocation(e.sheet, e.row, e.column)
}

type unitExcelRow struct {
	row     int
	name    string
	version *uint64
}

type brandExcelRow struct {
	row      int
	code     string
	name     string
	mnemonic string
	version  *uint64
}

type categoryExcelRow struct {
	row        int
	code       string
	name       string
	mnemonic   string
	parentCode string
	version    *uint64
}

type materialExcelRow struct {
	row           int
	code          string
	name          string
	mnemonic      string
	specification string
	alias         string
	brandCode     string
	categoryCode  string
	imageFileID   *uint
	version       *uint64
}

type materialUnitExcelRow struct {
	row        int
	material   string
	unitName   string
	factor     decimal.Decimal
	isBase     bool
	enabled    bool
}

type masterWorkbook struct {
	units         []unitExcelRow
	brands        []brandExcelRow
	categories    []categoryExcelRow
	materials     []materialExcelRow
	materialUnits []materialUnitExcelRow
}

func masterTemplate(kind string) ([]byte, error) {
	book := excelize.NewFile()
	defer book.Close()
	switch kind {
	case "units":
		if err := writeSheet(book, unitSheetName, []excelColumn{{"单位名称", true}, {"版本号", false}}, [][]string{{"个", ""}}); err != nil {
			return nil, err
		}
	case "brands":
		if err := writeSheet(book, brandSheetName, []excelColumn{{"编号", true}, {"品牌名称", true}, {"助记符", true}, {"版本号", false}}, [][]string{{"B001", "品牌", "pp", ""}}); err != nil {
			return nil, err
		}
	case "categories":
		if err := writeSheet(book, categorySheetName, []excelColumn{{"编号", true}, {"类别名称", true}, {"助记符", true}, {"上级类别编号", true}, {"版本号", false}}, [][]string{{"C001", "类别", "lb", "ROOT", ""}}); err != nil {
			return nil, err
		}
	case "materials":
		if err := writeSheet(book, materialSheetName, materialColumns(), [][]string{{"M001", "物料", "wl", "", "", "B001", "C001", "", ""}}); err != nil {
			return nil, err
		}
		if err := writeSheet(book, materialUnitSheetName, materialUnitColumns(), [][]string{{"M001", "个", "1", "是", "是"}}); err != nil {
			return nil, err
		}
	default:
		return nil, domainError(ValidationFailed, "未知资料模板", "kind")
	}
	if book.GetSheetName(0) == "Sheet1" {
		_ = book.DeleteSheet("Sheet1")
	}
	return workbookBytes(book)
}

func materialColumns() []excelColumn {
	return []excelColumn{{"编号", true}, {"物料名称", true}, {"助记符", true}, {"规格", false}, {"别名", false}, {"品牌编号", true}, {"类别编号", true}, {"图片文件ID", false}, {"版本号", false}}
}

func materialUnitColumns() []excelColumn {
	return []excelColumn{{"物料编号", true}, {"单位名称", true}, {"换算系数", true}, {"是否基本单位", true}, {"是否启用", true}}
}

func writeSheet(book *excelize.File, name string, columns []excelColumn, rows [][]string) error {
	index, err := book.NewSheet(name)
	if err != nil {
		return err
	}
	book.SetActiveSheet(index)
	style, err := book.NewStyle(&excelize.Style{CustomNumFmt: stringPointer(excelTextFormat)})
	if err != nil {
		return err
	}
	for column, definition := range columns {
		header := definition.Name
		if definition.Required {
			header += "*"
		}
		cell, _ := excelize.CoordinatesToCellName(column+1, 1)
		if err := book.SetCellStr(name, cell, header); err != nil {
			return err
		}
	}
	for rowIndex, row := range rows {
		for column := range columns {
			cell, _ := excelize.CoordinatesToCellName(column+1, rowIndex+2)
			value := ""
			if column < len(row) {
				value = row[column]
			}
			if err := book.SetCellStr(name, cell, value); err != nil {
				return err
			}
			if err := book.SetCellStyle(name, cell, cell, style); err != nil {
				return err
			}
		}
	}
	return nil
}

func parseMasterWorkbook(kind string, contents []byte) (masterWorkbook, error) {
	book, err := excelize.OpenReader(bytes.NewReader(contents))
	if err != nil {
		return masterWorkbook{}, NewDomainError(ImportFailed, "Excel文件无法读取")
	}
	defer book.Close()
	var parsed masterWorkbook
	var problems []excelCellError
	switch kind {
	case "units":
		parsed.units, problems = readUnitSheet(book)
	case "brands":
		parsed.brands, problems = readBrandSheet(book)
	case "categories":
		parsed.categories, problems = readCategorySheet(book)
	case "materials":
		parsed.materials, parsed.materialUnits, problems = readMaterialSheets(book)
	default:
		return masterWorkbook{}, domainError(ValidationFailed, "未知资料导入类型", "kind")
	}
	if len(problems) > 0 {
		return masterWorkbook{}, importProblems(problems)
	}
	return parsed, nil
}

func readUnitSheet(book *excelize.File) ([]unitExcelRow, []excelCellError) {
	rows, problems := readRows(book, unitSheetName, []excelColumn{{"单位名称", true}, {"版本号", false}})
	result := make([]unitExcelRow, 0, len(rows))
	for _, row := range rows {
		version, versionProblem := optionalVersion(unitSheetName, row.number, "版本号", row.values["版本号"])
		if versionProblem != nil {
			problems = append(problems, *versionProblem)
		}
		result = append(result, unitExcelRow{row: row.number, name: strings.TrimSpace(row.values["单位名称"]), version: version})
	}
	return result, problems
}

func readBrandSheet(book *excelize.File) ([]brandExcelRow, []excelCellError) {
	rows, problems := readRows(book, brandSheetName, []excelColumn{{"编号", true}, {"品牌名称", true}, {"助记符", true}, {"版本号", false}})
	result := make([]brandExcelRow, 0, len(rows))
	for _, row := range rows {
		version, versionProblem := optionalVersion(brandSheetName, row.number, "版本号", row.values["版本号"])
		if versionProblem != nil {
			problems = append(problems, *versionProblem)
		}
		result = append(result, brandExcelRow{row: row.number, code: strings.TrimSpace(row.values["编号"]), name: strings.TrimSpace(row.values["品牌名称"]), mnemonic: strings.TrimSpace(row.values["助记符"]), version: version})
	}
	return result, problems
}

func readCategorySheet(book *excelize.File) ([]categoryExcelRow, []excelCellError) {
	rows, problems := readRows(book, categorySheetName, []excelColumn{{"编号", true}, {"类别名称", true}, {"助记符", true}, {"上级类别编号", true}, {"版本号", false}})
	result := make([]categoryExcelRow, 0, len(rows))
	seen := map[string]int{}
	for _, row := range rows {
		code := strings.TrimSpace(row.values["编号"])
		key := NormalizeUniqueKey(code)
		if key == "root" {
			problems = append(problems, excelCellError{categorySheetName, row.number, "编号", "类别编号不能为ROOT"})
		}
		if previous, exists := seen[key]; exists {
			problems = append(problems, excelCellError{categorySheetName, row.number, "编号", fmt.Sprintf("与第%d行重复", previous)})
		}
		seen[key] = row.number
		version, versionProblem := optionalVersion(categorySheetName, row.number, "版本号", row.values["版本号"])
		if versionProblem != nil {
			problems = append(problems, *versionProblem)
		}
		result = append(result, categoryExcelRow{row: row.number, code: code, name: strings.TrimSpace(row.values["类别名称"]), mnemonic: strings.TrimSpace(row.values["助记符"]), parentCode: strings.TrimSpace(row.values["上级类别编号"]), version: version})
	}
	return result, problems
}

func readMaterialSheets(book *excelize.File) ([]materialExcelRow, []materialUnitExcelRow, []excelCellError) {
	rows, problems := readRows(book, materialSheetName, materialColumns())
	materials := make([]materialExcelRow, 0, len(rows))
	seen := map[string]int{}
	for _, row := range rows {
		code := strings.TrimSpace(row.values["编号"])
		key := NormalizeUniqueKey(code)
		if previous, exists := seen[key]; exists {
			problems = append(problems, excelCellError{materialSheetName, row.number, "编号", fmt.Sprintf("与第%d行重复", previous)})
		}
		seen[key] = row.number
		version, versionProblem := optionalVersion(materialSheetName, row.number, "版本号", row.values["版本号"])
		if versionProblem != nil {
			problems = append(problems, *versionProblem)
		}
		imageID, imageProblem := optionalFileID(row.number, row.values["图片文件ID"])
		if imageProblem != nil {
			problems = append(problems, *imageProblem)
		}
		materials = append(materials, materialExcelRow{row: row.number, code: code, name: strings.TrimSpace(row.values["物料名称"]), mnemonic: strings.TrimSpace(row.values["助记符"]), specification: strings.TrimSpace(row.values["规格"]), alias: strings.TrimSpace(row.values["别名"]), brandCode: strings.TrimSpace(row.values["品牌编号"]), categoryCode: strings.TrimSpace(row.values["类别编号"]), imageFileID: imageID, version: version})
	}
	unitRows, unitProblems := readRows(book, materialUnitSheetName, materialUnitColumns())
	problems = append(problems, unitProblems...)
	problems = append(problems, validateMaterialCoverage(materials, unitRows)...)
	units := make([]materialUnitExcelRow, 0, len(unitRows))
	for _, row := range unitRows {
		factor, err := ParseDecimal(strings.TrimSpace(row.values["换算系数"]), FactorRule, "换算系数")
		base, baseProblem := excelBool(row.number, "是否基本单位", row.values["是否基本单位"])
		enabled, enabledProblem := excelBool(row.number, "是否启用", row.values["是否启用"])
		if err != nil {
			problems = append(problems, excelCellError{materialUnitSheetName, row.number, "换算系数", domainMessage(err)})
		}
		if baseProblem != nil {
			problems = append(problems, *baseProblem)
		}
		if enabledProblem != nil {
			problems = append(problems, *enabledProblem)
		}
		units = append(units, materialUnitExcelRow{row: row.number, material: strings.TrimSpace(row.values["物料编号"]), unitName: strings.TrimSpace(row.values["单位名称"]), factor: factor, isBase: base, enabled: enabled})
	}
	return materials, units, problems
}

func validateMaterialCoverage(materials []materialExcelRow, rows []excelDataRow) []excelCellError {
	wanted := map[string]struct{}{}
	for _, material := range materials {
		wanted[NormalizeUniqueKey(material.code)] = struct{}{}
	}
	covered := map[string]map[string]int{}
	var problems []excelCellError
	for _, row := range rows {
		code := NormalizeUniqueKey(row.values["物料编号"])
		if _, exists := wanted[code]; !exists {
			problems = append(problems, excelCellError{materialUnitSheetName, row.number, "物料编号", "物料主表中不存在"})
			continue
		}
		unit := NormalizeUniqueKey(row.values["单位名称"])
		if covered[code] == nil {
			covered[code] = map[string]int{}
		}
		if previous, exists := covered[code][unit]; exists {
			problems = append(problems, excelCellError{materialUnitSheetName, row.number, "单位名称", fmt.Sprintf("与第%d行重复", previous)})
		}
		covered[code][unit] = row.number
	}
	for _, material := range materials {
		if len(covered[NormalizeUniqueKey(material.code)]) == 0 {
			problems = append(problems, excelCellError{materialSheetName, material.row, "编号", "缺少完整单位集合"})
		}
	}
	return problems
}

type excelDataRow struct {
	number int
	values map[string]string
}

func readRows(book *excelize.File, sheet string, columns []excelColumn) ([]excelDataRow, []excelCellError) {
	rows, err := book.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		return nil, []excelCellError{{sheet, 1, columns[0].Name, "缺少工作表或表头"}}
	}
	headers := map[int]string{}
	for index, header := range rows[0] {
		headers[index] = strings.TrimSuffix(strings.TrimSpace(header), "*")
	}
	var problems []excelCellError
	for _, column := range columns {
		if !headerExists(headers, column.Name) {
			problems = append(problems, excelCellError{sheet, 1, column.Name, "缺少表头"})
		}
	}
	result := make([]excelDataRow, 0, len(rows)-1)
	for rowIndex := 1; rowIndex < len(rows); rowIndex++ {
		values := map[string]string{}
		blank := true
		for column, header := range headers {
			cell, _ := excelize.CoordinatesToCellName(column+1, rowIndex+1)
			value, _ := book.GetCellValue(sheet, cell, excelize.Options{RawCellValue: true})
			formula, formulaErr := book.GetCellFormula(sheet, cell)
			if strings.TrimSpace(value) != "" || strings.TrimSpace(formula) != "" {
				blank = false
			}
			if formulaErr == nil && strings.TrimSpace(formula) != "" {
				problems = append(problems, excelCellError{sheet, rowIndex + 1, header, "不导入公式单元格"})
				value = formula
			}
			values[header] = value
		}
		if blank {
			continue
		}
		for _, column := range columns {
			if column.Required && strings.TrimSpace(values[column.Name]) == "" {
				problems = append(problems, excelCellError{sheet, rowIndex + 1, column.Name, "不能为空"})
			}
		}
		result = append(result, excelDataRow{number: rowIndex + 1, values: values})
	}
	return result, problems
}

func headerExists(headers map[int]string, name string) bool {
	for _, header := range headers {
		if header == name {
			return true
		}
	}
	return false
}

func optionalVersion(sheet string, row int, column, value string) (*uint64, *excelCellError) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		problem := excelCellError{sheet, row, column, "版本号必须大于0"}
		return nil, &problem
	}
	return &parsed, nil
}

func optionalFileID(row int, value string) (*uint, *excelCellError) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		problem := excelCellError{materialSheetName, row, "图片文件ID", "图片文件ID无效"}
		return nil, &problem
	}
	id := uint(parsed)
	return &id, nil
}

func excelBool(row int, column, value string) (bool, *excelCellError) {
	switch strings.TrimSpace(value) {
	case "是":
		return true, nil
	case "否":
		return false, nil
	default:
		problem := excelCellError{materialUnitSheetName, row, column, "只能填写是或否"}
		return false, &problem
	}
}

func domainMessage(err error) string {
	domain, ok := AsDomainError(err)
	if !ok {
		return "值无效"
	}
	return domain.Message
}

func importProblems(problems []excelCellError) error {
	root := NewDomainError(ImportFailed, "导入失败")
	for _, problem := range problems {
		root.WithErrors(problem.domain())
	}
	return root
}

func workbookBytes(book *excelize.File) ([]byte, error) {
	var output bytes.Buffer
	if err := book.Write(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func stringPointer(value string) *string { return &value }
