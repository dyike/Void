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

// --- Text Size Management ---
export function applyTextSize(size: string) {
  const html = document.documentElement
  if (size === 'small') {
    html.style.zoom = '0.85'
  } else if (size === 'large') {
    html.style.zoom = '1.25'
  } else {
    html.style.zoom = '1'
  }
}

let currentTextSize = 'medium'

export function setTextSize(size: string) {
  currentTextSize = size
  applyTextSize(size)
  Events.Emit('TextSizeChanged' as any, size)
}

Events.On('TextSizeChanged', (e: any) => {
  const size = (e?.data ?? e) as string
  if (size && typeof size === 'string') {
    currentTextSize = size
    applyTextSize(size)
  }
})

// Re-apply theme and text size when window becomes visible (covers hotkey activation)
document.addEventListener('visibilitychange', () => {
  if (!document.hidden) {
    SettingsService.GetSettings()
      .then((s: any) => {
        const mode = s?.themeMode || 'system'
        if (mode !== currentMode) {
          currentMode = mode
          applyTheme(mode)
        }
        const size = s?.fontSize || 'medium'
        if (size !== currentTextSize) {
          currentTextSize = size
          applyTextSize(size)
        }
      })
      .catch(() => { })
  }
})

// Initialize theme and text size from settings
SettingsService.GetSettings()
  .then((s: any) => {
    const mode = s?.themeMode || 'system'
    currentMode = mode
    applyTheme(mode)

    const size = s?.fontSize || 'medium'
    currentTextSize = size
    applyTextSize(size)
  })
  .catch(() => {
    applyTheme('system')
    applyTextSize('medium')
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

