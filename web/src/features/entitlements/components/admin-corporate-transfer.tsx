import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  sideDrawerContentClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Textarea } from '@/components/ui/textarea'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import {
  addCorporateReceipt,
  adminReplyCorporateTransfer,
  approveCorporateTransfer,
  getAdminCorporateTransferTicket,
  getAdminCorporateTransfers,
  getCorporateCollectionAdmin,
  publishCorporateCollection,
  registerCorporateTransferRefund,
  rejectCorporateTransfer,
  requestCorporateTransferInfo,
  saveCorporateCollectionDraft,
  setCorporateTransferTicketOpen,
  uploadCorporateCollectionAsset,
  voidCorporateReceipt,
} from '../api'
import { corporateTransferReceiptStatusKey } from '../corporate-transfer-conversation'
import { corporateTransferStatusKey } from '../corporate-transfer-status'
import type {
  CorporateCollectionChannel,
  CorporateTransferApplication,
} from '../types'
import { AuthenticatedCorporateImage } from './authenticated-corporate-image'
import {
  CorporateTransferMessageList,
  CorporateTransferPaymentPrompt,
  CorporateTransferReviewResult,
} from './corporate-transfer-conversation'
import { CorporateTransferReplyComposer } from './corporate-transfer-reply-composer'

const defaultChannels: CorporateCollectionChannel[] = [
  {
    channel_type: 'enterprise_wechat',
    enabled: false,
    display_name: '企业微信',
    organization_name: '',
    instructions: '',
  },
  {
    channel_type: 'corporate_bank',
    enabled: false,
    display_name: '银行卡',
    organization_name: '',
    account_name: '',
    bank_name: '',
    bank_account: '',
    bank_branch: '',
    instructions: '',
  },
]

function updateChannel(
  channels: CorporateCollectionChannel[],
  channelType: CorporateCollectionChannel['channel_type'],
  patch: Partial<CorporateCollectionChannel>
): CorporateCollectionChannel[] {
  return channels.map((channel) =>
    channel.channel_type === channelType ? { ...channel, ...patch } : channel
  )
}

