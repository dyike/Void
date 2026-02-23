import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import { Search, X, Clipboard, Sparkles, Settings, ArrowLeft, Loader2, FileText } from 'lucide-react'
import * as CommandsService from '../bindings/github.com/dyike/Void/internal/service/commands/service.js'
import * as CalcService from '../bindings/github.com/dyike/Void/internal/service/calculator/service.js'
import ClipboardManager from './ClipboardManager'
import AiChatView from './AiChatView'
import SettingsView from './SettingsView'
import SnippetsView from './SnippetsView'
import ActionPanel, { buildActions } from './ActionPanel'
import ScriptCommandsView from './ScriptCommandsView'

// ─── Types ────────────────────────────────────────────────────────────────

interface CommandInfo {
  id: string
  title: string
  subtitle?: string
  iconDataUrl?: string
  iconEmoji?: string
  category: 'app' | 'settings' | 'system' | 'file' | 'snippet'
  path?: string
  score?: number
}

interface CalcResult {
  input: string
  result: string
  isValid: boolean
}

type View = 'launcher' | 'clipboard' | 'ai' | 'settings' | 'snippets' | 'scripts'

const CATEGORY_ORDER = ['system', 'app', 'file', 'settings']
const CATEGORY_LABELS: Record<string, string> = {
  system: 'Void',
  app: 'Applications',
  settings: 'System Settings',
  file: 'Files',
}

const SYSTEM_ICONS: Record<string, React.ReactNode> = {
  'system-clipboard-manager': <Clipboard size={15} />,
  'system-ai-chat': <Sparkles size={15} />,
  'system-snippets': <FileText size={15} />,
  'system-scripts': <FileText size={15} />,
  'system-settings': <Settings size={15} />,
}

function useDebounce<T>(value: T, ms: number): T {
  const [d, setD] = useState(value)
  useEffect(() => {
    const t = setTimeout(() => setD(value), ms)
    return () => clearTimeout(t)
  }, [value, ms])
  return d
}

// ─── App ──────────────────────────────────────────────────────────────────

