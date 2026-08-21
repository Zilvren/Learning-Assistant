const BASE = '/api'

let refreshing = null

export class ApiError extends Error {
  // constructor 初始化前端可展示的接口错误及其状态详情。
  constructor(message, status, detail = null) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.detail = detail
  }
}

// responseError 将非成功 HTTP 响应规范化为前端可展示的 ApiError。
async function responseError(res, fallback = '请求失败') {
  const payload = await res.json().catch(() => ({ detail: res.statusText }))
  const message = res.status === 503
    ? (payload.detail || '数据目录正在被其他操作占用，请稍后重试')
    : (payload.detail || fallback)
  return new ApiError(message, res.status, payload)
}

// request 是 JSON API 的统一入口，负责 Cookie、一次刷新令牌重试和响应解码。
async function request(method, path, body, retry = true) {
  const opts = { method, headers: {}, credentials: 'include' }
  if (body) {
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(body)
  }
  const res = await fetch(BASE + path, opts)
  if (res.status === 401 && retry && !path.startsWith('/auth/')) {
    const ok = await refreshAuth()
    if (ok) return request(method, path, body, false)
  }
  if (!res.ok) {
    throw await responseError(res)
  }
  const ct = res.headers.get('content-type') || ''
  if (ct.includes('application/pdf')) {
    return res.blob()
  }
  return res.json()
}

// requestBackupExport 单独处理二进制 ZIP 下载，避免走 JSON 解码分支。
async function requestBackupExport(retry = true) {
  const res = await fetch(BASE + '/backup/export', { credentials: 'include' })
  if (res.status === 401 && retry) {
    const ok = await refreshAuth()
    if (ok) return requestBackupExport(false)
  }
  if (!res.ok) {
    throw await responseError(res, '备份失败')
  }
  return res.blob()
}

// requestBackupImport 以原始文件体上传 ZIP 备份，并沿用登录过期重试策略。
async function requestBackupImport(file, retry = true) {
  const res = await fetch(BASE + '/backup/import', { method: 'POST', body: file, credentials: 'include' })
  if (res.status === 401 && retry) {
    const ok = await refreshAuth()
    if (ok) return requestBackupImport(file, false)
  }
  if (!res.ok) {
    throw await responseError(res, '导入失败')
  }
  return res.json()
}

// requestLibraryContent 读取笔记或文件原文，并从 ETag 提取乐观并发版本号。
async function requestLibraryContent(id, retry = true) {
  const res = await fetch(`${BASE}/library/items/${id}/content`, { credentials: 'include' })
  if (res.status === 401 && retry) {
    const ok = await refreshAuth()
    if (ok) return requestLibraryContent(id, false)
  }
  if (!res.ok) throw await responseError(res, '读取资料失败')
  return { content: await res.text(), version: Number((res.headers.get('etag') || '').replace(/\D/g, '') || 0), type: res.headers.get('content-type') || '' }
}

// uploadLibraryFile 用 FormData 上传资料库文件，不手动设置 multipart Content-Type。
async function uploadLibraryFile(file, parentId, retry = true) {
  const form = new FormData()
  form.append('file', file)
  if (parentId) form.append('parent_id', String(parentId))
  const res = await fetch(`${BASE}/library/uploads`, { method: 'POST', body: form, credentials: 'include' })
  if (res.status === 401 && retry) {
    const ok = await refreshAuth()
    if (ok) return uploadLibraryFile(file, parentId, false)
  }
  if (!res.ok) throw await responseError(res, '上传失败')
  return res.json()
}

// refreshAuth 合并并发刷新请求，避免多个接口同时触发重复的 refresh token 调用。
async function refreshAuth() {
  if (!refreshing) {
    refreshing = fetch(BASE + '/auth/refresh', { method: 'POST', credentials: 'include' })
      .then(res => res.ok)
      .catch(() => false)
      .finally(() => { refreshing = null })
  }
  return refreshing
}

// ocrImage 上传原始文件，并且只允许在凭据过期后刷新重试一次。
async function ocrImage(file, retry = true) {
  const blob = file instanceof Blob ? file : new Blob([file])
  const headers = {}
  if (file?.name) headers['X-OCR-Filename'] = encodeURIComponent(file.name)
  const resp = await fetch(BASE + '/ocr', { method: 'POST', body: blob, headers, credentials: 'include' })
  if (resp.status === 401 && retry) {
    const ok = await refreshAuth()
    if (ok) return ocrImage(file, false)
  }
  if (!resp.ok) throw await responseError(resp, 'OCR failed')
  return resp.json()
}

// pause 封装前端到后端的接口调用或响应处理。
const pause = (ms) => new Promise((resolve) => window.setTimeout(resolve, ms))

