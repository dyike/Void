import React, { useState, useRef, useEffect } from 'react'
import { Send, Loader2, ChevronDown, ChevronRight, Terminal, CheckCircle2, XCircle } from 'lucide-react'
import * as AIService from '../bindings/github.com/dyike/Void/internal/service/ai/service.js'
import { Events } from '@wailsio/runtime'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

interface Message {
    role: 'user' | 'assistant'
    content: string
    toolCalls?: ToolCall[]
}

interface ToolCall {
    name: string
    args: string
    result?: string
    isError?: boolean
    status: 'running' | 'done'
}

const ToolCallCard: React.FC<{ tool: ToolCall }> = ({ tool }) => {
    const [expanded, setExpanded] = useState(false)

    return (
        <div className={`tool-call-card ${tool.status} ${tool.isError ? 'error' : ''}`}>
            <div className="tool-call-header" onClick={() => setExpanded(!expanded)}>
                <span className="tool-call-chevron">
                    {expanded ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
                </span>
                <Terminal size={12} className="tool-call-icon" />
                <span className="tool-call-name">{tool.name}</span>
                {tool.status === 'running' ? (
                    <Loader2 size={12} className="tool-call-spinner" />
                ) : tool.isError ? (
                    <XCircle size={12} className="tool-call-status error" />
                ) : (
                    <CheckCircle2 size={12} className="tool-call-status success" />
                )}
            </div>
            {expanded && (
                <div className="tool-call-details">
                    <div className="tool-call-section">
                        <div className="tool-call-label">Input</div>
                        <pre className="tool-call-code">{formatJSON(tool.args)}</pre>
                    </div>
                    {tool.result && (
                        <div className="tool-call-section">
                            <div className="tool-call-label">Output</div>
                            <pre className="tool-call-code">{formatResult(tool.result)}</pre>
                        </div>
                    )}
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
        // If result has nested content, try to extract text
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
    const [input, setInput] = useState('')
    const [loading, setLoading] = useState(false)
    const [streamedContent, setStreamedContent] = useState('')
    const [agentAction, setAgentAction] = useState('')
    const [toolCalls, setToolCalls] = useState<ToolCall[]>([])
    const inputRef = useRef<HTMLTextAreaElement>(null)
    const bottomRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
        inputRef.current?.focus()

        const unsubUpdate = Events.On('AgentUpdate', (e: any) => {
            setStreamedContent(prev => prev + (e.data[0] || ''))
        })
        const unsubAction = Events.On('AgentAction', (e: any) => {
            setAgentAction(e.data[0] || '')
        })
        const unsubToolStart = Events.On('ToolCallStart', (e: any) => {
            const name = e.data[0] || 'unknown'
            const args = e.data[1] || '{}'
            console.log('[ToolCallStart]', name, args)
            setToolCalls(prev => [...prev, { name, args, status: 'running' }])
        })
        const unsubToolEnd = Events.On('ToolCallEnd', (e: any) => {
            const name = e.data[0] || ''
            const result = e.data[1] || ''
            const isError = e.data[2] || false
            console.log('[ToolCallEnd]', name, isError)
            setToolCalls(prev => {
                const updated = [...prev]
                for (let i = updated.length - 1; i >= 0; i--) {
                    if (updated[i].name === name && updated[i].status === 'running') {
                        updated[i] = { ...updated[i], result, isError, status: 'done' }
                        break
                    }
                }
                return updated
            })
        })

        return () => {
            unsubUpdate()
            unsubAction()
            unsubToolStart()
            unsubToolEnd()
        }
    }, [])

    useEffect(() => {
        bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
    }, [messages, streamedContent, agentAction, toolCalls])

    const send = async () => {
        const text = input.trim()
        if (!text || loading) return

        const userMsg: Message = { role: 'user', content: text }
        setMessages(prev => [...prev, userMsg])
        setInput('')
        setLoading(true)
        setStreamedContent('')
        setAgentAction('')
        setToolCalls([])

        try {
            const history = [...messages, userMsg].map(m => ({ role: m.role, content: m.content }))
            const result = await AIService.AskWithHistory(history as any)
            if (result && result.content) {
                setToolCalls(prev => {
                    setMessages(msgs => [...msgs, { role: 'assistant', content: result.content, toolCalls: prev.length > 0 ? prev : undefined }])
                    return []
                })
            } else if (result && result.error) {
                setToolCalls(prev => {
                    setMessages(msgs => [...msgs, { role: 'assistant', content: `Error: ${result.error}`, toolCalls: prev.length > 0 ? prev : undefined }])
                    return []
                })
            }
        } catch (e: any) {
            setToolCalls(prev => {
                setMessages(msgs => [...msgs, { role: 'assistant', content: `Error: ${e?.message || 'Unknown error'}`, toolCalls: prev.length > 0 ? prev : undefined }])
                return []
            })
        } finally {
            setLoading(false)
            setStreamedContent('')
            setAgentAction('')
            setToolCalls([])
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
        <div className="ai-chat">
            <div className="ai-messages">
                {messages.length === 0 && (
                    <div className="empty-state" style={{ marginTop: 32 }}>
                        <span style={{ fontSize: 28 }}>✨</span>
                        <span>Ask me anything</span>
                    </div>
                )}
                {messages.map((msg, i) => (
                    <div key={i} className={`ai-msg ${msg.role}`}>
                        <div className="ai-bubble">
                            {msg.role === 'assistant' && msg.toolCalls && msg.toolCalls.length > 0 && (
                                <div className="tool-calls-container" style={{ marginBottom: 8 }}>
                                    {msg.toolCalls.map((tc, j) => (
                                        <ToolCallCard key={j} tool={tc} />
                                    ))}
                                </div>
                            )}
                            {msg.role === 'assistant' ? (
                                <ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.content}</ReactMarkdown>
                            ) : (
                                msg.content
                            )}
                        </div>
                    </div>
                ))}
                {loading && (
                    <div className="ai-msg assistant">
                        <div className="ai-bubble" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                            {/* Tool call cards */}
                            {toolCalls.length > 0 && (
                                <div className="tool-calls-container">
                                    {toolCalls.map((tc, i) => (
                                        <ToolCallCard key={i} tool={tc} />
                                    ))}
                                </div>
                            )}
                            {/* Agent action indicator */}
                            {agentAction && (
                                <div className="agent-action-line">
                                    <Loader2 size={12} className="tool-call-spinner" />
                                    <span>{agentAction}</span>
                                </div>
                            )}
                            {/* Streamed content or thinking indicator */}
                            {streamedContent ? (
                                <div className="markdown-body" style={{ background: 'transparent', padding: 0 }}>
                                    <ReactMarkdown remarkPlugins={[remarkGfm]}>{streamedContent}</ReactMarkdown>
                                </div>
                            ) : !agentAction && toolCalls.length === 0 && (
                                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                                    <Loader2 size={14} className="tool-call-spinner" />
                                    Thinking…
                                </div>
                            )}
                        </div>
                    </div>
                )}
                <div ref={bottomRef} />
            </div>

            <div className="ai-input-row">
                <textarea
                    ref={inputRef}
                    className="ai-input"
                    placeholder="Ask anything… (Enter to send, Shift+Enter for new line)"
                    value={input}
                    onChange={e => setInput(e.target.value)}
                    onKeyDown={handleKeyDown}
                    rows={1}
                    style={{ minHeight: 36, maxHeight: 120 }}
                />
                <button
                    className="ai-send-btn"
                    onClick={send}
                    disabled={!input.trim() || loading}
                >
                    <Send size={15} />
                </button>
            </div>

            <style>{`
                @keyframes spin { to { transform: rotate(360deg); } }

                .tool-calls-container {
                    display: flex;
                    flex-direction: column;
                    gap: 6px;
                }

                .tool-call-card {
                    border: 1px solid rgba(255,255,255,0.08);
                    border-radius: 8px;
                    background: rgba(0,0,0,0.2);
                    overflow: hidden;
                    font-size: 12px;
                }
                .tool-call-card.error {
                    border-color: rgba(255,80,80,0.2);
                }

                .tool-call-header {
                    display: flex;
                    align-items: center;
                    gap: 6px;
                    padding: 6px 10px;
                    cursor: pointer;
                    user-select: none;
                    transition: background 0.15s;
                }
                .tool-call-header:hover {
                    background: rgba(255,255,255,0.04);
                }

                .tool-call-chevron {
                    opacity: 0.5;
                    display: flex;
                    align-items: center;
                }

                .tool-call-icon {
                    opacity: 0.5;
                }

                .tool-call-name {
                    font-weight: 600;
                    color: #a78bfa;
                    flex: 1;
                }

                .tool-call-spinner {
                    animation: spin 1s linear infinite;
                    opacity: 0.6;
                }

                .tool-call-status.success {
                    color: #4ade80;
                }
                .tool-call-status.error {
                    color: #f87171;
                }

                .tool-call-details {
                    border-top: 1px solid rgba(255,255,255,0.06);
                    padding: 8px 10px;
                    display: flex;
                    flex-direction: column;
                    gap: 8px;
                }

                .tool-call-section {}

                .tool-call-label {
                    font-size: 10px;
                    text-transform: uppercase;
                    letter-spacing: 0.5px;
                    color: rgba(255,255,255,0.4);
                    margin-bottom: 4px;
                    font-weight: 600;
                }

                .tool-call-code {
                    background: rgba(0,0,0,0.3);
                    border-radius: 6px;
                    padding: 8px 10px;
                    margin: 0;
                    font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
                    font-size: 11px;
                    line-height: 1.5;
                    color: rgba(255,255,255,0.75);
                    overflow-x: auto;
                    white-space: pre-wrap;
                    word-break: break-all;
                    max-height: 200px;
                    overflow-y: auto;
                }

                .agent-action-line {
                    display: flex;
                    align-items: center;
                    gap: 6px;
                    font-size: 12px;
                    opacity: 0.6;
                    font-style: italic;
                }
            `}</style>
        </div>
    )
}

export default AiChatView
