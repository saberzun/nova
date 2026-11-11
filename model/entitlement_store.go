package model

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func ReplaceEntitlementTypeGroups(typeId int, groupNames []string, operatorId int, reason string) error {
	if typeId <= 0 {
		return errors.New("invalid entitlement type id")
	}
	cleanNames := make([]string, 0, len(groupNames))
	seen := make(map[string]struct{}, len(groupNames))
	for _, groupName := range groupNames {
		groupName = strings.TrimSpace(groupName)
		if groupName == "" {
			continue
		}
		if _, ok := seen[groupName]; ok {
			continue
		}
		seen[groupName] = struct{}{}
		cleanNames = append(cleanNames, groupName)
	}
	sort.Strings(cleanNames)
	return DB.Transaction(func(tx *gorm.DB) error {
		var entitlementType EntitlementType
		if err := lockForUpdate(tx).Where("id = ?", typeId).First(&entitlementType).Error; err != nil {
			return err
		}
		var oldGroups []EntitlementTypeGroup
		if err := tx.Where("entitlement_type_id = ?", typeId).Order("group_name asc").Find(&oldGroups).Error; err != nil {
			return err
		}
		oldNames := make([]string, 0, len(oldGroups))
		for _, group := range oldGroups {
			oldNames = append(oldNames, group.GroupName)
		}
		newGroups := make([]EntitlementTypeGroup, 0, len(cleanNames))
		for _, groupName := range cleanNames {
			var policy AccessGroupPolicy
			if err := tx.Where("group_name = ?", groupName).First(&policy).Error; err != nil {
				return fmt.Errorf("access group %s has no funding policy: %w", groupName, err)
			}
			if policy.FundingSourceType != entitlementType.AssetKind {
				return fmt.Errorf("%w: type=%s group=%s", ErrEntitlementFundingMismatch, entitlementType.AssetKind, policy.FundingSourceType)
			}
			newGroups = append(newGroups, EntitlementTypeGroup{
				EntitlementTypeId:   typeId,
				AccessGroupPolicyId: policy.Id,
				GroupName:           groupName,
				CreatedBy:           operatorId,
			})
		}
		beforeJSON, err := common.Marshal(oldNames)
		if err != nil {
			return err
		}
		afterJSON, err := common.Marshal(cleanNames)
		if err != nil {
			return err
		}
		if err := tx.Where("entitlement_type_id = ?", typeId).Delete(&EntitlementTypeGroup{}).Error; err != nil {
			return err
		}
		if len(newGroups) > 0 {
			if err := tx.Create(&newGroups).Error; err != nil {
				return err
			}
		}
		entitlementType.Revision++
		if err := tx.Model(&entitlementType).Updates(map[string]interface{}{
			"revision":   entitlementType.Revision,
			"updated_at": common.GetTimestamp(),
		}).Error; err != nil {
			return err
		}
		return tx.Create(&EntitlementTypeChangeLog{
			EntitlementTypeId: typeId,
			Revision:          entitlementType.Revision,
			Action:            "replace_groups",
			BeforeJSON:        string(beforeJSON),
			AfterJSON:         string(afterJSON),
			OperatorId:        operatorId,
			Reason:            strings.TrimSpace(reason),
		}).Error
	})
}

func CreateProductOrder(userId int, skuId int, quantity int, rechargeAmountMinor int64) (*ProductOrder, error) {
	return CreateProductOrderWithGift(userId, skuId, quantity, rechargeAmountMinor, 0)
}

func CreateProductOrderWithGift(userId int, skuId int, quantity int, rechargeAmountMinor int64, giftDiscountCents int64) (*ProductOrder, error) {
	return createProductOrderWithGiftAndHook(userId, skuId, quantity, rechargeAmountMinor, giftDiscountCents, nil)
}

