import { createRouter, createWebHistory } from 'vue-router'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: () => import('./views/LibraryView.vue') },
    { path: '/search', component: () => import('./views/SearchView.vue') },
    { path: '/lectures/:id', component: () => import('./views/LectureView.vue') },
    { path: '/bookmarks', component: () => import('./views/BookmarksView.vue') },
    { path: '/settings', component: () => import('./views/SettingsView.vue') },
  ],
})