// waitForOCRTask 封装前端到后端的接口调用或响应处理。
async function waitForOCRTask(id, { attempts = 150, interval = 2000 } = {}) {
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    const task = await request('GET', `/ocr/tasks/${id}`)
    if (task.status === 'succeeded') return task
    if (task.status === 'failed') throw new ApiError(task.error_message || 'OCR 识别失败', 422, task)
    await pause(interval)
  }
  throw new ApiError('OCR 识别仍在处理中，请稍后在设置中心的任务列表查看结果', 408)
}

// api 集中声明页面可调用的后端接口；简单的一行包装函数直接映射到 request。
export const api = {
// getLibraryItems 封装对应的后端接口调用。
	getLibraryItems: ({ parentId = null, all = false, kind = '', query = '', tag = '', review = false, due = false, trashed = false } = {}) => {
		const p = new URLSearchParams()
		if (parentId) p.set('parent_id', parentId)
		if (all) p.set('all', 'true')
		if (kind) p.set('kind', kind)
		if (query) p.set('q', query)
		if (tag) p.set('tag', tag)
		if (review) p.set('review', 'true')
		if (due) p.set('due', 'true')
		if (trashed) p.set('trashed', 'true')
		return request('GET', `/library/items?${p}`)
	},
// getLibraryItem 封装对应的后端接口调用。
	getLibraryItem: (id) => request('GET', `/library/items/${id}`),
// createLibraryItem 封装对应的后端接口调用。
	createLibraryItem: (data) => request('POST', '/library/items', data),
// updateLibraryItem 封装对应的后端接口调用。
	updateLibraryItem: (id, data) => request('PATCH', `/library/items/${id}`, data),
// trashLibraryItem 封装对应的后端接口调用。
	trashLibraryItem: (id) => request('DELETE', `/library/items/${id}`),
// restoreLibraryItem 封装对应的后端接口调用。
	restoreLibraryItem: (id) => request('POST', `/library/items/${id}/restore`),
// purgeLibraryItem 封装对应的后端接口调用。
	purgeLibraryItem: (id) => request('DELETE', `/library/items/${id}/purge`),
// duplicateLibraryItem 封装对应的后端接口调用。
	duplicateLibraryItem: (id, parentId = null) => request('POST', `/library/items/${id}/duplicate`, { parent_id: parentId }),
	getLibraryContent: requestLibraryContent,
// getLibraryPreview 封装对应的后端接口调用。
	getLibraryPreview: (id) => request('GET', `/library/items/${id}/preview`),
// saveLibraryContent 封装对应的后端接口调用。
	saveLibraryContent: (id, data) => request('PUT', `/library/items/${id}/content`, data),
	uploadLibraryFile,
// batchLibraryItems 封装对应的后端接口调用。
	batchLibraryItems: (action, ids, parentId = null) => request('POST', '/library/batch', { action, ids, parent_id: parentId }),
// getLibraryVersions 封装对应的后端接口调用。
	getLibraryVersions: (id) => request('GET', `/library/items/${id}/versions`),
// restoreLibraryVersion 封装对应的后端接口调用。
	restoreLibraryVersion: (id, versionId) => request('POST', `/library/items/${id}/versions/${versionId}/restore`),
// getLibraryTags 封装对应的后端接口调用。
	getLibraryTags: () => request('GET', '/library/tags'),
// getLibraryReviews 封装对应的后端接口调用。
	getLibraryReviews: () => request('GET', '/library/reviews'),
// reviewLibraryNote 封装对应的后端接口调用。
	reviewLibraryNote: (id, rating = 'good') => request('POST', `/library/items/${id}/review`, { rating }),
// getReviewInbox 封装对应的后端接口调用。
	getReviewInbox: () => request('GET', '/review/inbox'),
// searchLearning 封装对应的后端接口调用。
	searchLearning: (query) => request('GET', `/search?q=${encodeURIComponent(query)}`),
// getSubjects 封装对应的后端接口调用。
  getSubjects: () => request('GET', '/subjects'),
// authStatus 封装对应的后端接口调用。
  authStatus: () => request('GET', '/auth/status', null, false),
// me 封装对应的后端接口调用。
  me: () => request('GET', '/auth/me', null, false),
// login 封装对应的后端接口调用。
  login: (account, password) => request('POST', '/auth/login', { account, password }, false),
// register 封装对应的后端接口调用。
  register: (username, email, password) => request('POST', '/auth/register', { username, email, password }, false),
// verifyEmail 封装对应的后端接口调用。
  verifyEmail: (token) => request('POST', '/auth/verify-email', { token }, false),
// resendVerification 封装对应的后端接口调用。
  resendVerification: (email) => request('POST', '/auth/resend-verification', { email }, false),
// refreshAuth 封装对应的后端接口调用。
  refreshAuth: () => request('POST', '/auth/refresh', null, false),
// logout 封装对应的后端接口调用。
  logout: () => request('POST', '/auth/logout', null, false),
// addSubject 封装对应的后端接口调用。
  addSubject: (name) => request('POST', '/subjects', { name }),
// deleteSubject 封装对应的后端接口调用。
  deleteSubject: (name) => request('DELETE', '/subjects/' + encodeURIComponent(name)),
  ocrImage,
// getOCRTasks 封装对应的后端接口调用。
	getOCRTasks: () => request('GET', '/ocr/tasks'),
// getOCRTask 封装对应的后端接口调用。
	getOCRTask: (id) => request('GET', `/ocr/tasks/${id}`),
// retryOCRTask 封装对应的后端接口调用。
	retryOCRTask: (id) => request('POST', `/ocr/tasks/${id}/retry`),
	waitForOCRTask,
// saveToken 封装对应的后端接口调用。
  saveToken: (token) => request('PUT', '/settings/token', { token }),
// clearToken 封装对应的后端接口调用。
  clearToken: () => request('DELETE', '/settings/token'),
// getToken 封装对应的后端接口调用。
  getToken: () => request('GET', '/settings/token'),
// getDeepSeekToken 封装对应的后端接口调用。
	getDeepSeekToken: () => request('GET', '/settings/deepseek'),
// saveDeepSeekToken 封装对应的后端接口调用。
	saveDeepSeekToken: (token) => request('PUT', '/settings/deepseek', { token }),
// clearDeepSeekToken 封装对应的后端接口调用。
	clearDeepSeekToken: () => request('DELETE', '/settings/deepseek'),
// saveDeepSeekModel 封装对应的后端接口调用。
	saveDeepSeekModel: (model) => request('PUT', '/settings/deepseek/model', { model }),
// getAIHarnessStatus 封装对应的后端接口调用。
	getAIHarnessStatus: () => request('GET', '/ai/harness'),
// aiChat 封装对应的后端接口调用。
	aiChat: (data) => request('POST', '/ai/chat', data),
// getAIConversation 封装对应的后端接口调用。
	getAIConversation: () => request('GET', '/ai/conversation'),
// saveAIConversation 封装对应的后端接口调用。
	saveAIConversation: (conversations) => request('PUT', '/ai/conversation', { conversations }),
// clearAIConversation 封装对应的后端接口调用。
	clearAIConversation: () => request('DELETE', '/ai/conversation'),
// saveUsername 封装对应的后端接口调用。
	  saveUsername: (name) => request('PUT', '/settings/username', { name }),
// exportBackup 封装对应的后端接口调用。
  exportBackup: () => requestBackupExport(),
// importBackup 封装对应的后端接口调用。
  importBackup: (file) => requestBackupImport(file),
// getVersion 封装对应的后端接口调用。
  getVersion: () => request('GET', '/version'),
// checkUpdate 封装对应的后端接口调用。
  checkUpdate: (force = false) => request('GET', `/update/check?force=${force ? 'true' : 'false'}`),
// applyUpdate 封装对应的后端接口调用。
  applyUpdate: () => request('POST', '/update/apply'),
// getErrors 封装对应的后端接口调用。
	getErrors: (subject, keyword, tag, reason_tag) => {
    const p = new URLSearchParams()
    if (subject && subject !== '全部') p.set('subject', subject)
    if (keyword) p.set('keyword', keyword)
    if (tag) p.set('tag', tag)
    if (reason_tag) p.set('reason_tag', reason_tag)
    return request('GET', '/errors?' + p.toString())
	},
// getError 封装对应的后端接口调用。
	getError: (id) => request('GET', `/errors/${id}`),
  // getTags 发起并处理后端 API 请求。
  getTags: () => request('GET', '/tags'),
  // addError 发起并处理后端 API 请求。
  addError: (data) => request('POST', '/errors', data),
  // reviewError 发起并处理后端 API 请求。
	reviewError: (id, rating = 'good') => request('PUT', `/errors/${id}/review`, { rating }),
// deleteError 封装对应的后端接口调用。
  deleteError: (id) => request('DELETE', `/errors/${id}`),
// updateError 封装对应的后端接口调用。
  updateError: (id, data) => request('PUT', `/errors/${id}`, data),
// getDailyPush 封装对应的后端接口调用。
  getDailyPush: () => request('GET', '/daily-push'),
// getLearningActivity 封装对应的后端接口调用。
	getLearningActivity: (year = null) => request('GET', `/dashboard/activity${year ? `?year=${encodeURIComponent(year)}` : ''}`),
// getDailyPlan 封装对应的后端接口调用。
	getDailyPlan: () => request('GET', '/dashboard/plan'),
// saveDailyGoal 封装对应的后端接口调用。
	saveDailyGoal: (goal) => request('PUT', '/dashboard/plan', goal),
// recordFocusSession 封装对应的后端接口调用。
	recordFocusSession: (minutes, clientKey) => request('POST', '/dashboard/focus-sessions', { minutes, client_key: clientKey }),
// getWeeklyReport 封装对应的后端接口调用。
	getWeeklyReport: () => request('GET', '/dashboard/weekly-report'),
}
