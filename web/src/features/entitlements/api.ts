/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { api } from "@/lib/api";

import type {
  AccessGroupPolicy,
  ApiResponse,
  Entitlement,
  EntitlementAdjustment,
  EntitlementType,
  EntitlementTypeChangeLog,
  EpayResponse,
  PageData,
  Product,
  ProductOrder,
  ProductSKU,
  SKUReferences,
  TopupInfo,
  UsageCharge,
  UserEntitlementData,
  GiftSelfData,
  GiftSettings,
  StoreNotice,
  StoreNoticeAdmin,
  StoreNoticeRevision,
  CorporateCollectionChannel,
  CorporateTransferApplication,
  CorporateTransferCreation,
  CorporateTransferTicketDetail,
  CorporateTransferTicketListItem,
  CorporateReceiptVerification,
  CorporateTransferAvailability,
} from "./types";

export async function getStoreProducts(): Promise<ApiResponse<Product[]>> {
  return (await api.get("/api/store/products")).data;
}

export async function getStoreOrders(): Promise<ApiResponse<ProductOrder[]>> {
  return (await api.get("/api/store/orders")).data;
}

export async function createStoreOrder(data: {
  sku_id: number;
  quantity: number;
  amount_minor?: number;
  gift_discount_cents?: number;
}): Promise<ApiResponse<ProductOrder>> {
  return (await api.post("/api/store/orders", data)).data;
}

export async function createCorporateTransferOrder(
  data: {
    sku_id: number;
    quantity: number;
    amount_minor?: number;
    gift_discount_cents?: number;
    prior_order_no?: string;
  },
  idempotencyKey: string,
): Promise<ApiResponse<CorporateTransferCreation>> {
  return (
    await api.post("/api/store/corporate-transfers", data, {
      headers: { "Idempotency-Key": idempotencyKey },
    })
  ).data;
}

export async function getCorporateTransferAvailability(): Promise<
  ApiResponse<CorporateTransferAvailability>
> {
  return (await api.get("/api/store/corporate-transfers/availability")).data;
}

export async function getCorporateTransferTickets(): Promise<
  ApiResponse<CorporateTransferTicketListItem[]>
> {
  return (await api.get("/api/store/corporate-transfers/tickets")).data;
}

export async function getCorporateTransferUnreadCount(): Promise<
  ApiResponse<{ count: number }>
> {
  return (await api.get("/api/store/corporate-transfers/tickets-unread")).data;
}

export async function getCorporateTransferTicket(
  ticketNo: string,
): Promise<ApiResponse<CorporateTransferTicketDetail>> {
  return (await api.get(`/api/store/corporate-transfers/tickets/${ticketNo}`))
    .data;
}

export async function uploadCorporateTransferEvidence(
  applicationNo: string,
  files: File[],
  message = "",
): Promise<ApiResponse> {
  const form = new FormData();
  files.forEach((file) => form.append("files", file));
  if (message) form.append("message", message);
  return (
    await api.post(
      `/api/store/corporate-transfers/${applicationNo}/evidence`,
      form,
    )
  ).data;
}

export async function replyCorporateTransferTicket(
  applicationNo: string,
  body: string,
  files: File[] = [],
): Promise<ApiResponse> {
  if (files.length > 0) {
    const form = new FormData();
    form.append("body", body);
    files.forEach((file) => form.append("files", file));
    return (
      await api.post(
        `/api/store/corporate-transfers/${applicationNo}/reply`,
        form,
      )
    ).data;
  }
  return (
    await api.post(`/api/store/corporate-transfers/${applicationNo}/reply`, {
      body,
    })
  ).data;
}

export async function cancelCorporateTransfer(
  applicationNo: string,
): Promise<ApiResponse> {
  return (
    await api.post(`/api/store/corporate-transfers/${applicationNo}/cancel`)
  ).data;
}

export async function getAdminCorporateTransfers(params?: {
  page?: number;
  page_size?: number;
  status?: string;
  keyword?: string;
  user_id?: number;
  channel?: string;
}): Promise<ApiResponse<PageData<CorporateTransferApplication>>> {
  return (await api.get("/api/store/admin/corporate-transfers", { params }))
    .data;
}

export async function getAdminCorporateTransferTicket(
  ticketNo: string,
): Promise<ApiResponse<CorporateTransferTicketDetail>> {
  return (
    await api.get(`/api/store/admin/corporate-transfers/tickets/${ticketNo}`)
  ).data;
}

