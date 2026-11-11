package model

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type Entitlement struct {
	Id                 int     `json:"id"`
	UserId             int     `json:"user_id" gorm:"index;index:idx_entitlement_candidate,priority:1;not null"`
	EntitlementTypeId  int     `json:"entitlement_type_id" gorm:"index;not null"`
	AssetKind          string  `json:"asset_kind" gorm:"type:varchar(32);index;index:idx_entitlement_candidate,priority:2;not null"`
	ProductId          int     `json:"product_id" gorm:"index"`
	SKUId              int     `json:"sku_id" gorm:"index"`
	OrderItemId        int     `json:"order_item_id" gorm:"index"`
	QuantityIndex      int     `json:"quantity_index"`
	FulfillmentKey     *string `json:"fulfillment_key" gorm:"type:varchar(128);uniqueIndex"`
	State              string  `json:"state" gorm:"type:varchar(32);index;index:idx_entitlement_candidate,priority:3;not null"`
	TotalQuota         int64   `json:"total_quota" gorm:"bigint;not null"`
	UsedQuota          int64   `json:"used_quota" gorm:"bigint;not null"`
	ReservedQuota      int64   `json:"reserved_quota" gorm:"bigint;not null"`
	DailyQuota         int64   `json:"daily_quota" gorm:"bigint;not null"`
	ResetTimezone      string  `json:"reset_timezone" gorm:"type:varchar(64);not null"`
	StartAt            int64   `json:"start_at" gorm:"bigint;index"`
	ExpireAt           int64   `json:"expire_at" gorm:"bigint;index"`
	ActivationDeadline int64   `json:"activation_deadline" gorm:"bigint"`
	SortOrder          int     `json:"sort_order" gorm:"index"`
	SourceType         string  `json:"source_type" gorm:"type:varchar(32);index"`
	SourceId           int     `json:"source_id" gorm:"index"`
	FulfillmentBatchId string  `json:"fulfillment_batch_id" gorm:"type:varchar(64);index"`
	CreatedAt          int64   `json:"created_at" gorm:"bigint"`
	UpdatedAt          int64   `json:"updated_at" gorm:"bigint"`
}

func (e *Entitlement) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if e.State == "" {
		e.State = EntitlementStateActive
	}
	if e.ResetTimezone == "" {
		e.ResetTimezone = "Asia/Shanghai"
	}
	e.CreatedAt, e.UpdatedAt = now, now
	return nil
}

func (e *Entitlement) BeforeUpdate(_ *gorm.DB) error { e.UpdatedAt = common.GetTimestamp(); return nil }

func (e *Entitlement) AvailableQuota() int64 {
	remaining := e.TotalQuota - e.UsedQuota - e.ReservedQuota
	if remaining < 0 {
		return 0
	}
	return remaining
}

type EntitlementDailyUsage struct {
	Id            int   `json:"id"`
	EntitlementId int   `json:"entitlement_id" gorm:"uniqueIndex:idx_entitlement_daily_period;index;not null"`
	PeriodStart   int64 `json:"period_start" gorm:"bigint;uniqueIndex:idx_entitlement_daily_period;index;not null"`
	PeriodEnd     int64 `json:"period_end" gorm:"bigint;index;not null"`
	UsedQuota     int64 `json:"used_quota" gorm:"bigint;not null"`
	ReservedQuota int64 `json:"reserved_quota" gorm:"bigint;not null"`
	CreatedAt     int64 `json:"created_at" gorm:"bigint"`
	UpdatedAt     int64 `json:"updated_at" gorm:"bigint"`
}

func (u *EntitlementDailyUsage) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	u.CreatedAt, u.UpdatedAt = now, now
	return nil
}

func (u *EntitlementDailyUsage) BeforeUpdate(_ *gorm.DB) error {
	u.UpdatedAt = common.GetTimestamp()
	return nil
}

func (u *EntitlementDailyUsage) AvailableQuota(limit int64) int64 {
	if limit <= 0 {
		return int64(^uint64(0) >> 1)
	}
	remaining := limit - u.UsedQuota - u.ReservedQuota
	if remaining < 0 {
		return 0
	}
	return remaining
}

type EntitlementAdjustmentLedger struct {
	Id             int    `json:"id"`
	IdempotencyKey string `json:"idempotency_key" gorm:"type:varchar(128);uniqueIndex;not null"`
	EntitlementId  int    `json:"entitlement_id" gorm:"index;not null"`
	OperatorId     int    `json:"operator_id" gorm:"index"`
	DeltaQuota     int64  `json:"delta_quota" gorm:"bigint;not null"`
	Reason         string `json:"reason" gorm:"type:varchar(255);not null"`
	CreatedAt      int64  `json:"created_at" gorm:"bigint"`
}

func (l *EntitlementAdjustmentLedger) BeforeCreate(_ *gorm.DB) error {
	l.CreatedAt = common.GetTimestamp()
	return nil
}

func entitlementDailyPeriod(now time.Time, timezone string) (int64, int64) {
	location, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil {
		location = time.UTC
	}
	local := now.In(location)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	return start.Unix(), start.AddDate(0, 0, 1).Unix()
}

