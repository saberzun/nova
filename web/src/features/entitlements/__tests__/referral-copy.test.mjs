import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import test from 'node:test'
import vm from 'node:vm'
import { transformSync } from 'esbuild'
import React from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import * as query from '@tanstack/react-query'
import * as reactI18next from 'react-i18next'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createInstance } from 'i18next'
import { I18nextProvider } from 'react-i18next'

const require = createRequire(import.meta.url)
const source = readFileSync(new URL('../referral.tsx', import.meta.url), 'utf8')
const code = transformSync(source, { loader: 'tsx', format: 'cjs', jsx: 'automatic' }).code
const box = (props) => React.createElement('div', null, props.children)
const layout = Object.assign(box, { Title: box, Content: box })
const module = { exports: {} }
vm.runInNewContext(code, {
  module, exports: module.exports, window: { location: { origin: 'https://test.example' } },
  require: (name) => {
    // Presentation primitives are irrelevant to the referral text. Query and i18n are real.
    if (name === '@tanstack/react-query') return query
    if (name === 'react-i18next') return reactI18next
    if (name === '@/components/layout') return { SectionPageLayout: layout }
    if (name.startsWith('@/components/')) return new Proxy({}, { get: () => box })
    if (name === './api') return { getGiftSelf: async () => { throw new Error('Unexpected request') } }
    if (name === './lib') return { formatDate: () => '' }
    return require(name)
  },
})

async function render(settings, language = 'zh') {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  if (settings) {
    client.setQueryData(['gift', 'self'], {
    success: true,
    data: { settings, account: { available_cents: 0 }, rewards: [], aff_code: 'fixture', aff_count: 0 },
    })
  }
  const i18n = createInstance()
  await i18n.init({ lng: language, fallbackLng: 'en', keySeparator: false, resources: {
    [language]: JSON.parse(readFileSync(new URL(`../../../i18n/locales/${language}.json`, import.meta.url), 'utf8')),
  } })
  try {
    return renderToStaticMarkup(React.createElement(QueryClientProvider, { client },
      React.createElement(I18nextProvider, { i18n }, React.createElement(module.exports.ReferralGiftPage))))
  } finally { client.clear() }
}

const settings = { referral_enabled: true, first_rate_bps: 1500, recurring_rate_bps: 700 }
test('configured 15% and 7% rates appear instead of hard-coded defaults', async () => {
  const html = await render(settings)
  assert.match(html, /15%/)
  assert.match(html, /7%/)
  assert.doesNotMatch(html, /(?<![\d.])(?:10|5)%/)
})
test('fractional and zero rates are preserved', async () => {
  const html = await render({ ...settings, first_rate_bps: 1250, recurring_rate_bps: 0 }, 'en')
  assert.match(html, /12\.5%/)
  assert.match(html, /0%/)
})
test('disabled referral program does not promise rewards', async () => {
  const html = await render({ ...settings, referral_enabled: false })
  assert.match(html, /邀请返现暂未启用/)
  assert.doesNotMatch(html, /15%|7%|10%|5%/)
})
test('unloaded settings do not display default rates', async () => {
  const html = await render(undefined)
  assert.doesNotMatch(html, /(?<![\d.])(?:10|5)%/)
})
