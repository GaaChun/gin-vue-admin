package service

import (
	"sync"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
)

func TestActiveIdentifierPolicy(t *testing.T) {
	records := []IdentifierRecord{{ID: 1, Value: " 件 ", Deleted: false}}
	p := InspectIdentifierPresence("件", records)
	if err := ValidateIdentifierAvailability(ReusableMasterIdentifier, p, "name"); err == nil {
		t.Fatal("active unit duplicate accepted")
	}
	brand := InspectIdentifierPresence("BR-1", []IdentifierRecord{{ID: 2, Value: "br-1"}})
	if err := ValidateIdentifierAvailability(ReusableMasterIdentifier, brand, "code"); err == nil {
		t.Fatal("active brand code duplicate accepted")
	}
	records[0].Deleted = true
	p = InspectIdentifierPresence("件", records)
	if err := ValidateIdentifierAvailability(ReusableMasterIdentifier, p, "name"); err != nil {
		t.Fatalf("soft-deleted code not reusable: %v", err)
	}
	usage := SummarizeReferenceUsage(1, []ReferenceLink{{TargetID: 1, Deleted: false}})
	if err := usage.CanDeleteRecord(); err == nil {
		t.Fatal("externally referenced record accepted for deletion")
	}
}
func TestCategoryStateSerializesConflictingCommands(t *testing.T) {
	state := NewCategoryState(map[uint]uint{1: 0, 2: 0})
	first := state.Move(1, 2)
	second := state.Move(2, 1)
	if first != nil && second != nil {
		t.Fatal("unexpected failures")
	}
	if first == nil && second == nil {
		t.Fatal("inverse moves both committed")
	}
	state = NewCategoryState(map[uint]uint{1: 0})
	if err := state.Delete(1); err != nil {
		t.Fatal(err)
	}
	if err := state.Insert(2, 1); err == nil {
		t.Fatal("insert under deleted parent committed")
	}
}

func TestCategoryMaterialReferenceCreationCannotRaceDelete(t *testing.T) {
	state := NewCategoryState(map[uint]uint{1: 0})
	var group sync.WaitGroup
	group.Add(2)
	var deleteErr, createErr error
	go func() { defer group.Done(); deleteErr = state.Delete(1) }()
	go func() { defer group.Done(); createErr = state.CreateMaterialReference(1) }()
	group.Wait()
	if (deleteErr == nil) == (createErr == nil) {
		t.Fatalf("delete=%v create material reference=%v; exactly one command must commit", deleteErr, createErr)
	}
}
func TestMnemonicSuggestions(t *testing.T) {
	cases := map[string]string{"白菜": "bc", "ABC-12": "abc12", "---": ""}
	for input, want := range cases {
		if got := SuggestMnemonic(input); got != want {
			t.Fatalf("%q: got %q want %q", input, got, want)
		}
	}
}
func TestCategoryVirtualRootAndCycle(t *testing.T) {
	items := []model.Category{{ParentID: 0}, {ParentID: 0}, {ParentID: 1}}
	items[0].ID = 1
	items[1].ID = 2
	items[2].ID = 3
	tree := assembleCategoryTree(items)
	if len(tree) != 2 || len(tree[0].Children) != 1 {
		t.Fatal("virtual root tree incorrectly assembled")
	}
	state := NewCategoryState(map[uint]uint{1: 0, 2: 1, 3: 2})
	if err := state.Move(1, 3); err == nil {
		t.Fatal("descendant cycle accepted")
	}
}
func TestCategorySiblingAndImportPolicies(t *testing.T) {
	existing := map[uint]map[string]uint{1: {"same": 1}, 2: {"same": 2}}
	if CategorySiblingNameAvailable(existing, 1, 3, "SAME") {
		t.Fatal("same-level duplicate accepted")
	}
	if !CategorySiblingNameAvailable(existing, 3, 4, "same") {
		t.Fatal("different-level duplicate rejected")
	}
	if _, err := PrepareCategoryParentOrder([]CategoryDraft{{Code: "ROOT"}}); err == nil {
		t.Fatal("ROOT category code accepted")
	}
	order, err := PrepareCategoryParentOrder([]CategoryDraft{{Code: "child", ParentCode: "parent"}, {Code: "parent", ParentCode: "ROOT"}})
	if err != nil || len(order) != 2 || order[0] != "parent" {
		t.Fatalf("forward parent reference was not prepared: %v, %v", order, err)
	}
	state := NewCategoryState(map[uint]uint{1: 0, 2: 1})
	if err := state.Delete(1); err == nil {
		t.Fatal("category with an active child accepted for deletion")
	}
}