func createProductOrderWithGiftAndHook(userId int, skuId int, quantity int, rechargeAmountMinor int64, giftDiscountCents int64, hook func(*gorm.DB, *ProductOrder) error) (*ProductOrder, error) {
	if userId <= 0 || skuId <= 0 || quantity <= 0 || quantity > 100 {
		return nil, errors.New("invalid product order")
	}
	if giftDiscountCents < 0 {
		return nil, errors.New("gift discount cannot be negative")
	}
	var created ProductOrder
	err := DB.Transaction(func(tx *gorm.DB) error {
		var sku ProductSKU
		if err := lockForUpdate(tx).Where("id = ? AND status = ? AND deleted_at = 0", skuId, ProductStatusActive).First(&sku).Error; err != nil {
			return err
		}
		var product Product
		if err := tx.Where("id = ? AND status = ?", sku.ProductId, ProductStatusActive).First(&product).Error; err != nil {
			return err
		}
		var entitlementType EntitlementType
		if err := tx.Where("id = ? AND status = ?", sku.EntitlementTypeId, EntitlementTypeStatusActive).First(&entitlementType).Error; err != nil {
			return err
		}
		if sku.GrantTotalQuota <= 0 || sku.PriceAmountMinor < 0 {
			return errors.New("invalid sku grant or price")
		}
		unitPriceAmountMinor := sku.PriceAmountMinor
		grantTotalQuota := sku.GrantTotalQuota
		grantDailyQuota := sku.GrantDailyQuota
		if product.Category == ProductCategorySubscription {
			if quantity > 1 && !sku.MultiQuantityEnabled {
				return errors.New("sku does not allow multiple quantities")
			}
			if rechargeAmountMinor != 0 {
				return errors.New("subscription sku does not accept a recharge amount")
			}
			if entitlementType.AssetKind != EntitlementAssetSubscription || sku.ValiditySeconds <= 0 {
				return errors.New("invalid subscription sku")
			}
		} else if product.Category == ProductCategoryRecharge {
			if entitlementType.AssetKind != EntitlementAssetStoredValue || sku.ActivationPolicy != ActivationPolicyImmediate {
				return errors.New("invalid recharge sku")
			}
			if quantity != 1 || sku.PriceAmountMinor <= 0 {
				return errors.New("recharge orders require one pricing rule sku")
			}
			sku.NormalizeRechargeAmountBounds()
			if sku.MaxRechargeAmountMinor < sku.MinRechargeAmountMinor ||
				rechargeAmountMinor < sku.MinRechargeAmountMinor ||
				rechargeAmountMinor > sku.MaxRechargeAmountMinor {
				return errors.New("recharge amount is outside the allowed range")
			}
			quotaDecimal := decimal.NewFromInt(sku.GrantTotalQuota).
				Mul(decimal.NewFromInt(rechargeAmountMinor)).
				Div(decimal.NewFromInt(sku.PriceAmountMinor))
			quota, clamp := common.QuotaFromDecimalChecked(quotaDecimal)
			if clamp != nil {
				return clamp
			}
			if quota <= 0 {
				return errors.New("recharge amount grants no quota")
			}
			unitPriceAmountMinor = rechargeAmountMinor
			grantTotalQuota = int64(quota)
			if sku.GrantDailyQuota > 0 {
				dailyQuotaDecimal := decimal.NewFromInt(sku.GrantDailyQuota).
					Mul(decimal.NewFromInt(rechargeAmountMinor)).
					Div(decimal.NewFromInt(sku.PriceAmountMinor))
				dailyQuota, dailyClamp := common.QuotaFromDecimalChecked(dailyQuotaDecimal)
				if dailyClamp != nil {
					return dailyClamp
				}
				grantDailyQuota = int64(dailyQuota)
			}
		} else {
			return errors.New("unsupported product category")
		}
		if unitPriceAmountMinor > math.MaxInt64/int64(quantity) {
			return errors.New("product order amount overflow")
		}
		if sku.StockLimited && sku.Stock < int64(quantity) {
			return errors.New("sku stock insufficient")
		}
		if sku.PurchaseLimit > 0 {
			var purchased int64
			if err := tx.Model(&ProductOrderItem{}).
				Joins("JOIN product_orders ON product_orders.id = product_order_items.order_id").
				Where("product_orders.user_id = ? AND product_order_items.sku_id = ? AND product_orders.status IN ?", userId, skuId, []string{ProductOrderStatusPaid, ProductOrderStatusFulfilled}).
				Select("COALESCE(SUM(product_order_items.quantity), 0)").Scan(&purchased).Error; err != nil {
				return err
			}
			if purchased+int64(quantity) > int64(sku.PurchaseLimit) {
				return errors.New("sku purchase limit exceeded")
			}
		}
		originalAmount := unitPriceAmountMinor * int64(quantity)
		if giftDiscountCents > originalAmount {
			return errors.New("gift discount exceeds product amount")
		}
		if giftDiscountCents > 0 {
			var giftSettings GiftSettings
			settingsResult := tx.Order("id asc").Limit(1).Find(&giftSettings)
			if settingsResult.Error != nil {
				return settingsResult.Error
			}
			if settingsResult.RowsAffected == 0 || !giftSettings.CheckoutEnabled {
				return errors.New("gift checkout is disabled")
			}
		}
		cashPayable := originalAmount - giftDiscountCents
		orderNo := fmt.Sprintf("PO-%d-%s", common.GetTimestamp(), common.GetRandomString(10))
		order := ProductOrder{
			OrderNo:             orderNo,
			UserId:              userId,
			Status:              ProductOrderStatusPending,
			TotalAmountMinor:    cashPayable,
			OriginalAmountCents: originalAmount,
			GiftDiscountCents:   giftDiscountCents,
			CashPayableCents:    cashPayable,
			Currency:            sku.Currency,
			ExpiresAt:           getDBTimestampTx(tx) + ProductOrderPaymentTimeout,
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		item := ProductOrderItem{
			OrderId:               order.Id,
			ProductId:             product.Id,
			SKUId:                 sku.Id,
			EntitlementTypeId:     entitlementType.Id,
			ProductName:           product.Name,
			SKUName:               sku.Name,
			Quantity:              quantity,
			UnitPriceAmountMinor:  unitPriceAmountMinor,
			GrantTotalQuota:       grantTotalQuota,
			GrantDailyQuota:       grantDailyQuota,
			ValiditySeconds:       sku.ValiditySeconds,
			ActivationPolicy:      sku.ActivationPolicy,
			ActivationDeadlineSec: sku.ActivationDeadlineSec,
			StockReserved:         sku.StockLimited,
		}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		if giftDiscountCents > 0 {
			if err := reserveGiftTx(tx, userId, order.Id, giftDiscountCents, order.ExpiresAt); err != nil {
				return err
			}
			order.GiftStatus = GiftHoldStatusReserved
			if err := tx.Model(&order).Update("gift_status", GiftHoldStatusReserved).Error; err != nil {
				return err
			}
		}
		if sku.StockLimited {
			result := tx.Model(&ProductSKU{}).Where("id = ? AND stock >= ?", sku.Id, quantity).
				Update("stock", gorm.Expr("stock - ?", quantity))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errors.New("sku stock insufficient")
			}
		}
		order.Items = []ProductOrderItem{item}
		created = order
		if hook != nil {
			return hook(tx, &created)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func CompleteProductOrder(orderNo string, providerTradeNo string, paymentMethod string) error {
	return CompleteProductOrderWithProvider(orderNo, providerTradeNo, "", paymentMethod, "")
}

func PrepareProductOrderPayment(orderNo string, userId int, paymentProvider string, paymentMethod string) (*ProductOrder, error) {
	orderNo = strings.TrimSpace(orderNo)
	paymentProvider = strings.TrimSpace(paymentProvider)
	paymentMethod = strings.TrimSpace(paymentMethod)
	if orderNo == "" || userId <= 0 || paymentProvider == "" || paymentMethod == "" {
		return nil, errors.New("invalid product order payment")
	}
	var prepared ProductOrder
	expired := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		var order ProductOrder
		if err := lockForUpdate(tx).Where("order_no = ? AND user_id = ?", orderNo, userId).First(&order).Error; err != nil {
			return err
		}
		if order.Status != ProductOrderStatusPending {
			return fmt.Errorf("order cannot be paid from status %s", order.Status)
		}
		if order.ExpiresAt == 0 {
			order.ExpiresAt = order.CreatedAt + ProductOrderPaymentTimeout
		}
		if order.ExpiresAt <= getDBTimestampTx(tx) {
			if err := cancelProductOrderTx(tx, &order, "payment timeout", true); err != nil {
				return err
			}
			expired = true
			return nil
		}
		if order.CashPayableCents <= 0 && order.TotalAmountMinor <= 0 {
			return errors.New("free order does not require payment")
		}
		if order.PaymentProvider != "" && order.PaymentProvider != paymentProvider {
			return ErrPaymentMethodMismatch
		}
		if err := tx.Model(&order).Updates(map[string]interface{}{
			"payment_provider": paymentProvider,
			"payment_method":   paymentMethod,
			"expires_at":       order.ExpiresAt,
			"updated_at":       common.GetTimestamp(),
		}).Error; err != nil {
			return err
		}
		order.PaymentProvider = paymentProvider
		order.PaymentMethod = paymentMethod
		prepared = order
		return nil
	})
	if err != nil {
		return nil, err
	}
	if expired {
		return nil, ErrProductOrderExpired
	}
	return &prepared, nil
}

func ResetProductOrderPaymentPreparation(orderNo string, userId int, paymentProvider string, reason string) error {
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" || userId <= 0 || strings.TrimSpace(paymentProvider) == "" {
		return errors.New("invalid product order payment reset")
	}
	result := DB.Model(&ProductOrder{}).
		Where("order_no = ? AND user_id = ? AND status = ? AND payment_provider = ?", orderNo, userId, ProductOrderStatusPending, paymentProvider).
		Updates(map[string]interface{}{
			"payment_provider": "",
			"payment_method":   "",
			"status_reason":    strings.TrimSpace(reason),
			"updated_at":       common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func MarkProductOrderPaymentException(orderNo string, reason string, providerPayload string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var order ProductOrder
		result := lockForUpdate(tx).Where("order_no = ?", orderNo).Limit(1).Find(&order)
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}
		if order.Status != ProductOrderStatusPending && order.Status != ProductOrderStatusCancelled {
			return nil
		}
		if order.Status == ProductOrderStatusPending {
			if err := releaseGiftTx(tx, &order, "payment exception"); err != nil {
				return err
			}
		}
		updates := map[string]interface{}{
			"status":        ProductOrderStatusPaymentException,
			"status_reason": strings.TrimSpace(reason),
			"updated_at":    common.GetTimestamp(),
		}
		if order.GiftDiscountCents > 0 {
			updates["gift_status"] = GiftHoldStatusReleased
		}
		if strings.TrimSpace(providerPayload) != "" {
			updates["provider_payload"] = providerPayload
		}
		return tx.Model(&order).Updates(updates).Error
	})
}

func CompleteProductOrderWithProvider(orderNo string, providerTradeNo string, expectedPaymentProvider string, paymentMethod string, providerPayload string) error {
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return errors.New("order number is empty")
	}
	expired := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		err := completeProductOrderWithProviderTx(tx, orderNo, providerTradeNo, expectedPaymentProvider, paymentMethod, providerPayload)
		if !errors.Is(err, ErrProductOrderExpired) {
			return err
		}
		var order ProductOrder
		if err := lockForUpdate(tx).Preload("Items").Where("order_no = ?", orderNo).First(&order).Error; err != nil {
			return err
		}
		if err := cancelProductOrderTx(tx, &order, "payment timeout", true); err != nil {
			return err
		}
		expired = true
		return nil
	})
	if err != nil {
		return err
	}
	if expired {
		return ErrProductOrderExpired
	}
	return nil
}

