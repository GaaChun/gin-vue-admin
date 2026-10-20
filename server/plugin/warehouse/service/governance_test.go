package service

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	warehouseRequest "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
)

func TestCreateAuditDoesNotTrustDeliveryDTO(t *testing.T) {
	var input warehouseRequest.CreateDeliveryInput
	clientPayload := []byte(`{
		"reservationToken":"token",
		"deliveredAt":"2026-10-15T08:00:00Z",
		"brandId":3,
		"details":[{"materialId":4,"materialUnitId":5,"quantity":"1","unitPrice":"0","lineNo":1,"amount":"999","baseQuantity":"99"}],
		"createdAt":"2026-10-01T00:00:00Z",
		"updatedAt":"2026-10-02T00:00:00Z",
		"createdBy":99,
		"updatedBy":98,
		"reviewedBy":97,
		"version":99,
		"status":"reviewed",
		"number":"SK261015000001",
		"brandNameSnapshot":"伪造品牌",
		"totalAmount":"999"
	}`)
	if err := json.Unmarshal(clientPayload, &input); err != nil {
		t.Fatalf("decode delivery input: %v", err)
	}
	serialized, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("encode delivery input: %v", err)
	}
	for _, forbidden := range [][]byte{
		[]byte(`"createdAt"`), []byte(`"updatedAt"`), []byte(`"createdBy"`),
		[]byte(`"updatedBy"`), []byte(`"reviewedBy"`), []byte(`"version"`),
		[]byte(`"status"`), []byte(`"number"`), []byte(`"brandNameSnapshot"`),
		[]byte(`"lineNo"`), []byte(`"baseQuantity"`), []byte(`"amount"`),
		[]byte(`"totalAmount"`),
	} {
		if bytes.Contains(serialized, forbidden) {
			t.Fatalf("write DTO exposed server-owned field %s", forbidden)
		}
	}

	record := model.BusinessModel{CreatedBy: 99, UpdatedBy: 98, Version: 42}
	ApplyCreateAudit(&record, 7, time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC))
	if record.CreatedBy != 7 || record.UpdatedBy != 7 || record.Version != 1 {
		t.Fatal("create audit must be established from the authorized actor")
	}
	if record.ReviewedBy != nil || record.ReviewedAt != nil {
		t.Fatal("new delivery audit must not inherit a client reviewer")
	}
}

func TestApplyDeleteAuditRecordsDedicatedActorAndFlag(t *testing.T) {
	record := model.BusinessModel{UpdatedBy: 8, Version: 4}
	ApplyDeleteAudit(&record, 19)
	if record.DeletedBy == nil || *record.DeletedBy != 19 {
		t.Fatalf("deleted_by must record the authorized deleting actor: %#v", record.DeletedBy)
	}
	if !record.IsDeleted {
		t.Fatal("is_deleted must be true after deletion")
	}
	if record.UpdatedBy != 19 || record.Version != 5 {
		t.Fatalf("deletion must also update last editor and increment version: updated_by=%d version=%d", record.UpdatedBy, record.Version)
	}
}

func TestPrepareDeliveryCommandPreservesStateOnFailure(t *testing.T) {
	now := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	current := DeliveryCommandState{
		Audit:  model.BusinessModel{CreatedBy: 7, UpdatedBy: 7, Version: 3},
		Status: DeliveryDraft,
	}

	stale := uint64(2)
	unchanged, err := PrepareDeliveryCommand(current, &stale, 8, now, DeliveryReview)
	requireCommandState(t, unchanged, current)
	requireDomainCode(t, err, VersionConflict)
	if domain, ok := AsDomainError(err); !ok || domain.CurrentVersion != 3 ||
		domain.BusinessError().CurrentVersion != 3 {
		t.Fatal("version conflict must expose the authoritative current version")
	}

	zero := uint64(0)
	unchanged, err = PrepareDeliveryCommand(current, &zero, 8, now, DeliveryReview)
	requireCommandState(t, unchanged, current)
	requireDomainCode(t, err, ValidationFailed)

	next, err := PrepareDeliveryCommand(current, nil, 8, now, DeliveryEdit)
	if err != nil {
		t.Fatalf("missing version should permit a valid edit: %v", err)
	}
	if next.Audit.CreatedBy != 7 || next.Status != DeliveryDraft ||
		next.Audit.Version != 4 || next.Audit.UpdatedBy != 8 {
		t.Fatal("successful draft edit must preserve creator and increment once")
	}
	if current.Audit.Version != 3 || current.Audit.UpdatedBy != 7 {
		t.Fatal("successful command preparation must not mutate its input state")
	}
}