export async function adminReplyCorporateTransfer(
  applicationNo: string,
  body: string,
  internal = false,
  files: File[] = [],
): Promise<ApiResponse> {
  if (files.length > 0) {
    const form = new FormData();
    form.append("body", body);
    form.append("internal", String(internal));
    files.forEach((file) => form.append("files", file));
    return (
      await api.post(
        `/api/store/admin/corporate-transfers/${applicationNo}/reply`,
        form,
      )
    ).data;
  }
  return (
    await api.post(
      `/api/store/admin/corporate-transfers/${applicationNo}/reply`,
      { body, internal },
    )
  ).data;
}

export async function setCorporateTransferTicketOpen(
  applicationNo: string,
  open: boolean,
): Promise<ApiResponse> {
  return (
    await api.post(
      `/api/store/admin/corporate-transfers/${applicationNo}/ticket-state`,
      { open },
    )
  ).data;
}

export async function requestCorporateTransferInfo(
  applicationNo: string,
  reason: string,
): Promise<ApiResponse> {
  return (
    await api.post(
      `/api/store/admin/corporate-transfers/${applicationNo}/request-info`,
      { reason },
    )
  ).data;
}

export async function rejectCorporateTransfer(
  applicationNo: string,
  reason: string,
  internalNote: string,
): Promise<ApiResponse> {
  return (
    await api.post(
      `/api/store/admin/corporate-transfers/${applicationNo}/reject`,
      { reason, internal_note: internalNote },
    )
  ).data;
}

export async function addCorporateReceipt(
  applicationNo: string,
  data: Omit<
    CorporateReceiptVerification,
    "id" | "status" | "void_reason" | "created_at"
  >,
): Promise<ApiResponse<CorporateReceiptVerification>> {
  return (
    await api.post(
      `/api/store/admin/corporate-transfers/${applicationNo}/receipts`,
      data,
    )
  ).data;
}

export async function voidCorporateReceipt(
  receiptId: number,
  reason: string,
): Promise<ApiResponse> {
  return (
    await api.post(
      `/api/store/admin/corporate-transfers/receipts/${receiptId}/void`,
      { reason },
    )
  ).data;
}

export async function approveCorporateTransfer(
  applicationNo: string,
): Promise<ApiResponse> {
  return (
    await api.post(
      `/api/store/admin/corporate-transfers/${applicationNo}/approve`,
    )
  ).data;
}

export async function registerCorporateTransferRefund(
  applicationNo: string,
  data: {
    amount_cents: number;
    channel: "enterprise_wechat" | "corporate_bank";
    external_reference: string;
    refunded_at: number;
    reason: string;
  },
): Promise<ApiResponse> {
  return (
    await api.post(
      `/api/store/admin/corporate-transfers/${applicationNo}/refund`,
      data,
    )
  ).data;
}

export async function getCorporateCollectionAdmin(): Promise<
  ApiResponse<{
    state: { current_revision: number };
    draft?: CorporateCollectionChannel[];
    published_snapshot?: {
      revision: number;
      channels: CorporateCollectionChannel[];
    };
  }>
> {
  return (await api.get("/api/store/admin/corporate-transfer/collection")).data;
}

export async function saveCorporateCollectionDraft(
  channels: CorporateCollectionChannel[],
): Promise<ApiResponse> {
  return (
    await api.put("/api/store/admin/corporate-transfer/collection/draft", {
      channels,
    })
  ).data;
}

export async function publishCorporateCollection(
  expectedRevision: number,
): Promise<ApiResponse> {
  return (
    await api.post("/api/store/admin/corporate-transfer/collection/publish", {
      expected_revision: expectedRevision,
    })
  ).data;
}

export async function uploadCorporateCollectionAsset(
  file: File,
): Promise<ApiResponse<{ id: number }>> {
  const form = new FormData();
  form.append("file", file);
  return (
    await api.post(
      "/api/store/admin/corporate-transfer/collection/assets",
      form,
    )
  ).data;
}

export async function getGiftSelf(): Promise<ApiResponse<GiftSelfData>> {
  return (await api.get("/api/store/gift")).data;
}

export async function getStoreNotice(): Promise<ApiResponse<StoreNotice>> {
  return (await api.get("/api/store/notice")).data;
}

export async function dismissStoreNotice(
  revision: number,
): Promise<ApiResponse> {
  return (await api.post("/api/store/notice/dismiss", { revision })).data;
}

