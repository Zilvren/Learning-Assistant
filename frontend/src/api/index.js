const BASE = '/api'

let refreshing = null

export class ApiError extends Error {
  constructor(message, status, detail = null) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.detail = detail
  }
}

async function responseError(res, fallback = '请求失败') {
  const payload = await res.json().catch(() => ({ detail: res.statusText }))
  const message = res.status === 503
    ? (payload.detail || '数据目录正在被其他操作占用，请稍后重试')
    : (payload.detail || fallback)
  return new ApiError(message, res.status, payload)
}

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

async function requestBackupExport() {
  const res = await fetch(BASE + '/backup/export', { credentials: 'include' })
  if (res.status === 401) {
    const ok = await refreshAuth()
    if (ok) return requestBackupExport()
  }
  if (!res.ok) {
    throw await responseError(res, '备份失败')
  }
  return res.blob()
}

async function requestBackupImport(file) {
  const res = await fetch(BASE + '/backup/import', { method: 'POST', body: file, credentials: 'include' })
  if (res.status === 401) {
    const ok = await refreshAuth()
    if (ok) return requestBackupImport(file)
  }
  if (!res.ok) {
    throw await responseError(res, '导入失败')
  }
  return res.json()
}

async function requestLibraryContent(id) {
  const res = await fetch(`${BASE}/library/items/${id}/content`, { credentials: 'include' })
  if (!res.ok) throw await responseError(res, '读取资料失败')
  return { content: await res.text(), version: Number((res.headers.get('etag') || '').replace(/\D/g, '') || 0), type: res.headers.get('content-type') || '' }
}

async function uploadLibraryFile(file, parentId) {
  const form = new FormData()
  form.append('file', file)
  if (parentId) form.append('parent_id', String(parentId))
  const res = await fetch(`${BASE}/library/uploads`, { method: 'POST', body: form, credentials: 'include' })
  if (!res.ok) throw await responseError(res, '上传失败')
  return res.json()
}

async function refreshAuth() {
  if (!refreshing) {
    refreshing = fetch(BASE + '/auth/refresh', { method: 'POST', credentials: 'include' })
      .then(res => res.ok)
      .catch(() => false)
      .finally(() => { refreshing = null })
  }
  return refreshing
}

export const api = {
	getLibraryItems: ({ parentId = null, kind = '', query = '', tag = '', review = false, due = false, trashed = false } = {}) => {
		const p = new URLSearchParams()
		if (parentId) p.set('parent_id', parentId)
		if (kind) p.set('kind', kind)
		if (query) p.set('q', query)
		if (tag) p.set('tag', tag)
		if (review) p.set('review', 'true')
		if (due) p.set('due', 'true')
		if (trashed) p.set('trashed', 'true')
		return request('GET', `/library/items?${p}`)
	},
	getLibraryItem: (id) => request('GET', `/library/items/${id}`),
	createLibraryItem: (data) => request('POST', '/library/items', data),
	updateLibraryItem: (id, data) => request('PATCH', `/library/items/${id}`, data),
	trashLibraryItem: (id) => request('DELETE', `/library/items/${id}`),
	restoreLibraryItem: (id) => request('POST', `/library/items/${id}/restore`),
	purgeLibraryItem: (id) => request('DELETE', `/library/items/${id}/purge`),
	duplicateLibraryItem: (id, parentId = null) => request('POST', `/library/items/${id}/duplicate`, { parent_id: parentId }),
	getLibraryContent: requestLibraryContent,
	saveLibraryContent: (id, data) => request('PUT', `/library/items/${id}/content`, data),
	uploadLibraryFile,
	batchLibraryItems: (action, ids, parentId = null) => request('POST', '/library/batch', { action, ids, parent_id: parentId }),
	getLibraryVersions: (id) => request('GET', `/library/items/${id}/versions`),
	restoreLibraryVersion: (id, versionId) => request('POST', `/library/items/${id}/versions/${versionId}/restore`),
	getLibraryTags: () => request('GET', '/library/tags'),
	getLibraryReviews: () => request('GET', '/library/reviews'),
	reviewLibraryNote: (id) => request('POST', `/library/items/${id}/review`),
  getSubjects: () => request('GET', '/subjects'),
  authStatus: () => request('GET', '/auth/status', null, false),
  me: () => request('GET', '/auth/me', null, false),
  login: (account, password) => request('POST', '/auth/login', { account, password }, false),
  register: (username, email, password) => request('POST', '/auth/register', { username, email, password }, false),
  verifyEmail: (token) => request('POST', '/auth/verify-email', { token }, false),
  resendVerification: (email) => request('POST', '/auth/resend-verification', { email }, false),
  refreshAuth: () => request('POST', '/auth/refresh', null, false),
  logout: () => request('POST', '/auth/logout', null, false),
  addSubject: (name) => request('POST', '/subjects', { name }),
  deleteSubject: (name) => request('DELETE', '/subjects/' + encodeURIComponent(name)),
  ocrImage: async (file) => {
    const blob = file instanceof Blob ? file : new Blob([file])
    const resp = await fetch(BASE + '/ocr', { method: 'POST', body: blob, credentials: 'include' })
    if (resp.status === 401) {
      const ok = await refreshAuth()
      if (ok) return api.ocrImage(file)
    }
    if (!resp.ok) throw await responseError(resp, 'OCR failed')
    return resp.json()
  },
  saveToken: (token) => request('PUT', '/settings/token', { token }),
  clearToken: () => request('DELETE', '/settings/token'),
  getToken: () => request('GET', '/settings/token'),
  saveUsername: (name) => request('PUT', '/settings/username', { name }),
  exportBackup: () => requestBackupExport(),
  importBackup: (file) => requestBackupImport(file),
  getVersion: () => request('GET', '/version'),
  checkUpdate: (force = false) => request('GET', `/update/check?force=${force ? 'true' : 'false'}`),
  applyUpdate: () => request('POST', '/update/apply'),
  getErrors: (subject, keyword, tag, reason_tag) => {
    const p = new URLSearchParams()
    if (subject && subject !== '全部') p.set('subject', subject)
    if (keyword) p.set('keyword', keyword)
    if (tag) p.set('tag', tag)
    if (reason_tag) p.set('reason_tag', reason_tag)
    return request('GET', '/errors?' + p.toString())
  },
  getTags: () => request('GET', '/tags'),
  addError: (data) => request('POST', '/errors', data),
  reviewError: (id) => request('PUT', `/errors/${id}/review`),
  deleteError: (id) => request('DELETE', `/errors/${id}`),
  updateError: (id, data) => request('PUT', `/errors/${id}`, data),
  getDailyPush: () => request('GET', '/daily-push'),
  getLearningActivity: () => request('GET', '/dashboard/activity'),
}
