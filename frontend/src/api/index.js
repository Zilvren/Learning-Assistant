const BASE = '/api'

async function request(method, path, body) {
  const opts = { method, headers: {} }
  if (body) {
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(body)
  }
  const res = await fetch(BASE + path, opts)
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

export const api = {
  getSubjects: () => request('GET', '/subjects'),
  addSubject: (name) => request('POST', '/subjects', { name }),
  deleteSubject: (name) => request('DELETE', '/subjects/' + encodeURIComponent(name)),
  ocrImage: async (file) => {
    const blob = file instanceof Blob ? file : new Blob([file])
    const resp = await fetch(BASE + '/ocr', { method: 'POST', body: blob })
    if (!resp.ok) throw new Error((await resp.json()).detail || 'OCR failed')
    return resp.json()
  },
  saveToken: (token) => request('PUT', '/settings/token', { token }),
  getToken: () => request('GET', '/settings/token'),
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
  getStats: () => request('GET', '/stats'),
  getDailyPush: () => request('GET', '/daily-push'),
}
