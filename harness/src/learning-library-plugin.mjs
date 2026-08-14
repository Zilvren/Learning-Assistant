import { defineTool } from '@deepseek-ai/dsh-tools'

export const name = 'learning-library'
export const inject = ['tools']

function bridgeURL() {
  const value = String(process.env.LEARNING_ASSISTANT_BRIDGE_URL || '').trim()
  if (!/^http:\/\/(?:127\.0\.0\.1|localhost)(?::\d+)?$/u.test(value)) {
    throw new Error('学习资料库桥接地址未配置或不是本机地址')
  }
  return value
}

function capabilityToken() {
  const value = String(process.env.LEARNING_ASSISTANT_HARNESS_TOKEN || '').trim()
  if (value.length < 32) throw new Error('学习资料库会话授权已失效')
  return value
}

async function callBridge(tool, args) {
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), 45_000)
  try {
    const response = await fetch(`${bridgeURL()}/internal/harness/tools/${tool}`, {
      method: 'POST',
      headers: {
        authorization: `Bearer ${capabilityToken()}`,
        'content-type': 'application/json',
      },
      body: JSON.stringify(args),
      signal: controller.signal,
    })
    const body = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(String(body?.detail?.message || body?.message || '资料库工具调用失败'))
    return body.result
  } finally {
    clearTimeout(timer)
  }
}

function jsonOutput() {
  return {
    schema: { type: 'object', additionalProperties: true },
    render: (_args, value) => [{ type: 'text', text: JSON.stringify(value) }],
  }
}

// This plugin deliberately exposes no “apply” or generic filesystem tool. A
// prepared change is collected by Go and rendered in the existing preview UI;
// only a later browser confirmation can save it to PostgreSQL.
export function apply(ctx) {
  ctx.tools.register(defineTool({
    name: 'list_library_paths',
    description: 'List readable folders and notes within the current conversation\'s allowed learning-library scope. Use it to disambiguate a path before reading or preparing a note change.',
    parameters: {
      query: { type: 'string', description: 'Optional words used to narrow returned paths.' },
      limit: { type: 'number', description: 'Maximum paths to return, between 1 and 80.' },
    },
    output: jsonOutput(),
    execute: (args) => callBridge('list_paths', args),
  }))

  ctx.tools.register(defineTool({
    name: 'search_library',
    description: 'Search the allowed learning-library scope for relevant notes. Returns titles, paths, tags, and short excerpts; call read_library_note only for notes you need in full.',
    parameters: {
      query: { type: 'string', required: true, description: 'Learning topic, question, or keywords to search for.' },
      limit: { type: 'number', description: 'Maximum results to return, between 1 and 12.' },
    },
    output: jsonOutput(),
    execute: (args) => callBridge('search', args),
  }))

  ctx.tools.register(defineTool({
    name: 'read_library_note',
    description: 'Read one readable note from the current allowed learning-library scope by its numeric id. Never request files outside the scope.',
    parameters: {
      item_id: { type: 'number', required: true, description: 'The note id returned by list_library_paths or search_library.' },
    },
    output: jsonOutput(),
    execute: (args) => callBridge('read_note', args),
  }))

  ctx.tools.register(defineTool({
    name: 'prepare_note_change',
    description: 'Prepare a Markdown or text note creation/update preview at an explicit library path. Before calling, resolve an ambiguous path with list_library_paths. This never saves anything. After success, tell the user a preview is ready and ask them to confirm it in the app.',
    parameters: {
      path: { type: 'string', required: true, description: 'Explicit destination such as daily/20260814.md. A filename is required.' },
      content: { type: 'string', required: true, description: 'The complete Markdown or text content proposed for the note.' },
    },
    output: jsonOutput(),
    execute: (args) => callBridge('prepare_note_change', args),
  }))
}