function CollectionConfiguration() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const configuration = useQuery({
    queryKey: ['corporate-transfer', 'collection-admin'],
    queryFn: getCorporateCollectionAdmin,
  })
  const [channels, setChannels] = useState(defaultChannels)
  useEffect(() => {
    if (configuration.data?.data?.draft) {
      setChannels(configuration.data.data.draft)
    }
  }, [configuration.data?.data?.draft])
  const save = useMutation({
    mutationFn: async () => {
      const response = await saveCorporateCollectionDraft(channels)
      if (!response.success) {
        throw new Error(response.message || t('Save failed'))
      }
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['corporate-transfer', 'collection-admin'],
      })
      toast.success(t('Draft saved'))
    },
    onError: (error) => toast.error(error.message),
  })
  const publish = useMutation({
    mutationFn: async () => {
      const draftResponse = await saveCorporateCollectionDraft(channels)
      if (!draftResponse.success) {
        throw new Error(draftResponse.message || t('Save failed'))
      }
      const response = await publishCorporateCollection(
        configuration.data?.data?.state.current_revision ?? 0
      )
      if (!response.success) {
        throw new Error(response.message || t('Publish failed'))
      }
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['corporate-transfer', 'collection-admin'],
      })
      toast.success(t('Collection information published'))
    },
    onError: (error) => toast.error(error.message),
  })
  const uploadQr = useMutation({
    mutationFn: uploadCorporateCollectionAsset,
    onSuccess: (response) => {
      if (!response.success || !response.data) {
        toast.error(response.message || t('Upload failed'))
        return
      }
      setChannels((current) =>
        updateChannel(current, 'enterprise_wechat', {
          qr_code_attachment_id: response.data?.id,
        })
      )
      toast.success(t('QR code uploaded; save the draft to keep it'))
    },
    onError: (error) => toast.error(error.message),
  })
  const wechat =
    channels.find((channel) => channel.channel_type === 'enterprise_wechat') ??
    defaultChannels[0]
  const bank =
    channels.find((channel) => channel.channel_type === 'corporate_bank') ??
    defaultChannels[1]

  return (
    <div className='space-y-5'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <div>
          <h3 className='text-lg font-semibold'>
            {t('Collection configuration')}
          </h3>
          <p className='text-muted-foreground text-sm'>
            {t('Draft changes affect new applications only after publication.')}
          </p>
        </div>
        <Badge variant='secondary'>
          {t('Published revision {{revision}}', {
            revision: configuration.data?.data?.state.current_revision ?? 0,
          })}
        </Badge>
      </div>
      <div className='grid gap-5 xl:grid-cols-2'>
        <Card>
          <CardHeader>
            <CardTitle>{t('Enterprise WeChat')}</CardTitle>
          </CardHeader>
          <CardContent className='space-y-4'>
            <label className='flex items-center gap-2 text-sm'>
              <input
                type='checkbox'
                checked={wechat.enabled}
                onChange={(event) =>
                  setChannels((current) =>
                    updateChannel(current, 'enterprise_wechat', {
                      enabled: event.target.checked,
                    })
                  )
                }
              />
              {t('Enabled')}
            </label>
            <Input
              value={wechat.display_name}
              aria-label={t('Channel name')}
              placeholder={t('Channel name')}
              onChange={(event) =>
                setChannels((current) =>
                  updateChannel(current, 'enterprise_wechat', {
                    display_name: event.target.value,
                  })
                )
              }
            />
            <Input
              value={wechat.organization_name}
              aria-label={t('Organization name')}
              placeholder={t('Organization name')}
              onChange={(event) =>
                setChannels((current) =>
                  updateChannel(current, 'enterprise_wechat', {
                    organization_name: event.target.value,
                  })
                )
              }
            />
            <label className='grid gap-2 text-sm'>
              <span>{t('Enterprise WeChat collection QR code')}</span>
              <Input
                type='file'
                accept='image/jpeg,image/png,image/webp'
                onChange={(event) => {
                  const file = event.target.files?.[0]
                  if (file) uploadQr.mutate(file)
                }}
              />
            </label>
            {wechat.qr_code_attachment_id ? (
              <AuthenticatedCorporateImage
                path={`/api/store/corporate-transfers/collection-assets/${wechat.qr_code_attachment_id}`}
                alt={t('Enterprise WeChat collection QR code')}
                className='max-h-64 rounded-xl border object-contain'
              />
            ) : null}
            <Textarea
              value={wechat.instructions ?? ''}
              placeholder={t('Payment instructions')}
              onChange={(event) =>
                setChannels((current) =>
                  updateChannel(current, 'enterprise_wechat', {
                    instructions: event.target.value,
                  })
                )
              }
            />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>{t('Corporate bank')}</CardTitle>
          </CardHeader>
          <CardContent className='space-y-4'>
            <label className='flex items-center gap-2 text-sm'>
              <input
                type='checkbox'
                checked={bank.enabled}
                onChange={(event) =>
                  setChannels((current) =>
                    updateChannel(current, 'corporate_bank', {
                      enabled: event.target.checked,
                    })
                  )
                }
              />
              {t('Enabled')}
            </label>
            <Input
              value={bank.display_name}
              placeholder={t('Channel name')}
              onChange={(event) =>
                setChannels((current) =>
                  updateChannel(current, 'corporate_bank', {
                    display_name: event.target.value,
                  })
                )
              }
            />
            <Input
              value={bank.organization_name}
              placeholder={t('Organization name')}
              onChange={(event) =>
                setChannels((current) =>
                  updateChannel(current, 'corporate_bank', {
                    organization_name: event.target.value,
                  })
                )
              }
            />
            <Input
              value={bank.account_name ?? ''}
              placeholder={t('Account name')}
              onChange={(event) =>
                setChannels((current) =>
                  updateChannel(current, 'corporate_bank', {
                    account_name: event.target.value,
                  })
                )
              }
            />
            <Input
              value={bank.bank_name ?? ''}
              placeholder={t('Bank name')}
              onChange={(event) =>
                setChannels((current) =>
                  updateChannel(current, 'corporate_bank', {
                    bank_name: event.target.value,
                  })
                )
              }
            />
            <Input
              value={bank.bank_account ?? ''}
              placeholder={t('Bank account')}
              onChange={(event) =>
                setChannels((current) =>
                  updateChannel(current, 'corporate_bank', {
                    bank_account: event.target.value,
                  })
                )
              }
            />
            <Input
              value={bank.bank_branch ?? ''}
              placeholder={t('Bank branch')}
              onChange={(event) =>
                setChannels((current) =>
                  updateChannel(current, 'corporate_bank', {
                    bank_branch: event.target.value,
                  })
                )
              }
            />
            <Textarea
              value={bank.instructions ?? ''}
              placeholder={t('Payment instructions')}
              onChange={(event) =>
                setChannels((current) =>
                  updateChannel(current, 'corporate_bank', {
                    instructions: event.target.value,
                  })
                )
              }
            />
          </CardContent>
        </Card>
      </div>
      <div className='flex gap-3'>
        <Button
          variant='outline'
          disabled={save.isPending || publish.isPending}
          onClick={() => save.mutate()}
        >
          {t('Save draft')}
        </Button>
        <Button
          disabled={save.isPending || publish.isPending}
          onClick={() => publish.mutate()}
        >
          {t('Publish')}
        </Button>
      </div>
    </div>
  )
}

