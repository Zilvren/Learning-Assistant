import { renderMd } from "./markdown.js"

const colors = {}
const colorPool = ['#0EA5E9','#8B5CF6','#10B981','#F97316','#EC4899','#F59E0B','#6366F1','#14B8A6','#F43F5E','#EAB308']
// subjectColor 完成纯前端数据处理。
function subjectColor(name){if(colors[name])return colors[name];let h=0;for(let i=0;i<name.length;i++)h=((h<<5)-h+name.charCodeAt(i))|0;return colorPool[Math.abs(h)%colorPool.length]}

const exportStyles = {
  detailed: { label: "详细复盘", title: "错题本", pageClass: "style-detailed" },
  compact: { label: "紧凑打印", title: "错题速览", pageClass: "style-compact" },
  practice: { label: "练习自测", title: "错题自测卷", pageClass: "style-practice" },
}

// exportPdfReport 完成纯前端数据处理。
export async function exportPdfReport(errors, options = {}){
  const styleId = typeof options === "string" ? options : options.style || "detailed"
  const style = exportStyles[styleId] || exportStyles.detailed
  const iframe = document.createElement("iframe")
  Object.assign(iframe.style,{
    position:"fixed",top:"0",left:"0",width:"0",height:"0",
    border:"none",opacity:"0",pointerEvents:"none",zIndex:"-9999",
  })
  document.body.appendChild(iframe)

  const doc = iframe.contentDocument || iframe.contentWindow.document
  doc.open()
  doc.write(buildHtml(errors, styleId))
  doc.close()
  doc.title = `${style.title}_${new Date().toISOString().slice(0,10)}`

  await new Promise(r => setTimeout(r, 3000))
  try {
    iframe.contentWindow.focus()
    iframe.contentWindow.print()
  } finally {
    setTimeout(() => iframe.remove(), 5000)
  }
}

// tagChips 完成纯前端数据处理。
function tagChips(tags) {
  if (!tags?.length) return ''
  return `<div class="tags">${tags.map(t=>`<span>${escapeHtml(t)}</span>`).join('')}</div>`
}

// escapeHtml 完成纯前端数据处理。
function escapeHtml(value = "") {
  return String(value).replace(/[&<>"']/g, ch => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#39;",
  }[ch]))
}

// hasText 完成纯前端数据处理。
function hasText(value) {
  const text = (value || "").trim()
  return text && text !== "未记录"
}

// groupedBySubject 完成纯前端数据处理。
function groupedBySubject(errors) {
  return [...new Set(errors.map(e => e.subject))].map(subject => ({
    subject,
    items: errors.filter(e => e.subject === subject),
  })).filter(group => group.items.length)
}

// errorTitle 完成纯前端数据处理。
function errorTitle(e) {
  return escapeHtml(e.title || `未命名错题 #${e.id}`)
}

// answerHtml 完成纯前端数据处理。
function answerHtml(e) {
  const chunks = []
  if (hasText(e.wrong)) chunks.push(`<div class="wrong"><span class="label">错答</span><div class="md">${renderMd(e.wrong)}</div></div>`)
  if (hasText(e.correct)) chunks.push(`<div class="correct"><span class="label">正解</span><div class="md">${renderMd(e.correct)}</div></div>`)
  if (hasText(e.reason)) chunks.push(`<div class="reason"><span class="label">错因</span><div class="md">${renderMd(e.reason)}</div></div>`)
  return chunks.join("")
}

// renderDetailed 完成纯前端数据处理。
function renderDetailed(errors) {
  let html = ""
  for (const { subject, items } of groupedBySubject(errors)) {
    html += `<h2 class="subject" style="color:${subjectColor(subject)}">${escapeHtml(subject)}（${items.length}）</h2>`
    for (const e of items) {
      html += `<article class="card">
        <span class="card-title">#${e.id} ${errorTitle(e)}</span>
        <div class="block"><span class="label">题目</span><div class="md">${renderMd(e.question)}</div></div>`
      const w = hasText(e.wrong)
      const c = hasText(e.correct)
      if (w || c) {
        html += '<div class="answer-grid">'
        if (w) html += `<div class="wrong"><span class="label">错答</span><div class="md">${renderMd(e.wrong)}</div></div>`
        if (c) html += `<div class="correct"><span class="label">正解</span><div class="md">${renderMd(e.correct)}</div></div>`
        html += '</div>'
      }
      if (hasText(e.reason)) {
        html += `<div class="block"><span class="label">错因</span><div class="md">${renderMd(e.reason)}</div></div>`
      }
      html += tagChips(e.tags)
      if (e.reason_tags?.length) html += `<div class="reason-tag-line"><span>错因</span>${tagChips(e.reason_tags)}</div>`
      html += `</article>`
    }
  }
  return html
}

