import { mount } from 'svelte'
import '@fontsource/vazirmatn/400.css'
import '@fontsource/vazirmatn/500.css'
import '@fontsource/vazirmatn/600.css'
import '@fontsource/vazirmatn/700.css'
import './app.css'
import { initTheme } from './lib/themeService.js'
import App from './App.svelte'

// Apply saved theme immediately before UI render
initTheme()

const app = mount(App, {
  target: document.getElementById('app'),
})

export default app
