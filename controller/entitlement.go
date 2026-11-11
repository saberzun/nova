package controller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AdminListAccessGroupPolicies(c *gin.Context) {
	var policies []model.AccessGroupPolicy
	if err := model.DB.Order("group_name asc").Find(&policies).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, policies)
}

func AdminUpsertAccessGroupPolicy(c *gin.Context) {
	var request model.AccessGroupPolicy
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	request.GroupName = strings.TrimSpace(request.GroupName)
	request.FundingSourceType = strings.TrimSpace(request.FundingSourceType)
	if request.GroupName == "" {
		common.ApiErrorMsg(c, "分组不能为空")
		return
	}
	if _, ok := ratio_setting.GetGroupRatioCopy()[request.GroupName]; !ok {
		common.ApiErrorMsg(c, "New API 分组倍率配置中不存在该分组")
		return
	}
	if request.FundingSourceType != model.EntitlementAssetSubscription &&
		request.FundingSourceType != model.EntitlementAssetStoredValue {
		common.ApiErrorMsg(c, "无效的资金来源类型")
		return
	}
	var existing model.AccessGroupPolicy
	query := model.DB.Where("group_name = ?", request.GroupName).Limit(1).Find(&existing)
	if query.Error != nil {
		common.ApiError(c, query.Error)
		return
	}
	if query.RowsAffected > 0 {
		if existing.FundingSourceType != request.FundingSourceType {
			var references int64
			if err := model.DB.Model(&model.EntitlementTypeGroup{}).
				Where("access_group_policy_id = ?", existing.Id).Count(&references).Error; err != nil {
				common.ApiError(c, err)
				return
			}
			if references > 0 {
				common.ApiErrorMsg(c, "该分组已被权益类型引用，不能改变资金来源")
				return
			}
		}
		existing.FundingSourceType = request.FundingSourceType
		if err := model.DB.Save(&existing).Error; err != nil {
			common.ApiError(c, err)
			return
		}
		common.ApiSuccess(c, existing)
		return
	}
	if err := model.DB.Create(&request).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, request)
}

func AdminListEntitlementTypes(c *gin.Context) {
	var types []model.EntitlementType
	if err := model.DB.Preload("Groups").Order("id desc").Find(&types).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, types)
}

func validateEntitlementType(entitlementType *model.EntitlementType) error {
	if entitlementType == nil || strings.TrimSpace(entitlementType.Code) == "" || strings.TrimSpace(entitlementType.Name) == "" {
		return errors.New("权益类型编码和名称不能为空")
	}
	if entitlementType.AssetKind != model.EntitlementAssetSubscription && entitlementType.AssetKind != model.EntitlementAssetStoredValue {
		return errors.New("权益类型必须是 subscription 或 stored_value")
	}
	if entitlementType.MeterType == "" {
		entitlementType.MeterType = model.EntitlementMeterQuota
	}
	if entitlementType.MeterType != model.EntitlementMeterQuota {
		return errors.New("MVP 只支持 quota 计量")
	}
	if entitlementType.Status != model.EntitlementTypeStatusActive &&
		entitlementType.Status != model.EntitlementTypeStatusDisabled &&
		entitlementType.Status != model.EntitlementTypeStatusArchived {
		return errors.New("权益类型状态无效")
	}
	return nil
}

func AdminCreateEntitlementType(c *gin.Context) {
	var entitlementType model.EntitlementType
	if err := c.ShouldBindJSON(&entitlementType); err != nil {
		common.ApiError(c, err)
		return
	}
	entitlementType.Id = 0
	if entitlementType.Status == "" {
		entitlementType.Status = model.EntitlementTypeStatusActive
	}
	if err := validateEntitlementType(&entitlementType); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if err := model.DB.Create(&entitlementType).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.type_create", map[string]interface{}{"id": entitlementType.Id, "code": entitlementType.Code})
	common.ApiSuccess(c, entitlementType)
}