func completeProductOrderWithProviderTx(tx *gorm.DB, orderNo string, providerTradeNo string, expectedPaymentProvider string, paymentMethod string, providerPayload string) error {
	var order ProductOrder
	if err := lockForUpdate(tx).Preload("Items").Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		return err
	}
	if order.Status == ProductOrderStatusFulfilled {
		return nil
	}
	if expectedPaymentProvider != "" && order.PaymentProvider != expectedPaymentProvider {
		return ErrPaymentMethodMismatch
	}
	if order.Status != ProductOrderStatusPending && order.Status != ProductOrderStatusPaid {
		return fmt.Errorf("order cannot be fulfilled from status %s", order.Status)
	}
	now := getDBTimestampTx(tx)
	if order.Status == ProductOrderStatusPending && order.PaymentProvider != PaymentProviderCorporateTransfer {
		deadline := order.ExpiresAt
		if deadline == 0 {
			deadline = order.CreatedAt + ProductOrderPaymentTimeout
		}
		if deadline <= now {
			return ErrProductOrderExpired
		}
	}
	if order.OriginalAmountCents == 0 {
		order.OriginalAmountCents = order.TotalAmountMinor + order.GiftDiscountCents
	}
	if order.CashPayableCents == 0 && order.TotalAmountMinor > 0 {
		order.CashPayableCents = order.TotalAmountMinor
	}
	if err := captureGiftTx(tx, &order); err != nil {
		return err
	}
	order.CashPaidCents = order.CashPayableCents
	batchId := fmt.Sprintf("FUL-%d-%s", order.Id, common.GetRandomString(8))
	for _, item := range order.Items {
		var entitlementType EntitlementType
		if err := tx.Where("id = ?", item.EntitlementTypeId).First(&entitlementType).Error; err != nil {
			return err
		}
		for quantityIndex := 0; quantityIndex < item.Quantity; quantityIndex++ {
			fulfillmentKey := fmt.Sprintf("%d:%d", item.Id, quantityIndex)
			var existingCount int64
			if err := tx.Model(&Entitlement{}).Where("fulfillment_key = ?", fulfillmentKey).Count(&existingCount).Error; err != nil {
				return err
			}
			if existingCount > 0 {
				continue
			}
			state := EntitlementStateActive
			startAt := now
			expireAt := int64(0)
			activationDeadline := int64(0)
			if entitlementType.AssetKind == EntitlementAssetSubscription {
				switch item.ActivationPolicy {
				case ActivationPolicyManual:
					state, startAt = EntitlementStatePending, 0
					if item.ActivationDeadlineSec > 0 {
						activationDeadline = now + item.ActivationDeadlineSec
					}
				case ActivationPolicyDeferred:
					var lastExpire int64
					if err := tx.Model(&Entitlement{}).
						Where("user_id = ? AND entitlement_type_id = ? AND state IN ?", order.UserId, item.EntitlementTypeId, []string{EntitlementStateActive, EntitlementStateQueued}).
						Select("COALESCE(MAX(expire_at), 0)").Scan(&lastExpire).Error; err != nil {
						return err
					}
					if lastExpire > startAt {
						startAt, state = lastExpire, EntitlementStateQueued
					}
				}
				if startAt > 0 {
					expireAt = startAt + item.ValiditySeconds
				}
			}
			entitlement := Entitlement{
				UserId:             order.UserId,
				EntitlementTypeId:  item.EntitlementTypeId,
				AssetKind:          entitlementType.AssetKind,
				ProductId:          item.ProductId,
				SKUId:              item.SKUId,
				OrderItemId:        item.Id,
				QuantityIndex:      quantityIndex,
				FulfillmentKey:     &fulfillmentKey,
				State:              state,
				TotalQuota:         item.GrantTotalQuota,
				DailyQuota:         item.GrantDailyQuota,
				ResetTimezone:      "Asia/Shanghai",
				StartAt:            startAt,
				ExpireAt:           expireAt,
				ActivationDeadline: activationDeadline,
				SourceType:         "order",
				SourceId:           order.Id,
				FulfillmentBatchId: batchId,
			}
			if err := tx.Create(&entitlement).Error; err != nil {
				return err
			}
		}
	}
	tradeNo := strings.TrimSpace(providerTradeNo)
	updates := map[string]interface{}{
		"status":                ProductOrderStatusFulfilled,
		"payment_method":        strings.TrimSpace(paymentMethod),
		"original_amount_cents": order.OriginalAmountCents,
		"cash_payable_cents":    order.CashPayableCents,
		"cash_paid_cents":       order.CashPaidCents,
		"paid_at":               now,
		"fulfilled_at":          now,
		"updated_at":            common.GetTimestamp(),
	}
	if order.GiftDiscountCents > 0 {
		updates["gift_status"] = GiftHoldStatusCaptured
	}
	if expectedPaymentProvider != "" {
		updates["payment_provider"] = expectedPaymentProvider
	}
	if strings.TrimSpace(providerPayload) != "" {
		updates["provider_payload"] = providerPayload
	}
	if tradeNo != "" {
		updates["provider_trade_no"] = tradeNo
	}
	if err := tx.Model(&order).Updates(updates).Error; err != nil {
		return err
	}
	order.PaymentMethod = strings.TrimSpace(paymentMethod)
	return createReferralRewardTx(tx, &order, now)
}

