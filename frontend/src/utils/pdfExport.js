import { renderMd } from "./markdown.js"

const colors = {}
const colorPool = ['#0EA5E9','#8B5CF6','#10B981','#F97316','#EC4899','#F59E0B','#6366F1','#14B8A6','#F43F5E','#EAB308']
function subjectColor(name){if(colors[name])return colors[name];let h=0;for(let i=0;i<name.length;i++)h=((h<<5)-h+name.charCodeAt(i))|0;return colorPool[Math.abs(h)%colorPool.length]}

export async function exportPdfReport(errors, stats){
  const iframe = document.createElement("iframe")
  Object.assign(iframe.style,{
    position:"fixed",top:"0",left:"0",width:"0",height:"0",
    border:"none",opacity:"0",pointerEvents:"none",zIndex:"-9999",
  })
  document.body.appendChild(iframe)

  const doc = iframe.contentDocument || iframe.contentWindow.document
  doc.open()
  doc.write(buildHtml(errors))
  doc.close()
  doc.title = `错题本_${new Date().toISOString().slice(0,10)}`

  await new Promise(r => setTimeout(r, 3000))
  try {
    iframe.contentWindow.focus()
    iframe.contentWindow.print()
  } finally {
    setTimeout(() => iframe.remove(), 5000)
  }
}

function tagChips(tags) {
  if (!tags?.length) return ''
  return `<div style="display:flex;gap:3px;flex-wrap:wrap;margin-top:2px">${tags.map(t=>`<span style="display:inline-block;padding:1px 5px;border-radius:3px;background:#f1f5f9;color:#64748b;font-size:8px;border:1px solid #e2e8f0">${t}</span>`).join('')}</div>`
}

function buildHtml(errors){
  const now = new Date()
  const date = now.toISOString().slice(0,10)
  const total = errors.length

  let html = ""
  for(const s of [...new Set(errors.map(e=>e.subject))]){
    const se = errors.filter(e=>e.subject===s)
    if(!se.length) continue

    html += `<h2 class="subject" style="color:${subjectColor(s)};font-size:13px;font-weight:600;margin:18px 0 6px;padding-bottom:3px;border-bottom:1.5px solid currentColor;opacity:.8;page-break-after:avoid">${s}（${se.length}）</h2>`

    for(const e of se){
      html += `<div class="card" style="margin-bottom:8px;padding:0 0 6px 0;border-bottom:1px solid #f1f5f9;page-break-inside:avoid">
        <span style="font-size:11px;font-weight:600;color:#0f172a;display:block;margin-bottom:3px">#${e.id} ${e.title||''}</span>`
      // Question
      html += `<div style="margin-bottom:3px"><span class="label">题目</span><div class="md">${renderMd(e.question)}</div></div>`
      // Wrong / Correct
      const w = e.wrong && e.wrong !== '未记录'
      const c = e.correct && e.correct !== '未记录'
      if(w||c){
        html += '<div style="margin-bottom:3px;display:flex;gap:6px">'
        if(w) html += `<div class="wrong"><span class="label">错答</span><div class="md">${renderMd(e.wrong)}</div></div>`
        if(c) html += `<div class="correct"><span class="label">正解</span><div class="md">${renderMd(e.correct)}</div></div>`
        html += '</div>'
      }
      // Reason
      if(e.reason&&e.reason!=='未记录'){
        html += `<div style="margin-bottom:2px"><span class="label">错因</span><div class="md">${renderMd(e.reason)}</div></div>`
      }
      // Tags
      html += tagChips(e.tags)
      if (e.reason_tags?.length) {
        html += `<span style="font-size:7px;color:#94a3b8;margin-right:4px">错因:</span>` + tagChips(e.reason_tags)
      }
      html += `</div>`
    }
  }

  return `<!DOCTYPE html><html lang="zh-CN"><head>
<meta charset="utf-8">
<title>错题本 ${date}</title>
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
  .subject{page-break-after:avoid}
  .card{page-break-inside:avoid}
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
  .wrong{flex:1;padding:6px 8px;border-radius:6px;background:rgba(239,68,68,.06);border:1px solid rgba(239,68,68,.2);border-left:3px solid #ef4444}
  .correct{flex:1;padding:6px 8px;border-radius:6px;background:rgba(16,185,129,.04);border:1px solid rgba(16,185,129,.15);border-left:3px solid #10b981}
  .wrong .label{color:#ef4444}.correct .label{color:#16a34a}
  .katex{font-size:1.4em}.katex-display{font-size:1.55em;margin:4px 0;overflow-x:auto}
  .wrong .katex,.correct .katex{font-size:1.45em}
  .wrong .katex-display,.correct .katex-display{font-size:1.6em}
</style></head><body>

<div class="header">
  <h1>错题本</h1>
  <p>${date}</p>
  <div class="count">共 ${total} 道错题</div>
</div>

${html}

</body></html>`
}
