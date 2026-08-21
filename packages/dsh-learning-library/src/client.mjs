export const DEFAULT_BRIDGE_URL_ENV = 'DSH_LEARNING_LIBRARY_BRIDGE_URL'
export const DEFAULT_TOKEN_ENV = 'DSH_LEARNING_LIBRARY_CAPABILITY_TOKEN'
export const DEFAULT_TOOL_PATH_PREFIX = '/v1/learning-library/tools'
export const DEFAULT_REQUEST_TIMEOUT_MS = 45_000

// stringOption 完成当前模块定义的局部处理。
function stringOption(value, fallback) {
  const normalized = String(value ?? '').trim()
  return normalized || fallback
}

// environmentName 完成当前模块定义的局部处理。
function environmentName(value, optionName) {
  const name = stringOption(value, '')
  if (!/^[A-Za-z_][A-Za-z0-9_]*$/u.test(name)) {
    throw new Error(`${optionName} must be a valid environment variable name`)
  }
  return name
}

// normalizeToolPathPrefix 完成当前模块定义的局部处理。
function normalizeToolPathPrefix(value) {
  const path = stringOption(value, DEFAULT_TOOL_PATH_PREFIX)
  if (!path.startsWith('/') || path.includes('..') || /[?#]/u.test(path)) {
    throw new Error('toolPathPrefix must be an absolute path without query, fragment, or parent segments')
  }
  return path.replace(/\/+$/u, '') || '/'
}

// normalizeTimeout 完成当前模块定义的局部处理。
function normalizeTimeout(value) {
  const timeout = Number(value ?? DEFAULT_REQUEST_TIMEOUT_MS)
  if (!Number.isFinite(timeout) || timeout < 1_000 || timeout > 120_000) {
    throw new Error('requestTimeoutMs must be between 1000 and 120000')
  }
  return Math.round(timeout)
}

// readBridgeURL 完成当前模块定义的局部处理。
function readBridgeURL(env, bridgeUrlEnv) {
  const raw = String(env[bridgeUrlEnv] ?? '').trim()
  if (!raw) throw new Error(`Learning-library bridge URL is missing in ${bridgeUrlEnv}`)

  let url
  try {
    url = new URL(raw)
  } catch {
    throw new Error(`Learning-library bridge URL in ${bridgeUrlEnv} is invalid`)
  }

  if (url.username || url.password || url.search || url.hash) {
    throw new Error('Learning-library bridge URL must not include credentials, query, or fragment')
  }

  const host = url.hostname.toLowerCase()
  const isLoopback = host === '127.0.0.1' || host === 'localhost' || host === '[::1]' || host === '::1'
  if (url.protocol !== 'https:' && !(url.protocol === 'http:' && isLoopback)) {
    throw new Error('Learning-library bridge must use HTTPS, except for a loopback HTTP bridge')
  }

  return url.toString().replace(/\/+$/u, '')
}

// readCapabilityToken 完成当前模块定义的局部处理。
function readCapabilityToken(env, tokenEnv) {
  const token = String(env[tokenEnv] ?? '').trim()
  if (token.length < 16) throw new Error(`Learning-library capability token in ${tokenEnv} is missing or too short`)
  return token
}

// errorMessage 完成当前模块定义的局部处理。
function errorMessage(body, fallback) {
  const detail = body?.detail
  if (typeof detail === 'string' && detail.trim()) return detail
  if (typeof detail?.message === 'string' && detail.message.trim()) return detail.message
  if (typeof body?.error?.message === 'string' && body.error.message.trim()) return body.error.message
  if (typeof body?.message === 'string' && body.message.trim()) return body.message
  return fallback
}

// toolURL 完成当前模块定义的局部处理。
function toolURL(baseURL, toolPathPrefix, toolName) {
  if (!/^[a-z][a-z0-9_]*$/u.test(toolName)) throw new Error('Invalid learning-library tool name')
  return `${baseURL}${toolPathPrefix}/${encodeURIComponent(toolName)}`
}

/**
 * 创建供插件使用、由能力令牌保护的最小桥接层。
 * URL 和令牌值刻意只从宿主环境读取。
 */
export function createLearningLibraryBridge(config = {}, dependencies = {}) {
  const env = dependencies.env ?? process.env
  const fetchImpl = dependencies.fetchImpl ?? globalThis.fetch
  if (typeof fetchImpl !== 'function') throw new Error('A fetch implementation is required')

  const bridgeUrlEnv = environmentName(config.bridgeUrlEnv ?? DEFAULT_BRIDGE_URL_ENV, 'bridgeUrlEnv')
  const tokenEnv = environmentName(config.tokenEnv ?? DEFAULT_TOKEN_ENV, 'tokenEnv')
  const toolPathPrefix = normalizeToolPathPrefix(config.toolPathPrefix)
  const requestTimeoutMs = normalizeTimeout(config.requestTimeoutMs)

  return {
    async call(toolName, args = {}) {
      const controller = new AbortController()
      const timer = setTimeout(() => controller.abort(), requestTimeoutMs)
      try {
        const response = await fetchImpl(toolURL(
          readBridgeURL(env, bridgeUrlEnv),
          toolPathPrefix,
          toolName,
        ), {
          method: 'POST',
          headers: {
            authorization: `Bearer ${readCapabilityToken(env, tokenEnv)}`,
            'content-type': 'application/json',
          },
          body: JSON.stringify(args),
          signal: controller.signal,
        })
        const body = await response.json().catch(() => ({}))
        if (!response.ok) throw new Error(errorMessage(body, `Learning-library bridge request failed (${response.status})`))
        return body.result
      } catch (error) {
        if (error?.name === 'AbortError') throw new Error(`Learning-library bridge request timed out after ${requestTimeoutMs}ms`)
        throw error
      } finally {
        clearTimeout(timer)
      }
    },
  }
}