func AdminUpdateEntitlementType(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var request model.EntitlementType
	if id <= 0 || c.ShouldBindJSON(&request) != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	var existing model.EntitlementType
	if err := model.DB.Where("id = ?", id).First(&existing).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	var entitlementCount int64
	if err := model.DB.Model(&model.Entitlement{}).Where("entitlement_type_id = ?", id).Count(&entitlementCount).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if entitlementCount > 0 && (request.Code != existing.Code || request.AssetKind != existing.AssetKind || request.MeterType != existing.MeterType) {
		common.ApiErrorMsg(c, "已有权益引用后不能修改 code、asset_kind 或 meter_type")
		return
	}
	if err := validateEntitlementType(&request); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	nextRevision := existing.Revision + 1
	updates := map[string]interface{}{
		"code":        strings.TrimSpace(request.Code),
		"name":        strings.TrimSpace(request.Name),
		"description": request.Description,
		"asset_kind":  request.AssetKind,
		"meter_type":  request.MeterType,
		"status":      request.Status,
		"revision":    nextRevision,
		"updated_at":  common.GetTimestamp(),
	}
	beforeJSON, err := common.Marshal(map[string]interface{}{
		"code": existing.Code, "name": existing.Name, "description": existing.Description,
		"asset_kind": existing.AssetKind, "meter_type": existing.MeterType, "status": existing.Status,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	afterJSON, err := common.Marshal(map[string]interface{}{
		"code": updates["code"], "name": updates["name"], "description": updates["description"],
		"asset_kind": updates["asset_kind"], "meter_type": updates["meter_type"], "status": updates["status"],
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&existing).Updates(updates).Error; err != nil {
			return err
		}
		return tx.Create(&model.EntitlementTypeChangeLog{
			EntitlementTypeId: id,
			Revision:          nextRevision,
			Action:            "update_type",
			BeforeJSON:        string(beforeJSON),
			AfterJSON:         string(afterJSON),
			OperatorId:        c.GetInt("id"),
			Reason:            "admin update",
		}).Error
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.type_update", map[string]interface{}{"id": id})
	common.ApiSuccess(c, nil)
}

type replaceEntitlementTypeGroupsRequest struct {
	Groups []string `json:"groups"`
	Reason string   `json:"reason"`
}

func AdminReplaceEntitlementTypeGroups(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var request replaceEntitlementTypeGroupsRequest
	if id <= 0 || c.ShouldBindJSON(&request) != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.ReplaceEntitlementTypeGroups(id, request.Groups, c.GetInt("id"), request.Reason); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.type_groups", map[string]interface{}{"id": id, "reason": request.Reason})
	common.ApiSuccess(c, nil)
}

func AdminListEntitlementTypeChangeLogs(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var logs []model.EntitlementTypeChangeLog
	if id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.DB.Where("entitlement_type_id = ?", id).Order("id desc").Limit(200).Find(&logs).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, logs)
}

func AdminListProducts(c *gin.Context) {
	var products []model.Product
	query := model.DB.Preload("SKUs", "deleted_at = 0")
	if c.Query("include_deleted") == "true" {
		query = model.DB.Preload("SKUs")
	}
	if err := query.Order("sort_order desc, id desc").Find(&products).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, products)
}

func AdminCreateProduct(c *gin.Context) {
	var product model.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		common.ApiError(c, err)
		return
	}
	product.Id = 0
	if product.Status == "" {
		product.Status = model.ProductStatusDraft
	}
	if strings.TrimSpace(product.Code) == "" || strings.TrimSpace(product.Name) == "" ||
		(product.Category != model.ProductCategorySubscription && product.Category != model.ProductCategoryRecharge) ||
		!validProductStatus(product.Status) {
		common.ApiErrorMsg(c, "商品编码、名称或分类无效")
		return
	}
	if err := model.DB.Create(&product).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.product_create", map[string]interface{}{"id": product.Id, "code": product.Code})
	common.ApiSuccess(c, product)
}

func validProductStatus(status string) bool {
	return status == model.ProductStatusDraft || status == model.ProductStatusActive ||
		status == model.ProductStatusPaused || status == model.ProductStatusArchived
}

