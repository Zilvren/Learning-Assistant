import markdownit from "markdown-it"
import mark from "markdown-it-mark"
import katex from "katex"

const md = markdownit({
  html: false,
  linkify: true,
  breaks: true,
}).use(mark)

// Inline math: $...$
const inlineRegex = /\$(.+?)\$/g
// Block math: $$...$$
const blockRegex = /\$\$(.+?)\$\$/gs

function renderKatex(text, isBlock) {
  try {
    return katex.renderToString(text, {
      throwOnError: false,
      displayMode: isBlock,
      output: "html",
    })
  } catch {
    const escaped = escapeHtml(text)
    return isBlock ? `<div class="katex-error">${escaped}</div>` : `<span class="katex-error">${escaped}</span>`
  }
}

function escapeHtml(value = "") {
  return String(value).replace(/[&<>"']/g, ch => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#39;",
  }[ch]))
}

function normalizeMathDelimiters(src) {
  const blockMatches = src.match(/\$\$/g) || []
  const singleDollarCount = (src.match(/(?<!\$)\$(?!\$)/g) || []).length
  let normalized = src

  if (blockMatches.length % 2 !== 0) {
    normalized += "$$"
  }

  if (singleDollarCount % 2 !== 0) {
    normalized += "$"
  }

  return normalized
}

function removeLooseDollars(html) {
  return html
    .replace(/(^|[\s>])\$([\s<.,，。；;:：!?！？)]|$)/g, "$1$2")
    .replace(/([\s(（])\$([\s<]|$)/g, "$1$2")
}

function normalizeSafeDataImages(src) {
  return src.replace(
    /<img\s+[^>]*src=["'](data:image\/(?:png|jpe?g|gif|webp);base64,[A-Za-z0-9+/=]+)["'][^>]*>/gi,
    "![image]($1)"
  )
}

function mathPlaceholder(index) {
  return `\uE000ST_MATH_${index}\uE000`
}

function replaceMathWithPlaceholders(src) {
  const placeholders = []
  let result = src.replace(blockRegex, (_, code) => {
    const token = mathPlaceholder(placeholders.length)
    placeholders.push({ token, html: renderKatex(code.trim(), true) })
    return token
  })

  result = result.replace(inlineRegex, (_, code) => {
    const token = mathPlaceholder(placeholders.length)
    placeholders.push({ token, html: renderKatex(code.trim(), false) })
    return token
  })

  return { result, placeholders }
}

export function renderMd(src) {
  if (!src) return ""
  const normalized = normalizeMathDelimiters(normalizeSafeDataImages(src))
  const { result, placeholders } = replaceMathWithPlaceholders(normalized)
  let html = md.render(result)
  for (const item of placeholders) {
    html = html.split(item.token).join(item.html)
  }
  html = removeLooseDollars(html)
  // Add default width to base64 data URI images that don't already have width/height
  html = html.replace(/<img\s+src="(data:image\/[^"]+)"(?![^>]*\bwidth\b)(?![^>]*\bheight\b)([^>]*)>/g,
    '<img src="$1" width="400"$2>')
  return html
}

export { katex }
