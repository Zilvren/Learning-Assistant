import markdownit from "markdown-it"
import mark from "markdown-it-mark"
import katex from "katex"
import hljs from "highlight.js/lib/common"

// highlightCode 优先按指定语言高亮，语言未知或解析失败时回退到安全转义文本。
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

// embeddedImageWidth 从图片 title 元数据取宽度，并把范围限制在安全的展示尺寸内。
function embeddedImageWidth(value) {
  const match = /(?:^|;)width=(\d{1,4})(?:;|$)/.exec(String(value || ""))
  const width = Number(match?.[1])
  return Number.isInteger(width) && width >= 120 && width <= 1200 ? width : 400
}

// embeddedImageAlignment 从图片 title 元数据读取左、中、右对齐方式。
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

// headingID 为标题生成稳定、可用于目录跳转的 HTML id。
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

// renderKatex 将公式渲染为 HTML；公式异常时仍返回可读的错误占位内容。
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

// escapeHtml 转义用户文本，避免 Markdown 渲染中的 HTML 注入。
function escapeHtml(value = "") {
  return String(value).replace(/[&<>"']/g, ch => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#39;",
  }[ch]))
}

// normalizeMathDelimiters 自动补齐未成对的数学分隔符，避免后续占位替换失配。
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

// removeLooseDollars 清理正文中不构成公式的孤立美元符号。
function removeLooseDollars(html) {
  return html
    .replace(/(^|[\s>])\$([\s<.,，。；;:：!?！？)]|$)/g, "$1$2")
    .replace(/([\s(（])\$([\s<]|$)/g, "$1$2")
}

// normalizeSafeDataImages 将允许的内嵌 Base64 图片规范成受控的 Markdown 图片语法。
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

// mathPlaceholder 创建不会与用户正文冲突的公式临时标记。
function mathPlaceholder(index) {
  return `\uE000ST_MATH_${index}\uE000`
}

// alignmentPlaceholder 创建图片/内容对齐块的临时标记。
function alignmentPlaceholder(index) {
  return `\uE001ST_ALIGN_${index}\uE001`
}

// replaceAlignmentBlocks 先抽离自定义对齐块，避免 Markdown 解析打散其边界。
function replaceAlignmentBlocks(src) {
  const blocks = []
  const result = src.replace(/^\[\[align:(left|center|right)\]\][ \t]*\r?\n([\s\S]*?)\r?\n\[\[\/align\]\][ \t]*$/gm, (_, alignment, content) => {
    const token = alignmentPlaceholder(blocks.length)
    blocks.push({ token, alignment, content })
    return token
  })
  return { result, blocks }
}

// replaceMathWithPlaceholders 在 Markdown 渲染前保护公式 HTML，防止被再次转义。
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

// renderMd 是笔记预览入口：依次处理对齐、图片、公式、Markdown 与安全清理。
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

// extractOutline 从 Markdown token 中提取标题层级，供右侧目录或锚点跳转使用。
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