export async function getAdminStoreNotice(): Promise<
  ApiResponse<StoreNoticeAdmin>
> {
  return (await api.get("/api/store/admin/notice")).data;
}

export async function saveStoreNoticeDraft(data: {
  content: string;
  expected_draft_version: number;
}): Promise<ApiResponse<StoreNoticeAdmin>> {
  return (await api.put("/api/store/admin/notice/draft", data)).data;
}

export async function publishStoreNotice(data: {
  expected_revision: number;
  expected_draft_version: number;
}): Promise<ApiResponse<StoreNoticeRevision>> {
  return (await api.post("/api/store/admin/notice/publish", data)).data;
}

export async function getGiftSettings(): Promise<ApiResponse<GiftSettings>> {
  return (await api.get("/api/entitlement/admin/gift/settings")).data;
}

export async function saveGiftSettings(
  data: GiftSettings,
): Promise<ApiResponse> {
  return (await api.put("/api/entitlement/admin/gift/settings", data)).data;
}

export async function adjustUserGift(
  userId: number,
  data: { delta_cents: number; reason: string; idempotency_key: string },
): Promise<ApiResponse> {
  return (await api.post(`/api/entitlement/admin/users/${userId}/gift`, data))
    .data;
}

export async function payStoreOrderEpay(
  orderNo: string,
  paymentMethod: string,
): Promise<EpayResponse> {
  return (
    await api.post(`/api/store/orders/${orderNo}/epay`, {
      payment_method: paymentMethod,
    })
  ).data;
}

export async function cancelStoreOrder(orderNo: string): Promise<ApiResponse> {
  return (await api.post(`/api/store/orders/${orderNo}/cancel`)).data;
}

export async function getTopupInfo(): Promise<ApiResponse<TopupInfo>> {
  return (await api.get("/api/user/topup/info")).data;
}

export async function getSelfEntitlements(): Promise<
  ApiResponse<UserEntitlementData>
> {
  return (await api.get("/api/entitlement/self")).data;
}

export async function activateEntitlement(id: number): Promise<ApiResponse> {
  return (await api.post(`/api/entitlement/${id}/activate`)).data;
}

export async function updateEntitlementPriorities(
  ids: number[],
): Promise<ApiResponse> {
  return (await api.put("/api/entitlement/priorities", { ids })).data;
}

export async function getUsageCharges(): Promise<
  ApiResponse<PageData<UsageCharge>>
> {
  return (await api.get("/api/entitlement/usage-charges")).data;
}

export async function getGroupPolicies(): Promise<
  ApiResponse<AccessGroupPolicy[]>
> {
  return (await api.get("/api/entitlement/admin/group-policies")).data;
}

export async function saveGroupPolicy(data: {
  group_name: string;
  funding_source_type: string;
}): Promise<ApiResponse<AccessGroupPolicy>> {
  return (await api.post("/api/entitlement/admin/group-policies", data)).data;
}

export async function getEntitlementTypes(): Promise<
  ApiResponse<EntitlementType[]>
> {
  return (await api.get("/api/entitlement/admin/types")).data;
}

export async function createEntitlementType(
  data: Partial<EntitlementType>,
): Promise<ApiResponse<EntitlementType>> {
  return (await api.post("/api/entitlement/admin/types", data)).data;
}

export async function updateEntitlementType(
  id: number,
  data: Partial<EntitlementType>,
): Promise<ApiResponse> {
  return (await api.put(`/api/entitlement/admin/types/${id}`, data)).data;
}

export async function replaceTypeGroups(
  id: number,
  groups: string[],
  reason: string,
): Promise<ApiResponse> {
  return (
    await api.put(`/api/entitlement/admin/types/${id}/groups`, {
      groups,
      reason,
    })
  ).data;
}

export async function getTypeChangeLogs(
  id: number,
): Promise<ApiResponse<EntitlementTypeChangeLog[]>> {
  return (await api.get(`/api/entitlement/admin/types/${id}/change-logs`)).data;
}

export async function getAdminProducts(
  includeDeleted = false,
): Promise<ApiResponse<Product[]>> {
  return (
    await api.get("/api/entitlement/admin/products", {
      params: { include_deleted: includeDeleted },
    })
  ).data;
}

export async function createProduct(
  data: Partial<Product>,
): Promise<ApiResponse<Product>> {
  return (await api.post("/api/entitlement/admin/products", data)).data;
}

