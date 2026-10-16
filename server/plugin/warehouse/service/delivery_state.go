package service

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	"github.com/shopspring/decimal"
)

// DeliveryState models numbering and note commands without claiming database lock proof.
type DeliveryState struct {
	nextID       uint
	now          time.Time
	counters     map[string]uint
	reservations map[string]model.NumberReservation
	notes        map[uint]model.DeliveryNote
	details      map[uint]model.DeliveryDetail
	brands       map[uint]model.Brand
	materials    map[uint]model.Material
	units        map[uint]model.Unit
	categories   map[uint]model.Category
	bindings     map[uint]model.MaterialUnit
	names        map[uint]string
}

func NewDeliveryState(now time.Time) *DeliveryState {
	return &DeliveryState{nextID: 1, now: now, counters: map[string]uint{}, reservations: map[string]model.NumberReservation{}, notes: map[uint]model.DeliveryNote{}, details: map[uint]model.DeliveryDetail{}, brands: map[uint]model.Brand{}, materials: map[uint]model.Material{}, units: map[uint]model.Unit{}, categories: map[uint]model.Category{}, bindings: map[uint]model.MaterialUnit{}, names: map[uint]string{}}
}

func (s *DeliveryState) Reserve(owner uint) (model.NumberReservation, error) {
	date := businessDate(s.now)
	key := date.Format("2006-01-02")
	if s.counters[key] == 999999 {
		return model.NumberReservation{}, NewDomainError(NumberExhausted, "当日送货单号已用尽")
	}
	s.counters[key]++
	number := fmt.Sprintf("SK%s%06d", date.Format("060102"), s.counters[key])
	token := randomToken()
	item := model.NumberReservation{Number: number, Token: token, OwnerID: owner}
	item.ID = s.allocate()
	ApplyCreateAudit(&item.BusinessModel, owner, s.now)
	s.reservations[token] = item
	return item, nil
}

func (s *DeliveryState) Create(actor uint, in req.CreateDeliveryInput) (model.DeliveryNote, error) {
	working := s.clone()
	reservation, ok := working.reservations[in.ReservationToken]
	if !ok || reservation.OwnerID != actor || reservation.ConsumedNoteID != nil {
		return model.DeliveryNote{}, NewDomainError(StateConflict, "预占编号不可用").WithField("reservationToken")
	}
	note, details, err := working.compose(0, in.BrandID, in.DeliveredAt, in.Details, nil)
	if err != nil {
		return model.DeliveryNote{}, err
	}
	note.ID = working.allocate()
	note.Number = reservation.Number
	note.Status = string(DeliveryDraft)
	note.CreatedByNameSnapshot = working.names[actor]
	ApplyCreateAudit(&note.BusinessModel, actor, working.now)
	for index := range details {
		details[index].ID = working.allocate()
		details[index].NoteID = note.ID
		ApplyCreateAudit(&details[index].BusinessModel, actor, working.now)
		working.details[details[index].ID] = details[index]
	}
	consumed := note.ID
	reservation.ConsumedNoteID = &consumed
	working.reservations[reservation.Token] = reservation
	working.notes[note.ID] = note
	*s = *working
	return note, nil
}

func (s *DeliveryState) Update(actor uint, in req.UpdateDeliveryInput) (model.DeliveryNote, error) {
	current, ok := s.notes[in.ID]
	if !ok {
		return model.DeliveryNote{}, notFound()
	}
	if _, err := PrepareDeliveryCommand(commandOf(current), in.Version, actor, s.now, DeliveryEdit); err != nil {
		return model.DeliveryNote{}, err
	}
	working := s.clone()
	existing := working.detailsFor(current.ID)
	note, details, err := working.compose(current.ID, in.BrandID, in.DeliveredAt, in.Details, existing)
	if err != nil {
		return model.DeliveryNote{}, err
	}
	note.BusinessModel = current.BusinessModel
	note.Number = current.Number
	note.Status = current.Status
	note.CreatedByNameSnapshot = current.CreatedByNameSnapshot
	ApplyUpdateAudit(&note.BusinessModel, actor)
	working.replaceDetails(note.ID, actor, existing, details)
	working.notes[note.ID] = note
	*s = *working
	return note, nil
}

