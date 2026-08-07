const STORAGE_KEY = "library:last-path"

export function libraryPath(folderId) {
  return folderId ? `/library/${encodeURIComponent(String(folderId))}` : "/library"
}

export function rememberLibraryPath(folderId) {
  try {
    localStorage.setItem(STORAGE_KEY, libraryPath(folderId))
  } catch {
    // 本地存储不可用时仍可正常使用资料库。
  }
}

export function rememberedLibraryPath() {
  try {
    const path = localStorage.getItem(STORAGE_KEY)
    return /^\/library(?:\/[^/?#]+)?$/.test(path || "") ? path : "/library"
  } catch {
    return "/library"
  }
}
