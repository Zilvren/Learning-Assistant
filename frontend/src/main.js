import { createApp } from "vue"
import App from "./App.vue"
import { router } from "./router/index.js"
import "./style.css"
import "./styles/compact-error-cards.css"
import "./styles/library.css"

// 先挂载可恢复的启动界面；认证初始化由路由守卫触发，失败时仍可重试。
createApp(App)
  .use(router)
  .mount("#app")
