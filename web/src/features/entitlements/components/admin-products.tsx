/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Trash2 } from "lucide-react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { z } from "zod";

import { ConfirmDialog } from "@/components/confirm-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import {
  NativeSelect,
  NativeSelectOption,
} from "@/components/ui/native-select";
import { useSystemConfig } from "@/hooks/use-system-config";
import { formatQuota } from "@/lib/format";

import {
  createProduct,
  createProductSKU,
  deleteProductSKU,
  getAdminProducts,
  getSKUReferences,
  restoreProductSKU,
  getEntitlementTypes,
  updateProduct,
  updateProductSKU,
} from "../api";
import {
  hasAtMostDecimalPlaces,
  majorAmountToMinorUnits,
  minorUnitsToMajorAmount,
  quotaUnitsToUSD,
  usdToQuotaUnits,
} from "../form-units";
import { getSKUDeletionDecision } from "../sku-deletion";
import type { ProductSKU } from "../types";
import { AdminFormField } from "./admin-form-field";

const productSchema = z.object({
  code: z.string().trim().min(1),
  name: z.string().trim().min(1),
  description: z.string(),
  category: z.enum(["subscription", "recharge"]),
  status: z.enum(["draft", "active", "paused", "archived"]),
  sort_order: z.number().int(),
});

const currencyAmountSchema = z
  .number()
  .min(0)
  .refine((value) => hasAtMostDecimalPlaces(value, 2), {
    message: "Use no more than 2 decimal places",
  });

const quotaAmountSchema = z
  .number()
  .min(0)
  .refine((value) => hasAtMostDecimalPlaces(value, 4), {
    message: "Use no more than 4 decimal places",
  });

const skuSchema = z.object({
  product_id: z.number().int().positive(),
  entitlement_type_id: z.number().int().positive(),
  code: z.string().trim().min(1),
  name: z.string().trim().min(1),
  price_amount: currencyAmountSchema,
  grant_total_quota_usd: quotaAmountSchema.refine((value) => value > 0),
  grant_daily_quota_usd: quotaAmountSchema,
  min_recharge_amount: currencyAmountSchema,
  max_recharge_amount: currencyAmountSchema,
  validity_seconds: z.number().int().min(0),
  activation_policy: z.enum(["immediate", "manual", "deferred"]),
  activation_deadline_seconds: z.number().int().min(0),
  stock: z.number().int().min(0),
  stock_limited: z.boolean(),
  purchase_limit: z.number().int().min(0),
  multi_quantity_enabled: z.boolean(),
  status: z.enum(["draft", "active", "paused", "archived"]),
  sort_order: z.number().int(),
});

type ProductForm = z.infer<typeof productSchema>;
type SKUForm = z.infer<typeof skuSchema>;