func AdminUpdateProduct(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var request model.Product
	if id <= 0 || c.ShouldBindJSON(&request) != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	var existing model.Product
	if err := model.DB.Where("id = ?", id).First(&existing).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if strings.TrimSpace(request.Code) == "" || strings.TrimSpace(request.Name) == "" ||
		(request.Category != model.ProductCategorySubscription && request.Category != model.ProductCategoryRecharge) ||
		!validProductStatus(request.Status) {
		common.ApiErrorMsg(c, "商品编码、名称、分类或状态无效")
		return
	}
	if request.Category != existing.Category {
		var skuCount int64
		if err := model.DB.Model(&model.ProductSKU{}).Where("product_id = ?", id).Count(&skuCount).Error; err != nil {
			common.ApiError(c, err)
			return
		}
		if skuCount > 0 {
			common.ApiErrorMsg(c, "已有 SKU 后不能修改商品分类")
			return
		}
	}
	if err := model.DB.Model(&existing).Updates(map[string]interface{}{
		"code":            strings.TrimSpace(request.Code),
		"name":            strings.TrimSpace(request.Name),
		"description":     request.Description,
		"category":        request.Category,
		"status":          request.Status,
		"sort_order":      request.SortOrder,
		"visibility_rule": request.VisibilityRule,
		"updated_at":      common.GetTimestamp(),
	}).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.product_update", map[string]interface{}{"id": id})
	common.ApiSuccess(c, nil)
}

func validateProductSKU(sku *model.ProductSKU) error {
	if sku == nil || sku.ProductId <= 0 || sku.EntitlementTypeId <= 0 || strings.TrimSpace(sku.Code) == "" || strings.TrimSpace(sku.Name) == "" {
		return errors.New("SKU 参数不完整")
	}
	if sku.PriceAmountMinor < 0 || sku.GrantTotalQuota <= 0 || sku.GrantDailyQuota < 0 || sku.ValiditySeconds < 0 {
		return errors.New("SKU 价格或额度无效")
	}
	if !validProductStatus(sku.Status) {
		return errors.New("SKU 状态无效")
	}
	if !sku.StockLimited {
		sku.Stock = 0
	} else if sku.Stock < 0 {
		return errors.New("SKU 库存无效")
	}
	var product model.Product
	if err := model.DB.Where("id = ?", sku.ProductId).First(&product).Error; err != nil {
		return err
	}
	var entitlementType model.EntitlementType
	if err := model.DB.Where("id = ?", sku.EntitlementTypeId).First(&entitlementType).Error; err != nil {
		return err
	}
	if product.Category == model.ProductCategorySubscription {
		if entitlementType.AssetKind != model.EntitlementAssetSubscription || sku.ValiditySeconds <= 0 {
			return errors.New("订阅 SKU 必须关联 subscription Type 并配置有效期")
		}
	} else if product.Category == model.ProductCategoryRecharge {
		if entitlementType.AssetKind != model.EntitlementAssetStoredValue || sku.ActivationPolicy != model.ActivationPolicyImmediate {
			return errors.New("充值 SKU 必须关联 stored_value Type 并立即生效")
		}
		if sku.PriceAmountMinor <= 0 {
			return errors.New("充值 SKU 必须配置大于零的计价基准")
		}
		sku.NormalizeRechargeAmountBounds()
		if sku.MinRechargeAmountMinor < model.DefaultRechargeMinAmount ||
			sku.MaxRechargeAmountMinor < sku.MinRechargeAmountMinor ||
			sku.MaxRechargeAmountMinor > model.DefaultRechargeMaxAmount {
			return errors.New("充值金额范围无效")
		}
		if sku.Status != model.ProductStatusArchived {
			var existing int64
			if err := model.DB.Model(&model.ProductSKU{}).
				Where("product_id = ? AND id <> ? AND status <> ?", sku.ProductId, sku.Id, model.ProductStatusArchived).
				Count(&existing).Error; err != nil {
				return err
			}
			if existing > 0 {
				return errors.New("充值商品只能配置一个有效 SKU 作为计价规则")
			}
		}
	}
	return nil
}