func (s *DeliveryState) Review(actor uint, in req.ReviewDeliveryInput, reviewer string) (model.DeliveryNote, error) {
	return s.transition(actor, in.VersionedID, DeliveryReview, reviewer)
}

func (s *DeliveryState) Unreview(actor uint, in req.UnreviewDeliveryInput) (model.DeliveryNote, error) {
	return s.transition(actor, in.VersionedID, DeliveryUnreview, "")
}

func (s *DeliveryState) transition(actor uint, id req.VersionedID, action DeliveryAction, reviewer string) (model.DeliveryNote, error) {
	current, ok := s.notes[id.ID]
	if !ok {
		return model.DeliveryNote{}, notFound()
	}
	next, err := PrepareDeliveryCommand(commandOf(current), id.Version, actor, s.now, action)
	if err != nil {
		return model.DeliveryNote{}, err
	}
	current.Status = string(next.Status)
	current.BusinessModel = next.Audit
	if action == DeliveryReview {
		current.ReviewedByNameSnapshot = stringPointer(reviewer)
	}
	if action == DeliveryUnreview {
		current.ReviewedByNameSnapshot = nil
	}
	s.notes[current.ID] = current
	return current, nil
}

func (s *DeliveryState) compose(noteID, brandID uint, delivered time.Time, inputs []req.DeliveryDetailInput, existing map[uint]model.DeliveryDetail) (model.DeliveryNote, []model.DeliveryDetail, error) {
	if len(inputs) == 0 {
		return model.DeliveryNote{}, nil, domainError(ValidationFailed, "至少需要一条完整明细", "details")
	}
	brand, ok := s.brands[brandID]
	if !ok {
		return model.DeliveryNote{}, nil, domainError(ValidationFailed, "品牌不存在", "brandId")
	}
	note := model.DeliveryNote{DeliveredAt: delivered.UTC(), BrandID: brand.ID, BrandCodeSnapshot: brand.Code, BrandNameSnapshot: brand.Name, Status: string(DeliveryDraft)}
	if previous, found := s.notes[noteID]; found && previous.BrandID == brandID {
		note.BrandCodeSnapshot, note.BrandNameSnapshot = previous.BrandCodeSnapshot, previous.BrandNameSnapshot
	}
	lines := make([]LineAmounts, 0, len(inputs))
	details := make([]model.DeliveryDetail, 0, len(inputs))
	for index, input := range inputs {
		detail, err := s.composeDetail(noteID, brand.ID, uint(index+1), input, existing)
		if err != nil {
			if domain, ok := AsDomainError(err); ok {
				domain.LineNo = index + 1
			}
			return model.DeliveryNote{}, nil, err
		}
		lines = append(lines, LineAmounts{BaseQuantity: detail.BaseQuantity, Amount: detail.Amount})
		details = append(details, detail)
	}
	note.TotalAmount = SumAmounts(lines)
	if noteID != 0 {
		note.ID = noteID
	}
	return note, details, nil
}

