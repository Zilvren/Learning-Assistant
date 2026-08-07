import{c as g,B as p,q as u}from"./index-DidPj4PJ.js";/**
 * @license lucide-vue-next v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const S=g("Clock3Icon",[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["polyline",{points:"12 6 12 12 16.5 12",key:"1aq6pp"}]]);/**
 * @license lucide-vue-next v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const q=g("LightbulbIcon",[["path",{d:"M15 14c.2-1 .7-1.7 1.5-2.5 1-.9 1.5-2.2 1.5-3.5A6 6 0 0 0 6 8c0 1 .2 2.2 1.5 3.5.7.7 1.3 1.5 1.5 2.5",key:"1gvzjb"}],["path",{d:"M9 18h6",key:"x1upvd"}],["path",{d:"M10 22h4",key:"ceow96"}]]);/**
 * @license lucide-vue-next v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const x=g("LoaderCircleIcon",[["path",{d:"M21 12a9 9 0 1 1-6.219-8.56",key:"13zald"}]]);/**
 * @license lucide-vue-next v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const E=g("TagIcon",[["path",{d:"M12.586 2.586A2 2 0 0 0 11.172 2H4a2 2 0 0 0-2 2v7.172a2 2 0 0 0 .586 1.414l8.704 8.704a2.426 2.426 0 0 0 3.42 0l6.58-6.58a2.426 2.426 0 0 0 0-3.42z",key:"vktsd0"}],["circle",{cx:"7.5",cy:"7.5",r:".5",fill:"currentColor",key:"kqv944"}]]);function L(r){const e=(r||"").trim();return!!e&&e!=="未记录"}function C(r,e=new Date().toISOString().slice(0,10)){var l;const n=r.next_review||((l=r.created)==null?void 0:l.slice(0,10)),t=(r.review_count||0)+1;return n?n<e?`逾期 ${n} · 第 ${t} 轮`:n===e?`今日到期 · 第 ${t} 轮`:`下次 ${n} · 第 ${t} 轮`:`第 ${t} 轮复习`}function I(r,e=new Date().toISOString().slice(0,10)){var n;return(r.next_review||((n=r.created)==null?void 0:n.slice(0,10))||e)<=e}function k({subject:r="全部",keyword:e="",mode:n="全部"}={}){const t=String(e??"").trim()||null;return[r==="全部"?null:r,n==="题目"||n==="全部"?t:null,n==="题目标签"?t:null,n==="错因标签"?t:null]}function d(r,e){return r instanceof Error?r:r&&typeof r.message=="string"?new Error(r.message):new Error(typeof r=="string"&&r?r:e)}function A(r){return Array.isArray(r)?r.map(e=>!e||typeof e!="object"?e:{...e,tags:Array.isArray(e.tags)?e.tags:[],reason_tags:Array.isArray(e.reason_tags)?e.reason_tags:[]}):[]}function _(){const r=u([]),e=u([]),n=u(!1),t=u(!1),l=u(null),y=u(!1),v=u(null);let c=0,o=0,a=!1;async function b(i={}){if(a)return null;const s=++c;n.value=!0,l.value=null;try{const f=await p.getErrors(...k(i));return a||s!==c?null:(r.value=A(f==null?void 0:f.errors),t.value=!0,r.value)}catch(f){return a||s!==c||(l.value=d(f,"错题加载失败"),t.value=!0),null}finally{!a&&s===c&&(n.value=!1)}}async function h(){if(a)return null;const i=++o;y.value=!0,v.value=null;try{const s=await p.getSubjects();return a||i!==o?null:(e.value=Array.isArray(s==null?void 0:s.subjects)?s.subjects:[],e.value)}catch(s){return a||i!==o||(v.value=d(s,"科目加载失败")),null}finally{!a&&i===o&&(y.value=!1)}}function w(){a=!0,c+=1,o+=1,n.value=!1,y.value=!1}return{errors:r,subjects:e,loading:n,loaded:t,error:l,subjectsLoading:y,subjectsError:v,refresh:b,loadSubjects:h,dispose:w}}export{S as C,q as L,E as T,x as a,L as h,I as i,C as r,_ as u};