func restoreProductOrderStockTx(tx *gorm.DB, order *ProductOrder) error {
	if order == nil || order.StockRestored {
		return nil
	}
	if len(order.Items) == 0 {
		if err := tx.Where("order_id = ?", order.Id).Find(&order.Items).Error; err != nil {
			return err
		}
	}
	for _, item := range order.Items {
		if !item.StockReserved || item.Quantity <= 0 {
			continue
		}
		if err := tx.Model(&ProductSKU{}).Where("id = ?", item.SKUId).
			Update("stock", gorm.Expr("stock + ?", item.Quantity)).Error; err != nil {
			return err
		}
	}
	order.StockRestored = true
	return nil
}

func cancelProductOrderTx(tx *gorm.DB, order *ProductOrder, reason string, paymentTimedOut bool) error {
	if order == nil {
		return errors.New("product order is nil")
	}
	if order.Status == ProductOrderStatusCancelled {
		return nil
	}
	if order.Status != ProductOrderStatusPending {
		return fmt.Errorf("order cannot be cancelled from status %s", order.Status)
	}
	if paymentTimedOut {
		if order.ExpiresAt == 0 {
			order.ExpiresAt = order.CreatedAt + ProductOrderPaymentTimeout
		}
		if order.ExpiresAt > getDBTimestampTx(tx) {
			return errors.New("product order payment deadline has not elapsed")
		}
	}
	if order.PaymentProvider != "" && !(paymentTimedOut && order.PaymentProvider == PaymentProviderEpay) {
		return errors.New("order payment was initiated and must be reconciled before cancellation")
	}
	if err := releaseGiftTx(tx, order, reason); err != nil {
		return err
	}
	if err := restoreProductOrderStockTx(tx, order); err != nil {
		return err
	}
	now := getDBTimestampTx(tx)
	updates := map[string]interface{}{
		"status":         ProductOrderStatusCancelled,
		"status_reason":  strings.TrimSpace(reason),
		"cancelled_at":   now,
		"stock_restored": order.StockRestored,
		"updated_at":     common.GetTimestamp(),
	}
	if paymentTimedOut {
		updates["expires_at"] = order.ExpiresAt
	}
	if order.GiftDiscountCents > 0 {
		updates["gift_status"] = GiftHoldStatusReleased
	}
	return tx.Model(order).Updates(updates).Error
}