export function AdminProducts() {
  const { t } = useTranslation();
  const { currency } = useSystemConfig();
  const quotaPerUnit = currency.quotaPerUnit;
  const activationPolicyLabels = {
    immediate: t("Immediate"),
    manual: t("Manual activation"),
    deferred: t("Deferred"),
  };
  const queryClient = useQueryClient();
  const [editingProductId, setEditingProductId] = useState<number | null>(null);
  const [editingSKUId, setEditingSKUId] = useState<number | null>(null);
  const [deletingSKU, setDeletingSKU] = useState<ProductSKU | null>(null);
  const [showDeleted, setShowDeleted] = useState(false);
  const products = useQuery({
    queryKey: ["entitlement-admin", "products", showDeleted],
    queryFn: () => getAdminProducts(showDeleted),
  });
  const references = useQuery({
    queryKey: ["entitlement-admin", "sku-references", deletingSKU?.id],
    queryFn: () => getSKUReferences(deletingSKU?.id ?? 0),
    enabled: deletingSKU !== null,
    staleTime: 0,
  });
  const skuReferences =
    !references.isError && references.data?.success
      ? references.data.data
      : undefined;
  const deletion = getSKUDeletionDecision(skuReferences);
  const types = useQuery({
    queryKey: ["entitlement-admin", "types"],
    queryFn: getEntitlementTypes,
  });
  const productForm = useForm<ProductForm>({
    resolver: zodResolver(productSchema),
    defaultValues: {
      code: "",
      name: "",
      description: "",
      category: "subscription",
      status: "active",
      sort_order: 0,
    },
  });
  const skuForm = useForm<SKUForm>({
    resolver: zodResolver(skuSchema),
    defaultValues: {
      product_id: 0,
      entitlement_type_id: 0,
      code: "",
      name: "",
      price_amount: 0,
      grant_total_quota_usd: 1,
      grant_daily_quota_usd: 0,
      min_recharge_amount: 1,
      max_recharge_amount: 10000,
      validity_seconds: 2592000,
      activation_policy: "immediate",
      activation_deadline_seconds: 0,
      stock: 0,
      stock_limited: false,
      purchase_limit: 0,
      multi_quantity_enabled: false,
      status: "active",
      sort_order: 0,
    },
  });
  const invalidate = async () => {
    await queryClient.invalidateQueries({
      queryKey: ["entitlement-admin", "products"],
    });
  };
  const saveProduct = useMutation({
    mutationFn: (input: { id: number | null; values: ProductForm }) =>
      input.id
        ? updateProduct(input.id, {
            ...input.values,
            visibility_rule:
              products.data?.data?.find((product) => product.id === input.id)
                ?.visibility_rule ?? "",
          })
        : createProduct({ ...input.values, visibility_rule: "" }),
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t("Save failed"));
        return;
      }
      await invalidate();
      productForm.reset();
      setEditingProductId(null);
      toast.success(t("Product saved"));
    },
  });
  const saveSKU = useMutation({
    mutationFn: (input: { id: number | null; values: SKUForm }) => {
      const {
        price_amount,
        grant_total_quota_usd,
        grant_daily_quota_usd,
        min_recharge_amount,
        max_recharge_amount,
        ...values
      } = input.values;
      const payload = {
        ...values,
        currency: "CNY",
        price_amount_minor: majorAmountToMinorUnits(price_amount),
        grant_total_quota: usdToQuotaUnits(grant_total_quota_usd, quotaPerUnit),
        grant_daily_quota: usdToQuotaUnits(grant_daily_quota_usd, quotaPerUnit),
        min_recharge_amount_minor: majorAmountToMinorUnits(min_recharge_amount),
        max_recharge_amount_minor: majorAmountToMinorUnits(max_recharge_amount),
      };
      return input.id
        ? updateProductSKU(input.id, payload)
        : createProductSKU(payload);
    },
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t("Save failed"));
        return;
      }
      await invalidate();
      skuForm.reset();
      setEditingSKUId(null);
      toast.success(t("SKU saved"));
    },
  });
  const removeSKU = useMutation({
    mutationFn: deleteProductSKU,
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t("Delete failed"));
        return;
      }
      await invalidate();
      if (editingSKUId === deletingSKU?.id) {
        skuForm.reset();
        setEditingSKUId(null);
      }
      setDeletingSKU(null);
      toast.success(t("SKU deleted"));
    },
    onError: () => toast.error(t("Delete failed")),
  });
  const restoreSKU = useMutation({
    mutationFn: restoreProductSKU,
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t("Restore failed"));
        return;
      }
      await invalidate();
      toast.success(t("SKU restored as archived. Edit it to publish again."));
    },
    onError: () => toast.error(t("Restore failed")),
  });
  const selectedSKUProduct = (products.data?.data ?? []).find(
    (product) => product.id === skuForm.watch("product_id"),
  );
  const isRechargeSKU = selectedSKUProduct?.category === "recharge";
  const rechargeAlreadyHasRule =
    isRechargeSKU &&
    !editingSKUId &&
    (selectedSKUProduct?.skus ?? []).some((sku) => sku.status !== "archived");

  return (
    <div className="space-y-4">
      <div className="grid gap-4 xl:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>
              {editingProductId ? t("Edit product") : t("Create product")}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <form
              className="grid gap-3 sm:grid-cols-2"
              onSubmit={productForm.handleSubmit((values) =>
                saveProduct.mutate({ id: editingProductId, values }),
              )}
            >
              <AdminFormField label={t("Product code")}>
                <Input {...productForm.register("code")} />
              </AdminFormField>
              <AdminFormField label={t("Product name")}>
                <Input {...productForm.register("name")} />
              </AdminFormField>
              <AdminFormField
                className="sm:col-span-2"
                label={t("Description")}
              >
                <Input {...productForm.register("description")} />
              </AdminFormField>
              <AdminFormField label={t("Category")}>
                <NativeSelect
                  className="w-full"
                  {...productForm.register("category")}
                >
                  <NativeSelectOption value="subscription">
                    {t("Subscription")}
                  </NativeSelectOption>
                  <NativeSelectOption value="recharge">
                    {t("Recharge")}
                  </NativeSelectOption>
                </NativeSelect>
              </AdminFormField>
              <AdminFormField label={t("Status")}>
                <NativeSelect
                  className="w-full"
                  {...productForm.register("status")}
                >
                  <NativeSelectOption value="draft">
                    {t("draft")}
                  </NativeSelectOption>
                  <NativeSelectOption value="active">
                    {t("active")}
                  </NativeSelectOption>
                  <NativeSelectOption value="paused">
                    {t("paused")}
                  </NativeSelectOption>
                  <NativeSelectOption value="archived">
                    {t("archived")}
                  </NativeSelectOption>
                </NativeSelect>
              </AdminFormField>
              <AdminFormField label={t("Sort order")}>
                <Input
                  type="number"
                  {...productForm.register("sort_order", {
                    valueAsNumber: true,
                  })}
                />
              </AdminFormField>
              <div className="flex items-end gap-2">
                <Button type="submit" disabled={saveProduct.isPending}>
                  {editingProductId ? t("Save") : t("Create")}
                </Button>
                {editingProductId ? (
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      setEditingProductId(null);
                      productForm.reset();
                    }}
                  >
                    {t("Cancel")}
                  </Button>
                ) : null}
              </div>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>
              {editingSKUId ? t("Edit SKU") : t("Create SKU")}
            </CardTitle>
          </CardHeader>
          <CardContent>
            <form
              className="grid gap-3 sm:grid-cols-2"
              onSubmit={skuForm.handleSubmit((values) =>
                saveSKU.mutate({ id: editingSKUId, values }),
              )}
            >
              <AdminFormField label={t("Product")}>
                <NativeSelect
                  className="w-full"
                  {...skuForm.register("product_id", { valueAsNumber: true })}
                >
                  <NativeSelectOption value={0}>
                    {t("Select product")}
                  </NativeSelectOption>
                  {(products.data?.data ?? []).map((product) => (
                    <NativeSelectOption key={product.id} value={product.id}>
                      {product.name}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </AdminFormField>
              <AdminFormField label={t("Entitlement type")}>
                <NativeSelect
                  className="w-full"
                  {...skuForm.register("entitlement_type_id", {
                    valueAsNumber: true,
                  })}
                >
                  <NativeSelectOption value={0}>
                    {t("Select type")}
                  </NativeSelectOption>
                  {(types.data?.data ?? []).map((type) => (
                    <NativeSelectOption key={type.id} value={type.id}>
                      {type.name}
                    </NativeSelectOption>
                  ))}
                </NativeSelect>
              </AdminFormField>
              <AdminFormField label={t("SKU code")}>
                <Input {...skuForm.register("code")} />
              </AdminFormField>
              <AdminFormField label={t("SKU name")}>
                <Input {...skuForm.register("name")} />
              </AdminFormField>
              <AdminFormField
                label={
                  isRechargeSKU
                    ? t("Recharge pricing basis (CNY)")
                    : t("Price (CNY)")
                }
              >
                <Input
                  type="number"
                  min="0"
                  step="0.01"
                  {...skuForm.register("price_amount", {
                    valueAsNumber: true,
                  })}
                />
              </AdminFormField>
              <AdminFormField label={t("Grant total quota (USD)")}>
                <Input
                  type="number"
                  min="0.0001"
                  step="0.0001"
                  {...skuForm.register("grant_total_quota_usd", {
                    valueAsNumber: true,
                  })}
                />
              </AdminFormField>
              {isRechargeSKU ? (
                <>
                  <p className="text-muted-foreground text-sm sm:col-span-2">
                    {t(
                      "Recharge price and quota define the conversion rate. Customers choose the actual amount at checkout.",
                    )}
                  </p>
                  <AdminFormField label={t("Minimum recharge amount (CNY)")}>
                    <Input
                      type="number"
                      min="0"
                      step="0.01"
                      {...skuForm.register("min_recharge_amount", {
                        valueAsNumber: true,
                      })}
                    />
                  </AdminFormField>
                  <AdminFormField label={t("Maximum recharge amount (CNY)")}>
                    <Input
                      type="number"
                      min="0"
                      step="0.01"
                      {...skuForm.register("max_recharge_amount", {
                        valueAsNumber: true,
                      })}
                    />
                  </AdminFormField>
                  {rechargeAlreadyHasRule ? (
                    <p className="text-destructive text-sm sm:col-span-2">
                      {t(
                        "This recharge product already has an active pricing rule. Edit the existing SKU instead.",
                      )}
                    </p>
                  ) : null}
                </>
              ) : null}
              <AdminFormField label={t("Grant daily quota (USD)")}>
                <Input
                  type="number"
                  min="0"
                  step="0.0001"
                  {...skuForm.register("grant_daily_quota_usd", {
                    valueAsNumber: true,
                  })}
                />
              </AdminFormField>
              <p className="text-muted-foreground text-xs sm:col-span-2">
                {t(
                  "Quota is entered in USD and converted automatically using $1 = {{quota}} internal units.",
                  { quota: quotaPerUnit.toLocaleString() },
                )}
              </p>
              <AdminFormField label={t("Validity seconds")}>
                <Input
                  type="number"
                  {...skuForm.register("validity_seconds", {
                    valueAsNumber: true,
                  })}
                />
              </AdminFormField>
              <AdminFormField label={t("Activation policy")}>
                <NativeSelect
                  className="w-full"
                  {...skuForm.register("activation_policy")}
                >
                  <NativeSelectOption value="immediate">
                    {t("Immediate")}
                  </NativeSelectOption>
                  <NativeSelectOption value="manual">
                    {t("Manual activation")}
                  </NativeSelectOption>
                  <NativeSelectOption value="deferred">
                    {t("Deferred")}
                  </NativeSelectOption>
                </NativeSelect>
              </AdminFormField>
              <AdminFormField label={t("Activation deadline seconds")}>
                <Input
                  type="number"
                  {...skuForm.register("activation_deadline_seconds", {
                    valueAsNumber: true,
                  })}
                />
              </AdminFormField>
              <AdminFormField label={t("Stock")}>
                <Input
                  type="number"
                  disabled={!skuForm.watch("stock_limited")}
                  {...skuForm.register("stock", { valueAsNumber: true })}
                />
              </AdminFormField>
              <label className="flex items-center gap-2 text-sm">
                <Checkbox
                  checked={skuForm.watch("stock_limited")}
                  onCheckedChange={(checked) =>
                    skuForm.setValue("stock_limited", checked === true)
                  }
                />
                {t("Limit stock")}
              </label>
              <AdminFormField label={t("Purchase limit, 0 means unlimited")}>
                <Input
                  type="number"
                  {...skuForm.register("purchase_limit", {
                    valueAsNumber: true,
                  })}
                />
              </AdminFormField>
              {!isRechargeSKU ? (
                <label className="flex items-center gap-2 text-sm">
                  <Checkbox
                    checked={skuForm.watch("multi_quantity_enabled")}
                    onCheckedChange={(checked) =>
                      skuForm.setValue(
                        "multi_quantity_enabled",
                        checked === true,
                      )
                    }
                  />
                  {t("Allow multiple quantities")}
                </label>
              ) : null}
              <AdminFormField label={t("Status")}>
                <NativeSelect
                  className="w-full"
                  {...skuForm.register("status")}
                >
                  <NativeSelectOption value="draft">
                    {t("draft")}
                  </NativeSelectOption>
                  <NativeSelectOption value="active">
                    {t("active")}
                  </NativeSelectOption>
                  <NativeSelectOption value="paused">
                    {t("paused")}
                  </NativeSelectOption>
                  <NativeSelectOption value="archived">
                    {t("archived")}
                  </NativeSelectOption>
                </NativeSelect>
              </AdminFormField>
              <AdminFormField label={t("Sort order")}>
                <Input
                  type="number"
                  {...skuForm.register("sort_order", { valueAsNumber: true })}
                />
              </AdminFormField>
              <div className="flex gap-2 sm:col-span-2">
                <Button
                  type="submit"
                  disabled={saveSKU.isPending || rechargeAlreadyHasRule}
                >
                  {editingSKUId ? t("Save") : t("Create")}
                </Button>
                {editingSKUId ? (
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => {
                      setEditingSKUId(null);
                      skuForm.reset();
                    }}
                  >
                    {t("Cancel")}
                  </Button>
                ) : null}
              </div>
            </form>
          </CardContent>
        </Card>
      </div>

      <label className="flex items-center gap-2 text-sm">
        <Checkbox
          checked={showDeleted}
          onCheckedChange={(checked) => setShowDeleted(checked === true)}
        />
        {t("Show deleted SKUs")}
      </label>
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {(products.data?.data ?? []).map((product) => (
          <Card key={product.id}>
            <CardHeader>
              <div className="flex items-start justify-between gap-2">
                <div>
                  <CardTitle>{product.name}</CardTitle>
                  <div className="mt-2 flex gap-2">
                    <Badge variant="secondary">
                      {t(
                        product.category === "subscription"
                          ? "Subscription"
                          : "Recharge",
                      )}
                    </Badge>
                    <Badge variant="outline">{t(product.status)}</Badge>
                  </div>
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    setEditingProductId(product.id);
                    productForm.reset({
                      code: product.code,
                      name: product.name,
                      description: product.description,
                      category: product.category,
                      status: product.status,
                      sort_order: product.sort_order,
                    });
                  }}
                >
                  {t("Edit")}
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-2">
              {(product.skus ?? []).map((sku) => (
                <div
                  key={sku.id}
                  className="bg-muted/50 rounded-lg p-3 text-sm"
                >
                  <div className="flex justify-between gap-2">
                    <div>
                      <strong>{sku.name}</strong>
                      <Badge className="ml-2" variant="outline">
                        {t(sku.status)}
                      </Badge>
                    </div>
                    <span>
                      {product.category === "recharge"
                        ? `${t("Rule")} ¥${(sku.price_amount_minor / 100).toFixed(2)}`
                        : `¥${(sku.price_amount_minor / 100).toFixed(2)}`}
                    </span>
                  </div>
                  <p className="text-muted-foreground">
                    {t("Total quota")}: {formatQuota(sku.grant_total_quota)}
                  </p>
                  <p className="text-muted-foreground">
                    {t("Daily quota")}:{" "}
                    {sku.grant_daily_quota > 0
                      ? formatQuota(sku.grant_daily_quota)
                      : t("Unlimited")}{" "}
                    · {activationPolicyLabels[sku.activation_policy]}
                  </p>
                  {product.category === "recharge" ? (
                    <p className="text-muted-foreground">
                      {t("Recharge range: {{min}}–{{max}}", {
                        min: `¥${(sku.min_recharge_amount_minor / 100).toFixed(2)}`,
                        max: `¥${(sku.max_recharge_amount_minor / 100).toFixed(2)}`,
                      })}
                    </p>
                  ) : null}
                  <div className="mt-2 flex gap-2">
                    {sku.deleted_at ? (
                      <>
                        <Badge variant="secondary">{t("Deleted")}</Badge>
                        <Button
                          size="sm"
                          disabled={restoreSKU.isPending}
                          onClick={() => restoreSKU.mutate(sku.id)}
                        >
                          {t("Restore SKU")}
                        </Button>
                      </>
                    ) : (
                      <>
                        <Button
                          size="sm"
                          variant="outline"
                          onClick={() => {
                            setEditingSKUId(sku.id);
                            skuForm.reset({
                              product_id: sku.product_id,
                              entitlement_type_id: sku.entitlement_type_id,
                              code: sku.code,
                              name: sku.name,
                              price_amount: minorUnitsToMajorAmount(
                                sku.price_amount_minor,
                              ),
                              grant_total_quota_usd: quotaUnitsToUSD(
                                sku.grant_total_quota,
                                quotaPerUnit,
                              ),
                              grant_daily_quota_usd: quotaUnitsToUSD(
                                sku.grant_daily_quota,
                                quotaPerUnit,
                              ),
                              min_recharge_amount: minorUnitsToMajorAmount(
                                sku.min_recharge_amount_minor,
                              ),
                              max_recharge_amount: minorUnitsToMajorAmount(
                                sku.max_recharge_amount_minor,
                              ),
                              validity_seconds: sku.validity_seconds,
                              activation_policy: sku.activation_policy,
                              activation_deadline_seconds:
                                sku.activation_deadline_seconds,
                              stock: sku.stock,
                              stock_limited: sku.stock_limited,
                              purchase_limit: sku.purchase_limit,
                              multi_quantity_enabled:
                                sku.multi_quantity_enabled,
                              status: sku.status,
                              sort_order: sku.sort_order,
                            });
                          }}
                        >
                          {t("Edit SKU")}
                        </Button>
                        <Button
                          size="sm"
                          variant="destructive"
                          onClick={() => setDeletingSKU(sku)}
                        >
                          <Trash2 aria-hidden="true" />
                          {t("Delete SKU")}
                        </Button>
                      </>
                    )}
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        ))}
      </div>
      {deletingSKU ? (
        <ConfirmDialog
          open
          onOpenChange={(open) => !open && setDeletingSKU(null)}
          title={t("Delete SKU")}
          desc={
            <div className="space-y-2">
              <p>{t("Delete SKU {{name}}?", { name: deletingSKU.name })}</p>
              {skuReferences ? (
                <>
                  <p>
                    {t(
                      "Linked orders: {{orders}}. Linked entitlements: {{entitlements}}.",
                      {
                        orders: skuReferences.orders,
                        entitlements: skuReferences.entitlements,
                      },
                    )}
                  </p>
                  {deletion.mode === "blocked" ? (
                    <p>
                      {t(
                        "This SKU has {{count}} pending orders. Cancel them or wait for expiry before deleting.",
                        { count: skuReferences.pending_orders },
                      )}
                    </p>
                  ) : null}
                  {deletion.mode === "logical" ? (
                    <p>
                      {t(
                        "The SKU will be removed from the store and default admin list. Historical orders and existing entitlements will be preserved.",
                      )}
                    </p>
                  ) : null}
                  {deletion.mode === "permanent" ? (
                    <p>
                      {t(
                        "This SKU has no linked records and will be permanently deleted. This cannot be undone.",
                      )}
                    </p>
                  ) : null}
                  {deletion.confirmEntitlements ? (
                    <p>
                      {t(
                        "{{count}} ongoing entitlements will remain usable under their original rules. Confirm deletion to keep them unchanged.",
                        { count: skuReferences.ongoing_entitlements },
                      )}
                    </p>
                  ) : null}
                </>
              ) : (
                <p>
                  {references.isPending
                    ? t("Loading")
                    : t(
                        "Unable to load SKU references. Close this dialog and retry.",
                      )}
                </p>
              )}
            </div>
          }
          confirmText={t("Delete")}
          handleConfirm={() =>
            removeSKU.mutate({
              id: deletingSKU.id,
              confirmEntitlements: deletion.confirmEntitlements,
            })
          }
          disabled={!deletion.canDelete || references.isFetching}
          isLoading={removeSKU.isPending}
          destructive
        />
      ) : null}
    </div>
  );
}