func (s *DeliveryState) composeDetail(noteID, brandID, line uint, input req.DeliveryDetailInput, existing map[uint]model.DeliveryDetail) (model.DeliveryDetail, error) {
	quantity, err := ParseDecimal(input.Quantity, QuantityRule, "quantity")
	if err != nil {
		return model.DeliveryDetail{}, err
	}
	price, err := ParseDecimal(input.UnitPrice, PriceRule, "unitPrice")
	if err != nil {
		return model.DeliveryDetail{}, err
	}
	if input.ID != nil {
		old, ok := existing[*input.ID]
		if !ok || old.NoteID != noteID {
			return model.DeliveryDetail{}, domainError(ValidationFailed, "明细不属于当前送货单", "details")
		}
		if old.MaterialID == input.MaterialID && old.MaterialUnitID == input.MaterialUnitID && s.notes[noteID].BrandID == brandID {
			amounts, err := CalculateLine(quantity, price, old.Factor)
			if err != nil {
				return model.DeliveryDetail{}, err
			}
			old.Quantity, old.UnitPrice, old.BaseQuantity, old.Amount = quantity, price, amounts.BaseQuantity, amounts.Amount
			old.LineNo = line
			return old, nil
		}
	}
	material, ok := s.materials[input.MaterialID]
	if !ok || material.BrandID != brandID {
		return model.DeliveryDetail{}, domainError(ValidationFailed, "物料品牌与送货单不一致", "details")
	}
	binding, ok := s.bindings[input.MaterialUnitID]
	if !ok || binding.MaterialID != material.ID || !binding.Enabled {
		return model.DeliveryDetail{}, domainError(ValidationFailed, "物料单位不可用", "details")
	}
	category := s.categories[material.CategoryID]
	unit := s.units[binding.UnitID]
	base := s.baseBinding(material.ID)
	amounts, err := CalculateLine(quantity, price, binding.Factor)
	if err != nil {
		return model.DeliveryDetail{}, err
	}
	detail := model.DeliveryDetail{NoteID: noteID, LineNo: line, MaterialID: material.ID, MaterialUnitID: binding.ID, UnitID: binding.UnitID, BaseUnitID: base.UnitID, CategoryID: category.ID, MaterialCodeSnapshot: material.Code, MaterialNameSnapshot: material.Name, SpecificationSnapshot: material.Specification, CategoryCodeSnapshot: category.Code, CategoryNameSnapshot: category.Name, UnitNameSnapshot: unit.Name, BaseUnitNameSnapshot: s.units[base.UnitID].Name, Factor: binding.Factor, Quantity: quantity, UnitPrice: price, BaseQuantity: amounts.BaseQuantity, Amount: amounts.Amount}
	if input.ID != nil {
		detail.ID = *input.ID
		detail.BusinessModel = existing[*input.ID].BusinessModel
	}
	return detail, nil
}

func (s *DeliveryState) replaceDetails(noteID, actor uint, existing map[uint]model.DeliveryDetail, submitted []model.DeliveryDetail) {
	kept := map[uint]struct{}{}
	for _, detail := range submitted {
		if detail.ID == 0 {
			detail.ID = s.allocate()
			detail.NoteID = noteID
			ApplyCreateAudit(&detail.BusinessModel, actor, s.now)
		} else {
			ApplyUpdateAudit(&detail.BusinessModel, actor)
			kept[detail.ID] = struct{}{}
		}
		s.details[detail.ID] = detail
	}
	for id := range existing {
		if _, ok := kept[id]; !ok {
			delete(s.details, id)
		}
	}
}

func (s *DeliveryState) detailsFor(noteID uint) map[uint]model.DeliveryDetail {
	result := map[uint]model.DeliveryDetail{}
	for id, detail := range s.details {
		if detail.NoteID == noteID {
			result[id] = detail
		}
	}
	return result
}

func (s *DeliveryState) baseBinding(materialID uint) model.MaterialUnit {
	for _, binding := range s.bindings {
		if binding.MaterialID == materialID && binding.IsBase {
			return binding
		}
	}
	return model.MaterialUnit{}
}

func (s *DeliveryState) clone() *DeliveryState {
	next := NewDeliveryState(s.now)
	next.nextID = s.nextID
	for key, value := range s.counters {
		next.counters[key] = value
	}
	for key, value := range s.reservations {
		next.reservations[key] = value
	}
	for key, value := range s.notes {
		next.notes[key] = value
	}
	for key, value := range s.details {
		next.details[key] = value
	}
	next.brands, next.materials, next.units, next.categories, next.bindings, next.names = s.brands, s.materials, s.units, s.categories, s.bindings, s.names
	return next
}

func (s *DeliveryState) allocate() uint { id := s.nextID; s.nextID++; return id }

func commandOf(note model.DeliveryNote) DeliveryCommandState {
	return DeliveryCommandState{Audit: note.BusinessModel, Status: DeliveryStatus(note.Status)}
}

func businessDate(now time.Time) time.Time {
	local := now.In(time.FixedZone("Asia/Shanghai", 8*60*60))
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, local.Location())
}

func randomToken() string {
	var value [16]byte
	_, _ = rand.Read(value[:])
	return fmt.Sprintf("%x", value)
}

func decimalValue(value string) decimal.Decimal {
	parsed, _ := decimal.NewFromString(value)
	return parsed
}