const App: React.FC = () => {
  const [query, setQuery] = useState('')
  const [commands, setCommands] = useState<CommandInfo[]>([])
  const [fileResults, setFileResults] = useState<CommandInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [fileSearching, setFileSearching] = useState(false)
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [view, setView] = useState<View>('launcher')
  const [showActions, setShowActions] = useState(false)
  const [calcResult, setCalcResult] = useState<CalcResult | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)
  const itemRefs = useRef<(HTMLDivElement | null)[]>([])
  const debouncedQuery = useDebounce(query, 200)

  // ── Load commands ─────────────────────────────────────────────────────
  const fetchCommands = useCallback(async () => {
    try {
      const r = await CommandsService.GetCommands()
      setCommands(r as CommandInfo[])
    } catch (e) { console.error(e) }
    finally { setLoading(false) }
  }, [])

  useEffect(() => { fetchCommands() }, [fetchCommands])

  // ── File search ───────────────────────────────────────────────────────
  useEffect(() => {
    if (!debouncedQuery.trim() || debouncedQuery.length < 2) {
      setFileResults([]); return
    }
    setFileSearching(true)
    CommandsService.SearchFiles(debouncedQuery)
      .then(r => setFileResults((r || []) as CommandInfo[]))
      .catch(() => setFileResults([]))
      .finally(() => setFileSearching(false))
  }, [debouncedQuery])

  // ── Calculator ────────────────────────────────────────────────────────
  useEffect(() => {
    if (!debouncedQuery.trim()) { setCalcResult(null); return }
    CalcService.IsLikelyExpression(debouncedQuery).then(isExpr => {
      if (!isExpr) { setCalcResult(null); return }
      CalcService.Evaluate(debouncedQuery)
        .then(r => setCalcResult(r as CalcResult))
        .catch(() => setCalcResult(null))
    }).catch(() => setCalcResult(null))
  }, [debouncedQuery])

  // ── Filter + group ────────────────────────────────────────────────────
  const filtered = useMemo(() => {
    const base = query.trim()
      ? commands.filter(c =>
        c.title.toLowerCase().includes(query.toLowerCase()) ||
        (c.subtitle?.toLowerCase().includes(query.toLowerCase()) ?? false)
      )
      : commands
    return fileResults.length > 0 ? [...base, ...fileResults] : base
  }, [commands, fileResults, query])

  const grouped = useMemo(() => {
    const map = new Map<string, CommandInfo[]>()
    for (const cmd of filtered) {
      if (!map.has(cmd.category)) map.set(cmd.category, [])
      map.get(cmd.category)!.push(cmd)
    }
    return CATEGORY_ORDER
      .filter(cat => map.has(cat))
      .map(cat => ({ label: CATEGORY_LABELS[cat] || cat, items: map.get(cat)! }))
  }, [filtered])

  const flat = useMemo(() => grouped.flatMap(g => g.items), [grouped])

  useEffect(() => { setSelectedIndex(0) }, [query])
  useEffect(() => {
    itemRefs.current[selectedIndex]?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  }, [selectedIndex])
  useEffect(() => {
    if (view === 'launcher') setTimeout(() => inputRef.current?.focus(), 50)
  }, [view])

  // ── Performer actions ─────────────────────────────────────────────────
  const handleAction = useCallback(async (actionId: string) => {
    const cmd = flat[selectedIndex]
    if (!cmd) return
    switch (actionId) {
      case 'open':
        launchCmd(cmd); break
      case 'reveal':
        if (cmd.path) await CommandsService.OpenURL('file://' + (cmd.path ? cmd.path.split('/').slice(0, -1).join('/') : '')).catch(() => { })
        break
      case 'copy-path':
        if (cmd.path) await navigator.clipboard.writeText(cmd.path).catch(() => { })
        break
      case 'copy-name':
        await navigator.clipboard.writeText(cmd.title).catch(() => { })
        break
    }
  }, [flat, selectedIndex])

  // ── Launch ────────────────────────────────────────────────────────────
  const launchCmd = useCallback(async (cmd: CommandInfo) => {
    if (cmd.id === 'system-clipboard-manager') { setView('clipboard'); return }
    if (cmd.id === 'system-ai-chat') { setView('ai'); return }
    if (cmd.id === 'system-settings') { setView('settings'); return }
    if (cmd.id === 'system-snippets') { setView('snippets'); return }
    if (cmd.id === 'system-scripts') { setView('scripts'); return }

    // macOS system actions
    const SYSTEM_ACTIONS = new Set([
      'system-lock-screen', 'system-sleep', 'system-empty-trash',
      'system-restart', 'system-shutdown', 'system-open-finder',
      'system-open-terminal', 'system-show-desktop',
    ])
    if (SYSTEM_ACTIONS.has(cmd.id)) {
      await CommandsService.ExecuteSystemCommand(cmd.id).catch(console.error)
      return
    }

    // Record frecency
    CommandsService.RecordUsage(cmd.id).catch(() => { })

    if (cmd.category === 'app' && cmd.path) {
      await CommandsService.LaunchApp(cmd.path).catch(console.error)
    } else if (cmd.category === 'settings' && cmd.path) {
      await CommandsService.OpenURL('file://' + cmd.path).catch(console.error)
    } else if (cmd.category === 'file' && cmd.path) {
      await CommandsService.OpenURL('file://' + cmd.path).catch(console.error)
    }
  }, [])

  // Copy calc result to clipboard
  const copyCalcResult = useCallback(() => {
    if (calcResult?.result) navigator.clipboard.writeText(calcResult.result).catch(() => { })
  }, [calcResult])

  // ── Keyboard ──────────────────────────────────────────────────────────
  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (view !== 'launcher') return
    if (showActions) return // handled by ActionPanel

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
        if (calcResult?.isValid) { copyCalcResult(); break }
        if (flat[selectedIndex]) launchCmd(flat[selectedIndex])
        break
      case 'k':
        if (e.metaKey || e.ctrlKey) {
          e.preventDefault()
          if (flat[selectedIndex]) setShowActions(true)
        }
        break
      case 'Escape':
        if (query) { setQuery('') }
        break
    }
  }, [view, showActions, flat, selectedIndex, launchCmd, calcResult, copyCalcResult, query])

  // ─── Icon renderer ─────────────────────────────────────────────────────
  const renderIcon = (cmd: CommandInfo) => {
    const sys = SYSTEM_ICONS[cmd.id]
    if (sys) return sys
    if (cmd.iconDataUrl) return (
      <img src={cmd.iconDataUrl} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover', borderRadius: 6 }} />
    )
    return <span style={{ fontSize: 17 }}>{cmd.iconEmoji || '📦'}</span>
  }

  // ─── Sub-views ─────────────────────────────────────────────────────────
  if (view !== 'launcher') {
    const onBack = () => { setView('launcher'); setQuery('') }
    const title = { clipboard: 'Clipboard History', ai: 'AI Chat', settings: 'Void Settings', snippets: 'Snippets', scripts: 'Script Commands' }[view]
    return (
      <div className="launcher" onKeyDown={handleKeyDown}>
        <div className="search-bar">
          <button className="search-clear" onClick={onBack} style={{ color: 'rgba(255,255,255,0.45)' }}>
            <ArrowLeft size={16} />
          </button>
          <span style={{ fontSize: 14, fontWeight: 500, color: 'rgba(255,255,255,0.7)' }}>{title}</span>
        </div>
        <div className="view-overlay">
          {view === 'clipboard' && <ClipboardManager onClose={onBack} />}
          {view === 'ai' && <AiChatView />}
          {view === 'settings' && <SettingsView />}
          {view === 'snippets' && <SnippetsView />}
          {view === 'scripts' && <ScriptCommandsView />}
        </div>
      </div>
    )
  }

  // ─── Launcher ──────────────────────────────────────────────────────────
  const selectedCmd = flat[selectedIndex]
  let flatIdx = 0

  return (
    <div className="launcher" onKeyDown={handleKeyDown}>
      {/* Search bar */}
      <div className="search-bar">
        <Search size={16} className="search-icon" />
        <input
          ref={inputRef}
          className="search-input"
          placeholder="Search apps, files, snippets…"
          value={query}
          onChange={e => setQuery(e.target.value)}
          autoFocus
        />
        {(loading || fileSearching) && (
          <Loader2 size={14} style={{ color: 'rgba(255,255,255,0.3)', animation: 'spin 1s linear infinite', flexShrink: 0 }} />
        )}
        {query && (
          <button className="search-clear" onClick={() => setQuery('')}>
            <X size={14} />
          </button>
        )}
      </div>

      {/* Calculator result bar */}
      {calcResult?.isValid && (
        <div
          className="calc-bar"
          onClick={copyCalcResult}
          title="Click to copy"
        >
          <span className="calc-expr">{calcResult.input}</span>
          <span className="calc-eq">=</span>
          <span className="calc-result">{calcResult.result}</span>
          <span className="calc-copy-hint">↵ copy</span>
        </div>
      )}

      {/* Command list */}
      <div className="command-list">
        {flat.length === 0 && !loading ? (
          <div className="empty-state"><span>No results for "{query}"</span></div>
        ) : (
          grouped.map(group => (
            <div key={group.label}>
              <div className="section-label">{group.label}</div>
              {group.items.map(cmd => {
                const idx = flatIdx++
                const sel = idx === selectedIndex
                return (
                  <div
                    key={cmd.id}
                    ref={el => { itemRefs.current[idx] = el }}
                    className={`command-item${sel ? ' selected' : ''}`}
                    onClick={() => launchCmd(cmd)}
                    onMouseEnter={() => setSelectedIndex(idx)}
                  >
                    <div
                      className="command-icon"
                      style={SYSTEM_ICONS[cmd.id] ? { background: 'rgba(99,102,241,0.15)' } : undefined}
                    >
                      {renderIcon(cmd)}
                    </div>
                    <div className="command-text">
                      <div className="command-title">{cmd.title}</div>
                      {cmd.subtitle && <div className="command-subtitle">{cmd.subtitle}</div>}
                    </div>
                    {sel && (
                      <span className="command-hint" style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                        <span title="Actions (⌘K)" style={{ cursor: 'pointer', opacity: 0.55 }}>⌘K</span>
                        <span>↵</span>
                      </span>
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
        <span className="footer-left">{loading ? 'Loading…' : `${flat.length} results`}</span>
        <div className="footer-actions">
          {selectedCmd && (
            <>
              <button className="footer-btn" onClick={() => setShowActions(true)} title="Actions">
                Actions <kbd>⌘K</kbd>
              </button>
              <button className="footer-btn primary" onClick={() => launchCmd(selectedCmd)}>
                Open <kbd>↵</kbd>
              </button>
            </>
          )}
        </div>
      </div>

      {/* Action Panel */}
      {showActions && selectedCmd && (
        <ActionPanel
          title={selectedCmd.title}
          subtitle={selectedCmd.subtitle}
          actions={buildActions(selectedCmd as any)}
          onAction={handleAction}
          onClose={() => setShowActions(false)}
        />
      )}

      <style>{`
        @keyframes spin { to { transform: rotate(360deg); } }
        .calc-bar {
          display: flex; align-items: center; gap: 8px;
          padding: 8px 18px;
          background: rgba(99,102,241,0.12);
          border-bottom: 1px solid rgba(99,102,241,0.18);
          cursor: pointer;
          transition: background .12s;
          flex-shrink: 0;
        }
        .calc-bar:hover { background: rgba(99,102,241,0.20); }
        .calc-expr { font-size: 13px; color: rgba(255,255,255,0.45); font-family: monospace; }
        .calc-eq   { font-size: 13px; color: rgba(255,255,255,0.35); }
        .calc-result { font-size: 15px; font-weight: 600; color: rgba(180,183,255,0.95); font-family: monospace; flex: 1; }
        .calc-copy-hint { font-size: 11px; color: rgba(255,255,255,0.25); }
      `}</style>
    </div>
  )
}

export default App
