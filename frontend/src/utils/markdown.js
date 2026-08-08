import markdownit from "markdown-it"
import mark from "markdown-it-mark"
import katex from "katex"
import hljs from "highlight.js/lib/common"

function highlightCode(code, language = "") {
  try {
    if (language && hljs.getLanguage(language)) return hljs.highlight(code, { language }).value
    return hljs.highlightAuto(code).value
  } catch {
    return escapeHtml(code)
  }
}

const md = markdownit({
  html: false,
  linkify: true,
  breaks: true,
  highlight: highlightCode,
}).use(mark)

const safeDataImageSource = /^data:image\/(?:png|jpe?g|gif|webp);base64,[A-Za-z0-9+/=]+$/i
const defaultImageRenderer = md.renderer.rules.image

function embeddedImageWidth(value) {
  const match = /(?:^|;)width=(\d{1,4})(?:;|$)/.exec(String(value || ""))
  const width = Number(match?.[1])
  return Number.isInteger(width) && width >= 120 && width <= 1200 ? width : 400
}

function embeddedImageAlignment(value) {
  return /(?:^|;)align=(left|center|right)(?:;|$)/.exec(String(value || ""))?.[1] || "left"
}

md.renderer.rules.image = (tokens, index, options, env, self) => {
  const token = tokens[index]
  const src = token.attrGet("src") || ""
  if (!safeDataImageSource.test(src)) return defaultImageRenderer ? defaultImageRenderer(tokens, index, options, env, self) : self.renderToken(tokens, index, options)

  const alt = token.content || "图片"
  const width = embeddedImageWidth(token.attrGet("title"))
  const alignment = embeddedImageAlignment(token.attrGet("title"))
  return `<img class="markdown-image--align-${alignment}" src="${escapeHtml(src)}" alt="${escapeHtml(alt)}" width="${width}">`
}

function headingID(text, index) {
  const slug = String(text || "")
    .toLowerCase()
    .replace(/<[^>]+>/g, "")
    .replace(/[^\p{L}\p{N}]+/gu, "-")
    .replace(/^-+|-+$/g, "")
  return `section-${index}-${slug || "heading"}`
}

md.core.ruler.push("heading_anchors", (state) => {
  let index = 0
  for (let i = 0; i < state.tokens.length; i++) {
    const token = state.tokens[i]
    if (token.type !== "heading_open") continue
    const text = state.tokens[i + 1]?.content || ""
    token.attrSet("id", headingID(text, index))
    token.attrSet("data-outline-index", String(index))
    index++
  }
})

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
    (tag, dataUrl) => {
      const alt = /\balt=["']([^"']*)["']/i.exec(tag)?.[1] || "图片"
      const widthMatch = /\bwidth=["']?(\d{1,4})/i.exec(tag)
      const width = embeddedImageWidth(widthMatch ? `width=${widthMatch[1]}` : "")
      return `![${alt}](${dataUrl} "width=${width}")`
    }
  )
}

function mathPlaceholder(index) {
  return `\uE000ST_MATH_${index}\uE000`
}

function alignmentPlaceholder(index) {
  return `\uE001ST_ALIGN_${index}\uE001`
}

function replaceAlignmentBlocks(src) {
  const blocks = []
  const result = src.replace(/^\[\[align:(left|center|right)\]\][ \t]*\r?\n([\s\S]*?)\r?\n\[\[\/align\]\][ \t]*$/gm, (_, alignment, content) => {
    const token = alignmentPlaceholder(blocks.length)
    blocks.push({ token, alignment, content })
    return token
  })
  return { result, blocks }
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
  const { result: alignedSource, blocks } = replaceAlignmentBlocks(src)
  const normalized = normalizeMathDelimiters(normalizeSafeDataImages(alignedSource))
  const { result, placeholders } = replaceMathWithPlaceholders(normalized)
  let html = md.render(result)
  for (const item of placeholders) {
    html = html.split(item.token).join(item.html)
  }
  for (const block of blocks) {
    const rendered = `<div class="markdown-align markdown-align--${block.alignment}">${renderMd(block.content)}</div>`
    html = html.replace(`<p>${block.token}</p>\n`, rendered)
  }
  html = removeLooseDollars(html)
  return html
}

export function extractOutline(src) {
  if (!src) return []
  const tokens = md.parse(src, {})
  const outline = []
  let index = 0
  for (let i = 0; i < tokens.length; i++) {
    if (tokens[i].type !== "heading_open") continue
    const inline = tokens[i + 1]
    const text = inline?.children?.map((token) => token.content).join("") || inline?.content || "未命名章节"
    outline.push({ index, level: Number(tokens[i].tag.slice(1)), text, id: headingID(text, index) })
    index++
  }
  return outline
}

export { katex }