// renderCompact 完成纯前端数据处理。
function renderCompact(errors) {
  let html = ""
  for (const { subject, items } of groupedBySubject(errors)) {
    html += `<h2 class="subject" style="color:${subjectColor(subject)}">${escapeHtml(subject)}（${items.length}）</h2>`
    for (const e of items) {
      html += `<article class="card compact-card">
        <div class="compact-head"><strong>#${e.id}</strong><span>${errorTitle(e)}</span></div>
        <div class="md">${renderMd(e.question)}</div>`
      if (hasText(e.correct) || hasText(e.reason)) {
        html += `<div class="compact-answer">
          ${hasText(e.correct) ? `<div><span class="label">正解</span><div class="md">${renderMd(e.correct)}</div></div>` : ""}
          ${hasText(e.reason) ? `<div><span class="label">错因</span><div class="md">${renderMd(e.reason)}</div></div>` : ""}
        </div>`
      }
      html += tagChips(e.tags)
      html += `</article>`
    }
  }
  return html
}

// renderPractice 完成纯前端数据处理。
function renderPractice(errors) {
  let questions = ""
  let answers = ""
  for (const { subject, items } of groupedBySubject(errors)) {
    questions += `<h2 class="subject" style="color:${subjectColor(subject)}">${escapeHtml(subject)}（${items.length}）</h2>`
    answers += `<h2 class="subject" style="color:${subjectColor(subject)}">${escapeHtml(subject)}（${items.length}）</h2>`
    for (const e of items) {
      questions += `<article class="card practice-question">
        <span class="card-title">#${e.id} ${errorTitle(e)}</span>
        <div class="md">${renderMd(e.question)}</div>
        ${tagChips(e.tags)}
        <div class="answer-space"></div>
      </article>`
      answers += `<article class="card answer-key">
        <span class="card-title">#${e.id} ${errorTitle(e)}</span>
        ${answerHtml(e) || '<div class="muted">未记录答案或解析</div>'}
      </article>`
    }
  }
  return `${questions}<section class="answer-section"><h1>答案与解析</h1>${answers}</section>`
}

// buildBody 完成纯前端数据处理。
function buildBody(errors, styleId) {
  if (styleId === "compact") return renderCompact(errors)
  if (styleId === "practice") return renderPractice(errors)
  return renderDetailed(errors)
}

