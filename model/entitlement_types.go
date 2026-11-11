package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	EntitlementAssetSubscription = "subscription"
	EntitlementAssetStoredValue  = "stored_value"
	EntitlementAssetSystemWallet = "system_wallet"
	EntitlementMeterQuota        = "quota"

	EntitlementStatePending   = "pending"
	EntitlementStateQueued    = "queued"
	EntitlementStateActive    = "active"
	EntitlementStatePaused    = "paused"
	EntitlementStateDepleted  = "depleted"
	EntitlementStateExpired   = "expired"
	EntitlementStateCancelled = "cancelled"

	EntitlementTypeStatusActive   = "active"
	EntitlementTypeStatusDisabled = "disabled"
	EntitlementTypeStatusArchived = "archived"

	ProductCategorySubscription = "subscription"
	ProductCategoryRecharge     = "recharge"
	DefaultRechargeMinAmount    = int64(100)
	DefaultRechargeMaxAmount    = int64(1_000_000)
	ProductStatusDraft          = "draft"
	ProductStatusActive         = "active"
	ProductStatusPaused         = "paused"
	ProductStatusArchived       = "archived"

	ActivationPolicyImmediate = "immediate"
	ActivationPolicyManual    = "manual"
	ActivationPolicyDeferred  = "deferred"

	ProductOrderStatusPending          = "pending"
	ProductOrderStatusPaid             = "paid"
	ProductOrderStatusFulfilled        = "fulfilled"
	ProductOrderStatusCancelled        = "cancelled"
	ProductOrderStatusRefunded         = "refunded"
	ProductOrderStatusPaymentException = "payment_exception"
	ProductOrderPaymentTimeout         = int64(2 * 60 * 60)
)

var (
	ErrEntitlementQuotaInsufficient = errors.New("entitlement quota insufficient")
	ErrEntitlementFundingMismatch   = errors.New("entitlement funding source mismatch")
	ErrEntitlementChargeRefunded    = errors.New("entitlement charge already refunded")
	ErrProductOrderExpired          = errors.New("product order expired")
)

type AccessGroupPolicy struct {
	Id                int    `json:"id"`
	GroupName         string `json:"group_name" gorm:"type:varchar(64);uniqueIndex;not null"`
	FundingSourceType string `json:"funding_source_type" gorm:"type:varchar(32);index;not null"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt         int64  `json:"updated_at" gorm:"bigint"`
}

func (p *AccessGroupPolicy) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	p.GroupName = strings.TrimSpace(p.GroupName)
	p.FundingSourceType = strings.TrimSpace(p.FundingSourceType)
	p.CreatedAt, p.UpdatedAt = now, now
	return nil
}

func (p *AccessGroupPolicy) BeforeUpdate(_ *gorm.DB) error {
	p.UpdatedAt = common.GetTimestamp()
	return nil
}

type EntitlementType struct {
	Id          int                    `json:"id"`
	Code        string                 `json:"code" gorm:"type:varchar(64);uniqueIndex;not null"`
	Name        string                 `json:"name" gorm:"type:varchar(128);not null"`
	Description string                 `json:"description" gorm:"type:text"`
	AssetKind   string                 `json:"asset_kind" gorm:"type:varchar(32);index;not null"`
	MeterType   string                 `json:"meter_type" gorm:"type:varchar(32);not null"`
	Status      string                 `json:"status" gorm:"type:varchar(32);index;not null"`
	Revision    int64                  `json:"revision" gorm:"bigint;not null"`
	CreatedAt   int64                  `json:"created_at" gorm:"bigint"`
	UpdatedAt   int64                  `json:"updated_at" gorm:"bigint"`
	Groups      []EntitlementTypeGroup `json:"groups,omitempty" gorm:"foreignKey:EntitlementTypeId"`
}

func (t *EntitlementType) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	t.Code, t.Name = strings.TrimSpace(t.Code), strings.TrimSpace(t.Name)
	if t.MeterType == "" {
		t.MeterType = EntitlementMeterQuota
	}
	if t.Status == "" {
		t.Status = EntitlementTypeStatusActive
	}
	if t.Revision <= 0 {
		t.Revision = 1
	}
	t.CreatedAt, t.UpdatedAt = now, now
	return nil
}

func (t *EntitlementType) BeforeUpdate(_ *gorm.DB) error {
	t.UpdatedAt = common.GetTimestamp()
	return nil
}

type EntitlementTypeGroup struct {
	Id                  int    `json:"id"`
	EntitlementTypeId   int    `json:"entitlement_type_id" gorm:"uniqueIndex:idx_entitlement_type_group;index;not null"`
	AccessGroupPolicyId int    `json:"access_group_policy_id" gorm:"index;not null"`
	GroupName           string `json:"group_name" gorm:"type:varchar(64);uniqueIndex:idx_entitlement_type_group;index;not null"`
	CreatedBy           int    `json:"created_by"`
	CreatedAt           int64  `json:"created_at" gorm:"bigint"`
}

func (g *EntitlementTypeGroup) BeforeCreate(_ *gorm.DB) error {
	g.GroupName = strings.TrimSpace(g.GroupName)
	g.CreatedAt = common.GetTimestamp()
	return nil
}

type EntitlementTypeChangeLog struct {
	Id                int    `json:"id"`
	EntitlementTypeId int    `json:"entitlement_type_id" gorm:"index;not null"`
	Revision          int64  `json:"revision" gorm:"bigint;index;not null"`
	Action            string `json:"action" gorm:"type:varchar(64);not null"`
	BeforeJSON        string `json:"before_json" gorm:"type:text"`
	AfterJSON         string `json:"after_json" gorm:"type:text"`
	OperatorId        int    `json:"operator_id" gorm:"index"`
	Reason            string `json:"reason" gorm:"type:varchar(255)"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint"`
}