func TestDeliveryStateTransitionsRejectRepeatedActions(t *testing.T) {
	now := time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC)
	draft := DeliveryCommandState{
		Audit:  model.BusinessModel{CreatedBy: 7, UpdatedBy: 7, Version: 1},
		Status: DeliveryDraft,
	}
	reviewed, err := PrepareDeliveryCommand(draft, nil, 8, now, DeliveryReview)
	if err != nil {
		t.Fatalf("review draft: %v", err)
	}
	if reviewed.Status != DeliveryReviewed || reviewed.Audit.Version != 2 ||
		reviewed.Audit.ReviewedBy == nil || *reviewed.Audit.ReviewedBy != 8 ||
		reviewed.Audit.ReviewedAt == nil {
		t.Fatal("review must transition and establish reviewer audit")
	}

	unchanged, err := PrepareDeliveryCommand(reviewed, nil, 9, now, DeliveryReview)
	requireCommandState(t, unchanged, reviewed)
	requireDomainCode(t, err, StateConflict)

	draftAgain, err := PrepareDeliveryCommand(reviewed, nil, 9, now, DeliveryUnreview)
	if err != nil {
		t.Fatalf("unreview reviewed note: %v", err)
	}
	if draftAgain.Status != DeliveryDraft || draftAgain.Audit.Version != 3 ||
		draftAgain.Audit.ReviewedBy != nil || draftAgain.Audit.ReviewedAt != nil {
		t.Fatal("unreview must clear audit and return to draft")
	}

	unchanged, err = PrepareDeliveryCommand(draftAgain, nil, 9, now, DeliveryUnreview)
	requireCommandState(t, unchanged, draftAgain)
	requireDomainCode(t, err, StateConflict)
}

func TestIdentifierReuseAndReferenceProtection(t *testing.T) {
	records := []IdentifierRecord{{ID: 11, Value: "  Brand-A ", Deleted: true}}
	presence := InspectIdentifierPresence("brand-a", records)
	if presence.Active || !presence.Historical {
		t.Fatal("normalized soft-deleted code should be historical but inactive")
	}
	if err := ValidateIdentifierAvailability(ReusableMasterIdentifier, presence, "code"); err != nil {
		t.Fatalf("soft-deleted master code should be reusable: %v", err)
	}
	requireDomainCode(t, ValidateIdentifierAvailability(PermanentDeliveryNumber, presence, "number"), DuplicateValue)

	records = append(records, IdentifierRecord{ID: 12, Value: "BRAND-A"})
	requireDomainCode(t,
		ValidateIdentifierAvailability(ReusableMasterIdentifier, InspectIdentifierPresence(" brand-a ", records), "code"),
		DuplicateValue,
	)

	links := []ReferenceLink{
		{TargetID: 11},
		{TargetID: 12, InternalAggregateItem: true},
		{TargetID: 12, Deleted: true},
	}
	oldUsage := SummarizeReferenceUsage(11, links)
	requireDomainCode(t, oldUsage.CanDeleteAggregate(), ReferenceInUse)
	newUsage := SummarizeReferenceUsage(12, links)
	if err := newUsage.CanDeleteAggregate(); err != nil {
		t.Fatalf("internal material-unit binding must not block aggregate deletion: %v", err)
	}
	requireDomainCode(t, newUsage.CanDeleteRecord(), ReferenceInUse)
	if links[1].Deleted {
		t.Fatal("reference inspection must not modify input links")
	}
}

func TestNormalizeUniqueKeyAndMnemonic(t *testing.T) {
	if got := NormalizeUniqueKey("  Brand A  "); got != "brand a" {
		t.Fatalf("normalized %q", got)
	}
	if got := SuggestMnemonic("白 菜-A1"); got != "bca1" {
		t.Fatalf("mnemonic %q", got)
	}
}

func requireCommandState(t *testing.T, got, want DeliveryCommandState) {
	t.Helper()
	if got.Status != want.Status || got.Audit.CreatedBy != want.Audit.CreatedBy ||
		got.Audit.UpdatedBy != want.Audit.UpdatedBy || got.Audit.Version != want.Audit.Version ||
		got.Audit.ReviewedBy != want.Audit.ReviewedBy || got.Audit.ReviewedAt != want.Audit.ReviewedAt {
		t.Fatalf("command state changed on failure: got %#v want %#v", got, want)
	}
}

func requireDomainCode(t *testing.T, err error, want ErrorCode) {
	t.Helper()
	domain, ok := AsDomainError(err)
	if !ok || domain.Code != want {
		t.Fatalf("error code = %v, want %s", err, want)
	}
}