export async function updateProduct(
  id: number,
  data: Partial<Product>,
): Promise<ApiResponse> {
  return (await api.put(`/api/entitlement/admin/products/${id}`, data)).data;
}

export async function createProductSKU(
  data: Partial<ProductSKU>,
): Promise<ApiResponse<ProductSKU>> {
  return (await api.post("/api/entitlement/admin/skus", data)).data;
}

export async function updateProductSKU(
  id: number,
  data: Partial<ProductSKU>,
): Promise<ApiResponse<ProductSKU>> {
  return (await api.put(`/api/entitlement/admin/skus/${id}`, data)).data;
}

export async function deleteProductSKU(input: {
  id: number;
  confirmEntitlements: boolean;
}): Promise<ApiResponse> {
  return (
    await api.delete(`/api/entitlement/admin/skus/${input.id}`, {
      params: { confirm_entitlements: input.confirmEntitlements },
    })
  ).data;
}

export async function getSKUReferences(
  id: number,
): Promise<ApiResponse<SKUReferences>> {
  return (await api.get(`/api/entitlement/admin/skus/${id}/references`)).data;
}

export async function restoreProductSKU(id: number): Promise<ApiResponse> {
  return (await api.post(`/api/entitlement/admin/skus/${id}/restore`)).data;
}

export async function getAdminOrders(params?: {
  page?: number;
  page_size?: number;
  status?: string;
  user_id?: number;
  keyword?: string;
}): Promise<ApiResponse<PageData<ProductOrder>>> {
  return (await api.get("/api/entitlement/admin/orders", { params })).data;
}

export async function completeProductOrder(
  orderNo: string,
): Promise<ApiResponse> {
  return (
    await api.post(`/api/entitlement/admin/orders/${orderNo}/complete`, {
      provider_trade_no: `manual-${orderNo}`,
      payment_method: "manual",
    })
  ).data;
}

export async function cancelProductOrder(
  orderNo: string,
  reason: string,
): Promise<ApiResponse> {
  return (
    await api.post(`/api/entitlement/admin/orders/${orderNo}/cancel`, {
      reason,
    })
  ).data;
}

export async function refundProductOrder(
  orderNo: string,
  reason: string,
): Promise<ApiResponse> {
  return (
    await api.post(`/api/entitlement/admin/orders/${orderNo}/refund`, {
      reason,
    })
  ).data;
}

export async function resetProductOrderPayment(
  orderNo: string,
  reason: string,
): Promise<ApiResponse> {
  return (
    await api.post(`/api/entitlement/admin/orders/${orderNo}/reset-payment`, {
      reason,
    })
  ).data;
}

export async function getUserEntitlements(
  userId: number,
): Promise<ApiResponse<UserEntitlementData>> {
  return (await api.get(`/api/entitlement/admin/users/${userId}/entitlements`))
    .data;
}

export async function grantEntitlement(
  userId: number,
  data: Partial<Entitlement>,
): Promise<ApiResponse<Entitlement>> {
  return (
    await api.post(`/api/entitlement/admin/users/${userId}/entitlements`, data)
  ).data;
}

export async function grantProductSKUEntitlement(
  userId: number,
  skuId: number,
): Promise<ApiResponse<Entitlement>> {
  return (
    await api.post(
      `/api/entitlement/admin/users/${userId}/entitlements/from-sku`,
      { sku_id: skuId },
    )
  ).data;
}

export async function pauseEntitlement(id: number): Promise<ApiResponse> {
  return (await api.post(`/api/entitlement/admin/entitlements/${id}/pause`))
    .data;
}

export async function resumeEntitlement(id: number): Promise<ApiResponse> {
  return (await api.post(`/api/entitlement/admin/entitlements/${id}/resume`))
    .data;
}

export async function adjustEntitlement(
  id: number,
  data: { delta_quota: number; reason: string; idempotency_key: string },
): Promise<ApiResponse> {
  return (
    await api.post(`/api/entitlement/admin/entitlements/${id}/adjust`, data)
  ).data;
}

export async function getEntitlementAdjustments(
  id: number,
): Promise<ApiResponse<EntitlementAdjustment[]>> {
  return (
    await api.get(`/api/entitlement/admin/entitlements/${id}/adjustments`)
  ).data;
}

export async function revokeEntitlement(id: number): Promise<ApiResponse> {
  return (await api.post(`/api/entitlement/admin/entitlements/${id}/revoke`))
    .data;
}
