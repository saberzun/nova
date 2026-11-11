/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

export type AssetKind = "subscription" | "stored_value";
export type EntityStatus = "draft" | "active" | "paused" | "archived";

export interface ApiResponse<T = unknown> {
  success: boolean;
  message?: string;
  data?: T;
}

export interface StoreNotice {
  published: boolean;
  revision: number;
  content: string;
  published_at: number;
  dismissed: boolean;
}

export interface StoreNoticeAdmin {
  current_revision: number;
  current_content: string;
  published_at: number;
  draft_content: string;
  draft_version: number;
  draft_updated_by: number;
  draft_updated_at: number;
}

export interface StoreNoticeRevision {
  id: number;
  revision: number;
  content: string;
  content_hash: string;
  summary: string;
  published_by: number;
  published_at: number;
}

export interface PageData<T> {
  page: number;
  page_size: number;
  total: number;
  items: T[];
}

export interface AccessGroupPolicy {
  id: number;
  group_name: string;
  funding_source_type: AssetKind;
  created_at: number;
  updated_at: number;
}

export interface EntitlementTypeGroup {
  id: number;
  entitlement_type_id: number;
  access_group_policy_id: number;
  group_name: string;
}

export interface EntitlementType {
  id: number;
  code: string;
  name: string;
  description: string;
  asset_kind: AssetKind;
  meter_type: "quota";
  status: "active" | "disabled" | "archived";
  revision: number;
  groups?: EntitlementTypeGroup[];
}

export interface ProductSKU {
  deleted_at?: number;
  id: number;
  code: string;
  product_id: number;
  entitlement_type_id: number;
  name: string;
  price_amount_minor: number;
  currency: string;
  grant_total_quota: number;
  grant_daily_quota: number;
  min_recharge_amount_minor: number;
  max_recharge_amount_minor: number;
  validity_seconds: number;
  activation_policy: "immediate" | "manual" | "deferred";
  activation_deadline_seconds: number;
  stock: number;
  stock_limited: boolean;
  purchase_limit: number;
  multi_quantity_enabled: boolean;
  status: EntityStatus;
  sort_order: number;
  entitlement_type?: EntitlementType;
}

export interface SKUReferences {
  orders: number;
  pending_orders: number;
  entitlements: number;
  ongoing_entitlements: number;
}

export interface Product {
  id: number;
  code: string;
  name: string;
  description: string;
  category: "subscription" | "recharge";
  status: EntityStatus;
  sort_order: number;
  visibility_rule: string;
  skus?: ProductSKU[];
}

export interface ProductOrderItem {
  id: number;
  product_name: string;
  sku_name: string;
  quantity: number;
  unit_price_amount_minor: number;
  grant_total_quota: number;
}

export interface ProductOrder {
  id: number;
  order_no: string;
  user_id: number;
  status:
    | "pending"
    | "paid"
    | "fulfilled"
    | "cancelled"
    | "refunded"
    | "payment_exception";
  payment_method: string;
  payment_provider: string;
  total_amount_minor: number;
  original_amount_cents: number;
  promotion_discount_cents: number;
  gift_discount_cents: number;
  cash_payable_cents: number;
  cash_paid_cents: number;
  gift_status: string;
  currency: string;
  expires_at: number;
  created_at: number;
  paid_at: number;
  fulfilled_at: number;
  cancelled_at: number;
  refunded_at: number;
  status_reason: string;
  items?: ProductOrderItem[];
  corporate_ticket_no?: string;
  corporate_transfer_status?: CorporateTransferStatus;
}

export interface Entitlement {
  id: number;
  user_id: number;
  entitlement_type_id: number;
  asset_kind: AssetKind;
  product_id: number;
  sku_id: number;
  state:
    | "pending"
    | "queued"
    | "active"
    | "paused"
    | "depleted"
    | "expired"
    | "cancelled";
  total_quota: number;
  used_quota: number;
  reserved_quota: number;
  daily_quota: number;
  reset_timezone: string;
  start_at: number;
  expire_at: number;
  activation_deadline: number;
  sort_order: number;
  source_type: string;
  created_at: number;
}

export interface EntitlementAdjustment {
  id: number;
  entitlement_id: number;
  operator_id: number;
  delta_quota: number;
  reason: string;
  created_at: number;
}

export interface UserEntitlementData {
  entitlements: Entitlement[];
  types: EntitlementType[];
}

export interface UsageChargeAllocation {
  id: number;
  entitlement_id: number;
  reserved_quota: number;
  settled_quota: number;
  refunded_quota: number;
  state: string;
}

export interface UsageCharge {
  id: number;
  request_id: string;
  access_group: string;
  asset_kind: AssetKind;
  model_name: string;
  estimated_quota: number;
  reserved_quota: number;
  settled_quota: number;
  uncovered_quota: number;
  state: string;
  created_at: number;
  allocations?: UsageChargeAllocation[];
}

