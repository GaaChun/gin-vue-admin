package service

import (
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/shopspring/decimal"
)

func TestDeliveryNumbersAreUniqueAcrossBrandsAndDays(t *testing.T) {
	now := time.Date(2026, 10, 15, 18, 0, 0, 0, time.FixedZone("Asia/Shanghai", 8*3600))
	state := NewDeliveryState(now)
	first, err := state.Reserve(1)
	second, secondErr := state.Reserve(2)
	if err != nil || secondErr != nil || first.Number == second.Number || first.Number != "SK261015000001" || second.Number != "SK261015000002" {
		t.Fatalf("numbers=%s %s errors=%v %v", first.Number, second.Number, err, secondErr)
	}
	state.now = now.Add(24 * time.Hour)
	next, err := state.Reserve(1)
	if err != nil || next.Number != "SK261016000001" {
		t.Fatalf("next day number=%s err=%v", next.Number, err)
	}
	exhausted := NewDeliveryState(now)
	exhausted.counters[businessDate(now).Format("2006-01-02")] = 999999
	if _, err := exhausted.Reserve(1); err == nil {
		t.Fatal("exhausted sequence accepted")
	}
}

func TestSequentialReservationsStayUnique(t *testing.T) {
	state := NewDeliveryState(time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC))
	numbers := map[string]struct{}{}
	for range 20 {
		item, err := state.Reserve(1)
		if err != nil {
			t.Fatal(err)
		}
		numbers[item.Number] = struct{}{}
	}
	if len(numbers) != 20 || state.counters["2026-10-15"] != 20 {
		t.Fatalf("unique=%d counter=%d", len(numbers), state.counters["2026-10-15"])
	}
}

func TestReservationCannotBeStolenOrReused(t *testing.T) {
	state := deliveryFixture(time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC))
	reserved, _ := state.Reserve(4)
	if _, err := state.Create(5, deliveryInput(reserved.Token, "1", "1")); err == nil {
		t.Fatal("another user consumed reservation")
	}
	note, err := state.Create(4, deliveryInput(reserved.Token, "2.0000", "3.0000"))
	if err != nil || note.Number != reserved.Number {
		t.Fatalf("note=%#v err=%v", note, err)
	}
	if _, err := state.Create(4, deliveryInput(reserved.Token, "2", "3")); err == nil {
		t.Fatal("consumed reservation accepted again")
	}
	state.now = state.now.Add(24 * time.Hour)
	later, _ := state.Reserve(4)
	if later.Number[:8] == reserved.Number[:8] {
		t.Fatal("cross-day reservation changed an existing number")
	}
}

func TestDeliverySnapshotSurvivesMasterDataRename(t *testing.T) {
	state := deliveryFixture(time.Now())
	reserved, _ := state.Reserve(4)
	note, err := state.Create(4, deliveryInput(reserved.Token, "0.3333", "0.05"))
	if err != nil || note.TotalAmount.StringFixed(2) != "0.02" {
		t.Fatalf("amount=%s err=%v", note.TotalAmount, err)
	}
	material := state.materials[1]
	material.Name = "大白菜"
	state.materials[1] = material
	binding := state.bindings[1]
	binding.Factor = decimal.RequireFromString("3")
	state.bindings[1] = binding
	detailID := firstDetail(state, note.ID)
	updated, err := state.Update(4, req.UpdateDeliveryInput{VersionedID: req.VersionedID{ID: note.ID}, DeliveredAt: note.DeliveredAt, BrandID: 1, Details: []req.DeliveryDetailInput{{ID: &detailID, MaterialID: 1, MaterialUnitID: 1, Quantity: "2", UnitPrice: "0.05"}}})
	if err != nil {
		t.Fatal(err)
	}
	detail := state.details[detailID]
	if detail.MaterialNameSnapshot != "白菜" || !detail.Factor.Equal(decimal.NewFromInt(1)) || updated.TotalAmount.StringFixed(2) != "0.10" {
		t.Fatalf("snapshot changed: %#v total=%s", detail, updated.TotalAmount)
	}
}

func TestMixedCustomerMaterialsAreAllowedAndFailedUpdateDoesNotMutateNote(t *testing.T) {
	state := deliveryFixture(time.Now())
	brand := model.Brand{Code: "B2", Name: "另一客户"}
	brand.ID = 2
	state.brands[2] = brand
	material := model.Material{Code: "M2", Name: "另一物料", BrandID: 2, CategoryID: 1}
	material.ID = 2
	state.materials[2] = material
	binding := state.bindings[1]
	binding.ID = 2
	binding.MaterialID = 2
	state.bindings[2] = binding
	reserved, _ := state.Reserve(4)
	note, err := state.Create(4, req.CreateDeliveryInput{ReservationToken: reserved.Token, DeliveredAt: time.Now(), BrandID: 1, Details: []req.DeliveryDetailInput{
		{MaterialID: 1, MaterialUnitID: 1, Quantity: "1", UnitPrice: "1"},
		{MaterialID: 2, MaterialUnitID: 2, Quantity: "1", UnitPrice: "1"},
	}})
	if err != nil || len(state.detailsFor(note.ID)) != 2 {
		t.Fatalf("mixed-customer materials rejected: note=%#v err=%v", note, err)
	}
	before := state.details[firstDetail(state, note.ID)]
	_, err = state.Update(4, req.UpdateDeliveryInput{VersionedID: req.VersionedID{ID: note.ID}, DeliveredAt: note.DeliveredAt, BrandID: 1, Details: []req.DeliveryDetailInput{{ID: &before.ID, MaterialID: 1, MaterialUnitID: 1, Quantity: "0", UnitPrice: "9"}}})
	after := state.details[before.ID]
	if err == nil || !after.Quantity.Equal(before.Quantity) || state.notes[note.ID].Version != 1 {
		t.Fatalf("failed update changed note: err=%v quantity=%s version=%d", err, after.Quantity, state.notes[note.ID].Version)
	}
}