func CancelProductOrder(orderNo string, userId int, reason string) error {
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return errors.New("order number is empty")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var order ProductOrder
		query := lockForUpdate(tx).Preload("Items").Where("order_no = ?", orderNo)
		if userId > 0 {
			query = query.Where("user_id = ?", userId)
		}
		if err := query.First(&order).Error; err != nil {
			return err
		}
		return cancelProductOrderTx(tx, &order, reason, false)
	})
}

func RefundProductOrder(orderNo string, operatorId int, reason string) error {
	orderNo = strings.TrimSpace(orderNo)
	reason = strings.TrimSpace(reason)
	if orderNo == "" || operatorId <= 0 || reason == "" {
		return errors.New("invalid product order refund")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		return refundProductOrderTx(tx, orderNo, operatorId, reason)
	})
}

func refundProductOrderTx(tx *gorm.DB, orderNo string, operatorId int, reason string) error {
	var order ProductOrder
	if err := lockForUpdate(tx).Preload("Items").Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		return err
	}
	if order.Status == ProductOrderStatusRefunded {
		return nil
	}
	if order.Status != ProductOrderStatusFulfilled && order.Status != ProductOrderStatusPaid {
		return fmt.Errorf("order cannot be refunded from status %s", order.Status)
	}
	var entitlements []Entitlement
	if err := lockForUpdate(tx).Where("source_type = ? AND source_id = ?", "order", order.Id).Find(&entitlements).Error; err != nil {
		return err
	}
	for _, entitlement := range entitlements {
		if entitlement.UsedQuota != 0 || entitlement.ReservedQuota != 0 {
			return errors.New("order contains consumed or reserved entitlement quota")
		}
	}
	if err := tx.Model(&Entitlement{}).Where("source_type = ? AND source_id = ?", "order", order.Id).
		Update("state", EntitlementStateCancelled).Error; err != nil {
		return err
	}
	if err := refundGiftTx(tx, &order, reason); err != nil {
		return err
	}
	if err := reverseReferralRewardTx(tx, &order, reason); err != nil {
		return err
	}
	if err := restoreProductOrderStockTx(tx, &order); err != nil {
		return err
	}
	now := getDBTimestampTx(tx)
	updates := map[string]interface{}{
		"status":             ProductOrderStatusRefunded,
		"status_reason":      reason,
		"refunded_at":        now,
		"refund_operator_id": operatorId,
		"stock_restored":     order.StockRestored,
		"updated_at":         common.GetTimestamp(),
	}
	if order.GiftDiscountCents > 0 {
		updates["gift_status"] = GiftHoldStatusRefunded
	}
	return tx.Model(&order).Updates(updates).Error
}