function TransferReviewDetail(props: {
  application: CorporateTransferApplication
}) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [channel, setChannel] = useState<
    'enterprise_wechat' | 'corporate_bank'
  >('enterprise_wechat')
  const [reference, setReference] = useState('')
  const [amountYuan, setAmountYuan] = useState('')
  const [receivedAt, setReceivedAt] = useState('')
  const [payerName, setPayerName] = useState('')
  const [reason, setReason] = useState('')
  const [internalNote, setInternalNote] = useState('')
  const [adminReply, setAdminReply] = useState('')
  const [adminReplyFiles, setAdminReplyFiles] = useState<File[]>([])
  const [refundReference, setRefundReference] = useState('')
  const [approvalConfirmOpen, setApprovalConfirmOpen] = useState(false)
  const detail = useQuery({
    queryKey: [
      'corporate-transfer',
      'admin-ticket',
      props.application.ticket?.ticket_no,
    ],
    queryFn: () =>
      getAdminCorporateTransferTicket(
        props.application.ticket?.ticket_no ?? ''
      ),
    enabled: Boolean(props.application.ticket?.ticket_no),
  })
  const refresh = async () => {
    await queryClient.invalidateQueries({
      queryKey: ['corporate-transfer', 'admin'],
    })
    await queryClient.invalidateQueries({
      queryKey: ['corporate-transfer', 'admin-ticket'],
    })
    await queryClient.invalidateQueries({
      queryKey: ['corporate-transfer', 'unread'],
    })
    await queryClient.invalidateQueries({
      queryKey: ['entitlement-store', 'orders'],
    })
  }
  const receiptTotal = useMemo(
    () =>
      (detail.data?.data?.receipts ?? [])
        .filter((receipt) => receipt.status === 'active')
        .reduce((sum, receipt) => sum + receipt.amount_cents, 0),
    [detail.data?.data?.receipts]
  )
  const mutate = useMutation({
    mutationFn: async (action: 'receipt' | 'info' | 'reject' | 'approve') => {
      let response
      if (action === 'receipt') {
        response = await addCorporateReceipt(props.application.application_no, {
          channel,
          external_reference: reference,
          amount_cents: Math.round(Number(amountYuan) * 100),
          received_at: Math.floor(new Date(receivedAt).getTime() / 1000),
          payer_name: payerName,
          admin_note: '',
        })
      } else if (action === 'info') {
        response = await requestCorporateTransferInfo(
          props.application.application_no,
          reason
        )
      } else if (action === 'reject') {
        response = await rejectCorporateTransfer(
          props.application.application_no,
          reason,
          internalNote
        )
      } else {
        response = await approveCorporateTransfer(
          props.application.application_no
        )
      }
      if (response && !response.success) {
        throw new Error(response.message || t('Operation failed'))
      }
    },
    onSuccess: async () => {
      setApprovalConfirmOpen(false)
      await refresh()
      toast.success(t('Operation completed'))
    },
    onError: (error) => toast.error(error.message),
  })
  const voidReceipt = useMutation({
    mutationFn: async (receiptId: number) => {
      const voidReason = window.prompt(t('Reason for voiding this receipt'))
      if (!voidReason) return
      const response = await voidCorporateReceipt(receiptId, voidReason)
      if (!response.success) {
        throw new Error(response.message || t('Operation failed'))
      }
    },
    onSuccess: refresh,
    onError: (error) => toast.error(error.message),
  })
  const refund = useMutation({
    mutationFn: async () => {
      if (
        !window.confirm(
          t('Confirm that the offline refund has already completed?')
        )
      ) {
        return
      }
      const response = await registerCorporateTransferRefund(
        props.application.application_no,
        {
          amount_cents: props.application.order?.cash_paid_cents ?? 0,
          channel,
          external_reference: refundReference,
          refunded_at: Math.floor(Date.now() / 1000),
          reason,
        }
      )
      if (!response.success) {
        throw new Error(response.message || t('Operation failed'))
      }
    },
    onSuccess: refresh,
    onError: (error) => toast.error(error.message),
  })
  const ticketAction = useMutation({
    mutationFn: async (action: 'reply' | 'internal' | 'toggle') => {
      const response =
        action === 'reply' || action === 'internal'
          ? await adminReplyCorporateTransfer(
              props.application.application_no,
              adminReply,
              action === 'internal',
              adminReplyFiles
            )
          : await setCorporateTransferTicketOpen(
              props.application.application_no,
              detail.data?.data?.ticket.status !== 'open'
            )
      if (!response.success) {
        throw new Error(response.message || t('Operation failed'))
      }
    },
    onSuccess: async () => {
      setAdminReply('')
      setAdminReplyFiles([])
      await refresh()
    },
    onError: (error) => toast.error(error.message),
  })
  const data = detail.data?.data
  if (!data) return <p className='text-muted-foreground'>{t('Loading...')}</p>
  const payable = data.application.order?.cash_payable_cents ?? 0

  return (
    <div className='space-y-5'>
      <Card>
        <CardContent className='grid gap-4 pt-6 sm:grid-cols-2 lg:grid-cols-4'>
          <div>
            <p className='text-muted-foreground text-xs'>{t('Application')}</p>
            <p className='font-mono text-sm'>
              {data.application.application_no}
            </p>
          </div>
          <div>
            <p className='text-muted-foreground text-xs'>{t('Order')}</p>
            <p className='font-mono text-sm'>
              {data.application.order?.order_no}
            </p>
          </div>
          <div>
            <p className='text-muted-foreground text-xs'>{t('Cash payable')}</p>
            <p className='font-semibold'>¥{(payable / 100).toFixed(2)}</p>
          </div>
          <div>
            <p className='text-muted-foreground text-xs'>
              {t('Verified receipts')}
            </p>
            <p className='font-semibold'>¥{(receiptTotal / 100).toFixed(2)}</p>
          </div>
        </CardContent>
      </Card>
      <CorporateTransferPaymentPrompt
        application={data.application}
        collection={data.collection}
        collectionOutdated={data.collection_outdated}
        terminal={['cancelled', 'expired', 'rejected', 'fulfilled'].includes(
          data.application.status
        )}
      />
      <Card>
        <CardHeader>
          <CardTitle>{t('Conversation')}</CardTitle>
        </CardHeader>
        <CardContent className='space-y-4'>
          <CorporateTransferMessageList
            messages={data.messages}
            viewerRole='admin'
            application={data.application}
          />
          <CorporateTransferReviewResult application={data.application} />
          <CorporateTransferReplyComposer
            value={adminReply}
            files={adminReplyFiles}
            pending={ticketAction.isPending}
            placeholder={t('Reply to user')}
            onValueChange={setAdminReply}
            onFilesChange={setAdminReplyFiles}
            onSubmit={() => ticketAction.mutate('reply')}
            submitLabel={t('Send reply')}
            secondarySubmitLabel={t('Add internal note')}
            onSecondarySubmit={() => ticketAction.mutate('internal')}
            trailingAction={
              <Button
                variant='outline'
                disabled={ticketAction.isPending}
                onClick={() => ticketAction.mutate('toggle')}
              >
                {data.ticket.status === 'open'
                  ? t('Close ticket')
                  : t('Reopen ticket')}
              </Button>
            }
          />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>{t('Receipt verification')}</CardTitle>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='grid gap-3 md:grid-cols-5'>
            <NativeSelect
              value={channel}
              onChange={(event) =>
                setChannel(event.target.value as typeof channel)
              }
            >
              <NativeSelectOption value='enterprise_wechat'>
                {t('Enterprise WeChat')}
              </NativeSelectOption>
              <NativeSelectOption value='corporate_bank'>
                {t('Corporate bank')}
              </NativeSelectOption>
            </NativeSelect>
            <Input
              value={reference}
              placeholder={t('External transaction reference')}
              onChange={(event) => setReference(event.target.value)}
            />
            <Input
              type='number'
              step='0.01'
              value={amountYuan}
              placeholder={t('Received amount')}
              onChange={(event) => setAmountYuan(event.target.value)}
            />
            <Input
              type='datetime-local'
              value={receivedAt}
              onChange={(event) => setReceivedAt(event.target.value)}
            />
            <Input
              value={payerName}
              placeholder={t('Payer name (optional)')}
              onChange={(event) => setPayerName(event.target.value)}
            />
          </div>
          <Button
            disabled={
              !reference.trim() ||
              !amountYuan ||
              !receivedAt ||
              mutate.isPending
            }
            onClick={() => mutate.mutate('receipt')}
          >
            {t('Add verified receipt')}
          </Button>
          <div className='space-y-2'>
            {(data.receipts ?? []).map((receipt) => (
              <div
                key={receipt.id}
                className='flex flex-wrap items-center justify-between gap-3 rounded-lg border p-3 text-sm'
              >
                <div>
                  <span className='font-mono'>
                    {receipt.external_reference}
                  </span>
                  <span className='ml-3'>
                    ¥{(receipt.amount_cents / 100).toFixed(2)}
                  </span>
                  <Badge className='ml-3' variant='secondary'>
                    {t(corporateTransferReceiptStatusKey(receipt.status))}
                  </Badge>
                </div>
                {receipt.status === 'active' &&
                data.application.status !== 'fulfilled' ? (
                  <Button
                    size='sm'
                    variant='outline'
                    onClick={() => voidReceipt.mutate(receipt.id)}
                  >
                    {t('Void')}
                  </Button>
                ) : null}
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>{t('Review decision')}</CardTitle>
        </CardHeader>
        <CardContent className='space-y-3'>
          <Textarea
            value={reason}
            placeholder={t(
              'User-visible reason for more information or rejection'
            )}
            onChange={(event) => setReason(event.target.value)}
          />
          <Textarea
            value={internalNote}
            placeholder={t('Internal note (not visible to user)')}
            onChange={(event) => setInternalNote(event.target.value)}
          />
          <div className='flex flex-wrap gap-3'>
            <Button
              variant='outline'
              disabled={!reason.trim() || mutate.isPending}
              onClick={() => mutate.mutate('info')}
            >
              {t('Request more information')}
            </Button>
            <Button
              variant='destructive'
              disabled={!reason.trim() || mutate.isPending}
              onClick={() => mutate.mutate('reject')}
            >
              {t('Reject')}
            </Button>
            <Button
              disabled={
                receiptTotal !== payable ||
                mutate.isPending ||
                data.application.status !== 'under_review'
              }
              onClick={() => setApprovalConfirmOpen(true)}
            >
              {t('Confirm receipt and fulfill')}
            </Button>
          </div>
        </CardContent>
      </Card>
      <ConfirmDialog
        open={approvalConfirmOpen}
        onOpenChange={(open) => {
          if (mutate.isPending) return
          setApprovalConfirmOpen(open)
          if (!open) toast.info(t('Entitlement grant cancelled'))
        }}
        title={t('Confirm receipt and fulfill')}
        desc={t(
          'Confirm receipt of {{amount}} and grant the locked entitlement? This cannot be undone directly.',
          {
            amount: new Intl.NumberFormat(undefined, {
              style: 'currency',
              currency: 'CNY',
            }).format(receiptTotal / 100),
          }
        )}
        confirmText={t('Confirm receipt and fulfill')}
        isLoading={mutate.isPending}
        disabled={
          receiptTotal !== payable ||
          data.application.status !== 'under_review'
        }
        handleConfirm={() => mutate.mutate('approve')}
      />
      {data.application.status === 'fulfilled' &&
      data.application.order?.status === 'fulfilled' ? (
        <Card>
          <CardHeader>
            <CardTitle>{t('Offline refund registration')}</CardTitle>
          </CardHeader>
          <CardContent className='space-y-3'>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Register only after the Enterprise WeChat or bank refund has actually completed.'
              )}
            </p>
            <NativeSelect
              value={channel}
              onChange={(event) =>
                setChannel(event.target.value as typeof channel)
              }
            >
              <NativeSelectOption value='enterprise_wechat'>
                {t('Enterprise WeChat')}
              </NativeSelectOption>
              <NativeSelectOption value='corporate_bank'>
                {t('Corporate bank')}
              </NativeSelectOption>
            </NativeSelect>
            <Input
              value={refundReference}
              placeholder={t('External refund reference')}
              onChange={(event) => setRefundReference(event.target.value)}
            />
            <Textarea
              value={reason}
              placeholder={t('Refund reason')}
              onChange={(event) => setReason(event.target.value)}
            />
            <Button
              variant='destructive'
              disabled={
                !refundReference.trim() || !reason.trim() || refund.isPending
              }
              onClick={() => refund.mutate()}
            >
              {t('Register completed refund')}
            </Button>
          </CardContent>
        </Card>
      ) : null}
    </div>
  )
}

