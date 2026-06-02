import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import { Search, X, ArrowLeft, Loader2 } from 'lucide-react'
import * as CommandsService from '../bindings/github.com/dyike/Void/internal/service/commands/service.js'
import * as CalcService from '../bindings/github.com/dyike/Void/internal/service/calculator/service.js'
import * as SettingsService from '../bindings/github.com/dyike/Void/internal/service/settings/service.js'
import * as SnippetsService from '../bindings/github.com/dyike/Void/internal/service/snippets/service.js'
import ClipboardManager from './ClipboardManager'
import AiChatView from './AiChatView'
import SnippetsView from './SnippetsView'
import ActionPanel, { buildActions } from './ActionPanel'
import ScriptCommandsView from './ScriptCommandsView'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"
import { DynamicIcon } from "./DynamicIcon"

// ─── Types ────────────────────────────────────────────────────────────────

interface CommandInfo {
  id: string
  title: string
  subtitle?: string
  iconDataUrl?: string
  icon?: string
  category: 'app' | 'settings' | 'system' | 'file' | 'snippet'
  path?: string
  score?: number
}

interface CalcResult {
  input: string
  result: string
  isValid: boolean
}

interface Snippet {
  id: string
  name: string
  content: string
  prefix?: string
}

type View = 'launcher' | 'clipboard' | 'ai' | 'settings' | 'snippets' | 'scripts'

