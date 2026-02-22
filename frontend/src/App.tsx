import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import { Search, X, Clipboard, Sparkles, Settings, ArrowLeft, Loader2 } from 'lucide-react'
import * as CommandsService from '../bindings/github.com/dyike/Void/internal/service/commands/service.js'
import ClipboardManager from './ClipboardManager'
import AiChatView from './AiChatView'
import SettingsView from './SettingsView'

// Binding types
interface CommandInfo {
  id: string
  title: string
  subtitle?: string
  iconDataUrl?: string
  iconEmoji?: string
  category: 'app' | 'settings' | 'system'
  path?: string
}

type View = 'launcher' | 'clipboard' | 'ai' | 'settings'

const CATEGORY_ORDER = ['system', 'app', 'settings']

const App: React.FC = () => {
  const [query, setQuery] = useState('')
  const [commands, setCommands] = useState<CommandInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [view, setView] = useState<View>('launcher')
  const inputRef = useRef<HTMLInputElement>(null)
  const itemRefs = useRef<(HTMLDivElement | null)[]>([])

  // ── Load commands ────────────────────────────────────────────────────
  const fetchCommands = useCallback(async () => {
    try {
      const result = await CommandsService.GetCommands()
      setCommands(result as CommandInfo[])
    } catch (e) {
      console.error('Failed to fetch commands:', e)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchCommands()
  }, [fetchCommands])

  // ── Filter + group commands ──────────────────────────────────────────
  const filtered = useMemo(() => {
    if (!query.trim()) return commands
    const q = query.toLowerCase()
    return commands.filter(cmd =>
      cmd.title.toLowerCase().includes(q) ||
      (cmd.subtitle?.toLowerCase().includes(q) ?? false)
    )
  }, [commands, query])

  const grouped = useMemo(() => {
    const map = new Map<string, CommandInfo[]>()
    for (const cmd of filtered) {
      const cat = cmd.category
      if (!map.has(cat)) map.set(cat, [])
      map.get(cat)!.push(cmd)
    }
    const result: Array<{ label: string; items: CommandInfo[] }> = []
    for (const cat of CATEGORY_ORDER) {
      if (map.has(cat)) {
        const label = cat === 'app' ? 'Applications' : cat === 'system' ? 'Void' : 'System Settings'
        result.push({ label, items: map.get(cat)! })
      }
    }
    return result
  }, [filtered])

  // Flat list for keyboard navigation
  const flat = useMemo(() => grouped.flatMap(g => g.items), [grouped])

  // ── Reset selection when query changes ───────────────────────────────
  useEffect(() => {
    setSelectedIndex(0)
  }, [query])

  // ── Scroll selected item into view ───────────────────────────────────
  useEffect(() => {
    itemRefs.current[selectedIndex]?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  }, [selectedIndex])

  // ── Focus search input ───────────────────────────────────────────────
  useEffect(() => {
    if (view === 'launcher') {
      setTimeout(() => inputRef.current?.focus(), 50)
    }
  }, [view])

  // ── Launch a command ─────────────────────────────────────────────────
  const launch = useCallback(async (cmd: CommandInfo) => {
    if (cmd.id === 'system-clipboard-manager') {
      setView('clipboard'); return
    }
    if (cmd.id === 'system-ai-chat') {
      setView('ai'); return
    }
    if (cmd.id === 'system-settings') {
      setView('settings'); return
    }
    if (cmd.category === 'app' && cmd.path) {
      try { await CommandsService.LaunchApp(cmd.path) } catch (e) { console.error(e) }
    }
  }, [])

  // ── Keyboard handling ────────────────────────────────────────────────
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (view !== 'launcher') return
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        setSelectedIndex(i => Math.min(i + 1, flat.length - 1))
        break
      case 'ArrowUp':
        e.preventDefault()
        setSelectedIndex(i => Math.max(i - 1, 0))
        break
      case 'Enter':
        e.preventDefault()
        if (flat[selectedIndex]) launch(flat[selectedIndex])
        break
      case 'Escape':
        setQuery('')
        break
    }
  }, [view, flat, selectedIndex, launch])

  // ─── Render icon ──────────────────────────────────────────────────────
  const renderIcon = (cmd: CommandInfo) => {
    if (cmd.iconDataUrl) {
      return <img src={cmd.iconDataUrl} alt="" />
    }
    return <span>{cmd.iconEmoji || '📦'}</span>
  }

  // ─── Category icon helper ─────────────────────────────────────────────
  const getCategoryIcon = (id: string) => {
    if (id === 'system-clipboard-manager') return <Clipboard size={15} />
    if (id === 'system-ai-chat') return <Sparkles size={15} />
    if (id === 'system-settings') return <Settings size={15} />
    return null
  }

  // ─── View rendering ───────────────────────────────────────────────────
  if (view !== 'launcher') {
    const onBack = () => { setView('launcher'); setQuery('') }
    return (
      <div className="launcher" onKeyDown={handleKeyDown}>
        {/* Back header */}
        <div className="search-bar">
          <button
            className="search-clear"
            onClick={onBack}
            style={{ color: 'rgba(255,255,255,0.45)' }}
            title="Back"
          >
            <ArrowLeft size={16} />
          </button>
          <span style={{ fontSize: 14, fontWeight: 500, color: 'rgba(255,255,255,0.7)' }}>
            {view === 'clipboard' ? 'Clipboard History'
              : view === 'ai' ? 'AI Chat'
                : 'Settings'}
          </span>
        </div>
        <div className="view-overlay">
          {view === 'clipboard' && <ClipboardManager onClose={onBack} />}
          {view === 'ai' && <AiChatView />}
          {view === 'settings' && <SettingsView />}
        </div>
      </div>
    )
  }

  // ─── Launcher main view ───────────────────────────────────────────────
  let flatIdx = 0

  return (
    <div className="launcher" onKeyDown={handleKeyDown}>
      {/* Search bar */}
      <div className="search-bar">
        <Search size={16} className="search-icon" />
        <input
          ref={inputRef}
          className="search-input"
          placeholder="Search applications and commands..."
          value={query}
          onChange={e => setQuery(e.target.value)}
          autoFocus
        />
        {query && (
          <button className="search-clear" onClick={() => setQuery('')}>
            <X size={14} />
          </button>
        )}
      </div>

      {/* Command list */}
      <div className="command-list">
        {loading ? (
          <div className="loading-spinner">
            <Loader2 size={16} style={{ animation: 'spin 1s linear infinite' }} />
            Loading commands...
          </div>
        ) : flat.length === 0 ? (
          <div className="empty-state">
            <span>No results for "{query}"</span>
          </div>
        ) : (
          grouped.map(group => (
            <div key={group.label}>
              <div className="section-label">{group.label}</div>
              {group.items.map(cmd => {
                const idx = flatIdx++
                const isSelected = idx === selectedIndex
                const sysIcon = getCategoryIcon(cmd.id)
                return (
                  <div
                    key={cmd.id}
                    ref={el => { itemRefs.current[idx] = el }}
                    className={`command-item${isSelected ? ' selected' : ''}`}
                    onClick={() => launch(cmd)}
                    onMouseEnter={() => setSelectedIndex(idx)}
                  >
                    <div className="command-icon" style={sysIcon ? { background: 'rgba(99,102,241,0.15)' } : {}}>
                      {sysIcon ?? renderIcon(cmd)}
                    </div>
                    <div className="command-text">
                      <div className="command-title">{cmd.title}</div>
                      {cmd.subtitle && (
                        <div className="command-subtitle">{cmd.subtitle}</div>
                      )}
                    </div>
                    {isSelected && (
                      <span className="command-hint">↵</span>
                    )}
                  </div>
                )
              })}
            </div>
          ))
        )}
      </div>

      {/* Footer */}
      <div className="footer">
        <span className="footer-left">
          {loading ? 'Loading…' : `${flat.length} results`}
        </span>
        <div className="footer-actions">
          {flat[selectedIndex] && (
            <button
              className="footer-btn primary"
              onClick={() => launch(flat[selectedIndex])}
            >
              Open <kbd>↵</kbd>
            </button>
          )}
        </div>
      </div>

      <style>{`
        @keyframes spin { to { transform: rotate(360deg); } }
      `}</style>
    </div>
  )
}

export default App