export function AdminCorporateTransfer() {
  const { t } = useTranslation()
  const role = useAuthStore((state) => state.auth.user?.role ?? 0)
  const [status, setStatus] = useState('under_review')
  const [keyword, setKeyword] = useState('')
  const [selected, setSelected] = useState<CorporateTransferApplication | null>(
    null
  )
  const applications = useQuery({
    queryKey: ['corporate-transfer', 'admin', status, keyword],
    queryFn: () =>
      getAdminCorporateTransfers({ status, keyword, page_size: 100 }),
  })
  return (
    <div className='space-y-8'>
      {role >= ROLE.SUPER_ADMIN ? <CollectionConfiguration /> : null}
      <div className='space-y-4'>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <div>
            <h3 className='text-lg font-semibold'>
              {t('Corporate transfer review')}
            </h3>
            <p className='text-muted-foreground text-sm'>
              {t(
                'Verify receipts against the actual collection account before fulfillment.'
              )}
            </p>
          </div>
          <div className='flex flex-wrap gap-3'>
            <Input
              value={keyword}
              placeholder={t('Application, order or external reference')}
              onChange={(event) => setKeyword(event.target.value)}
            />
            <NativeSelect
              value={status}
              onChange={(event) => setStatus(event.target.value)}
            >
              <NativeSelectOption value='under_review'>
                {t('Under review')}
              </NativeSelectOption>
              <NativeSelectOption value='needs_more_information'>
                {t('More information required')}
              </NativeSelectOption>
              <NativeSelectOption value='awaiting_evidence'>
                {t('Awaiting payment evidence')}
              </NativeSelectOption>
              <NativeSelectOption value='fulfilled'>
                {t('Fulfilled')}
              </NativeSelectOption>
              <NativeSelectOption value='rejected'>
                {t('Rejected')}
              </NativeSelectOption>
              <NativeSelectOption value='expired'>
                {t('Expired')}
              </NativeSelectOption>
            </NativeSelect>
            <Badge variant='secondary'>
              {t('Applications: {{count}}', {
                count: applications.data?.data?.total ?? 0,
              })}
            </Badge>
          </div>
        </div>
        <div className='overflow-x-auto rounded-xl border'>
          <table className='w-full text-sm'>
            <thead className='bg-muted/50 text-left'>
              <tr>
                <th className='p-3'>{t('Application')}</th>
                <th className='p-3'>{t('Order')}</th>
                <th className='p-3'>{t('Amount')}</th>
                <th className='p-3'>{t('Status')}</th>
                <th className='p-3'>{t('Actions')}</th>
              </tr>
            </thead>
            <tbody>
              {(applications.data?.data?.items ?? []).map((application) => (
                <tr key={application.id} className='border-t'>
                  <td className='p-3 font-mono text-xs'>
                    {application.application_no}
                  </td>
                  <td className='p-3 font-mono text-xs'>
                    {application.order?.order_no}
                  </td>
                  <td className='p-3'>
                    ¥
                    {(
                      (application.order?.cash_payable_cents ?? 0) / 100
                    ).toFixed(2)}
                  </td>
                  <td className='p-3'>
                    <Badge variant='secondary'>
                      {t(corporateTransferStatusKey(application.status))}
                    </Badge>
                  </td>
                  <td className='p-3'>
                    <Button
                      size='sm'
                      variant='outline'
                      onClick={() => setSelected(application)}
                    >
                      {t('Review')}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
      <Sheet
        open={Boolean(selected)}
        onOpenChange={(open) => {
          if (!open) setSelected(null)
        }}
      >
        <SheetContent
          side='right'
          className={sideDrawerContentClassName(
            'max-w-none sm:!max-w-[min(960px,92vw)]'
          )}
        >
          <SheetHeader className={sideDrawerHeaderClassName()}>
            <SheetTitle>{t('Review details')}</SheetTitle>
            <SheetDescription>
              {selected?.application_no ?? ''}
            </SheetDescription>
          </SheetHeader>
          <div className={sideDrawerFormClassName()}>
            {selected ? <TransferReviewDetail application={selected} /> : null}
          </div>
        </SheetContent>
      </Sheet>
    </div>
  )
}
