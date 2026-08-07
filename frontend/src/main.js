import { createApp } from "vue"
import App from "./App.vue"
import { router } from "./router/index.js"
import { useAuth } from "./store/auth.js"
import "./style.css"
import "./styles/compact-error-cards.css"
import "./styles/library.css"

async function bootstrap() {
  await useAuth().init()
  createApp(App)
    .use(router)
    .mount("#app")
}

bootstrap()