func ExpirePendingProductOrders(limit int) (int, error) {
	if limit <= 0 {
		return 0, nil
	}
	var orderNos []string
	now := GetDBTimestamp()
	if err := DB.Model(&ProductOrder{}).
		Where("status = ? AND (payment_provider IN ? OR payment_provider IS NULL) AND ((expires_at > 0 AND expires_at <= ?) OR ((expires_at = 0 OR expires_at IS NULL) AND created_at <= ?))",
			ProductOrderStatusPending, []string{"", PaymentProviderEpay}, now, now-ProductOrderPaymentTimeout).
		Order("id asc").Limit(limit).Pluck("order_no", &orderNos).Error; err != nil {
		return 0, err
	}
	expired := 0
	for _, orderNo := range orderNos {
		changed := false
		if err := DB.Transaction(func(tx *gorm.DB) error {
			var order ProductOrder
			if err := lockForUpdate(tx).Preload("Items").Where("order_no = ?", orderNo).First(&order).Error; err != nil {
				return err
			}
			deadline := order.ExpiresAt
			if deadline == 0 {
				deadline = order.CreatedAt + ProductOrderPaymentTimeout
			}
			if order.Status != ProductOrderStatusPending || deadline > getDBTimestampTx(tx) ||
				(order.PaymentProvider != "" && order.PaymentProvider != PaymentProviderEpay) {
				return nil
			}
			if err := cancelProductOrderTx(tx, &order, "payment timeout", true); err != nil {
				return err
			}
			changed = true
			return nil
		}); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			var current ProductOrder
			if lookupErr := DB.Select("status").Where("order_no = ?", orderNo).First(&current).Error; lookupErr == nil && current.Status != ProductOrderStatusPending {
				continue
			}
			return expired, err
		}
		if changed {
			expired++
		}
	}
	return expired, nil
}

