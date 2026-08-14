import assert from 'node:assert/strict'
import test from 'node:test'
import { apply, inject, name } from '../src/index.mjs'

test('registers the restricted tools as a DeepSeek Harness plugin', () => {
  const registered = []
  const context = {
    tools: {
      register: (tool) => registered.push(tool),
    },
  }

  apply(context, {
    bridgeUrlEnv: 'TEST_BRIDGE_URL',
    tokenEnv: 'TEST_CAPABILITY_TOKEN',
    toolPathPrefix: '/v1/learning-library/tools',
  })

  assert.equal(name, 'dsh-learning-library')
  assert.deepEqual(inject, ['tools'])
  assert.deepEqual(registered.map((tool) => tool.name), [
    'list_library_paths',
    'search_library',
    'read_library_note',
    'create_library_note',
    'update_library_note',
  ])
})
