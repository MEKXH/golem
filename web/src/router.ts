import { createRouter, createWebHistory } from 'vue-router'
import LandingPage from './pages/LandingPage.vue'
import ConsolePage from './pages/ConsolePage.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'landing', component: LandingPage },
    { path: '/console', name: 'console', component: ConsolePage },
  ],
  scrollBehavior() {
    return { top: 0 }
  },
})

export default router
