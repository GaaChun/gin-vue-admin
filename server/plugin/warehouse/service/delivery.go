package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	req "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/request"
	resp "github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model/response"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DeliveryService persists numbering and delivery commands in one transaction.
type DeliveryService struct {
	db  *gorm.DB
	now func() time.Time
}

func NewDeliveryService(db *gorm.DB) *DeliveryService {
	return &DeliveryService{db: db, now: time.Now}
}

func DefaultDeliveryService() *DeliveryService { return NewDeliveryService(global.GVA_DB) }

type DeliveryDetail struct {
	Note    model.DeliveryNote
	Details []model.DeliveryDetail
}

func (s *DeliveryService) ReserveNumber(ctx context.Context, actor uint) (*model.NumberReservation, error) {
	var result model.NumberReservation
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		item, err := reserveNumber(tx, actor, s.now())
		result = item
		return err
	})
	return &result, err
}

func (s *DeliveryService) Create(ctx context.Context, actor uint, actorName string, in req.CreateDeliveryInput) (*DeliveryDetail, error) {
	var result DeliveryDetail
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var reservation model.NumberReservation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("token = ? AND owner_id = ?", in.ReservationToken, actor).First(&reservation).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return NewDomainError(StateConflict, "预占编号不可用").WithField("reservationToken")
			}
			return err
		}
		if reservation.ConsumedNoteID != nil {
			return NewDomainError(StateConflict, "预占编号已使用").WithField("reservationToken")
		}
		note, details, err := composePersisted(tx, 0, in.BrandID, in.DeliveredAt, in.Details, nil)
		if err != nil {
			return err
		}
		note.Department = in.Department
		note.Number = reservation.Number
		note.Status = string(DeliveryDraft)
		note.CreatedByNameSnapshot = actorName
		ApplyCreateAudit(&note.BusinessModel, actor, s.now())
		if err := tx.Create(&note).Error; err != nil {
			return err
		}
		for index := range details {
			details[index].NoteID = note.ID
			ApplyCreateAudit(&details[index].BusinessModel, actor, s.now())
		}
		if err := tx.Create(&details).Error; err != nil {
			return err
		}
		if err := tx.Model(&reservation).Update("consumed_note_id", note.ID).Error; err != nil {
			return err
		}
		result = DeliveryDetail{Note: note, Details: details}
		return nil
	})
	return &result, err
}

func (s *DeliveryService) Update(ctx context.Context, actor uint, in req.UpdateDeliveryInput) (*DeliveryDetail, error) {
	var result DeliveryDetail
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.DeliveryNote
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, in.ID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return notFound()
			}
			return err
		}
		next, err := PrepareDeliveryCommand(commandOf(current), in.Version, actor, s.now(), DeliveryEdit)
		if err != nil {
			return err
		}
		var existing []model.DeliveryDetail
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("note_id = ?", current.ID).Order("line_no").Find(&existing).Error; err != nil {
			return err
		}
		byID := map[uint]model.DeliveryDetail{}
		for _, detail := range existing {
			byID[detail.ID] = detail
		}
		note, details, err := composePersisted(tx, current.ID, in.BrandID, in.DeliveredAt, in.Details, byID)
		if err != nil {
			return err
		}
		note.Department = in.Department
		note.BusinessModel = next.Audit
		note.Number, note.Status, note.CreatedByNameSnapshot = current.Number, current.Status, current.CreatedByNameSnapshot
		if err := tx.Save(&note).Error; err != nil {
			return err
		}
		kept := map[uint]struct{}{}
		for index := range details {
			detail := &details[index]
			if detail.ID == 0 {
				detail.NoteID = note.ID
				ApplyCreateAudit(&detail.BusinessModel, actor, s.now())
				if err := tx.Create(detail).Error; err != nil {
					return err
				}
				continue
			}
			ApplyUpdateAudit(&detail.BusinessModel, actor)
			if err := tx.Save(detail).Error; err != nil {
				return err
			}
			kept[detail.ID] = struct{}{}
		}
		for _, detail := range existing {
			if _, ok := kept[detail.ID]; ok {
				continue
			}
			ApplyUpdateAudit(&detail.BusinessModel, actor)
			if err := tx.Save(&detail).Error; err != nil {
				return err
			}
			if err := tx.Delete(&detail).Error; err != nil {
				return err
			}
		}
		result = DeliveryDetail{Note: note, Details: details}
		return nil
	})
	return &result, err
}

func (s *DeliveryService) Delete(ctx context.Context, actor uint, in req.DeleteInput) error {
	return s.transition(ctx, actor, in.VersionedID, DeliveryDelete, "")
}

func (s *DeliveryService) Review(ctx context.Context, actor uint, reviewer string, in req.ReviewDeliveryInput) (*model.DeliveryNote, error) {
	return s.transitionNote(ctx, actor, in.VersionedID, DeliveryReview, reviewer)
}

func (s *DeliveryService) Unreview(ctx context.Context, actor uint, in req.UnreviewDeliveryInput) (*model.DeliveryNote, error) {
	return s.transitionNote(ctx, actor, in.VersionedID, DeliveryUnreview, "")
}

func (s *DeliveryService) transition(ctx context.Context, actor uint, id req.VersionedID, action DeliveryAction, reviewer string) error {
	_, err := s.transitionNote(ctx, actor, id, action, reviewer)
	return err
}