const CATEGORY_ORDER = ['system', 'app', 'snippet', 'file', 'settings']
const CATEGORY_LABELS: Record<string, string> = {
  system: 'Void',
  app: 'Applications',
  snippet: 'Snippets',
  settings: 'System Settings',
  file: 'Files',
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
  const [snippetResults, setSnippetResults] = useState<CommandInfo[]>([])
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

  // ── Snippet search ────────────────────────────────────────────────────
  useEffect(() => {
    const q = debouncedQuery.trim()
    if (!q) {
      setSnippetResults([]); return
    }
    SnippetsService.GetSnippets(q)
      .then(r => {
        const results = ((r || []) as Snippet[]).slice(0, 8).map(sn => ({
          id: sn.id,
          title: sn.name,
          subtitle: sn.prefix ? `${sn.prefix} • ${sn.content}` : sn.content,
          icon: 'FileText',
          category: 'snippet' as const,
        }))
        setSnippetResults(results)
      })
      .catch(() => setSnippetResults([]))
  }, [debouncedQuery])

  // ── File search ───────────────────────────────────────────────────────
  useEffect(() => {
    const q = debouncedQuery.trim()
    if (!q || q.length < 2 || q.startsWith(';')) {
      setFileResults([])
      setFileSearching(false)
      return
    }
    setFileSearching(true)
    CommandsService.SearchFiles(q)
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
    const q = query.trim().toLowerCase()
    const base = q
      ? commands.filter(c => c.title.toLowerCase().includes(q))
      : commands
    // Only include file results with a reasonable fuzzy score
    const goodFiles = fileResults
      .filter(f => (f.score ?? 0) > 0)
      .slice(0, 10)
    return [...base, ...snippetResults, ...goodFiles]
  }, [commands, fileResults, query, snippetResults])

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

  // Re-focus search input when window becomes visible (e.g., activated via hotkey)
  useEffect(() => {
    const focusInput = () => {
      if (view === 'launcher') {
        setTimeout(() => inputRef.current?.focus(), 50)
      }
    }
    const handleVisibility = () => {
      if (!document.hidden) focusInput()
    }
    document.addEventListener('visibilitychange', handleVisibility)
    window.addEventListener('focus', focusInput)
    return () => {
      document.removeEventListener('visibilitychange', handleVisibility)
      window.removeEventListener('focus', focusInput)
    }
  }, [view])

  // ── Performer actions ─────────────────────────────────────────────────
  const handleAction = useCallback(async (actionId: string) => {
    const cmd = flat[selectedIndex]
    if (!cmd) return
    switch (actionId) {
      case 'open':
        launchCmd(cmd); break
      case 'reveal':
        if (cmd.path) await CommandsService.RevealInFinder(cmd.path).catch(() => { })
        break
      case 'quicklook':
        if (cmd.path) await CommandsService.QuickLook(cmd.path).catch(() => { })
        break
      case 'copy-path':
        if (cmd.path) await navigator.clipboard.writeText(cmd.path).catch(() => { })
        break
      case 'copy-name':
        await navigator.clipboard.writeText(cmd.title).catch(() => { })
        break
      case 'copy-snippet':
        await SnippetsService.CopyToClipboard(cmd.id).catch(() => { })
        break
    }
  }, [flat, selectedIndex])

  // ── Launch ────────────────────────────────────────────────────────────
  const launchCmd = useCallback(async (cmd: CommandInfo) => {
    if (cmd.id === 'system-clipboard-manager') { setView('clipboard'); return }
    if (cmd.id === 'system-ai-chat') { setView('ai'); return }
    if (cmd.id === 'system-snippets') { setView('snippets'); return }
    if (cmd.id === 'system-scripts') { setView('scripts'); return }

    // macOS system actions
    const SYSTEM_ACTIONS = new Set([
      'system-lock-screen', 'system-sleep', 'system-empty-trash',
      'system-restart', 'system-shutdown', 'system-open-finder',
      'system-open-terminal', 'system-show-desktop',
      'window-left-half', 'window-right-half', 'window-top-half', 'window-bottom-half',
      'window-top-left', 'window-top-right', 'window-bottom-left', 'window-bottom-right',
      'window-maximize', 'window-center',
    ])
    if (SYSTEM_ACTIONS.has(cmd.id)) {
      await CommandsService.ExecuteSystemCommand(cmd.id).catch(console.error)
      return
    }

    // Record frecency
    CommandsService.RecordUsage(cmd.id).catch(() => { })

    if (cmd.category === 'app' && cmd.path) {
      await CommandsService.LaunchApp(cmd.path).catch(console.error)
    } else if (cmd.category === 'snippet') {
      await SnippetsService.CopyToClipboard(cmd.id).catch(console.error)
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
      case ',':
        if (e.metaKey || e.ctrlKey) {
          e.preventDefault()
          SettingsService.OpenSettingsWindow().catch(console.error)
        }
        break
    }
  }, [view, showActions, flat, selectedIndex, launchCmd, calcResult, copyCalcResult, query])

  // ─── Icon renderer ─────────────────────────────────────────────────────
  const renderIcon = (cmd: CommandInfo) => {
    if (cmd.iconDataUrl) return (
      <img src={cmd.iconDataUrl} alt="" className="h-full w-full rounded-md object-cover" />
    )
    return <DynamicIcon name={cmd.icon} size={16} className="text-foreground/80 group-aria-selected:text-foreground" />
  }

  // ─── Sub-views ─────────────────────────────────────────────────────────
  if (view !== 'launcher') {
    const onBack = () => { setView('launcher'); setQuery('') }
    const title = { clipboard: 'Clipboard History', ai: 'AI Chat', snippets: 'Snippets', scripts: 'Script Commands' }[view]
    return (
      <div className="flex h-full w-full flex-col overflow-hidden rounded-lg border border-launcher-border bg-launcher/85 text-foreground backdrop-blur-3xl" onKeyDown={handleKeyDown}>
        <div className="flex items-center gap-2 border-b border-launcher-border px-4 py-3">
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7 text-muted-foreground hover:text-foreground"
            onClick={onBack}
          >
            <ArrowLeft size={16} />
          </Button>
          <span className="text-sm font-medium text-muted-foreground">{title}</span>
        </div>
        <div className="flex min-h-0 flex-1 flex-col">
          {view === 'clipboard' && <ClipboardManager onClose={onBack} />}
          {view === 'ai' && <AiChatView />}
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
    <div
      className="flex h-full w-full flex-col overflow-hidden rounded-lg border border-launcher-border bg-launcher/85 text-foreground backdrop-blur-3xl"
      onKeyDown={handleKeyDown}
      onMouseDown={(e) => {
        // Prevent clicks on non-input elements from stealing focus
        if (e.target !== inputRef.current) {
          e.preventDefault()
          inputRef.current?.focus()
        }
      }}
    >
      {/* Search bar */}
      <div className="flex items-center gap-2 border-b border-launcher-border px-4 py-3">
        <Search size={16} className="text-muted-foreground" />
        <Input
          ref={inputRef}
          className="h-8 border-0 bg-transparent px-0 text-[15px] font-medium text-foreground placeholder:text-muted-foreground/60 shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
          placeholder="Search apps, files, snippets…"
          value={query}
          onChange={e => setQuery(e.target.value)}
          autoFocus
        />
        {(loading || fileSearching) && (
          <Loader2 size={14} className="shrink-0 animate-spin text-muted-foreground/60" />
        )}
        {query && (
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7 text-muted-foreground hover:text-foreground"
            onClick={() => setQuery('')}
          >
            <X size={14} />
          </Button>
        )}
      </div>

      {/* Calculator result bar */}
      {calcResult?.isValid && (
        <div
          className="flex shrink-0 cursor-pointer items-center gap-2 border-b border-indigo-400/20 bg-indigo-500/15 px-4 py-2 transition hover:bg-indigo-500/25"
          onClick={copyCalcResult}
          title="Click to copy"
        >
          <span className="text-xs text-muted-foreground">{calcResult.input}</span>
          <span className="text-xs text-muted-foreground/60">=</span>
          <span className="flex-1 font-mono text-[15px] font-semibold text-indigo-700 dark:text-indigo-200/95">{calcResult.result}</span>
          <span className="text-[11px] text-muted-foreground/60">↵ copy</span>
        </div>
      )}

      {/* Command list */}
      <div className="flex-1 overflow-y-auto p-2">
        {flat.length === 0 && !loading ? (
          <div className="flex flex-col items-center justify-center gap-2 px-8 py-10 text-sm text-muted-foreground">
            <span>No results for "{query}"</span>
          </div>
        ) : (
          grouped.map(group => (
            <div key={group.label}>
              <div className="px-3 pb-1 pt-2 text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground/60">
                {group.label}
              </div>
              {group.items.map(cmd => {
                const idx = flatIdx++
                const sel = idx === selectedIndex
                return (
                  <div
                    key={cmd.id}
                    ref={el => { itemRefs.current[idx] = el }}
                    className={cn(
                      "flex cursor-pointer items-center gap-2 rounded-md border border-transparent px-3 py-2 transition",
                      sel ? "border-indigo-400/40 bg-indigo-500/25" : "hover:bg-accent"
                    )}
                    onClick={() => launchCmd(cmd)}
                    onMouseEnter={() => setSelectedIndex(idx)}
                  >
                    <div
                      className={cn(
                        "flex h-7 w-7 items-center justify-center rounded-md text-[17px]",
                        cmd.category === 'system' ? "bg-indigo-500/20 text-indigo-600 dark:text-indigo-100" : "bg-secondary"
                      )}
                    >
                      {renderIcon(cmd)}
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-[14px] font-medium text-foreground">{cmd.title}</div>
                      {cmd.subtitle && <div className="truncate text-[11px] text-muted-foreground">{cmd.subtitle}</div>}
                    </div>
                    {sel && (
                      <span className="flex items-center gap-2 text-[11px] text-muted-foreground">
                        <span title="Actions (⌘K)" className="cursor-pointer opacity-60">⌘K</span>
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
      <div className="flex items-center justify-between gap-2 border-t border-launcher-border px-3 py-2">
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7 text-muted-foreground hover:text-foreground"
          onClick={() => SettingsService.OpenSettingsWindow()}
          title="Settings"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
          </svg>
        </Button>
        <div className="flex items-center gap-2">
          {selectedCmd && (
            <>
              <Button
                variant="secondary"
                size="sm"
                className="bg-secondary text-secondary-foreground hover:bg-accent"
                onClick={() => setShowActions(true)}
                title="Actions"
              >
                Actions
                <kbd className="ml-1 inline-flex h-5 min-w-5 items-center justify-center rounded bg-secondary px-1 text-[11px] text-muted-foreground">⌘K</kbd>
              </Button>
              <Button
                size="sm"
                className="bg-indigo-500/30 text-indigo-600 dark:text-indigo-100 hover:bg-indigo-500/45"
                onClick={() => launchCmd(selectedCmd)}
              >
                {selectedCmd.category === 'snippet' ? 'Copy' : 'Open'}
                <kbd className="ml-1 inline-flex h-5 min-w-5 items-center justify-center rounded bg-indigo-500/30 px-1 text-[11px] text-indigo-600 dark:text-indigo-100">↵</kbd>
              </Button>
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

    </div>
  )
}

export default App
