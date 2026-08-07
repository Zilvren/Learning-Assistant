import { afterEach, vi } from "vitest"

if (!window.requestAnimationFrame) {
  window.requestAnimationFrame = (callback) => window.setTimeout(callback, 0)
}

if (!window.matchMedia) {
  window.matchMedia = vi.fn().mockImplementation((query) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

if (!Element.prototype.scrollIntoView) Element.prototype.scrollIntoView = vi.fn()

afterEach(() => {
  document.body.innerHTML = ""
  localStorage.clear()
  vi.restoreAllMocks()
})
