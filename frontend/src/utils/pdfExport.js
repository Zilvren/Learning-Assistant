import { renderMd } from "./markdown.js"

const colors = {}
const colorPool = ['#0EA5E9','#8B5CF6','#10B981','#F97316','#EC4899','#F59E0B','#6366F1','#14B8A6','#F43F5E','#EAB308']
function subjectColor(name){if(colors[name])return colors[name];let h=0;for(let i=0;i<name.length;i++)h=((h<<5)-h+name.charCodeAt(i))|0;return colorPool[Math.abs(h)%colorPool.length]}

function collectCss() {
  // Collect all stylesheet rules from the page so PDF matches web display
  let css = ''
  for (const sheet of document.styleSheets) {
    try {
      for (const rule of sheet.cssRules) {
        css += rule.cssText + '\n'
      }
    } catch(e) { /* CORS sheets throw */ }
  }
  return css
}

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

function buildHtml(errors){
  const now = new Date()
  const date = now.toISOString().slice(0,10)
  const total = errors.length
  const pageCss = collectCss()

  let html = ""
  for(const s of [...new Set(errors.map(e=>e.subject))]){
    const se = errors.filter(e=>e.subject===s)
    if(!se.length) continue

    html += `<h2 class="subject" style="color:${subjectColor(s)};font-size:14px;font-weight:600;margin:20px 0 8px;padding-bottom:4px;border-bottom:1.5px solid currentColor;opacity:.8">${s}（${se.length}）</h2>`

    for(const e of se){
      html += `<div class="card" style="margin-bottom:10px;page-break-inside:avoid">
        <span style="font-size:12px;font-weight:600;color:#0f172a;display:block;margin-bottom:5px">#${e.id}</span>`
      // Question
      html += `<div style="margin-bottom:5px"><span style="font-size:9px;font-weight:600;color:#64748b">题目</span><div class="markdown-body">${renderMd(e.question)}</div></div>`
      // Wrong / Correct
      const w = e.wrong && e.wrong !== '未记录'
      const c = e.correct && e.correct !== '未记录'
      if(w||c){
        if(w) html += `<div class="wrong-card" style="padding:10px 12px;border-radius:8px;margin-bottom:5px;background:rgba(239,68,68,.08);border:1px solid rgba(239,68,68,.25);border-left:4px solid #ef4444"><span style="font-size:9px;font-weight:600;color:#ef4444;display:block;margin-bottom:2px">错答</span><div class="markdown-body">${renderMd(e.wrong)}</div></div>`
        if(c) html += `<div class="correct-card" style="padding:10px 12px;border-radius:8px;margin-bottom:6px;background:rgba(16,185,129,.06);border:1px solid rgba(16,185,129,.2);border-left:4px solid #10b981"><span style="font-size:9px;font-weight:600;color:#16a34a;display:block;margin-bottom:2px">正解</span><div class="markdown-body">${renderMd(e.correct)}</div></div>`
      }
      // Reason
      if(e.reason&&e.reason!=='未记录'){
        html += `<div style="margin-bottom:5px"><span style="font-size:9px;font-weight:600;color:#64748b">错因</span><div class="markdown-body">${renderMd(e.reason)}</div></div>`
      }
      // Tags
      if(e.tags?.length){
        html += `<div style="margin-top:3px;display:flex;gap:4px;flex-wrap:wrap">${e.tags.map(t=>`<span style="display:inline-block;padding:1px 6px;border-radius:3px;background:#f8fafc;color:#64748b;font-size:9px;border:1px solid #f1f5f9">${t}</span>`).join('')}</div>`
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
  @page{size:A4;margin:1.5cm}
  body{font-family:"Noto Sans SC",-apple-system,sans-serif;color:#1e293b;max-width:100%;margin:0 auto}
  /* Scale entire page to 70% so web-sized content fits A4 */
  .pdf-root{zoom:0.7}
  .header{text-align:center;padding:20px 0 16px;border-bottom:1.5px solid #f1f5f9;margin-bottom:20px}
  .header h1{font-size:17px;font-weight:700;color:#0f172a;margin-bottom:2px}
  .header p{font-size:10px;color:#94a3b8}
  .header .count{font-size:10px;color:#6366f1;margin-top:4px}
  ${pageCss}
</style></head><body>
<div class="pdf-root">
<div class="header"><h1>错题本</h1><p>${date}</p><div class="count">共 ${total} 道错题</div></div>
${html}
</div>
</body></html>`
}
