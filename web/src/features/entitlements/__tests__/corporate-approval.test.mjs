import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import vm from 'node:vm'
import { transformSync } from 'esbuild'
import { MutationObserver, QueryClient } from '@tanstack/query-core'

const source = readFileSync(new URL('../components/admin-corporate-transfer.tsx', import.meta.url), 'utf8')
const start = source.indexOf('  const mutate = useMutation({', source.indexOf('function TransferReviewDetail'))
const end = source.indexOf('  const voidReceipt', start)
const mutationCode = transformSync(`${source.slice(start, end)}\nmutate`, { loader: 'ts' }).code
// Exercise the real JSX callbacks at the confirmation boundary without a browser.
const dialogStart = source.indexOf('      <ConfirmDialog', start)
const dialogEnd = source.indexOf('      />', dialogStart) + '      />'.length
const buttonEnd = source.indexOf("{t('Confirm receipt and fulfill')}", start)
const buttonStart = source.lastIndexOf('<Button', buttonEnd)
const buttonClose = source.indexOf('</Button>', buttonEnd) + '</Button>'.length
function render(fragment, context) {
  const code = transformSync(`(${fragment})`, { loader: 'tsx', jsxFactory: 'element' }).code
  return vm.runInNewContext(code, {
    element: (_type, props) => props,
    Button: 'button', ConfirmDialog: 'dialog', t: (key) => key,
    receiptTotal: 20000, payable: 20000,
    data: { application: { status: 'under_review' } },
    ...context,
  })
}

test('initial approval click opens confirmation without granting', () => {
  const states = [], requests = []
  const button = render(source.slice(buttonStart, buttonClose), {
    setApprovalConfirmOpen: (open) => states.push(open),
    mutate: { isPending: false, mutate: (action) => requests.push(action) },
  })
  button.onClick()
  assert.deepEqual(states, [true])
  assert.deepEqual(requests, [])
})

for (const pending of [false, true]) {
  test(`cancel ${pending ? 'during submission is ignored' : 'closes confirmation and reports cancellation without granting'}`, () => {
    const states = [], messages = [], requests = []
    const dialog = render(source.slice(dialogStart, dialogEnd), {
      approvalConfirmOpen: true,
      setApprovalConfirmOpen: (open) => states.push(open),
      toast: { info: (message) => messages.push(message) },
      mutate: { isPending: pending, mutate: (action) => requests.push(action) },
    })
    dialog.onOpenChange(false)
    assert.deepEqual(states, pending ? [] : [false])
    assert.deepEqual(messages, pending ? [] : ['Entitlement grant cancelled'])
    assert.deepEqual(requests, [])
    assert.equal(dialog.isLoading, pending)
  })
}

test('explicit confirmation submits approval', () => {
  const requests = []
  const dialog = render(source.slice(dialogStart, dialogEnd), {
    approvalConfirmOpen: true,
    mutate: { isPending: false, mutate: (action) => requests.push(action) },
  })
  dialog.handleConfirm()
  assert.deepEqual(requests, ['approve'])
})

for (const accepted of [true, false]) {
  test(`server approval ${accepted ? 'success closes confirmation and refreshes' : 'failure stays open and reports error'}`, async () => {
    const client = new QueryClient()
    const states = [], success = [], errors = []
    let refreshes = 0
    try {
      const mutation = vm.runInNewContext(mutationCode, {
        useMutation: (options) => new MutationObserver(client, options),
        props: { application: { application_no: 'TEST-APPROVAL' } },
        t: (key) => key,
        approveCorporateTransfer: async () => ({ success: accepted, message: 'Approval failed' }),
        setApprovalConfirmOpen: (open) => states.push(open),
        refresh: async () => { refreshes++ },
        toast: { success: (message) => success.push(message), error: (message) => errors.push(message) },
      })
      if (accepted) await mutation.mutate('approve')
      else await assert.rejects(mutation.mutate('approve'), /Approval failed/)
      assert.deepEqual(states, accepted ? [false] : [])
      assert.deepEqual(success, accepted ? ['Operation completed'] : [])
      assert.deepEqual(errors, accepted ? [] : ['Approval failed'])
      assert.equal(refreshes, accepted ? 1 : 0)
    } finally { client.clear() }
  })
}