func AdjustEntitlement(entitlementId int, operatorId int, deltaQuota int64, reason string, idempotencyKey string) error {
	reason = strings.TrimSpace(reason)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if entitlementId <= 0 || deltaQuota == 0 || reason == "" || idempotencyKey == "" {
		return errors.New("invalid entitlement adjustment")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var existing EntitlementAdjustmentLedger
		query := tx.Where("idempotency_key = ?", idempotencyKey).Limit(1).Find(&existing)
		if query.Error != nil {
			return query.Error
		}
		if query.RowsAffected > 0 {
			if existing.EntitlementId == entitlementId && existing.DeltaQuota == deltaQuota {
				return nil
			}
			return errors.New("idempotency key already used for another adjustment")
		}
		var entitlement Entitlement
		if err := lockForUpdate(tx).Where("id = ?", entitlementId).First(&entitlement).Error; err != nil {
			return err
		}
		newTotal := entitlement.TotalQuota + deltaQuota
		if newTotal < entitlement.UsedQuota+entitlement.ReservedQuota {
			return errors.New("adjusted quota cannot be less than used plus reserved quota")
		}
		ledger := EntitlementAdjustmentLedger{
			IdempotencyKey: idempotencyKey,
			EntitlementId:  entitlementId,
			OperatorId:     operatorId,
			DeltaQuota:     deltaQuota,
			Reason:         reason,
		}
		if err := tx.Create(&ledger).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{"total_quota": newTotal, "updated_at": common.GetTimestamp()}
		if entitlement.State == EntitlementStateDepleted && newTotal > entitlement.UsedQuota {
			updates["state"] = EntitlementStateActive
		}
		result := tx.Model(&Entitlement{}).Where("id = ?", entitlementId).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func RevokeEntitlement(entitlementId int) error {
	if entitlementId <= 0 {
		return errors.New("invalid entitlement id")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var entitlement Entitlement
		if err := lockForUpdate(tx).Where("id = ?", entitlementId).First(&entitlement).Error; err != nil {
			return err
		}
		if entitlement.State == EntitlementStateCancelled {
			return nil
		}
		if entitlement.ReservedQuota != 0 {
			return errors.New("entitlement still has in-flight reservations")
		}
		result := tx.Model(&Entitlement{}).Where("id = ? AND reserved_quota = 0", entitlementId).
			Updates(map[string]interface{}{"state": EntitlementStateCancelled, "updated_at": common.GetTimestamp()})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("entitlement still has in-flight reservations")
		}
		return nil
	})
}

func ActivateEntitlement(userId int, entitlementId int) error {
	if userId <= 0 || entitlementId <= 0 {
		return errors.New("invalid entitlement activation")
	}
	now := GetDBTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		var entitlement Entitlement
		if err := lockForUpdate(tx).Where("id = ? AND user_id = ?", entitlementId, userId).First(&entitlement).Error; err != nil {
			return err
		}
		if entitlement.State == EntitlementStateActive {
			return nil
		}
		if entitlement.State != EntitlementStatePending {
			return errors.New("entitlement is not pending activation")
		}
		if entitlement.ActivationDeadline > 0 && entitlement.ActivationDeadline <= now {
			return errors.New("entitlement activation deadline passed")
		}
		var item ProductOrderItem
		if err := tx.Where("id = ?", entitlement.OrderItemId).First(&item).Error; err != nil {
			return err
		}
		return tx.Model(&entitlement).Updates(map[string]interface{}{
			"state":      EntitlementStateActive,
			"start_at":   now,
			"expire_at":  now + item.ValiditySeconds,
			"updated_at": common.GetTimestamp(),
		}).Error
	})
}
