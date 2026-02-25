import React, { useState, useRef, useEffect } from 'react'
import { Send, Loader2, ChevronDown, ChevronRight, Terminal, CheckCircle2, XCircle } from 'lucide-react'
import * as AIService from '../bindings/github.com/dyike/Void/internal/service/ai/service.js'
import { Events } from '@wailsio/runtime'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

interface Message {
    role: 'user' | 'assistant'
    content: string
}

interface ToolCall {
    name: string
    args: string
    result?: string
    isError?: boolean
    status: 'running' | 'done'
    startTime: number
    duration?: number
}

interface Part {
    type: 'text' | 'tool'
    text?: string
    tool?: ToolCall
}

interface RenderBlock {
    id: string
    role: 'user' | 'assistant'
    parts: Part[]
}

interface AIEvent {
    type: 'delta' | 'tool_start' | 'tool_end' | 'action' | 'turn' | 'done'
    content?: string
    name?: string
    args?: string
    result?: string
    isError?: boolean
    turn?: number
}

const ToolCallCard: React.FC<{ tool: ToolCall }> = ({ tool }) => {
    const [expanded, setExpanded] = useState(false)

    return (
        <div
            className={cn(
                "rounded-lg border bg-white/5 text-xs",
                tool.isError ? "border-red-400/30" : "border-white/10"
            )}
        >
            <div
                className="flex cursor-pointer items-center gap-2 px-2.5 py-1.5 transition hover:bg-white/5"
                onClick={() => setExpanded(!expanded)}
            >
                <span className="flex items-center text-white/50">
                    {expanded ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
                </span>
                <Terminal size={12} className="text-white/50" />
                <span className="flex-1 font-semibold text-indigo-300">{tool.name}</span>
                {tool.status === 'running' ? (
                    <Loader2 size={12} className="animate-spin text-white/60" />
                ) : tool.isError ? (
                    <XCircle size={12} className="text-red-400" />
                ) : (
                    <CheckCircle2 size={12} className="text-emerald-400" />
                )}
                {tool.duration !== undefined && (
                    <span className="text-[10px] text-white/30">{tool.duration < 1000 ? `${tool.duration}ms` : `${(tool.duration / 1000).toFixed(1)}s`}</span>
                )}
            </div>
            {expanded && (
                <div className="border-t border-white/10 px-2.5 py-2">
                    <div className="space-y-2">
                        <div>
                            <div className="mb-1 text-[10px] font-semibold uppercase tracking-wide text-white/40">Input</div>
                            <pre className="max-h-52 overflow-auto whitespace-pre-wrap break-all rounded-md bg-black/30 px-2.5 py-2 font-mono text-[11px] leading-relaxed text-white/75">
                                {formatJSON(tool.args)}
                            </pre>
                        </div>
                        {tool.result && (
                            <div>
                                <div className="mb-1 text-[10px] font-semibold uppercase tracking-wide text-white/40">Output</div>
                                <pre className="max-h-52 overflow-auto whitespace-pre-wrap break-all rounded-md bg-black/30 px-2.5 py-2 font-mono text-[11px] leading-relaxed text-white/75">
                                    {formatResult(tool.result)}
                                </pre>
                            </div>
                        )}
                    </div>
                </div>
            )}
        </div>
    )
}

function formatJSON(s: string): string {
    try {
        return JSON.stringify(JSON.parse(s), null, 2)
    } catch {
        return s
    }
}

function formatResult(s: string): string {
    try {
        const parsed = JSON.parse(s)
        if (parsed?.content) {
            const texts = parsed.content
                .filter((b: any) => b.type === 'text')
                .map((b: any) => b.text)
                .join('\n')
            if (texts) return texts.length > 500 ? texts.slice(0, 500) + '…' : texts
        }
        const pretty = JSON.stringify(parsed, null, 2)
        return pretty.length > 500 ? pretty.slice(0, 500) + '…' : pretty
    } catch {
        return s.length > 500 ? s.slice(0, 500) + '…' : s
    }
}

const AiChatView: React.FC = () => {
    const [messages, setMessages] = useState<Message[]>([])
    const [blocks, setBlocks] = useState<RenderBlock[]>([])
    const [input, setInput] = useState('')
    const [loading, setLoading] = useState(false)
    const inputRef = useRef<HTMLTextAreaElement>(null)
    const bottomRef = useRef<HTMLDivElement>(null)
    const activeAssistantIdRef = useRef<string | null>(null)
    const hasStreamedRef = useRef(false)

    useEffect(() => {
        inputRef.current?.focus()

        const getOrCreateAssistantId = () => {
            let id = activeAssistantIdRef.current
            if (id) return id
            id = `assistant-${Date.now()}-${Math.random().toString(16).slice(2)}`
            activeAssistantIdRef.current = id
            return id
        }

        const appendText = (delta: string) => {
            if (!delta) return
            hasStreamedRef.current = true
            const id = getOrCreateAssistantId()
            setBlocks(prev => {
                const next = [...prev]
                let idx = next.findIndex(b => b.id === id)
                if (idx === -1) {
                    next.push({ id, role: 'assistant', parts: [] })
                    idx = next.length - 1
                }
                const block = next[idx]
                const lastPart = block.parts[block.parts.length - 1]
                if (lastPart?.type === 'text') {
                    const updatedPart = { ...lastPart, text: (lastPart.text || '') + delta }
                    const updatedParts = [...block.parts.slice(0, -1), updatedPart]
                    next[idx] = { ...block, parts: updatedParts }
                } else {
                    next[idx] = { ...block, parts: [...block.parts, { type: 'text', text: delta }] }
                }
                return next
            })
        }

        const appendToolStart = (name: string, args: string) => {
            const id = getOrCreateAssistantId()
            setBlocks(prev => {
                const next = [...prev]
                let idx = next.findIndex(b => b.id === id)
                if (idx === -1) {
                    next.push({ id, role: 'assistant', parts: [] })
                    idx = next.length - 1
                }
                const block = next[idx]
                next[idx] = { ...block, parts: [...block.parts, { type: 'tool', tool: { name, args, status: 'running', startTime: Date.now() } }] }
                return next
            })
        }

        const updateToolEnd = (name: string, result: string, isError: boolean) => {
            setBlocks(prev => {
                const next = [...prev]
                for (let i = next.length - 1; i >= 0; i--) {
                    const block = next[i]
                    for (let j = block.parts.length - 1; j >= 0; j--) {
                        const part = block.parts[j]
                        if (part.type === 'tool' && part.tool?.name === name && part.tool?.status === 'running') {
                            const updatedTool = { ...part.tool, result, isError, status: 'done' as const, duration: Date.now() - part.tool.startTime }
                            const updatedParts = [...block.parts]
                            updatedParts[j] = { ...part, tool: updatedTool }
                            next[i] = { ...block, parts: updatedParts }
                            return next
                        }
                    }
                }
                return next
            })
        }

        const unsub = Events.On('AIEvent', (e: any) => {
            const event = (e.data ?? e) as AIEvent
            if (!event || !event.type) return
            console.log('[AIEvent]', event.type, event)
            switch (event.type) {
                case 'delta':
                    appendText(event.content || '')
                    break
                case 'tool_start':
                    appendToolStart(event.name || 'unknown', event.args || '{}')
                    break
                case 'tool_end':
                    updateToolEnd(event.name || '', event.result || '', event.isError || false)
                    break
                case 'done':
                    break
            }
        })

        return () => {
            unsub()
        }
    }, [])

    useEffect(() => {
        bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
    }, [blocks])

    const send = async () => {
        const text = input.trim()
        if (!text || loading) return

        const userMsg: Message = { role: 'user', content: text }
        setMessages(prev => [...prev, userMsg])
        hasStreamedRef.current = false
        activeAssistantIdRef.current = null
        setBlocks(prev => [...prev, { id: `user-${Date.now()}`, role: 'user', parts: [{ type: 'text', text }] }])
        setInput('')
        setLoading(true)

        try {
            const history = [...messages, userMsg].map(m => ({ role: m.role, content: m.content }))
            const result = await AIService.AskWithHistory(history as any)
            if (result && result.content) {
                setMessages(msgs => [...msgs, { role: 'assistant', content: result.content }])
                if (!hasStreamedRef.current) {
                    setBlocks(prev => {
                        const next = [...prev]
                        let id = activeAssistantIdRef.current
                        if (!id) {
                            id = `assistant-${Date.now()}`
                            activeAssistantIdRef.current = id
                            next.push({ id, role: 'assistant', parts: [] })
                        }
                        const idx = next.findIndex(b => b.id === id)
                        if (idx === -1) return next
                        const block = next[idx]
                        next[idx] = { ...block, parts: [...block.parts, { type: 'text', text: result.content }] }
                        return next
                    })
                }
            } else if (result && result.error) {
                const content = `Error: ${result.error}`
                setMessages(msgs => [...msgs, { role: 'assistant', content }])
                setBlocks(prev => [...prev, { id: `assistant-${Date.now()}`, role: 'assistant', parts: [{ type: 'text', text: content }] }])
            }
        } catch (e: any) {
            const content = `Error: ${e?.message || 'Unknown error'}`
            setMessages(msgs => [...msgs, { role: 'assistant', content }])
            setBlocks(prev => [...prev, { id: `assistant-${Date.now()}`, role: 'assistant', parts: [{ type: 'text', text: content }] }])
        } finally {
            setLoading(false)
            setTimeout(() => inputRef.current?.focus(), 50)
        }
    }

    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault()
            send()
        }
    }

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <div className="flex-1 overflow-y-auto px-4 py-4">
                <div className="flex flex-col gap-3">
                    {blocks.length === 0 && (
                        <div className="mt-6 flex flex-col items-center gap-2 text-sm text-white/40">
                            <span className="text-2xl">✨</span>
                            <span>Ask me anything</span>
                        </div>
                    )}
                    {blocks.map(block => (
                        <div
                            key={block.id}
                            className={cn("flex", block.role === 'user' ? "justify-end" : "justify-start")}
                        >
                            <div
                                className={cn(
                                    "max-w-[80%] rounded-[14px] px-3.5 py-2 text-[13px] leading-relaxed",
                                    block.role === 'user'
                                        ? "rounded-br-[4px] bg-indigo-500/30 text-indigo-100"
                                        : "rounded-tl-[4px] border border-white/10 bg-white/5 text-white/90"
                                )}
                            >
                                {block.role === 'assistant' ? (
                                    (() => {
                                        const hasTools = block.parts.some(p => p.type === 'tool')
                                        return (
                                            <div className="flex flex-col">
                                                {block.parts.map((part, idx) => {
                                                    const isLast = idx === block.parts.length - 1
                                                    return (
                                                        <div key={`${block.id}-part-${idx}`} className={cn("flex gap-2", !isLast && "items-stretch")}>
                                                            {hasTools && (
                                                                <div className="flex flex-col items-center w-3 flex-shrink-0">
                                                                    <div className="mt-[7px] h-1.5 w-1.5 rounded-full flex-shrink-0 bg-indigo-300/60" />
                                                                    {!isLast && <div className="flex-1 w-px bg-white/10 my-0.5" />}
                                                                </div>
                                                            )}
                                                            <div className={cn("flex-1 min-w-0", !isLast && "pb-2")}>
                                                                {part.type === 'tool' && part.tool && (
                                                                    <ToolCallCard tool={part.tool} />
                                                                )}
                                                                {part.type === 'text' && (
                                                                    <div className="space-y-2 text-[13px] leading-relaxed [&_p]:m-0 [&_ul]:list-disc [&_ul]:pl-5 [&_ol]:list-decimal [&_ol]:pl-5 [&_li]:mt-1 [&_code]:rounded [&_code]:bg-black/40 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-[12px] [&_pre]:rounded-lg [&_pre]:bg-black/40 [&_pre]:p-3 [&_pre_code]:bg-transparent [&_pre_code]:p-0 [&_blockquote]:border-l-2 [&_blockquote]:border-indigo-400/40 [&_blockquote]:pl-3 [&_blockquote]:text-white/70 [&_table]:w-full [&_table]:border-collapse [&_th]:border [&_td]:border [&_th]:border-white/10 [&_td]:border-white/10 [&_th]:bg-white/5 [&_th]:px-2 [&_td]:px-2 [&_th]:py-1 [&_td]:py-1 [&_a]:text-indigo-200 [&_a]:underline [&_a]:underline-offset-2">
                                                                        <ReactMarkdown remarkPlugins={[remarkGfm]}>{part.text || ''}</ReactMarkdown>
                                                                    </div>
                                                                )}
                                                            </div>
                                                        </div>
                                                    )
                                                })}
                                            </div>
                                        )
                                    })()
                                ) : (
                                    block.parts[0]?.text
                                )}
                            </div>
                        </div>
                    ))}
                    {loading && blocks.length > 0 && blocks[blocks.length - 1].role !== 'assistant' && (
                        <div className="flex justify-start">
                            <div className="max-w-[80%] rounded-[14px] rounded-tl-[4px] border border-white/10 bg-white/5 px-3.5 py-2 text-[13px] text-white/80">
                                <div className="flex items-center gap-2 text-xs text-white/60">
                                    <Loader2 size={14} className="animate-spin text-white/60" />
                                    Thinking…
                                </div>
                            </div>
                        </div>
                    )}
                    <div ref={bottomRef} />
                </div>
            </div>

            <div className="flex items-end gap-2 border-t border-white/10 px-3 py-3">
                <textarea
                    ref={inputRef}
                    className="min-h-[36px] max-h-[140px] flex-1 resize-none rounded-md border border-white/10 bg-white/5 px-3 py-2 text-[13px] text-white/90 outline-none focus-visible:ring-1 focus-visible:ring-indigo-400/40"
                    placeholder="Ask anything… (Enter to send, Shift+Enter for new line)"
                    value={input}
                    onChange={e => setInput(e.target.value)}
                    onKeyDown={handleKeyDown}
                    rows={1}
                />
                <Button
                    size="icon"
                    className="h-9 w-9 bg-indigo-500/30 text-indigo-100 hover:bg-indigo-500/50"
                    onClick={send}
                    disabled={!input.trim() || loading}
                >
                    <Send size={15} />
                </Button>
            </div>
        </div>
    )
}

export default AiChatView
