package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

var (
	ErrSKUPendingOrders        = errors.New("SKU has pending orders")
	ErrSKUConfirmationRequired = errors.New("existing entitlements require confirmation")
)

type SKUReferences struct {
	Orders              int64 `json:"orders"`
	PendingOrders       int64 `json:"pending_orders"`
	Entitlements        int64 `json:"entitlements"`
	OngoingEntitlements int64 `json:"ongoing_entitlements"`
}

func GetSKUReferences(tx *gorm.DB, skuId int) (*SKUReferences, error) {
	refs := &SKUReferences{}
	items := tx.Model(&ProductOrderItem{}).Where(&ProductOrderItem{SKUId: skuId})
	if err := items.Distinct("order_id").Count(&refs.Orders).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&ProductOrder{}).Where("id IN (?) AND status = ?",
		tx.Model(&ProductOrderItem{}).Select("order_id").Where(&ProductOrderItem{SKUId: skuId}), ProductOrderStatusPending).
		Count(&refs.PendingOrders).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&Entitlement{}).Where(&Entitlement{SKUId: skuId}).Count(&refs.Entitlements).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&Entitlement{}).Where(&Entitlement{SKUId: skuId}).
		Where("state IN ?", []string{EntitlementStateActive, EntitlementStatePaused, EntitlementStateQueued, EntitlementStatePending}).
		Count(&refs.OngoingEntitlements).Error; err != nil {
		return nil, err
	}
	return refs, nil
}

func DeleteProductSKU(skuId int) error {
	return DeleteProductSKUWithConfirmation(skuId, false)
}

func DeleteProductSKUWithConfirmation(skuId int, confirmEntitlements bool) error {
	if skuId <= 0 {
		return errors.New("invalid SKU id")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var sku ProductSKU
		if err := lockForUpdate(tx).Where("id = ?", skuId).First(&sku).Error; err != nil {
			return err
		}
		if sku.DeletedAt != 0 {
			return nil
		}
		refs, err := GetSKUReferences(tx, skuId)
		if err != nil {
			return err
		}
		if refs.PendingOrders > 0 {
			return ErrSKUPendingOrders
		}
		if refs.OngoingEntitlements > 0 && !confirmEntitlements {
			return ErrSKUConfirmationRequired
		}
		if refs.Orders == 0 && refs.Entitlements == 0 {
			return tx.Delete(&sku).Error
		}
		return tx.Model(&sku).Updates(map[string]interface{}{"deleted_at": common.GetTimestamp(), "status": ProductStatusArchived}).Error
	})
}

// Restore into archived state so recovery never silently republishes a SKU.
func RestoreProductSKU(skuId int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var sku ProductSKU
		if err := lockForUpdate(tx).Where("id = ?", skuId).First(&sku).Error; err != nil {
			return err
		}
		if sku.DeletedAt == 0 {
			return nil
		}
		return tx.Model(&sku).Updates(map[string]interface{}{"deleted_at": 0, "status": ProductStatusArchived}).Error
	})
}
