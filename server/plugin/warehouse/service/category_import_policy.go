package service

import "strings"

type CategoryDraft struct {
	Code       string
	ParentCode string
}

func PrepareCategoryParentOrder(drafts []CategoryDraft) ([]string, error) {
	parents := make(map[string]string, len(drafts))
	for _, draft := range drafts {
		code := NormalizeUniqueKey(draft.Code)
		if code == "" || code == "root" {
			return nil, domainError(ValidationFailed, "类别编号不能为ROOT", "code")
		}
		if _, exists := parents[code]; exists {
			return nil, duplicate("code")
		}
		parents[code] = NormalizeUniqueKey(draft.ParentCode)
	}
	for code, parent := range parents {
		if parent != "" && parent != "root" {
			if _, exists := parents[parent]; !exists {
				return nil, domainError(ValidationFailed, "上级类别不存在", "parentCode")
			}
		}
		parents[code] = strings.TrimSpace(parent)
	}
	order := make([]string, 0, len(parents))
	visited := make(map[string]bool, len(parents))
	visiting := make(map[string]bool, len(parents))
	var visit func(string) error
	visit = func(code string) error {
		if visited[code] {
			return nil
		}
		if visiting[code] {
			return domainError(ValidationFailed, "类别层级不能成环", "parentCode")
		}
		visiting[code] = true
		parent := parents[code]
		if parent != "" && parent != "root" {
			if err := visit(parent); err != nil {
				return err
			}
		}
		visiting[code] = false
		visited[code] = true
		order = append(order, code)
		return nil
	}
	for code := range parents {
		if err := visit(code); err != nil {
			return nil, err
		}
	}
	return order, nil
}

func CategorySiblingNameAvailable(existing map[uint]map[string]uint, parentID, selfID uint, name string) bool {
	siblings := existing[parentID]
	id, exists := siblings[NormalizeUniqueKey(name)]
	return !exists || id == selfID
}
