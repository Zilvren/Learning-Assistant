const BASE = '/api'

let refreshing = null

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
    const err = await res.json().catch(() => ({ detail: res.statusText }))
    throw new Error(err.detail || '请求失败')
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
    const err = await res.json().catch(() => ({ detail: res.statusText }))
    throw new Error(err.detail || '备份失败')
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
    const err = await res.json().catch(() => ({ detail: res.statusText }))
    throw new Error(err.detail || '导入失败')
  }
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
  getSubjects: () => request('GET', '/subjects'),
  authStatus: () => request('GET', '/auth/status', null, false),
  me: () => request('GET', '/auth/me', null, false),
  login: (account, password) => request('POST', '/auth/login', { account, password }, false),
  register: (username, email, password) => request('POST', '/auth/register', { username, email, password }, false),
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
    if (!resp.ok) throw new Error((await resp.json()).detail || 'OCR failed')
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
}
