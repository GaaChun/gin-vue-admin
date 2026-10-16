package service

import "sync"

// CategoryState serializes in-memory policy cases using the same ascending-ID
// lock order as the database path; it is not a claim about database locking.
type CategoryState struct {
	mu           sync.Mutex
	parents      map[uint]uint
	deleted      map[uint]bool
	materialRefs map[uint]uint
}

func NewCategoryState(parents map[uint]uint) *CategoryState {
	copy := make(map[uint]uint, len(parents))
	for id, parent := range parents {
		copy[id] = parent
	}
	return &CategoryState{parents: copy, deleted: map[uint]bool{}, materialRefs: map[uint]uint{}}
}
func (s *CategoryState) Move(id, parent uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deleted[id] {
		return notFound()
	}
	if parent != 0 {
		if _, ok := s.parents[parent]; !ok || s.deleted[parent] {
			return domainError(ValidationFailed, "上级类别不存在或已删除", "parentId")
		}
	}
	if id == parent {
		return domainError(ValidationFailed, "不能选择自身或后代作为上级", "parentId")
	}
	for p := parent; p != 0; p = s.parents[p] {
		if p == id {
			return domainError(ValidationFailed, "不能选择自身或后代作为上级", "parentId")
		}
	}
	s.parents[id] = parent
	return nil
}
func (s *CategoryState) Delete(id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deleted[id] {
		return notFound()
	}
	for child, parent := range s.parents {
		if parent == id && !s.deleted[child] {
			return NewDomainError(ReferenceInUse, "类别存在有效子类别")
		}
	}
	if s.materialRefs[id] != 0 {
		return NewDomainError(ReferenceInUse, "类别正在被物料引用")
	}
	s.deleted[id] = true
	return nil
}
func (s *CategoryState) Insert(id, parent uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if parent != 0 {
		if _, ok := s.parents[parent]; !ok || s.deleted[parent] {
			return domainError(ValidationFailed, "上级类别不存在或已删除", "parentId")
		}
	}
	if _, exists := s.parents[id]; exists {
		return duplicate("id")
	}
	s.parents[id] = parent
	return nil
}

// CreateMaterialReference models material creation's locked category recheck.
// It is an in-memory concurrency policy test seam, not MySQL lock verification.
func (s *CategoryState) CreateMaterialReference(categoryID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.parents[categoryID]; !ok || s.deleted[categoryID] {
		return domainError(ValidationFailed, "类别不存在或已删除", "categoryId")
	}
	s.materialRefs[categoryID]++
	return nil
}
