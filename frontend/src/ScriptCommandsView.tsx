import React, { useState, useEffect, useRef, useCallback } from 'react'
import { Play, RefreshCw, FolderOpen, Search, X, Loader2, CheckCircle, AlertCircle } from 'lucide-react'
import * as ScriptsService from '../bindings/github.com/dyike/Void/internal/service/scripts/service.js'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

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
        <div className="flex min-h-0 flex-1 flex-col">
            <div className="flex items-center gap-2 border-b border-white/10 px-4 py-3">
                <Search size={14} className="text-white/40" />
                <Input
                    ref={searchRef}
                    className="h-8 border-0 bg-transparent px-0 text-[14px] text-white/90 placeholder:text-white/35 shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
                    placeholder="Search script commands…"
                    value={query}
                    onChange={e => setQuery(e.target.value)}
                />
                {query && (
                    <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7 text-white/40 hover:text-white/80"
                        onClick={() => setQuery('')}
                    >
                        <X size={13} />
                    </Button>
                )}
                <Button
                    variant="secondary"
                    size="sm"
                    className="ml-auto bg-white/10 text-white/70 hover:bg-white/20"
                    onClick={refresh}
                    title="Refresh"
                >
                    <RefreshCw size={13} />
                </Button>
                <Button
                    variant="secondary"
                    size="sm"
                    className="bg-white/10 text-white/70 hover:bg-white/20"
                    onClick={openDir}
                    title="Open scripts folder"
                >
                    <FolderOpen size={13} />
                </Button>
            </div>

            <div className="flex min-h-0 flex-1">
                <div className="w-[40%] overflow-y-auto border-r border-white/10 p-2">
                    {filtered.length === 0
                        ? <div className="flex flex-col items-center justify-center gap-1 px-4 py-8 text-xs text-white/40">
                            <span>No scripts found.</span>
                            <span className="text-white/30">Add .sh scripts to the scripts folder.</span>
                        </div>
                        : filtered.map(s => (
                            <div
                                key={s.id}
                                className={cn(
                                    "flex cursor-pointer items-center gap-2 rounded-md border border-transparent px-2 py-2 text-sm transition",
                                    selected?.id === s.id ? "border-indigo-400/40 bg-indigo-500/20" : "hover:bg-white/10"
                                )}
                                onClick={() => selectScript(s)}
                                onDoubleClick={() => run(s)}
                            >
                                <span className="text-[13px]">{s.iconEmoji}</span>
                                <span className="truncate text-[13px] text-white/80">{s.title}</span>
                                {s.packageName && (
                                    <span className="ml-1 text-[10px] text-white/35">{s.packageName}</span>
                                )}
                            </div>
                        ))
                    }
                </div>

                <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-4">
                    {selected
                        ? <>
                            <div className="text-sm font-semibold text-white/85">
                                {selected.iconEmoji} {selected.title}
                            </div>
                            {selected.description && (
                                <div className="text-xs text-white/50">
                                    {selected.description}
                                </div>
                            )}
                            <div className="font-mono text-[11px] text-white/35">
                                mode: {selected.mode}
                            </div>

                            {selected.arguments && selected.arguments.length > 0 && (
                                <div className="flex flex-col gap-2">
                                    {selected.arguments.map(arg => (
                                        <div key={arg.name}>
                                            <div className="mb-1 text-[11px] text-white/50">
                                                {arg.name}{arg.required ? ' *' : ''}
                                            </div>
                                            <Input
                                                className="h-8 border-white/10 bg-white/5 text-white/90 placeholder:text-white/35 focus-visible:ring-indigo-400/30"
                                                type={arg.type === 'password' ? 'password' : 'text'}
                                                placeholder={arg.placeholder || arg.name}
                                                value={args[arg.name] || ''}
                                                onChange={e => setArgs(a => ({ ...a, [arg.name]: e.target.value }))}
                                            />
                                        </div>
                                    ))}
                                </div>
                            )}

                            <Button
                                size="sm"
                                className="w-fit bg-indigo-500/30 text-indigo-100 hover:bg-indigo-500/45"
                                onClick={() => run(selected)}
                                disabled={status === 'running'}
                            >
                                {status === 'running'
                                    ? <><Loader2 size={13} className="animate-spin" /> Running…</>
                                    : <><Play size={13} /> Run <kbd className="ml-1 inline-flex h-5 min-w-5 items-center justify-center rounded bg-indigo-500/30 px-1 text-[11px] text-indigo-100">↵</kbd></>
                                }
                            </Button>

                            {result && (
                                <div className="flex min-h-0 flex-1 flex-col gap-2">
                                    <div className="flex items-center gap-2 text-xs">
                                        {status === 'success'
                                            ? <><CheckCircle size={13} className="text-emerald-400" /><span className="text-emerald-400">Exit 0</span></>
                                            : <><AlertCircle size={13} className="text-red-400" /><span className="text-red-400">Exit {result.exitCode}</span></>
                                        }
                                    </div>
                                    {result.output && (
                                        <pre className="flex-1 whitespace-pre-wrap break-all rounded-lg border border-white/10 bg-white/5 p-3 text-xs text-white/80">
                                            {result.output}
                                        </pre>
                                    )}
                                </div>
                            )}
                        </>
                        : <div className="flex flex-1 items-center justify-center text-xs text-white/40">Select a script to run</div>
                    }
                </div>
            </div>

            <div className="flex items-center justify-between gap-2 border-t border-white/10 px-3 py-2">
                <span className="text-xs text-white/35">{filtered.length} scripts</span>
                <div className="flex items-center gap-2">
                    {selected && status !== 'running' && (
                        <Button
                            size="sm"
                            className="bg-indigo-500/30 text-indigo-100 hover:bg-indigo-500/45"
                            onClick={() => run(selected)}
                        >
                            <Play size={13} /> Run <kbd className="ml-1 inline-flex h-5 min-w-5 items-center justify-center rounded bg-indigo-500/30 px-1 text-[11px] text-indigo-100">↵</kbd>
                        </Button>
                    )}
                </div>
            </div>
        </div>
    )
}

export default ScriptCommandsView
