/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import type { SKUReferences } from './types'

interface SKUDeletionDecision {
  canDelete: boolean
  confirmEntitlements: boolean
  mode: 'loading' | 'blocked' | 'logical' | 'permanent'
}

export function getSKUDeletionDecision(
  references: SKUReferences | undefined
): SKUDeletionDecision {
  if (!references) {
    return {
      canDelete: false,
      confirmEntitlements: false,
      mode: 'loading',
    }
  }
  if (references.pending_orders > 0) {
    return {
      canDelete: false,
      confirmEntitlements: false,
      mode: 'blocked',
    }
  }
  return {
    canDelete: true,
    confirmEntitlements: references.ongoing_entitlements > 0,
    mode:
      references.orders > 0 || references.entitlements > 0
        ? 'logical'
        : 'permanent',
  }
}
