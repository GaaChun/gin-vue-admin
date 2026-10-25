package service

import (
	"context"
	"fmt"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MasterCodeSuggestion struct {
	Code string `json:"code"`
}

// ReserveCode consumes a durable sequence value when the new-record editor opens.
// Cancelled editors may leave gaps; neither restart nor deletion reuses a value.
func (s *MasterService) ReserveCode(ctx context.Context, actor uint, resource string) (MasterCodeSuggestion, error) {
	var prefix string
	var target interface{}
	switch resource {
	case "brands":
		prefix, target = "KH", &model.Brand{}
	case "categories":
		prefix, target = "LB", &model.Category{}
	case "materials":
		prefix, target = "WL", &model.Material{}
	default:
		return MasterCodeSuggestion{}, domainError(ValidationFailed, "该资料不支持自动编号", "resource")
	}
	var result MasterCodeSuggestion
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		counter := model.MasterCodeCounter{Resource: resource}
		ApplyCreateAudit(&counter.BusinessModel, actor, time.Now())
		// The unique resource key and upsert serialize simultaneous first allocations.
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&counter).Error; err != nil {
			return err
		}
		counter = model.MasterCodeCounter{}
		if err := tx.Unscoped().Clauses(clause.Locking{Strength: "UPDATE"}).Where("resource = ?", resource).First(&counter).Error; err != nil {
			return err
		}
		for {
			if counter.LastValue >= 999999 {
				return NewDomainError(NumberExhausted, "基础资料编号流水已用尽")
			}
			counter.LastValue++
			code := fmt.Sprintf("%s%06d", prefix, counter.LastValue)
			var count int64
			// Skip manual/imported codes, including deleted history, without max+1.
			if err := tx.Unscoped().Model(target).Where("LOWER(TRIM(code)) = ?", NormalizeUniqueKey(code)).Count(&count).Error; err != nil {
				return err
			}
			if count != 0 {
				continue
			}
			ApplyUpdateAudit(&counter.BusinessModel, actor)
			if err := tx.Unscoped().Save(&counter).Error; err != nil {
				return err
			}
			result.Code = code
			return nil
		}
	})
	return result, dbError(err)
}