func refreshUserEntitlementStatesTx(tx *gorm.DB, userId int, now int64) error {
	if userId <= 0 {
		return nil
	}
	if err := tx.Model(&Entitlement{}).
		Where("user_id = ? AND state IN ? AND expire_at > 0 AND expire_at <= ? AND reserved_quota = 0", userId, []string{EntitlementStateActive, EntitlementStatePaused, EntitlementStateQueued}, now).
		Update("state", EntitlementStateExpired).Error; err != nil {
		return err
	}
	if err := tx.Model(&Entitlement{}).
		Where("user_id = ? AND state = ? AND activation_deadline > 0 AND activation_deadline <= ?", userId, EntitlementStatePending, now).
		Update("state", EntitlementStateExpired).Error; err != nil {
		return err
	}
	return tx.Model(&Entitlement{}).
		Where("user_id = ? AND state = ? AND start_at > 0 AND start_at <= ? AND (expire_at = 0 OR expire_at > ?)", userId, EntitlementStateQueued, now, now).
		Update("state", EntitlementStateActive).Error
}

func RefreshUserEntitlementStates(userId int) error {
	if userId <= 0 {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		return refreshUserEntitlementStatesTx(tx, userId, getDBTimestampTx(tx))
	})
}

func RefreshDueEntitlementStates(limit int) (int, error) {
	if limit <= 0 {
		return 0, nil
	}
	var userIds []int
	now := GetDBTimestamp()
	err := DB.Model(&Entitlement{}).
		Distinct("user_id").
		Where("(state IN ? AND expire_at > 0 AND expire_at <= ? AND reserved_quota = 0) OR (state = ? AND activation_deadline > 0 AND activation_deadline <= ?) OR (state = ? AND start_at > 0 AND start_at <= ? AND (expire_at = 0 OR expire_at > ?))",
			[]string{EntitlementStateActive, EntitlementStatePaused, EntitlementStateQueued}, now,
			EntitlementStatePending, now,
			EntitlementStateQueued, now, now).
		Order("user_id asc").Limit(limit).Pluck("user_id", &userIds).Error
	if err != nil {
		return 0, err
	}
	updatedUsers := 0
	for _, userId := range userIds {
		if err := RefreshUserEntitlementStates(userId); err != nil {
			return updatedUsers, err
		}
		updatedUsers++
	}
	return updatedUsers, nil
}

func GetUserEntitlementGroups(userId int) ([]string, error) {
	if userId <= 0 {
		return nil, nil
	}
	if err := RefreshUserEntitlementStates(userId); err != nil {
		return nil, err
	}
	now := GetDBTimestamp()
	var groups []string
	err := DB.Model(&Entitlement{}).
		Distinct("entitlement_type_groups.group_name").
		Joins("JOIN entitlement_types ON entitlement_types.id = entitlements.entitlement_type_id").
		Joins("JOIN entitlement_type_groups ON entitlement_type_groups.entitlement_type_id = entitlement_types.id").
		Joins("JOIN access_group_policies ON access_group_policies.id = entitlement_type_groups.access_group_policy_id").
		Where("entitlements.user_id = ?", userId).
		Where("entitlements.state = ?", EntitlementStateActive).
		Where("entitlements.start_at <= ?", now).
		Where("(entitlements.expire_at = 0 OR entitlements.expire_at > ?)", now).
		Where("entitlements.used_quota < entitlements.total_quota").
		Where("entitlement_types.status = ?", EntitlementTypeStatusActive).
		Where("entitlements.asset_kind = access_group_policies.funding_source_type").
		Order("entitlement_type_groups.group_name asc").
		Pluck("entitlement_type_groups.group_name", &groups).Error
	return groups, err
}

func UserCanUseEntitlementGroup(userId int, groupName string) (bool, error) {
	if userId <= 0 || strings.TrimSpace(groupName) == "" {
		return false, nil
	}
	if err := RefreshUserEntitlementStates(userId); err != nil {
		return false, err
	}
	now := GetDBTimestamp()
	var count int64
	err := DB.Model(&Entitlement{}).
		Joins("JOIN entitlement_types ON entitlement_types.id = entitlements.entitlement_type_id").
		Joins("JOIN entitlement_type_groups ON entitlement_type_groups.entitlement_type_id = entitlement_types.id").
		Joins("JOIN access_group_policies ON access_group_policies.id = entitlement_type_groups.access_group_policy_id").
		Where("entitlements.user_id = ?", userId).
		Where("entitlements.state = ?", EntitlementStateActive).
		Where("entitlements.start_at <= ?", now).
		Where("(entitlements.expire_at = 0 OR entitlements.expire_at > ?)", now).
		Where("entitlements.used_quota < entitlements.total_quota").
		Where("entitlement_types.status = ?", EntitlementTypeStatusActive).
		Where("entitlement_type_groups.group_name = ?", strings.TrimSpace(groupName)).
		Where("entitlements.asset_kind = access_group_policies.funding_source_type").
		Limit(1).
		Count(&count).Error
	return count > 0, err
}

