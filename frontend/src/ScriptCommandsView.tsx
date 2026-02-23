import React, { useState, useEffect, useRef, useCallback } from 'react'
import { Play, RefreshCw, FolderOpen, Search, X, Loader2, CheckCircle, AlertCircle } from 'lucide-react'
import * as ScriptsService from '../bindings/github.com/dyike/Void/internal/service/scripts/service.js'

interface Argument {
    name: string
    type: string
    placeholder: string
    required: boolean
}

interface ScriptCommand {
    id: string
    title: string
    description?: string
    mode: string
    iconEmoji: string
    scriptPath: string
    arguments?: Argument[]
    packageName?: string
}

interface ExecutionResult {
    commandId: string
    title: string
    mode: string
    exitCode: number
    stdout: string
    stderr: string
    output: string
    firstLine: string
    lastLine: string
}

type Status = 'idle' | 'running' | 'success' | 'error'

const ScriptCommandsView: React.FC = () => {
    const [scripts, setScripts] = useState<ScriptCommand[]>([])
    const [query, setQuery] = useState('')
    const [selected, setSelected] = useState<ScriptCommand | null>(null)
    const [result, setResult] = useState<ExecutionResult | null>(null)
    const [status, setStatus] = useState<Status>('idle')
    const [args, setArgs] = useState<Record<string, string>>({})
    const searchRef = useRef<HTMLInputElement>(null)

    const load = useCallback(async () => {
        try {
            const r = await ScriptsService.GetScriptCommands()
            setScripts((r || []) as ScriptCommand[])
        } catch (e) { console.error(e) }
    }, [])

    useEffect(() => { load(); searchRef.current?.focus() }, [load])

    const filtered = scripts.filter(s =>
        !query || s.title.toLowerCase().includes(query.toLowerCase()) ||
        (s.description?.toLowerCase().includes(query.toLowerCase()) ?? false)
    )

    const selectScript = (s: ScriptCommand) => {
        setSelected(s)
        setResult(null)
        setStatus('idle')
        setArgs({})
    }

    const run = async (s: ScriptCommand) => {
        setStatus('running')
        setResult(null)
        try {
            const r = await ScriptsService.Execute(s.id, args)
            setResult(r as ExecutionResult)
            setStatus((r as ExecutionResult).exitCode === 0 ? 'success' : 'error')
        } catch (e) {
            setStatus('error')
            setResult({ commandId: s.id, title: s.title, mode: s.mode, exitCode: 1, stdout: '', stderr: String(e), output: String(e), firstLine: '', lastLine: '' })
        }
    }

    const openDir = () => ScriptsService.OpenScriptsDir().catch(() => { })
    const refresh = () => { ScriptsService.RefreshScriptCommands(); load() }

    return (
        <div className="view-overlay" style={{ display: 'flex', flexDirection: 'column' }}>
            {/* Search + toolbar */}
            <div className="search-bar">
                <Search size={14} style={{ color: 'rgba(255,255,255,0.35)', flexShrink: 0 }} />
                <input
                    ref={searchRef}
                    className="search-input"
                    placeholder="Search script commands…"
                    value={query}
                    onChange={e => setQuery(e.target.value)}
                />
                {query && <button className="search-clear" onClick={() => setQuery('')}><X size={13} /></button>}
                <button className="footer-btn" onClick={refresh} title="Refresh"><RefreshCw size={13} /></button>
                <button className="footer-btn" onClick={openDir} title="Open scripts folder"><FolderOpen size={13} /></button>
            </div>

            {/* Split: list + detail */}
            <div className="clipboard-split" style={{ flex: 1, minHeight: 0 }}>
                {/* Left: command list */}
                <div className="clipboard-list">
                    {filtered.length === 0
                        ? <div className="empty-state" style={{ padding: 20, fontSize: 12 }}>
                            No scripts found.<br />
                            <span style={{ opacity: 0.5 }}>Add .sh scripts to the scripts folder.</span>
                        </div>
                        : filtered.map(s => (
                            <div
                                key={s.id}
                                className={`clip-item${selected?.id === s.id ? ' selected' : ''}`}
                                onClick={() => selectScript(s)}
                                onDoubleClick={() => run(s)}
                            >
                                <span className="clip-text">
                                    <span style={{ marginRight: 6 }}>{s.iconEmoji}</span>
                                    {s.title}
                                </span>
                                {s.packageName && (
                                    <span style={{ fontSize: 10, opacity: 0.4, marginLeft: 4 }}>{s.packageName}</span>
                                )}
                            </div>
                        ))
                    }
                </div>

                {/* Right: detail + output */}
                <div className="clipboard-preview" style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
                    {selected
                        ? <>
                            <div style={{ fontSize: 14, fontWeight: 600, color: 'rgba(255,255,255,0.85)' }}>
                                {selected.iconEmoji} {selected.title}
                            </div>
                            {selected.description && (
                                <div style={{ fontSize: 12, color: 'rgba(255,255,255,0.45)', lineHeight: 1.5 }}>
                                    {selected.description}
                                </div>
                            )}
                            <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.3)', fontFamily: 'monospace' }}>
                                mode: {selected.mode}
                            </div>

                            {/* Arguments */}
                            {selected.arguments && selected.arguments.length > 0 && (
                                <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
                                    {selected.arguments.map(arg => (
                                        <div key={arg.name}>
                                            <div style={{ fontSize: 11, color: 'rgba(255,255,255,0.45)', marginBottom: 3 }}>
                                                {arg.name}{arg.required ? ' *' : ''}
                                            </div>
                                            <input
                                                className="settings-input"
                                                style={{ width: '100%' }}
                                                type={arg.type === 'password' ? 'password' : 'text'}
                                                placeholder={arg.placeholder || arg.name}
                                                value={args[arg.name] || ''}
                                                onChange={e => setArgs(a => ({ ...a, [arg.name]: e.target.value }))}
                                            />
                                        </div>
                                    ))}
                                </div>
                            )}

                            <button
                                className="footer-btn primary"
                                style={{ alignSelf: 'flex-start', padding: '7px 16px' }}
                                onClick={() => run(selected)}
                                disabled={status === 'running'}
                            >
                                {status === 'running'
                                    ? <><Loader2 size={13} style={{ animation: 'spin 1s linear infinite' }} /> Running…</>
                                    : <><Play size={13} /> Run <kbd>↵</kbd></>
                                }
                            </button>

                            {/* Result */}
                            {result && (
                                <div style={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column', gap: 6 }}>
                                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 12 }}>
                                        {status === 'success'
                                            ? <><CheckCircle size={13} style={{ color: 'rgba(52,211,153,0.8)' }} /><span style={{ color: 'rgba(52,211,153,0.8)' }}>Exit 0</span></>
                                            : <><AlertCircle size={13} style={{ color: 'rgba(248,113,113,0.8)' }} /><span style={{ color: 'rgba(248,113,113,0.8)' }}>Exit {result.exitCode}</span></>
                                        }
                                    </div>
                                    {result.output && (
                                        <pre className="preview-content" style={{ flex: 1, minHeight: 0, overflowY: 'auto', whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
                                            {result.output}
                                        </pre>
                                    )}
                                </div>
                            )}
                        </>
                        : <div className="empty-state">Select a script to run</div>
                    }
                </div>
            </div>

            {/* Footer */}
            <div className="footer">
                <span className="footer-left">{filtered.length} scripts</span>
                <div className="footer-actions">
                    {selected && status !== 'running' && (
                        <button className="footer-btn primary" onClick={() => run(selected)}>
                            <Play size={13} /> Run <kbd>↵</kbd>
                        </button>
                    )}
                </div>
            </div>

            <style>{`@keyframes spin { to { transform: rotate(360deg) } }`}</style>
        </div>
    )
}

export default ScriptCommandsView