func TestReviewedDeliveryRejectsEditAndUnreviewClearsAudit(t *testing.T) {
	state := deliveryFixture(time.Now())
	reserved, _ := state.Reserve(4)
	note, _ := state.Create(4, deliveryInput(reserved.Token, "1", "1"))
	reviewed, err := state.Review(4, req.ReviewDeliveryInput{VersionedID: req.VersionedID{ID: note.ID}}, "审核人")
	if err != nil || reviewed.Status != "reviewed" || reviewed.ReviewedByNameSnapshot == nil {
		t.Fatalf("review=%#v err=%v", reviewed, err)
	}
	if _, err := state.Update(4, req.UpdateDeliveryInput{VersionedID: req.VersionedID{ID: note.ID}, DeliveredAt: note.DeliveredAt, BrandID: 1, Details: []req.DeliveryDetailInput{{MaterialID: 1, MaterialUnitID: 1, Quantity: "2", UnitPrice: "1"}}}); err == nil {
		t.Fatal("reviewed note updated")
	}
	if _, err := state.Review(4, req.ReviewDeliveryInput{VersionedID: req.VersionedID{ID: note.ID}}, "审核人"); err == nil {
		t.Fatal("double review accepted")
	}
	draft, err := state.Unreview(4, req.UnreviewDeliveryInput{VersionedID: req.VersionedID{ID: note.ID}})
	if err != nil || draft.Status != "draft" || draft.ReviewedByNameSnapshot != nil || draft.ReviewedBy != nil {
		t.Fatalf("unreview=%#v err=%v", draft, err)
	}
}

func deliveryFixture(now time.Time) *DeliveryState {
	state := NewDeliveryState(now)
	brand := model.Brand{Code: "B1", Name: "品牌"}
	brand.ID = 1
	material := model.Material{Code: "M1", Name: "白菜", BrandID: 1, CategoryID: 1}
	material.ID = 1
	category := model.Category{Code: "C1", Name: "蔬菜"}
	category.ID = 1
	unit := model.Unit{Name: "斤"}
	unit.ID = 1
	binding := model.MaterialUnit{MaterialID: 1, UnitID: 1, Factor: decimal.NewFromInt(1), IsBase: true, Enabled: true}
	binding.ID = 1
	state.brands[1], state.materials[1], state.categories[1], state.units[1], state.bindings[1] = brand, material, category, unit, binding
	state.names[4] = "创建人"
	return state
}

func deliveryInput(token, quantity, price string) req.CreateDeliveryInput {
	return req.CreateDeliveryInput{ReservationToken: token, DeliveredAt: time.Now(), BrandID: 1, Details: []req.DeliveryDetailInput{{MaterialID: 1, MaterialUnitID: 1, Quantity: quantity, UnitPrice: price}}}
}

func firstDetail(state *DeliveryState, noteID uint) uint {
	for id, detail := range state.details {
		if detail.NoteID == noteID {
			return id
		}
	}
	return 0
}

func TestDeliveryStateAtomicReviewAndReReviewOpinion(t *testing.T) {
	state := deliveryFixture(time.Now())
	reserved, _ := state.Reserve(4)
	in := deliveryInput(reserved.Token, "3", "99")
	amount := "10.00"
	in.Review, in.Details[0].Amount = true, &amount
	note, err := state.Create(4, in)
	if err != nil || note.Status != "reviewed" || note.Version != 1 || note.TotalAmount.StringFixed(2) != "10.00" {
		t.Fatalf("create=%#v err=%v", note, err)
	}
	detail := state.details[firstDetail(state, note.ID)]
	if detail.UnitPrice.StringFixed(4) != "3.3333" || detail.Amount.StringFixed(2) != amount {
		t.Fatalf("line=%#v", detail)
	}
	opinion := ""
	reReviewed, err := state.ReReview(4, req.ReReviewDeliveryInput{VersionedID: req.VersionedID{ID: note.ID}, Opinion: &opinion})
	if err != nil || reReviewed.Status != "reReviewed" || reReviewed.ReReviewOpinion == nil || *reReviewed.ReReviewOpinion != "" {
		t.Fatalf("empty opinion=%#v err=%v", reReviewed, err)
	}
	if _, err := state.Unreview(4, req.UnreviewDeliveryInput{VersionedID: req.VersionedID{ID: note.ID}}); err == nil {
		t.Fatal("reReviewed note unreviewed")
	}
	if _, err := state.CancelReReview(4, req.CancelReReviewDeliveryInput{VersionedID: req.VersionedID{ID: note.ID}}); err != nil {
		t.Fatal(err)
	}
	draft, err := state.Unreview(4, req.UnreviewDeliveryInput{VersionedID: req.VersionedID{ID: note.ID}})
	if err != nil { t.Fatal(err) }
	version := draft.Version
	updated, err := state.Update(4, req.UpdateDeliveryInput{VersionedID: req.VersionedID{ID: draft.ID, Version: &version}, DeliveredAt: draft.DeliveredAt, BrandID: 1, Details: in.Details, Review: true})
	if err != nil || updated.Status != "reviewed" || updated.Version != version+1 || updated.ReviewedBy == nil {
		t.Fatalf("update=%#v err=%v", updated, err)
	}
}
