/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { getSKUDeletionDecision } from '../sku-deletion.ts'

describe('SKU deletion confirmation', () => {
  test('disables deletion until references are available', () => {
    assert.equal(getSKUDeletionDecision(undefined).canDelete, false)
  })
  test('blocks deletion while pending orders exist even with entitlement confirmation', () => {
    const decision = getSKUDeletionDecision({
      orders: 2,
      pending_orders: 1,
      entitlements: 1,
      ongoing_entitlements: 1,
    })
    assert.equal(decision.canDelete, false)
    assert.equal(decision.mode, 'blocked')
  })
  test('allows logical deletion when only historical records remain', () => {
    const decision = getSKUDeletionDecision({
      orders: 3,
      pending_orders: 0,
      entitlements: 2,
      ongoing_entitlements: 0,
    })
    assert.equal(decision.canDelete, true)
    assert.equal(decision.mode, 'logical')
    assert.equal(decision.confirmEntitlements, false)
  })
  test('requires preserving ongoing entitlements when confirming deletion', () => {
    const decision = getSKUDeletionDecision({
      orders: 0,
      pending_orders: 0,
      entitlements: 1,
      ongoing_entitlements: 1,
    })
    assert.equal(decision.canDelete, true)
    assert.equal(decision.confirmEntitlements, true)
  })
  test('identifies an unused SKU as permanently deletable', () => {
    assert.equal(
      getSKUDeletionDecision({
        orders: 0,
        pending_orders: 0,
        entitlements: 0,
        ongoing_entitlements: 0,
      }).mode,
      'permanent'
    )
  })
})
