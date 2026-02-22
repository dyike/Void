import React, { useState, useRef, useEffect } from 'react'
import { Send, Loader2 } from 'lucide-react'
import * as AIService from '../bindings/github.com/dyike/Void/internal/service/ai/service.js'

interface Message {
    role: 'user' | 'assistant'
    content: string
}

const AiChatView: React.FC = () => {
    const [messages, setMessages] = useState<Message[]>([])
    const [input, setInput] = useState('')
    const [loading, setLoading] = useState(false)
    const inputRef = useRef<HTMLTextAreaElement>(null)
    const bottomRef = useRef<HTMLDivElement>(null)

    useEffect(() => {
        inputRef.current?.focus()
    }, [])

    useEffect(() => {
        bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
    }, [messages])

    const send = async () => {
        const text = input.trim()
        if (!text || loading) return

        const userMsg: Message = { role: 'user', content: text }
        setMessages(prev => [...prev, userMsg])
        setInput('')
        setLoading(true)

        try {
            // Build full history including new user message
            const history = [...messages, userMsg].map(m => ({ role: m.role, content: m.content }))
            const result = await AIService.AskWithHistory(history)
            if (result && result.content) {
                setMessages(prev => [...prev, { role: 'assistant', content: result.content }])
            } else if (result && result.error) {
                setMessages(prev => [...prev, { role: 'assistant', content: `Error: ${result.error}` }])
            }
        } catch (e: any) {
            setMessages(prev => [...prev, { role: 'assistant', content: `Error: ${e?.message || 'Unknown error'}` }])
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
                        <div className="ai-bubble">{msg.content}</div>
                    </div>
                ))}
                {loading && (
                    <div className="ai-msg assistant">
                        <div className="ai-bubble" style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                            <Loader2 size={14} style={{ animation: 'spin 1s linear infinite' }} />
                            Thinking…
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
      `}</style>
        </div>
    )
}

export default AiChatView