func (l *EntitlementTypeChangeLog) BeforeCreate(_ *gorm.DB) error {
	l.CreatedAt = common.GetTimestamp()
	return nil
}

type Product struct {
	Id             int          `json:"id"`
	Code           string       `json:"code" gorm:"type:varchar(64);uniqueIndex;not null"`
	Name           string       `json:"name" gorm:"type:varchar(128);not null"`
	Description    string       `json:"description" gorm:"type:text"`
	Category       string       `json:"category" gorm:"type:varchar(32);index;not null"`
	Status         string       `json:"status" gorm:"type:varchar(32);index;not null"`
	SortOrder      int          `json:"sort_order" gorm:"index"`
	VisibilityRule string       `json:"visibility_rule" gorm:"type:text"`
	CreatedAt      int64        `json:"created_at" gorm:"bigint"`
	UpdatedAt      int64        `json:"updated_at" gorm:"bigint"`
	SKUs           []ProductSKU `json:"skus,omitempty" gorm:"foreignKey:ProductId"`
}

func (p *Product) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	p.Code, p.Name = strings.TrimSpace(p.Code), strings.TrimSpace(p.Name)
	if p.Status == "" {
		p.Status = ProductStatusDraft
	}
	p.CreatedAt, p.UpdatedAt = now, now
	return nil
}

func (p *Product) BeforeUpdate(_ *gorm.DB) error { p.UpdatedAt = common.GetTimestamp(); return nil }

