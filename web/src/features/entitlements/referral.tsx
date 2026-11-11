/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { useQuery } from '@tanstack/react-query'
import { Gift, Share2, Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { SectionPageLayout } from '@/components/layout'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'

import { getGiftSelf } from './api'
import { formatDate } from './lib'

const formatCNY = (cents: number) =>
  new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'CNY',
  }).format(cents / 100)

export function ReferralGiftPage() {
  const { t } = useTranslation()
  const gift = useQuery({ queryKey: ['gift', 'self'], queryFn: getGiftSelf })
  const data = gift.data?.success ? gift.data.data : undefined
  const settings = data?.settings
  let referralDescription = t(gift.isPending ? 'Loading...' : 'Loading failed')
  if (settings) {
    referralDescription = t('Referral rewards are currently disabled')
    if (settings.referral_enabled) {
      referralDescription = t(
        'Earn {{firstRate}}% gift balance on a friend’s first eligible cash payment and {{recurringRate}}% on later eligible payments. Rewards become available after the cooling period.',
        {
          firstRate: settings.first_rate_bps / 100,
          recurringRate: settings.recurring_rate_bps / 100,
        }
      )
    }
  }
  const referralLink = data?.aff_code
    ? `${window.location.origin}/register?aff=${encodeURIComponent(data.aff_code)}`
    : ''

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Friend Referral')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        <div className='space-y-6'>
          <Card>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <Share2 className='size-5' aria-hidden='true' />
                {t('Invite friends and earn gift balance')}
              </CardTitle>
            </CardHeader>
            <CardContent className='space-y-4'>
              <p className='text-muted-foreground text-sm'>
                {referralDescription}
              </p>
              <div className='flex gap-2'>
                <Input value={referralLink} readOnly className='font-mono' />
                <CopyButton
                  value={referralLink}
                  tooltip={t('Copy referral link')}
                  aria-label={t('Copy referral link')}
                />
              </div>
            </CardContent>
          </Card>

          <div className='grid gap-4 md:grid-cols-4'>
            {[
              {
                label: t('Available gift'),
                value: formatCNY(data?.account.available_cents ?? 0),
                Icon: Gift,
              },
              {
                label: t('Pending gift'),
                value: formatCNY(data?.pending_cents ?? 0),
                Icon: Gift,
              },
              {
                label: t('Total referral gift'),
                value: formatCNY(data?.total_reward_cents ?? 0),
                Icon: Gift,
              },
              {
                label: t('Invited friends'),
                value: String(data?.aff_count ?? 0),
                Icon: Users,
              },
            ].map(({ label, value, Icon }) => (
              <Card key={label}>
                <CardContent className='pt-6'>
                  <Icon
                    className='text-muted-foreground mb-3 size-5'
                    aria-hidden='true'
                  />
                  <p className='text-muted-foreground text-sm'>{label}</p>
                  <p className='mt-1 text-2xl font-semibold'>{value}</p>
                </CardContent>
              </Card>
            ))}
          </div>

          <Card>
            <CardHeader>
              <CardTitle>{t('Referral reward history')}</CardTitle>
            </CardHeader>
            <CardContent className='overflow-x-auto p-0'>
              <table className='w-full text-sm'>
                <thead className='bg-muted/50 text-left'>
                  <tr>
                    <th className='p-3'>{t('Order number')}</th>
                    <th className='p-3'>{t('Friend')}</th>
                    <th className='p-3'>{t('Category')}</th>
                    <th className='p-3'>{t('Cash paid')}</th>
                    <th className='p-3'>{t('Reward rate')}</th>
                    <th className='p-3'>{t('Gift reward')}</th>
                    <th className='p-3'>{t('Status')}</th>
                    <th className='p-3'>{t('Created at')}</th>
                  </tr>
                </thead>
                <tbody>
                  {(data?.rewards ?? []).map((reward) => (
                    <tr key={reward.id} className='border-t'>
                      <td className='p-3 font-mono text-xs'>
                        {reward.source_order_no}
                      </td>
                      <td className='p-3'>{reward.friend}</td>
                      <td className='p-3'>{t(reward.source_category)}</td>
                      <td className='p-3'>
                        {formatCNY(reward.cash_paid_cents)}
                      </td>
                      <td className='p-3'>
                        {reward.reward_tier === 'first'
                          ? t('First')
                          : t('Recurring')}{' '}
                        {(reward.reward_rate_bps / 100).toFixed(2)}%
                      </td>
                      <td className='p-3'>{formatCNY(reward.reward_cents)}</td>
                      <td className='p-3'>
                        <Badge variant='secondary'>{t(reward.status)}</Badge>
                      </td>
                      <td className='p-3'>{formatDate(reward.created_at)}</td>
                    </tr>
                  ))}
                  {!gift.isPending && (data?.rewards.length ?? 0) === 0 ? (
                    <tr>
                      <td
                        colSpan={8}
                        className='text-muted-foreground p-8 text-center'
                      >
                        {t('No referral rewards yet')}
                      </td>
                    </tr>
                  ) : null}
                </tbody>
              </table>
            </CardContent>
          </Card>
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