// buildHtml 完成纯前端数据处理。
function buildHtml(errors, styleId = "detailed"){
  const now = new Date()
  const date = now.toISOString().slice(0,10)
  const total = errors.length
  const style = exportStyles[styleId] || exportStyles.detailed
  const bodyHtml = buildBody(errors, styleId)

  return `<!DOCTYPE html><html lang="zh-CN"><head>
<meta charset="utf-8">
<title>${style.title} ${date}</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/katex.min.css">
<link href="https://fonts.googleapis.com/css2?family=Noto+Sans+SC:wght@400;500;600;700&display=swap" rel="stylesheet">
<style>
  @page{size:A4;margin:1.2cm}
  *{margin:0;padding:0;box-sizing:border-box}
  body{font-family:"Noto Sans SC",-apple-system,sans-serif;color:#1e293b;font-size:11px;line-height:1.5;max-width:100%;margin:0 auto}
  .header{text-align:center;padding:12px 0 10px;border-bottom:1.5px solid #f1f5f9;margin-bottom:14px}
  .header h1{font-size:15px;font-weight:700;color:#0f172a;margin-bottom:1px;letter-spacing:.5px}
  .header p{font-size:9px;color:#94a3b8}
  .header .count{font-size:9px;color:#6366f1;margin-top:2px}
  .subject{font-size:13px;font-weight:600;margin:18px 0 6px;padding-bottom:3px;border-bottom:1.5px solid currentColor;opacity:.82;page-break-after:avoid}
  .card{page-break-inside:avoid;margin-bottom:8px;padding:0 0 7px;border-bottom:1px solid #f1f5f9}
  .card-title{font-size:11px;font-weight:600;color:#0f172a;display:block;margin-bottom:3px}
  .block{margin-bottom:3px}
  .label{font-size:7.5px;font-weight:600;color:#64748b;display:inline-block;margin-right:4px;text-transform:uppercase;letter-spacing:.3px}
  .md{font-size:11.5px;line-height:1.55;color:#1e293b;word-wrap:break-word}
  .md p{margin:2px 0}
  .md h1,.md h2,.md h3,.md h4{margin-top:3px;margin-bottom:2px;font-weight:600;font-size:1.05em}
  .md ul,.md ol{padding-left:16px;margin:2px 0}
  .md code{padding:1px 4px;border-radius:3px;background:#f1f5f9;font-size:.9em;font-family:"Consolas",monospace}
  .md pre{padding:6px 8px;border-radius:4px;background:#0f172a;color:#e2e8f0;font-size:9px;overflow-x:auto;margin:4px 0}
  .md pre code{background:none;padding:0;color:inherit}
  .md mark{background:#fef08a;color:#1e293b;padding:1px 3px;border-radius:2px;font-weight:600;-webkit-print-color-adjust:exact!important;print-color-adjust:exact!important}
  .md blockquote{border-left:3px solid #6366f1;padding:2px 8px;margin:3px 0;color:#64748b;background:#f8fafc;border-radius:0 3px 3px 0;font-size:.95em}
  .md img{max-width:100%;max-height:260px;border-radius:4px;margin:4px 0;object-fit:contain}
  .answer-grid{margin-bottom:3px;display:flex;gap:6px}
  .wrong{flex:1;padding:6px 8px;border-radius:6px;background:rgba(239,68,68,.06);border:1px solid rgba(239,68,68,.2);border-left:3px solid #ef4444}
  .correct{flex:1;padding:6px 8px;border-radius:6px;background:rgba(16,185,129,.04);border:1px solid rgba(16,185,129,.15);border-left:3px solid #10b981}
  .reason{padding:5px 8px;border-radius:6px;background:#f8fafc;border:1px solid #e2e8f0}
  .wrong .label{color:#ef4444}.correct .label{color:#16a34a}
  .tags{display:flex;gap:3px;flex-wrap:wrap;margin-top:2px}
  .tags span{display:inline-block;padding:1px 5px;border-radius:3px;background:#f1f5f9;color:#64748b;font-size:8px;border:1px solid #e2e8f0}
  .reason-tag-line{display:flex;align-items:flex-start;gap:4px;margin-top:2px}
  .reason-tag-line>span{font-size:7px;color:#94a3b8;margin-top:2px}
  .muted{font-size:10px;color:#94a3b8}
  .style-compact .header{margin-bottom:8px}
  .style-compact .subject{margin:10px 0 4px;font-size:11px}
  .style-compact .card{margin-bottom:4px;padding-bottom:4px}
  .style-compact .md{font-size:9.6px;line-height:1.42}
  .style-compact .card-title,.style-compact .compact-head{font-size:9.5px}
  .compact-head{display:flex;gap:5px;align-items:baseline;color:#0f172a;margin-bottom:2px}
  .compact-head strong{color:#6366f1}
  .compact-answer{display:grid;grid-template-columns:1fr 1fr;gap:5px;margin-top:3px}
  .style-compact .md img{max-height:150px}
  .style-practice .card{padding-bottom:10px;margin-bottom:10px}
  .style-practice .md{font-size:12px}
  .answer-space{height:62px;border:1px dashed #cbd5e1;border-radius:6px;margin-top:8px;background:linear-gradient(#fff,#fff) padding-box,repeating-linear-gradient(to bottom,transparent 0,transparent 18px,#e2e8f0 19px) border-box}
  .answer-section{page-break-before:always}
  .answer-section>h1{font-size:16px;text-align:center;margin:0 0 14px;color:#0f172a}
  .katex{font-size:1.4em}.katex-display{font-size:1.55em;margin:4px 0;overflow-x:auto}
  .wrong .katex,.correct .katex{font-size:1.45em}
  .wrong .katex-display,.correct .katex-display{font-size:1.6em}
</style></head><body>

<div class="${style.pageClass}">
  <div class="header">
  <h1>${style.title}</h1>
  <p>${date}</p>
  <div class="count">${style.label} · 共 ${total} 道错题</div>
  </div>

  ${bodyHtml}
</div>

</body></html>`
}