func AdminCreateProductSKU(c *gin.Context) {
	var sku model.ProductSKU
	if err := c.ShouldBindJSON(&sku); err != nil {
		common.ApiError(c, err)
		return
	}
	sku.Id = 0
	if sku.Status == "" {
		sku.Status = model.ProductStatusDraft
	}
	if err := validateProductSKU(&sku); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	if err := model.DB.Create(&sku).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.sku_create", map[string]interface{}{"id": sku.Id, "code": sku.Code})
	common.ApiSuccess(c, sku)
}

func AdminUpdateProductSKU(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var sku model.ProductSKU
	if id <= 0 || c.ShouldBindJSON(&sku) != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	var existing model.ProductSKU
	if err := model.DB.Where("id = ? AND deleted_at = 0", id).First(&existing).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	sku.Id = id
	if err := validateProductSKU(&sku); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	result := model.DB.Model(&existing).Where("deleted_at = 0").Updates(map[string]interface{}{
		"code":                      strings.TrimSpace(sku.Code),
		"product_id":                sku.ProductId,
		"entitlement_type_id":       sku.EntitlementTypeId,
		"name":                      strings.TrimSpace(sku.Name),
		"price_amount_minor":        sku.PriceAmountMinor,
		"currency":                  sku.Currency,
		"grant_total_quota":         sku.GrantTotalQuota,
		"grant_daily_quota":         sku.GrantDailyQuota,
		"min_recharge_amount_minor": sku.MinRechargeAmountMinor,
		"max_recharge_amount_minor": sku.MaxRechargeAmountMinor,
		"validity_seconds":          sku.ValiditySeconds,
		"activation_policy":         sku.ActivationPolicy,
		"activation_deadline_sec":   sku.ActivationDeadlineSec,
		"stock":                     sku.Stock,
		"stock_limited":             sku.StockLimited,
		"purchase_limit":            sku.PurchaseLimit,
		"multi_quantity_enabled":    sku.MultiQuantityEnabled,
		"status":                    sku.Status,
		"sort_order":                sku.SortOrder,
		"updated_at":                common.GetTimestamp(),
	})
	if result.Error != nil {
		common.ApiError(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		common.ApiErrorI18n(c, i18n.MsgSKUDeleted)
		return
	}
	recordManageAudit(c, "entitlement.sku_update", map[string]interface{}{"id": id})
	common.ApiSuccess(c, sku)
}

func AdminDeleteProductSKU(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	var sku model.ProductSKU
	if err := model.DB.Where("id = ?", id).First(&sku).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteProductSKUWithConfirmation(id, c.Query("confirm_entitlements") == "true"); err != nil {
		if errors.Is(err, model.ErrSKUPendingOrders) {
			common.ApiErrorI18n(c, i18n.MsgSKUPendingOrders)
			return
		}
		if errors.Is(err, model.ErrSKUConfirmationRequired) {
			common.ApiErrorI18n(c, i18n.MsgSKUConfirmEntitlements)
			return
		}
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.sku_delete", map[string]interface{}{"id": id, "code": sku.Code})
	common.ApiSuccess(c, nil)
}

func AdminGetSKUReferences(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var sku model.ProductSKU
	if id <= 0 {
		common.ApiErrorI18n(c, i18n.MsgSKUNotFound)
		return
	}
	if err := model.DB.First(&sku, id).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	refs, err := model.GetSKUReferences(model.DB, id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, refs)
}

func AdminRestoreProductSKU(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.RestoreProductSKU(id); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.sku_restore", map[string]interface{}{"id": id})
	common.ApiSuccess(c, nil)
}

func ListStoreProducts(c *gin.Context) {
	var products []model.Product
	if err := model.DB.Preload("SKUs", "status = ? AND deleted_at = 0", model.ProductStatusActive).
		Preload("SKUs.EntitlementType.Groups").
		Where("status = ?", model.ProductStatusActive).
		Order("sort_order desc, id desc").Find(&products).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	for productIndex := range products {
		if products[productIndex].Category != model.ProductCategoryRecharge {
			continue
		}
		for skuIndex := range products[productIndex].SKUs {
			products[productIndex].SKUs[skuIndex].NormalizeRechargeAmountBounds()
		}
	}
	common.ApiSuccess(c, products)
}

type createProductOrderRequest struct {
	SKUId             int   `json:"sku_id"`
	Quantity          int   `json:"quantity"`
	AmountMinor       int64 `json:"amount_minor"`
	GiftDiscountCents int64 `json:"gift_discount_cents"`
}

func CreateStoreOrder(c *gin.Context) {
	var request createProductOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	if request.Quantity == 0 {
		request.Quantity = 1
	}
	order, err := model.CreateProductOrderWithGift(c.GetInt("id"), request.SKUId, request.Quantity, request.AmountMinor, request.GiftDiscountCents)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if order.TotalAmountMinor == 0 {
		paymentMethod := "free"
		if order.GiftDiscountCents > 0 {
			paymentMethod = "gift"
		}
		if err := model.CompleteProductOrder(order.OrderNo, "", paymentMethod); err != nil {
			common.ApiError(c, err)
			return
		}
		order.Status = model.ProductOrderStatusFulfilled
	}
	common.ApiSuccess(c, order)
}

func GetGiftSelf(c *gin.Context) {
	userId := c.GetInt("id")
	account, err := model.GetGiftAccount(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	ledger, err := model.ListGiftLedger(userId, 100)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	rewards, err := model.ListReferralRewardsForInviter(userId, 100)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	rewardRows := make([]gin.H, 0, len(rewards))
	for _, reward := range rewards {
		friend := "***"
		var invitee model.User
		if queryErr := model.DB.Select("username", "email").Where("id = ?", reward.InviteeUserId).Limit(1).Find(&invitee).Error; queryErr == nil {
			identity := strings.TrimSpace(invitee.Email)
			if identity == "" {
				identity = strings.TrimSpace(invitee.Username)
			}
			if at := strings.Index(identity, "@"); at > 2 {
				friend = identity[:2] + "***" + identity[at:]
			} else if len(identity) > 2 {
				friend = identity[:2] + "***"
			}
		}
		rewardRows = append(rewardRows, gin.H{
			"id": reward.Id, "friend": friend, "source_order_no": reward.SourceOrderNo,
			"source_category": reward.SourceCategory, "reward_tier": reward.RewardTier,
			"cash_paid_cents": reward.CashPaidCents, "reward_rate_bps": reward.RewardRateBps,
			"reward_cents": reward.RewardCents, "status": reward.Status,
			"release_at": reward.ReleaseAt, "created_at": reward.CreatedAt,
		})
	}
	settings, err := model.GetGiftSettings()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var user model.User
	if err := model.DB.Select("id", "aff_code", "aff_count").Where("id = ?", userId).First(&user).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	var pendingCents int64
	var totalCents int64
	_ = model.DB.Model(&model.ReferralRewardEvent{}).
		Where("inviter_user_id = ? AND status = ?", userId, model.ReferralRewardPending).
		Select("COALESCE(SUM(reward_cents), 0)").Scan(&pendingCents).Error
	_ = model.DB.Model(&model.ReferralRewardEvent{}).
		Where("inviter_user_id = ? AND status IN ?", userId, []string{model.ReferralRewardReleased, model.ReferralRewardPending}).
		Select("COALESCE(SUM(reward_cents), 0)").Scan(&totalCents).Error
	common.ApiSuccess(c, gin.H{
		"account": account, "ledger": ledger, "rewards": rewardRows, "settings": settings,
		"aff_code": user.AffCode, "aff_count": user.AffCount,
		"pending_cents": pendingCents, "total_reward_cents": totalCents,
	})
}

func AdminGetGiftSettings(c *gin.Context) {
	settings, err := model.GetGiftSettings()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, settings)
}

func AdminSaveGiftSettings(c *gin.Context) {
	var settings model.GiftSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.SaveGiftSettings(&settings); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "gift.settings_update", map[string]interface{}{
		"checkout_enabled": settings.CheckoutEnabled, "referral_enabled": settings.ReferralEnabled,
		"first_rate_bps": settings.FirstRateBps, "recurring_rate_bps": settings.RecurringRateBps,
	})
	common.ApiSuccess(c, nil)
}

type adminGiftAdjustmentRequest struct {
	DeltaCents     int64  `json:"delta_cents"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
}

func AdminAdjustGift(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	var request adminGiftAdjustmentRequest
	if userId <= 0 || c.ShouldBindJSON(&request) != nil || request.DeltaCents == 0 ||
		strings.TrimSpace(request.Reason) == "" || strings.TrimSpace(request.IdempotencyKey) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.AdminAdjustGift(userId, c.GetInt("id"), request.DeltaCents, request.Reason, request.IdempotencyKey); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, userId, "gift.adjust", map[string]interface{}{
		"delta_cents": request.DeltaCents, "reason": request.Reason,
	})
	common.ApiSuccess(c, nil)
}

func AdminGetUserGift(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	if userId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	account, err := model.GetGiftAccount(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	ledger, err := model.ListGiftLedger(userId, 200)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"account": account, "ledger": ledger})
}

func AdminReleaseReferralRewards(c *gin.Context) {
	count, err := model.ReleaseDueReferralRewards(500)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "gift.referral_release", map[string]interface{}{"count": count})
	common.ApiSuccess(c, gin.H{"released": count})
}

func AdminAuditGiftAccounting(c *gin.Context) {
	report, err := model.AuditGiftAccounting()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, report)
}

type freezeLegacyQuotaRequest struct {
	BatchNo string `json:"batch_no"`
}

func AdminFreezeLegacyQuota(c *gin.Context) {
	var request freezeLegacyQuotaRequest
	if c.ShouldBindJSON(&request) != nil || strings.TrimSpace(request.BatchNo) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	count, err := model.FreezeLegacyQuota(request.BatchNo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "gift.legacy_quota_freeze", map[string]interface{}{"batch_no": request.BatchNo, "count": count})
	common.ApiSuccess(c, gin.H{"created": count})
}

func ListStoreOrders(c *gin.Context) {
	var orders []model.ProductOrder
	if err := model.DB.Preload("Items").Where("user_id = ?", c.GetInt("id")).Order("id desc").Find(&orders).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	for index := range orders {
		if orders[index].PaymentProvider != model.PaymentProviderCorporateTransfer {
			continue
		}
		var application model.CorporateTransferApplication
		if err := model.DB.Select("ticket_id", "status").Where("order_id = ?", orders[index].Id).Limit(1).Find(&application).Error; err != nil {
			common.ApiError(c, err)
			return
		}
		if application.TicketId == 0 {
			continue
		}
		var ticket model.SupportTicket
		if err := model.DB.Select("ticket_no").Where("id = ?", application.TicketId).First(&ticket).Error; err != nil {
			common.ApiError(c, err)
			return
		}
		orders[index].CorporateTicketNo = ticket.TicketNo
		orders[index].CorporateTransferStatus = application.Status
	}
	common.ApiSuccess(c, orders)
}

func CancelStoreOrder(c *gin.Context) {
	if err := model.CancelProductOrder(c.Param("order_no"), c.GetInt("id"), "cancelled by user"); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func AdminListProductOrders(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	query := model.DB.Model(&model.ProductOrder{})
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		query = query.Where("status = ?", status)
	}
	if userId, _ := strconv.Atoi(c.Query("user_id")); userId > 0 {
		query = query.Where("user_id = ?", userId)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		query = query.Where("order_no LIKE ?", "%"+keyword+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	var orders []model.ProductOrder
	if err := query.Preload("Items").Order("id desc").
		Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&orders).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(orders)
	common.ApiSuccess(c, pageInfo)
}

type completeProductOrderRequest struct {
	ProviderTradeNo string `json:"provider_trade_no"`
	PaymentMethod   string `json:"payment_method"`
}

func AdminCompleteProductOrder(c *gin.Context) {
	var request completeProductOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	var order model.ProductOrder
	if err := model.DB.Select("payment_provider").Where("order_no = ?", c.Param("order_no")).First(&order).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if order.PaymentProvider != "" {
		common.ApiErrorMsg(c, "已绑定支付渠道的订单不能手工完成")
		return
	}
	if err := model.CompleteProductOrder(c.Param("order_no"), request.ProviderTradeNo, request.PaymentMethod); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.order_complete", map[string]interface{}{"order_no": c.Param("order_no")})
	common.ApiSuccess(c, nil)
}

type productOrderStatusRequest struct {
	Reason string `json:"reason"`
}

func AdminCancelProductOrder(c *gin.Context) {
	var request productOrderStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	if strings.TrimSpace(request.Reason) == "" {
		common.ApiErrorMsg(c, "取消原因不能为空")
		return
	}
	if err := model.CancelProductOrder(c.Param("order_no"), 0, request.Reason); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.order_cancel", map[string]interface{}{"order_no": c.Param("order_no"), "reason": request.Reason})
	common.ApiSuccess(c, nil)
}

func AdminRefundProductOrder(c *gin.Context) {
	var request productOrderStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	if strings.TrimSpace(request.Reason) == "" {
		common.ApiErrorMsg(c, "退款原因不能为空")
		return
	}
	var order model.ProductOrder
	if err := model.DB.Select("payment_provider").Where("order_no = ?", c.Param("order_no")).First(&order).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if order.PaymentProvider == model.PaymentProviderCorporateTransfer {
		common.ApiErrorMsg(c, "对公转账订单必须通过退款登记流程处理")
		return
	}
	if err := model.RefundProductOrder(c.Param("order_no"), c.GetInt("id"), request.Reason); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.order_refund", map[string]interface{}{"order_no": c.Param("order_no"), "reason": request.Reason})
	common.ApiSuccess(c, nil)
}

func AdminResetProductOrderPayment(c *gin.Context) {
	var request productOrderStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	if strings.TrimSpace(request.Reason) == "" {
		common.ApiErrorMsg(c, "重置原因不能为空")
		return
	}
	var order model.ProductOrder
	if err := model.DB.Where("order_no = ?", c.Param("order_no")).First(&order).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if order.Status != model.ProductOrderStatusPending || order.PaymentProvider == "" {
		common.ApiErrorMsg(c, "订单不处于待对账状态")
		return
	}
	if order.PaymentProvider == model.PaymentProviderCorporateTransfer {
		common.ApiErrorMsg(c, "对公转账申请不能重置为普通支付订单")
		return
	}
	if err := model.ResetProductOrderPaymentPreparation(order.OrderNo, order.UserId, order.PaymentProvider, request.Reason); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.order_reset", map[string]interface{}{"order_no": order.OrderNo, "reason": request.Reason})
	common.ApiSuccess(c, nil)
}

func ListUserEntitlements(c *gin.Context) {
	userId := c.GetInt("id")
	if target := c.Param("id"); target != "" {
		parsed, _ := strconv.Atoi(target)
		if parsed > 0 {
			userId = parsed
		}
	}
	if err := model.RefreshUserEntitlementStates(userId); err != nil {
		common.ApiError(c, err)
		return
	}
	var entitlements []model.Entitlement
	if err := model.DB.Where("user_id = ?", userId).Order("sort_order asc, id desc").Find(&entitlements).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	typeIds := make([]int, 0, len(entitlements))
	for _, entitlement := range entitlements {
		typeIds = append(typeIds, entitlement.EntitlementTypeId)
	}
	var types []model.EntitlementType
	if len(typeIds) > 0 {
		_ = model.DB.Preload("Groups").Where("id IN ?", typeIds).Find(&types).Error
	}
	common.ApiSuccess(c, gin.H{"entitlements": entitlements, "types": types})
}

func ActivateUserEntitlement(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := model.ActivateEntitlement(c.GetInt("id"), id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

type updateEntitlementPrioritiesRequest struct {
	Ids []int `json:"ids"`
}

func UpdateEntitlementPriorities(c *gin.Context) {
	var request updateEntitlementPrioritiesRequest
	if c.ShouldBindJSON(&request) != nil || len(request.Ids) == 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	seen := make(map[int]struct{}, len(request.Ids))
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		for index, id := range request.Ids {
			if id <= 0 {
				return errors.New("权益 ID 无效")
			}
			if _, ok := seen[id]; ok {
				return errors.New("权益 ID 重复")
			}
			seen[id] = struct{}{}
			result := tx.Model(&model.Entitlement{}).Where("id = ? AND user_id = ?", id, c.GetInt("id")).Update("sort_order", index)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return gorm.ErrRecordNotFound
			}
		}
		return nil
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func ListUserUsageCharges(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	var charges []model.UsageCharge
	query := model.DB.Preload("Allocations").Where("user_id = ?", c.GetInt("id")).Order("id desc")
	if err := query.Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&charges).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	var total int64
	_ = model.DB.Model(&model.UsageCharge{}).Where("user_id = ?", c.GetInt("id")).Count(&total).Error
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(charges)
	common.ApiSuccess(c, pageInfo)
}

func AdminGrantEntitlement(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	var request model.Entitlement
	if userId <= 0 || c.ShouldBindJSON(&request) != nil || request.EntitlementTypeId <= 0 || request.TotalQuota <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	var entitlementType model.EntitlementType
	if err := model.DB.Where("id = ?", request.EntitlementTypeId).First(&entitlementType).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	now := time.Now().Unix()
	request.Id = 0
	request.UserId = userId
	request.AssetKind = entitlementType.AssetKind
	request.SourceType = "admin"
	request.SourceId = c.GetInt("id")
	if request.State == "" {
		request.State = model.EntitlementStateActive
	}
	if request.State == model.EntitlementStateActive && request.StartAt == 0 {
		request.StartAt = now
	}
	if err := model.DB.Create(&request).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, userId, "entitlement.grant", map[string]interface{}{"id": request.Id})
	common.ApiSuccess(c, request)
}

type grantProductSKUEntitlementRequest struct {
	SKUId int `json:"sku_id"`
}

func AdminGrantProductSKUEntitlement(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Param("id"))
	var request grantProductSKUEntitlementRequest
	if userId <= 0 || c.ShouldBindJSON(&request) != nil || request.SKUId <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	entitlement, err := model.GrantProductSKUEntitlement(userId, request.SKUId, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, userId, "entitlement.grant", map[string]interface{}{"id": entitlement.Id, "sku_id": request.SKUId})
	common.ApiSuccess(c, entitlement)
}

type adjustEntitlementRequest struct {
	DeltaQuota     int64  `json:"delta_quota"`
	Reason         string `json:"reason"`
	IdempotencyKey string `json:"idempotency_key"`
}

func AdminListEntitlementAdjustments(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	var ledgers []model.EntitlementAdjustmentLedger
	if err := model.DB.Where("entitlement_id = ?", id).Order("id desc").Limit(200).Find(&ledgers).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, ledgers)
}

func AdminAdjustEntitlement(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var request adjustEntitlementRequest
	if id <= 0 || c.ShouldBindJSON(&request) != nil || request.DeltaQuota == 0 || strings.TrimSpace(request.Reason) == "" || strings.TrimSpace(request.IdempotencyKey) == "" {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	err := model.AdjustEntitlement(id, c.GetInt("id"), request.DeltaQuota, request.Reason, request.IdempotencyKey)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.adjust", map[string]interface{}{"id": id, "delta_quota": request.DeltaQuota, "reason": request.Reason})
	common.ApiSuccess(c, nil)
}

func AdminRevokeEntitlement(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.RevokeEntitlement(id); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.revoke", map[string]interface{}{"id": id})
	common.ApiSuccess(c, nil)
}

func AdminPauseEntitlement(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.PauseEntitlement(id); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.pause", map[string]interface{}{"id": id})
	common.ApiSuccess(c, nil)
}

func AdminResumeEntitlement(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if err := model.ResumeEntitlement(id); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAudit(c, "entitlement.resume", map[string]interface{}{"id": id})
	common.ApiSuccess(c, nil)
}

func formatEntitlementError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("权益操作失败: %s", err.Error())
}
