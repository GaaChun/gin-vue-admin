package service

import (
	"strings"
	"time"
	"unicode"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	"github.com/mozillazg/go-pinyin"
)

// DeliveryStatus is the only persisted lifecycle state for a delivery note.
type DeliveryStatus string

const (
	// DeliveryDraft permits content changes, deletion, and review.
	DeliveryDraft DeliveryStatus = "draft"
	// DeliveryReviewed locks content until an explicit unreview action succeeds.
	DeliveryReviewed DeliveryStatus = "reviewed"
)

// DeliveryAction identifies a stateful delivery command.
type DeliveryAction string

const (
	// DeliveryEdit updates a draft note.
	DeliveryEdit DeliveryAction = "edit"
	// DeliveryDelete soft-deletes a draft note.
	DeliveryDelete DeliveryAction = "delete"
	// DeliveryReview transitions a draft note to reviewed.
	DeliveryReview DeliveryAction = "review"
	// DeliveryUnreview transitions a reviewed note back to draft.
	DeliveryUnreview DeliveryAction = "unreview"
)

// DeliveryCommandState is the in-memory command boundary shared by state and
// audit checks before a service persists the real note and its details.
type DeliveryCommandState struct {
	Audit  model.BusinessModel
	Status DeliveryStatus
}

// CheckVersion validates an explicitly provided optimistic-lock version.
func CheckVersion(current uint64, requested *uint64) error {
	if requested == nil {
		return nil
	}
	if *requested == 0 {
		return domainError(ValidationFailed, "版本号必须大于0", "version")
	}
	if *requested != current {
		return NewDomainError(VersionConflict, "记录已被其他操作修改").
			WithField("version").WithCurrentVersion(current)
	}
	return nil
}

// ValidateDeliveryAction rejects any action whose source state is not legal.
func ValidateDeliveryAction(status DeliveryStatus, action DeliveryAction) error {
	switch action {
	case DeliveryEdit, DeliveryDelete, DeliveryReview:
		if status == DeliveryDraft {
			return nil
		}
	case DeliveryUnreview:
		if status == DeliveryReviewed {
			return nil
		}
	}
	return NewDomainError(StateConflict, "当前状态不允许该操作").WithField("status")
}

// PrepareDeliveryCommand checks version and lifecycle state before changing a
// copied command state. On every error it returns the original state unchanged.
func PrepareDeliveryCommand(current DeliveryCommandState, requested *uint64, actorID uint, now time.Time, action DeliveryAction) (DeliveryCommandState, error) {
	if err := CheckVersion(current.Audit.Version, requested); err != nil {
		return current, err
	}
	if err := ValidateDeliveryAction(current.Status, action); err != nil {
		return current, err
	}

	next := current
	switch action {
	case DeliveryReview:
		next.Status = DeliveryReviewed
		ApplyReviewAudit(&next.Audit, actorID, now)
	case DeliveryUnreview:
		next.Status = DeliveryDraft
		ClearReviewAudit(&next.Audit, actorID)
	default:
		ApplyUpdateAudit(&next.Audit, actorID)
	}
	return next, nil
}

// ApplyCreateAudit establishes the only trusted initial audit state.
func ApplyCreateAudit(record *model.BusinessModel, actorID uint, now time.Time) {
	record.CreatedBy, record.UpdatedBy = actorID, actorID
	record.ReviewedBy, record.ReviewedAt = nil, nil
	record.DeletedBy, record.IsDeleted = nil, false
	record.Version = 1
}

// ApplyUpdateAudit increments once per successful command so concurrent edits stay detectable.
func ApplyUpdateAudit(record *model.BusinessModel, actorID uint) {
	record.UpdatedBy = actorID
	record.Version++
}

// ApplyReviewAudit records the reviewer and increments the version exactly once.
func ApplyReviewAudit(record *model.BusinessModel, actorID uint, now time.Time) {
	record.ReviewedBy = &actorID
	record.ReviewedAt = &now
	ApplyUpdateAudit(record, actorID)
}

// ClearReviewAudit removes current review attribution and increments the version once.
func ClearReviewAudit(record *model.BusinessModel, actorID uint) {
	record.ReviewedBy, record.ReviewedAt = nil, nil
	ApplyUpdateAudit(record, actorID)
}

// ApplyDeleteAudit records the deleting actor separately from the last editor.
func ApplyDeleteAudit(record *model.BusinessModel, actorID uint) {
	actor := actorID
	record.DeletedBy = &actor
	record.IsDeleted = true
	ApplyUpdateAudit(record, actorID)
}

