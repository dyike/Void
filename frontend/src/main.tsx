import React from 'react'
import ReactDOM from 'react-dom/client'
import './index.css'
import App from './App'
import SettingsView from './SettingsView'

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
