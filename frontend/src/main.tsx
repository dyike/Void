import React from 'react'
import ReactDOM from 'react-dom/client'
import './index.css'
import App from './App'
import SettingsView from './SettingsView'
import * as SettingsService from '../bindings/github.com/dyike/Void/internal/service/settings/service.js'
import { Events } from '@wailsio/runtime'

export function applyTheme(mode: string) {
  const html = document.documentElement
  if (mode === 'dark') {
    html.classList.add('dark')
  } else if (mode === 'light') {
    html.classList.remove('dark')
  } else {
    // system
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    if (prefersDark) {
      html.classList.add('dark')
    } else {
      html.classList.remove('dark')
    }
  }
}

// Listen for system theme changes
let currentMode = 'system'
const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
mediaQuery.addEventListener('change', () => {
  if (currentMode === 'system') {
    applyTheme('system')
  }
})

export function setThemeMode(mode: string) {
  currentMode = mode
  applyTheme(mode)
  // Broadcast to all windows (including main window) via Wails events
  Events.Emit('ThemeChanged' as any, mode)
}

// Listen for ThemeChanged events from other windows (e.g., Settings → Main)
Events.On('ThemeChanged', (e: any) => {
  const mode = (e?.data ?? e) as string
  if (mode && typeof mode === 'string') {
    currentMode = mode
    applyTheme(mode)
  }
})

// Re-apply theme when window becomes visible (covers hotkey activation)
document.addEventListener('visibilitychange', () => {
  if (!document.hidden) {
    SettingsService.GetSettings()
      .then((s: any) => {
        const mode = s?.themeMode || 'system'
        if (mode !== currentMode) {
          currentMode = mode
          applyTheme(mode)
        }
      })
      .catch(() => { })
  }
})

// Initialize theme from settings
SettingsService.GetSettings()
  .then((s: any) => {
    const mode = s?.themeMode || 'system'
    currentMode = mode
    applyTheme(mode)
  })
  .catch(() => {
    applyTheme('system')
  })

function Root() {
  const hash = window.location.hash
  if (hash === '#/settings') {
    return <SettingsView />
  }
  return <App />
}

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <Root />
  </React.StrictMode>,
)