// IdentifierReusePolicy distinguishes soft-deletable master identifiers from
// delivery note numbers, whose historical uniqueness is permanent.
type IdentifierReusePolicy string

const (
	// ReusableMasterIdentifier permits a code once every previous matching record is deleted.
	ReusableMasterIdentifier IdentifierReusePolicy = "reusable_master_identifier"
	// PermanentDeliveryNumber forbids reusing a number even after soft deletion.
	PermanentDeliveryNumber IdentifierReusePolicy = "permanent_delivery_number"
)

// IdentifierPresence summarizes matching records after applying the key's
// normalization rule. Historical includes active and soft-deleted records.
type IdentifierPresence struct {
	Active     bool
	Historical bool
}

// IdentifierRecord is the minimal persisted identity needed to inspect a
// normalized key while retaining the record ID for callers' historical links.
type IdentifierRecord struct {
	ID      uint
	Value   string
	Deleted bool
}

// InspectIdentifierPresence finds active and historical key matches using the
// same normalization rule used by active unique constraints.
func InspectIdentifierPresence(value string, records []IdentifierRecord) IdentifierPresence {
	normalized := NormalizeUniqueKey(value)
	var presence IdentifierPresence
	for _, record := range records {
		if NormalizeUniqueKey(record.Value) != normalized {
			continue
		}
		presence.Historical = true
		if !record.Deleted {
			presence.Active = true
		}
	}
	return presence
}

// ValidateIdentifierAvailability applies the identifier's reuse rule without
// coupling the rule to a database query or persistence model.
func ValidateIdentifierAvailability(policy IdentifierReusePolicy, presence IdentifierPresence, field string) error {
	switch policy {
	case ReusableMasterIdentifier:
		if !presence.Active {
			return nil
		}
		return NewDomainError(DuplicateValue, "存在未删除的相同编号").WithField(field)
	case PermanentDeliveryNumber:
		if !presence.Historical {
			return nil
		}
		return NewDomainError(DuplicateValue, "送货单号不可复用").WithField(field)
	default:
		return NewDomainError(ValidationFailed, "未知编号规则").WithField(field)
	}
}

// ReferenceUsage distinguishes an aggregate's own children from independent
// records that would retain an external reference after deletion.
type ReferenceUsage struct {
	InternalAggregateItems   uint
	ExternalActiveReferences uint
}

// ReferenceLink records a stable-ID reference. InternalAggregateItem is true
// only for a child that is deleted atomically with the target aggregate.
type ReferenceLink struct {
	TargetID              uint
	Deleted               bool
	InternalAggregateItem bool
}

// SummarizeReferenceUsage counts only non-deleted links to the exact target ID,
// so a new record reusing an old code never inherits the old record's references.
func SummarizeReferenceUsage(targetID uint, links []ReferenceLink) ReferenceUsage {
	var usage ReferenceUsage
	for _, link := range links {
		if link.Deleted || link.TargetID != targetID {
			continue
		}
		if link.InternalAggregateItem {
			usage.InternalAggregateItems++
			continue
		}
		usage.ExternalActiveReferences++
	}
	return usage
}

// CanDeleteAggregate permits an aggregate to soft-delete its own children, but
// rejects deletion while an active external reference exists.
func (usage ReferenceUsage) CanDeleteAggregate() error {
	if usage.ExternalActiveReferences == 0 {
		return nil
	}
	return NewDomainError(ReferenceInUse, "记录仍被有效业务引用").WithField("id")
}

// CanDeleteRecord rejects deletion of a standalone referenced record. Internal
// aggregate items count as references here because they cannot be cascaded.
func (usage ReferenceUsage) CanDeleteRecord() error {
	if usage.InternalAggregateItems == 0 && usage.ExternalActiveReferences == 0 {
		return nil
	}
	return NewDomainError(ReferenceInUse, "记录仍被有效业务引用").WithField("id")
}

// NormalizeUniqueKey matches the database's case-insensitive active-key semantics.
func NormalizeUniqueKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// SuggestMnemonic derives a lowercase initial-letter suggestion while leaving
// final ambiguity resolution to the user.
func SuggestMnemonic(name string) string {
	args := pinyin.NewArgs()
	args.Style = pinyin.FirstLetter
	args.Fallback = func(r rune, _ pinyin.Args) []string { return []string{string(r)} }
	parts := pinyin.Pinyin(strings.TrimSpace(name), args)
	var mnemonic strings.Builder
	for _, part := range parts {
		for _, r := range strings.Join(part, "") {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				mnemonic.WriteRune(unicode.ToLower(r))
			}
		}
	}
	return mnemonic.String()
}