type ProductSKU struct {
	Id                     int              `json:"id"`
	DeletedAt              int64            `json:"deleted_at" gorm:"bigint;index;not null;default:0"`
	Code                   string           `json:"code" gorm:"type:varchar(64);uniqueIndex;not null"`
	ProductId              int              `json:"product_id" gorm:"index;not null"`
	EntitlementTypeId      int              `json:"entitlement_type_id" gorm:"index;not null"`
	Name                   string           `json:"name" gorm:"type:varchar(128);not null"`
	PriceAmountMinor       int64            `json:"price_amount_minor" gorm:"bigint;not null"`
	Currency               string           `json:"currency" gorm:"type:varchar(8);not null"`
	GrantTotalQuota        int64            `json:"grant_total_quota" gorm:"bigint;not null"`
	GrantDailyQuota        int64            `json:"grant_daily_quota" gorm:"bigint;not null"`
	MinRechargeAmountMinor int64            `json:"min_recharge_amount_minor" gorm:"bigint;not null;default:0"`
	MaxRechargeAmountMinor int64            `json:"max_recharge_amount_minor" gorm:"bigint;not null;default:0"`
	ValiditySeconds        int64            `json:"validity_seconds" gorm:"bigint;not null"`
	ActivationPolicy       string           `json:"activation_policy" gorm:"type:varchar(32);not null"`
	ActivationDeadlineSec  int64            `json:"activation_deadline_seconds" gorm:"bigint;not null"`
	Stock                  int64            `json:"stock" gorm:"bigint;not null"`
	StockLimited           bool             `json:"stock_limited"`
	PurchaseLimit          int              `json:"purchase_limit" gorm:"not null"`
	MultiQuantityEnabled   bool             `json:"multi_quantity_enabled"`
	Status                 string           `json:"status" gorm:"type:varchar(32);index;not null"`
	SortOrder              int              `json:"sort_order" gorm:"index"`
	CreatedAt              int64            `json:"created_at" gorm:"bigint"`
	UpdatedAt              int64            `json:"updated_at" gorm:"bigint"`
	EntitlementType        *EntitlementType `json:"entitlement_type,omitempty" gorm:"foreignKey:EntitlementTypeId"`
}

func (s *ProductSKU) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	s.DeletedAt = 0
	s.Code, s.Name = strings.TrimSpace(s.Code), strings.TrimSpace(s.Name)
	if s.Currency == "" {
		s.Currency = "CNY"
	}
	if s.ActivationPolicy == "" {
		s.ActivationPolicy = ActivationPolicyImmediate
	}
	if s.Status == "" {
		s.Status = ProductStatusDraft
	}
	if s.Stock > 0 {
		s.StockLimited = true
	}
	s.CreatedAt, s.UpdatedAt = now, now
	return nil
}

func (s *ProductSKU) BeforeUpdate(_ *gorm.DB) error { s.UpdatedAt = common.GetTimestamp(); return nil }

func (s *ProductSKU) NormalizeRechargeAmountBounds() {
	if s.MinRechargeAmountMinor <= 0 {
		s.MinRechargeAmountMinor = DefaultRechargeMinAmount
	}
	if s.MaxRechargeAmountMinor <= 0 {
		s.MaxRechargeAmountMinor = DefaultRechargeMaxAmount
	}
}

type ProductOrder struct {
	Id                      int                `json:"id"`
	OrderNo                 string             `json:"order_no" gorm:"type:varchar(64);uniqueIndex;not null"`
	UserId                  int                `json:"user_id" gorm:"index;not null"`
	Status                  string             `json:"status" gorm:"type:varchar(32);index;not null"`
	PaymentMethod           string             `json:"payment_method" gorm:"type:varchar(32)"`
	PaymentProvider         string             `json:"payment_provider" gorm:"type:varchar(32);index"`
	ProviderTradeNo         *string            `json:"provider_trade_no" gorm:"type:varchar(128);uniqueIndex"`
	ProviderPayload         string             `json:"-" gorm:"type:text"`
	TotalAmountMinor        int64              `json:"total_amount_minor" gorm:"bigint;not null"`
	OriginalAmountCents     int64              `json:"original_amount_cents" gorm:"bigint;not null;default:0"`
	PromotionDiscountCents  int64              `json:"promotion_discount_cents" gorm:"bigint;not null;default:0"`
	GiftDiscountCents       int64              `json:"gift_discount_cents" gorm:"bigint;not null;default:0"`
	CashPayableCents        int64              `json:"cash_payable_cents" gorm:"bigint;not null;default:0"`
	CashPaidCents           int64              `json:"cash_paid_cents" gorm:"bigint;not null;default:0"`
	GiftStatus              string             `json:"gift_status" gorm:"type:varchar(32);index"`
	Currency                string             `json:"currency" gorm:"type:varchar(8);not null"`
	ExpiresAt               int64              `json:"expires_at" gorm:"bigint;index"`
	CreatedAt               int64              `json:"created_at" gorm:"bigint"`
	PaidAt                  int64              `json:"paid_at" gorm:"bigint"`
	FulfilledAt             int64              `json:"fulfilled_at" gorm:"bigint"`
	CancelledAt             int64              `json:"cancelled_at" gorm:"bigint"`
	RefundedAt              int64              `json:"refunded_at" gorm:"bigint"`
	RefundOperatorId        int                `json:"refund_operator_id" gorm:"index"`
	StatusReason            string             `json:"status_reason" gorm:"type:varchar(255)"`
	StockRestored           bool               `json:"stock_restored"`
	UpdatedAt               int64              `json:"updated_at" gorm:"bigint"`
	Items                   []ProductOrderItem `json:"items,omitempty" gorm:"foreignKey:OrderId"`
	CorporateTicketNo       string             `json:"corporate_ticket_no,omitempty" gorm:"-"`
	CorporateTransferStatus string             `json:"corporate_transfer_status,omitempty" gorm:"-"`
}

