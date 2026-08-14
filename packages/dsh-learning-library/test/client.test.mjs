import assert from 'node:assert/strict'
import test from 'node:test'
import { createLearningLibraryBridge } from '../src/client.mjs'

const token = '0123456789abcdef0123456789abcdef'

test('calls a configured HTTPS bridge with a bearer capability token', async () => {
  const calls = []
  const bridge = createLearningLibraryBridge({
    bridgeUrlEnv: 'TEST_BRIDGE_URL',
    tokenEnv: 'TEST_CAPABILITY_TOKEN',
    toolPathPrefix: '/bridge/v1/tools',
  }, {
    env: {
      TEST_BRIDGE_URL: 'https://bridge.example.test/base',
      TEST_CAPABILITY_TOKEN: token,
    },
    fetchImpl: async (url, init) => {
      calls.push({ url, init })
      return new Response(JSON.stringify({ result: { items: ['daily'] } }), { status: 200 })
    },
  })

  const result = await bridge.call('list_paths', { limit: 5 })

  assert.deepEqual(result, { items: ['daily'] })
  assert.equal(calls[0].url, 'https://bridge.example.test/base/bridge/v1/tools/list_paths')
  assert.equal(calls[0].init.headers.authorization, `Bearer ${token}`)
  assert.equal(calls[0].init.body, '{"limit":5}')
})

test('allows an HTTP bridge only on loopback', async () => {
  const baseConfig = { bridgeUrlEnv: 'TEST_BRIDGE_URL', tokenEnv: 'TEST_CAPABILITY_TOKEN' }
  const baseDependencies = { env: { TEST_CAPABILITY_TOKEN: token }, fetchImpl: async () => new Response('{}') }

  await assert.rejects(createLearningLibraryBridge(baseConfig, {
    ...baseDependencies,
    env: { ...baseDependencies.env, TEST_BRIDGE_URL: 'http://bridge.example.test' },
  }).call('search', {}), /must use HTTPS/u)

  const bridge = createLearningLibraryBridge(baseConfig, {
    ...baseDependencies,
    env: { ...baseDependencies.env, TEST_BRIDGE_URL: 'http://127.0.0.1:8000' },
  })
  await bridge.call('search', {})
})

test('reports a bridge error without exposing the capability token', async () => {
  const bridge = createLearningLibraryBridge({
    bridgeUrlEnv: 'TEST_BRIDGE_URL',
    tokenEnv: 'TEST_CAPABILITY_TOKEN',
  }, {
    env: { TEST_BRIDGE_URL: 'https://bridge.example.test', TEST_CAPABILITY_TOKEN: token },
    fetchImpl: async () => new Response(JSON.stringify({ detail: { message: 'Path is outside the granted scope' } }), { status: 403 }),
  })

  await assert.rejects(bridge.call('read_note', { item_id: 7 }), /outside the granted scope/u)
})

test('rejects unsafe bridge routing configuration', () => {
  assert.throws(() => createLearningLibraryBridge({ toolPathPrefix: '../tools' }), /absolute path/u)
  assert.throws(() => createLearningLibraryBridge({ bridgeUrlEnv: 'NOT-AN-ENV-NAME' }), /environment variable/u)
})