func (s *DeliveryService) transitionNote(ctx context.Context, actor uint, id req.VersionedID, action DeliveryAction, reviewer string) (*model.DeliveryNote, error) {
	var result model.DeliveryNote
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&result, id.ID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return notFound()
			}
			return err
		}
		next, err := PrepareDeliveryCommand(commandOf(result), id.Version, actor, s.now(), action)
		if err != nil {
			return err
		}
		result.BusinessModel = next.Audit
		result.Status = string(next.Status)
		if action == DeliveryReview {
			result.ReviewedByNameSnapshot = stringPointer(reviewer)
		}
		if action == DeliveryUnreview {
			result.ReviewedByNameSnapshot = nil
		}
		if err := tx.Save(&result).Error; err != nil {
			return err
		}
		if action == DeliveryDelete {
			if err := tx.Where("note_id = ?", result.ID).Delete(&model.DeliveryDetail{}).Error; err != nil {
				return err
			}
			return tx.Delete(&result).Error
		}
		return nil
	})
	return &result, err
}

func (s *DeliveryService) Detail(ctx context.Context, id uint) (*DeliveryDetail, error) {
	var result DeliveryDetail
	if err := s.db.WithContext(ctx).First(&result.Note, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, notFound()
		}
		return nil, err
	}
	return &result, s.db.WithContext(ctx).Where("note_id = ?", id).Order("line_no").Find(&result.Details).Error
}

func (s *DeliveryService) List(ctx context.Context, query req.DeliveryListQuery) (resp.Page[model.DeliveryNote], error) {
	query.PageQuery = query.PageQuery.WithDefaults()
	result := resp.Page[model.DeliveryNote]{Page: query.Page, PageSize: query.PageSize}
	db := s.db.WithContext(ctx).Model(&model.DeliveryNote{})
	if query.BrandID != nil {
		db = db.Where("brand_id = ?", *query.BrandID)
	}
	if query.Department != "" {
		db = db.Where("department = ?", query.Department)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Number != "" {
		db = db.Where("number = ?", query.Number)
	}
	if query.DeliveredFrom != nil {
		db = db.Where("delivered_at >= ?", *query.DeliveredFrom)
	}
	if query.DeliveredTo != nil {
		db = db.Where("delivered_at < ?", *query.DeliveredTo)
	}
	if err := db.Count(&result.Total).Error; err != nil {
		return result, err
	}
	err := db.Order("delivered_at desc, id desc").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&result.List).Error
	return result, err
}

// Departments returns distinct, nonblank department values from delivery notes.
func (s *DeliveryService) Departments(ctx context.Context) ([]string, error) {
	var departments []string
	err := s.db.WithContext(ctx).Model(&model.DeliveryNote{}).
		Distinct("department").Where("department <> ''").Order("department").Pluck("department", &departments).Error
	if err != nil {
		return nil, err
	}
	nonblank := departments[:0]
	for _, department := range departments {
		if strings.TrimSpace(department) != "" {
			nonblank = append(nonblank, department)
		}
	}
	return nonblank, nil
}

func reserveNumber(tx *gorm.DB, actor uint, now time.Time) (model.NumberReservation, error) {
	date := businessDate(now)
	counter := model.NumberCounter{BusinessDate: date}
	if err := tx.Where("business_date = ?", date).Attrs(counter).FirstOrCreate(&counter).Error; err != nil {
		return model.NumberReservation{}, err
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&counter, counter.ID).Error; err != nil {
		return model.NumberReservation{}, err
	}
	if counter.LastValue == 999999 {
		return model.NumberReservation{}, NewDomainError(NumberExhausted, "当日送货单号已用尽")
	}
	counter.LastValue++
	ApplyUpdateAudit(&counter.BusinessModel, actor)
	if err := tx.Save(&counter).Error; err != nil {
		return model.NumberReservation{}, err
	}
	item := model.NumberReservation{Number: formatDeliveryNumber(date, counter.LastValue), Token: randomToken(), OwnerID: actor}
	ApplyCreateAudit(&item.BusinessModel, actor, now)
	return item, tx.Create(&item).Error
}

func composePersisted(tx *gorm.DB, noteID, brandID uint, delivered time.Time, inputs []req.DeliveryDetailInput, existing map[uint]model.DeliveryDetail) (model.DeliveryNote, []model.DeliveryDetail, error) {
	state := NewDeliveryState(time.Now())
	if noteID != 0 {
		var current model.DeliveryNote
		if err := tx.First(&current, noteID).Error; err != nil {
			return model.DeliveryNote{}, nil, err
		}
		state.notes[noteID] = current
	}
	var brand model.Brand
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&brand, brandID).Error; err != nil {
		return model.DeliveryNote{}, nil, domainError(ValidationFailed, "客户不存在", "brandId")
	}
	state.brands[brand.ID] = brand
	for _, input := range inputs {
		var material model.Material
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&material, input.MaterialID).Error; err == nil {
			state.materials[material.ID] = material
		}
		var binding model.MaterialUnit
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&binding, input.MaterialUnitID).Error; err == nil {
			state.bindings[binding.ID] = binding
		}
	}
	for _, material := range state.materials {
		var category model.Category
		if err := tx.First(&category, material.CategoryID).Error; err == nil {
			state.categories[category.ID] = category
		}
		var bindings []model.MaterialUnit
		if err := tx.Where("material_id = ?", material.ID).Find(&bindings).Error; err == nil {
			for _, binding := range bindings {
				state.bindings[binding.ID] = binding
			}
		}
	}
	for _, binding := range state.bindings {
		var unit model.Unit
		if err := tx.First(&unit, binding.UnitID).Error; err == nil {
			state.units[unit.ID] = unit
		}
	}
	return state.compose(noteID, brandID, delivered, inputs, existing)
}

func formatDeliveryNumber(date time.Time, value uint) string {
	return fmt.Sprintf("SK%s%06d", date.Format("060102"), value)
}
