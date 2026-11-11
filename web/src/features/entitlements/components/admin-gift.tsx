/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'

import { adjustUserGift, getGiftSettings, saveGiftSettings } from '../api'
import type { GiftSettings } from '../types'
import { AdminFormField } from './admin-form-field'

const defaultSettings: GiftSettings = {
  checkout_enabled: false,
  referral_enabled: false,
  first_rate_bps: 1000,
  recurring_rate_bps: 500,
  min_cash_paid_cents: 100,
  reward_cap_cents: 10000,
  cooling_days: 7,
  large_grant_approval_cents: 50000,
}

export function AdminGift() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const query = useQuery({
    queryKey: ['entitlement-admin', 'gift-settings'],
    queryFn: getGiftSettings,
  })
  const [settings, setSettings] = useState(defaultSettings)
  const [userId, setUserId] = useState(0)
  const [giftYuan, setGiftYuan] = useState('0.00')
  const [reason, setReason] = useState('')

  useEffect(() => {
    if (query.data?.data) setSettings(query.data.data)
  }, [query.data?.data])

  const save = useMutation({
    mutationFn: saveGiftSettings,
    onSuccess: async (response) => {
      if (!response.success) {
        toast.error(response.message || t('Save failed'))
        return
      }
      await queryClient.invalidateQueries({
        queryKey: ['entitlement-admin', 'gift-settings'],
      })
      await queryClient.invalidateQueries({ queryKey: ['gift', 'self'] })
      toast.success(t('Gift settings saved'))
    },
  })
  const adjust = useMutation({
    mutationFn: () =>
      adjustUserGift(userId, {
        delta_cents: Math.round(Number(giftYuan) * 100),
        reason,
        idempotency_key: `admin-${userId}-${Date.now()}`,
      }),
    onSuccess: (response) => {
      if (!response.success) {
        toast.error(response.message || t('Operation failed'))
        return
      }
      setGiftYuan('0.00')
      setReason('')
      toast.success(t('Gift balance adjusted'))
    },
  })

  const setNumber = (field: keyof GiftSettings, value: number) =>
    setSettings((current) => ({ ...current, [field]: value }))

  return (
    <div className='grid gap-4 xl:grid-cols-2'>
      <Card>
        <CardHeader>
          <CardTitle>{t('Gift and referral settings')}</CardTitle>
        </CardHeader>
        <CardContent className='space-y-4'>
          <label className='flex items-center gap-2 text-sm'>
            <Checkbox
              checked={settings.checkout_enabled}
              onCheckedChange={(checked) =>
                setSettings((current) => ({
                  ...current,
                  checkout_enabled: checked === true,
                }))
              }
            />
            {t('Enable gift checkout')}
          </label>
          <label className='flex items-center gap-2 text-sm'>
            <Checkbox
              checked={settings.referral_enabled}
              onCheckedChange={(checked) =>
                setSettings((current) => ({
                  ...current,
                  referral_enabled: checked === true,
                }))
              }
            />
            {t('Enable referral rewards')}
          </label>
          <div className='grid gap-3 sm:grid-cols-2'>
            <AdminFormField label={t('First reward rate (%)')}>
              <Input
                type='number'
                min='0'
                max='100'
                step='0.01'
                value={settings.first_rate_bps / 100}
                onChange={(event) =>
                  setNumber(
                    'first_rate_bps',
                    Math.round(Number(event.target.value) * 100)
                  )
                }
              />
            </AdminFormField>
            <AdminFormField label={t('Recurring reward rate (%)')}>
              <Input
                type='number'
                min='0'
                max='100'
                step='0.01'
                value={settings.recurring_rate_bps / 100}
                onChange={(event) =>
                  setNumber(
                    'recurring_rate_bps',
                    Math.round(Number(event.target.value) * 100)
                  )
                }
              />
            </AdminFormField>
            <AdminFormField label={t('Minimum cash payment (CNY)')}>
              <Input
                type='number'
                min='0'
                step='0.01'
                value={settings.min_cash_paid_cents / 100}
                onChange={(event) =>
                  setNumber(
                    'min_cash_paid_cents',
                    Math.round(Number(event.target.value) * 100)
                  )
                }
              />
            </AdminFormField>
            <AdminFormField label={t('Reward cap per order (CNY)')}>
              <Input
                type='number'
                min='0.01'
                step='0.01'
                value={settings.reward_cap_cents / 100}
                onChange={(event) =>
                  setNumber(
                    'reward_cap_cents',
                    Math.round(Number(event.target.value) * 100)
                  )
                }
              />
            </AdminFormField>
            <AdminFormField label={t('Cooling period (days)')}>
              <Input
                type='number'
                min='1'
                max='365'
                value={settings.cooling_days}
                onChange={(event) =>
                  setNumber('cooling_days', Number(event.target.value))
                }
              />
            </AdminFormField>
          </div>
          <Button
            onClick={() => save.mutate(settings)}
            disabled={save.isPending}
          >
            {t('Save')}
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>{t('Adjust user gift balance')}</CardTitle>
        </CardHeader>
        <CardContent className='space-y-3'>
          <AdminFormField label={t('User ID')}>
            <Input
              type='number'
              min='1'
              value={userId || ''}
              onChange={(event) => setUserId(Number(event.target.value))}
            />
          </AdminFormField>
          <AdminFormField label={t('Amount (CNY, negative to deduct)')}>
            <Input
              type='number'
              step='0.01'
              value={giftYuan}
              onChange={(event) => setGiftYuan(event.target.value)}
            />
          </AdminFormField>
          <AdminFormField label={t('Reason')}>
            <Input
              value={reason}
              onChange={(event) => setReason(event.target.value)}
            />
          </AdminFormField>
          <Button
            onClick={() => adjust.mutate()}
            disabled={
              adjust.isPending ||
              userId <= 0 ||
              Number(giftYuan) === 0 ||
              !reason.trim()
            }
          >
            {t('Apply adjustment')}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