func GrantProductSKUEntitlement(userId int, skuId int, operatorId int) (*Entitlement, error) {
	if userId <= 0 || skuId <= 0 {
		return nil, errors.New("invalid user or SKU id")
	}
	var entitlement *Entitlement
	err := DB.Transaction(func(tx *gorm.DB) error {
		var sku ProductSKU
		if err := lockForUpdate(tx).Where("id = ? AND deleted_at = 0", skuId).First(&sku).Error; err != nil {
			return err
		}
		var product Product
		if err := tx.Where("id = ?", sku.ProductId).First(&product).Error; err != nil {
			return err
		}
		var entitlementType EntitlementType
		if err := tx.Where("id = ?", sku.EntitlementTypeId).First(&entitlementType).Error; err != nil {
			return err
		}
		if product.Category != ProductCategorySubscription || entitlementType.AssetKind != EntitlementAssetSubscription {
			return errors.New("SKU is not a subscription")
		}
		if product.Status != ProductStatusActive || sku.Status != ProductStatusActive {
			return errors.New("subscription SKU is not active")
		}
		if sku.ValiditySeconds <= 0 || sku.GrantTotalQuota <= 0 {
			return errors.New("subscription SKU configuration is invalid")
		}

		now := getDBTimestampTx(tx)
		state := EntitlementStateActive
		startAt := now
		expireAt := now + sku.ValiditySeconds
		activationDeadline := int64(0)
		switch sku.ActivationPolicy {
		case ActivationPolicyManual:
			state, startAt, expireAt = EntitlementStatePending, 0, 0
			if sku.ActivationDeadlineSec > 0 {
				activationDeadline = now + sku.ActivationDeadlineSec
			}
		case ActivationPolicyDeferred:
			var lastExpire int64
			if err := tx.Model(&Entitlement{}).
				Where("user_id = ? AND entitlement_type_id = ? AND state IN ?", userId, sku.EntitlementTypeId, []string{EntitlementStateActive, EntitlementStatePaused, EntitlementStateQueued}).
				Select("COALESCE(MAX(expire_at), 0)").Scan(&lastExpire).Error; err != nil {
				return err
			}
			if lastExpire > startAt {
				startAt, state = lastExpire, EntitlementStateQueued
			}
			expireAt = startAt + sku.ValiditySeconds
		}

		entitlement = &Entitlement{
			UserId:             userId,
			EntitlementTypeId:  sku.EntitlementTypeId,
			AssetKind:          EntitlementAssetSubscription,
			ProductId:          product.Id,
			SKUId:              sku.Id,
			State:              state,
			TotalQuota:         sku.GrantTotalQuota,
			DailyQuota:         sku.GrantDailyQuota,
			ResetTimezone:      "Asia/Shanghai",
			StartAt:            startAt,
			ExpireAt:           expireAt,
			ActivationDeadline: activationDeadline,
			SourceType:         "admin",
			SourceId:           operatorId,
		}
		return tx.Create(entitlement).Error
	})
	return entitlement, err
}

func PauseEntitlement(entitlementId int) error {
	if entitlementId <= 0 {
		return errors.New("invalid entitlement id")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var entitlement Entitlement
		if err := lockForUpdate(tx).Where("id = ?", entitlementId).First(&entitlement).Error; err != nil {
			return err
		}
		if entitlement.AssetKind != EntitlementAssetSubscription {
			return errors.New("only subscription entitlements can be paused")
		}
		if entitlement.State == EntitlementStatePaused {
			return nil
		}
		if entitlement.State != EntitlementStateActive {
			return errors.New("only active subscriptions can be paused")
		}
		return tx.Model(&entitlement).Updates(map[string]interface{}{
			"state":      EntitlementStatePaused,
			"updated_at": common.GetTimestamp(),
		}).Error
	})
}

func ResumeEntitlement(entitlementId int) error {
	if entitlementId <= 0 {
		return errors.New("invalid entitlement id")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var entitlement Entitlement
		if err := lockForUpdate(tx).Where("id = ?", entitlementId).First(&entitlement).Error; err != nil {
			return err
		}
		if entitlement.AssetKind != EntitlementAssetSubscription {
			return errors.New("only subscription entitlements can be resumed")
		}
		if entitlement.State == EntitlementStateActive {
			return nil
		}
		if entitlement.State != EntitlementStatePaused {
			return errors.New("only paused subscriptions can be resumed")
		}
		now := getDBTimestampTx(tx)
		state := EntitlementStateActive
		if entitlement.ExpireAt > 0 && entitlement.ExpireAt <= now {
			state = EntitlementStateExpired
		} else if entitlement.UsedQuota+entitlement.ReservedQuota >= entitlement.TotalQuota {
			state = EntitlementStateDepleted
		}
		return tx.Model(&entitlement).Updates(map[string]interface{}{
			"state":      state,
			"updated_at": common.GetTimestamp(),
		}).Error
	})
}