export interface PaymentMethod {
  type: string;
  name?: string;
}

export interface TopupInfo {
  enable_online_topup: boolean;
  epay_missing_configuration: EpayMissingConfiguration[];
  pay_methods: PaymentMethod[];
}

export type EpayMissingConfiguration =
  | "payment_compliance"
  | "gateway_address"
  | "merchant_id"
  | "merchant_key"
  | "payment_methods";

export interface CorporateTransferAvailability {
  available: boolean;
  revision: number;
}

export interface EpayResponse {
  message: string;
  data?: Record<string, string>;
  url?: string;
}

export interface EntitlementTypeChangeLog {
  id: number;
  revision: number;
  action: string;
  before_json: string;
  after_json: string;
  operator_id: number;
  reason: string;
  created_at: number;
}

export interface GiftAccount {
  user_id: number;
  available_cents: number;
  reserved_cents: number;
  debt_cents: number;
}

export interface GiftSettings {
  checkout_enabled: boolean;
  referral_enabled: boolean;
  first_rate_bps: number;
  recurring_rate_bps: number;
  min_cash_paid_cents: number;
  reward_cap_cents: number;
  cooling_days: number;
  large_grant_approval_cents: number;
}

export interface GiftLedgerEntry {
  id: number;
  business_type: string;
  business_id: string;
  available_delta_cents: number;
  reserved_delta_cents: number;
  debt_delta_cents: number;
  reason: string;
  created_at: number;
}

export interface ReferralRewardEvent {
  id: number;
  friend: string;
  source_order_no: string;
  source_category: "subscription" | "recharge";
  reward_tier: "first" | "recurring";
  cash_paid_cents: number;
  reward_rate_bps: number;
  reward_cents: number;
  status: "pending" | "released" | "cancelled" | "reversed";
  release_at: number;
  created_at: number;
}

export interface GiftSelfData {
  account: GiftAccount;
  ledger: GiftLedgerEntry[];
  rewards: ReferralRewardEvent[];
  settings: GiftSettings;
  aff_code: string;
  aff_count: number;
  pending_cents: number;
  total_reward_cents: number;
}

export type CorporateTransferStatus =
  | "awaiting_evidence"
  | "under_review"
  | "needs_more_information"
  | "approved"
  | "fulfilled"
  | "cancelled"
  | "expired"
  | "rejected";

export interface CorporateCollectionChannel {
  channel_type: "enterprise_wechat" | "corporate_bank";
  enabled: boolean;
  display_name: string;
  organization_name: string;
  qr_code_attachment_id?: number;
  account_name?: string;
  bank_name?: string;
  bank_account?: string;
  bank_branch?: string;
  instructions?: string;
}

export interface CorporateCollectionSnapshot {
  revision: number;
  channels: CorporateCollectionChannel[];
}

export interface CorporateTransferApplication {
  id: number;
  application_no: string;
  order_id: number;
  ticket_id: number;
  user_id: number;
  status: CorporateTransferStatus;
  evidence_deadline_at: number;
  evidence_submitted_at: number;
  user_visible_reason: string;
  prior_order_no: string;
  created_at: number;
  updated_at: number;
  order?: ProductOrder;
  ticket?: SupportTicket;
}

export interface SupportTicket {
  id: number;
  ticket_no: string;
  user_id: number;
  subject: string;
  status: "open" | "closed";
  related_order_id: number;
  related_application_id: number;
  created_at: number;
  updated_at: number;
  closed_at: number;
}

export interface SupportTicketAttachment {
  id: number;
  kind: "payment_evidence" | "reply" | "";
  original_name: string;
  content_type: string;
  byte_size: number;
  sha256: string;
  created_at: number;
}

export interface SupportTicketMessage {
  id: number;
  sender_user_id: number;
  sender_role: "user" | "admin" | "system";
  body: string;
  internal: boolean;
  created_at: number;
  attachments?: SupportTicketAttachment[];
}

export interface CorporateReceiptVerification {
  id: number;
  channel: "enterprise_wechat" | "corporate_bank";
  external_reference: string;
  amount_cents: number;
  received_at: number;
  payer_name: string;
  admin_note: string;
  status: "active" | "voided";
  void_reason: string;
  created_at: number;
}

export interface CorporateTransferCreation {
  order: ProductOrder;
  application: CorporateTransferApplication;
  ticket: SupportTicket;
}

export interface CorporateTransferTicketListItem {
  ticket: SupportTicket;
  application: CorporateTransferApplication;
}

export interface CorporateTransferTicketDetail {
  ticket: SupportTicket;
  application: CorporateTransferApplication;
  messages: SupportTicketMessage[];
  collection: CorporateCollectionSnapshot;
  collection_outdated?: boolean;
  receipts?: CorporateReceiptVerification[];
}