func (o *ProductOrder) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if o.Status == "" {
		o.Status = ProductOrderStatusPending
	}
	if o.Currency == "" {
		o.Currency = "CNY"
	}
	if o.ExpiresAt == 0 && o.Status == ProductOrderStatusPending {
		o.ExpiresAt = now + ProductOrderPaymentTimeout
	}
	o.CreatedAt, o.UpdatedAt = now, now
	return nil
}

func (o *ProductOrder) BeforeUpdate(_ *gorm.DB) error {
	o.UpdatedAt = common.GetTimestamp()
	return nil
}

func (o *ProductOrder) PayableCashCents() int64 {
	if o == nil {
		return 0
	}
	if o.CashPayableCents > 0 || o.OriginalAmountCents > 0 || o.GiftDiscountCents > 0 {
		return o.CashPayableCents
	}
	return o.TotalAmountMinor
}

type ProductOrderItem struct {
	Id                    int    `json:"id"`
	OrderId               int    `json:"order_id" gorm:"index;not null"`
	ProductId             int    `json:"product_id" gorm:"index;not null"`
	SKUId                 int    `json:"sku_id" gorm:"index;not null"`
	EntitlementTypeId     int    `json:"entitlement_type_id" gorm:"index;not null"`
	ProductName           string `json:"product_name" gorm:"type:varchar(128);not null"`
	SKUName               string `json:"sku_name" gorm:"type:varchar(128);not null"`
	Quantity              int    `json:"quantity" gorm:"not null"`
	UnitPriceAmountMinor  int64  `json:"unit_price_amount_minor" gorm:"bigint;not null"`
	GrantTotalQuota       int64  `json:"grant_total_quota" gorm:"bigint;not null"`
	GrantDailyQuota       int64  `json:"grant_daily_quota" gorm:"bigint;not null"`
	ValiditySeconds       int64  `json:"validity_seconds" gorm:"bigint;not null"`
	ActivationPolicy      string `json:"activation_policy" gorm:"type:varchar(32);not null"`
	ActivationDeadlineSec int64  `json:"activation_deadline_seconds" gorm:"bigint;not null"`
	StockReserved         bool   `json:"stock_reserved"`
	CreatedAt             int64  `json:"created_at" gorm:"bigint"`
}

func (i *ProductOrderItem) BeforeCreate(_ *gorm.DB) error {
	i.CreatedAt = common.GetTimestamp()
	return nil
}

func GetAccessGroupFundingType(groupName string) (string, bool, error) {
	groupName = strings.TrimSpace(groupName)
	if groupName == "" {
		return "", false, nil
	}
	var policy AccessGroupPolicy
	result := DB.Where("group_name = ?", groupName).Limit(1).Find(&policy)
	if result.Error != nil {
		// Keep token and relay paths compatible during rolling upgrades and in
		// narrowly-scoped tests which intentionally migrate only legacy tables.
		// Once the entitlement tables exist, all real query errors still surface.
		message := strings.ToLower(result.Error.Error())
		if strings.Contains(message, "no such table") ||
			strings.Contains(message, "doesn't exist") ||
			strings.Contains(message, "does not exist") ||
			strings.Contains(message, "undefined table") {
			return "", false, nil
		}
		return "", false, result.Error
	}
	if result.RowsAffected == 0 {
		return "", false, nil
	}
	return policy.FundingSourceType, true, nil
}

func AccessGroupPolicyTableAvailable() bool {
	return DB != nil && DB.Migrator().HasTable(&AccessGroupPolicy{})
}
