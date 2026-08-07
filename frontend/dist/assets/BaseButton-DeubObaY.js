import{c as u,d as a,e as s,g as i,I as n,n as o}from"./index-DidPj4PJ.js";/**
 * @license lucide-vue-next v0.468.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const y=u("UserRoundIcon",[["circle",{cx:"12",cy:"8",r:"5",key:"1hypcn"}],["path",{d:"M20 21a8 8 0 0 0-16 0",key:"rfgkzh"}]]),d=["type","disabled","aria-busy"],r={key:0,class:"ui-spinner","aria-hidden":"true"},b={__name:"BaseButton",props:{variant:{type:String,default:"default"},size:{type:String,default:"md"},busy:{type:Boolean,default:!1},type:{type:String,default:"button"}},setup(e){return(t,c)=>(a(),s("button",{type:e.type,class:o(["ui-button",[`ui-button--${e.variant}`,`ui-button--${e.size}`]]),disabled:e.busy||t.$attrs.disabled,"aria-busy":e.busy||void 0},[e.busy?(a(),s("span",r)):i("",!0),n(t.$slots,"icon"),n(t.$slots,"default")],10,d))}};export{y as U,b as _};
