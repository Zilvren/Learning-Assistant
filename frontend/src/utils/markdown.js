import markdownit from "markdown-it"
import mark from "markdown-it-mark"
import katex from "katex"

const md = markdownit({
  html: true,
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
    return isBlock ? `<div class="katex-error">${text}</div>` : `<span class="katex-error">${text}</span>`
  }
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

export function renderMd(src) {
  if (!src) return ""
  const normalized = normalizeMathDelimiters(src)
  // Process block math first
  let result = normalized.replace(blockRegex, (_, code) => renderKatex(code.trim(), true))
  // Then process inline math
  result = result.replace(inlineRegex, (_, code) => renderKatex(code.trim(), false))
  // Finally render markdown
  let html = md.render(result)
  html = removeLooseDollars(html)
  // Add default width to base64 data URI images that don't already have width/height
  html = html.replace(/<img\s+src="(data:image\/[^"]+)"(?![^>]*\bwidth\b)(?![^>]*\bheight\b)([^>]*)>/g,
    '<img src="$1" width="400"$2>')
  return html
}

export { katex }
