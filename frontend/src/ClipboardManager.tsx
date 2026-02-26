import React, { useState, useEffect, useRef, useCallback } from 'react'
import { FileText, Link, Image as ImageIcon, Trash2, Copy, Clipboard } from 'lucide-react'
import * as ClipboardService from '../bindings/github.com/dyike/Void/internal/service/clipboardservice.js'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

interface ClipboardItem {
    id: string
    type: 'text' | 'url' | 'image' | 'file'
    content: string
    preview?: string
    timestamp: number
    metadata?: Record<string, any>
}

interface Props { onClose: () => void }

const FILTER_TYPES = ['all', 'text', 'url', 'image', 'file'] as const
type FilterType = typeof FILTER_TYPES[number]

const ClipboardManager: React.FC<Props> = ({ onClose }) => {
    const [items, setItems] = useState<ClipboardItem[]>([])
    const [filter, setFilter] = useState<FilterType>('all')
    const [searchQuery, setSearchQuery] = useState('')
    const [selectedIndex, setSelectedIndex] = useState(0)
    const [loading, setLoading] = useState(true)
    const inputRef = useRef<HTMLInputElement>(null)

    const loadHistory = useCallback(async () => {
        try {
            const history = await ClipboardService.GetHistory()
            setItems(history as ClipboardItem[])
        } catch (e) {
            console.error('clipboard error:', e)
        } finally {
            setLoading(false)
        }
    }, [])

    useEffect(() => {
        loadHistory()
        inputRef.current?.focus()
        const interval = setInterval(() => loadHistory(), 1000)
        return () => clearInterval(interval)
    }, [loadHistory])

    const filtered = items
        .filter(item => filter === 'all' || item.type === filter)
        .filter(item => {
            if (!searchQuery.trim()) return true
            const q = searchQuery.toLowerCase()
            return (item.preview || item.content).toLowerCase().includes(q)
        })

    useEffect(() => { setSelectedIndex(0) }, [filter, searchQuery])

    const pasteItem = async (item?: ClipboardItem) => {
        const target = item ?? filtered[selectedIndex]
        if (!target) return
        try { await ClipboardService.PasteItem(target.id) } catch (e) { console.error(e) }
    }

    const copyItem = async () => {
        const target = filtered[selectedIndex]
        if (!target) return
        try { await ClipboardService.CopyItem(target.id) } catch (e) { console.error(e) }
    }

    const deleteItem = async (item?: ClipboardItem) => {
        const target = item ?? filtered[selectedIndex]
        if (!target) return
        try {
            await ClipboardService.DeleteItem(target.id)
            loadHistory()
        } catch (e) { console.error(e) }
    }

    const handleKeyDown = (e: React.KeyboardEvent) => {
        switch (e.key) {
            case 'ArrowDown':
                e.preventDefault()
                setSelectedIndex(i => Math.min(i + 1, filtered.length - 1))
                break
            case 'ArrowUp':
                e.preventDefault()
                setSelectedIndex(i => Math.max(i - 1, 0))
                break
            case 'Enter':
                e.preventDefault()
                pasteItem()
                break
            case 'Escape':
                onClose()
                break
        }
    }

    const selectedItem = filtered[selectedIndex]

    const formatDate = (ts: number) =>
        new Date(ts).toLocaleString(undefined, {
            month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit',
        })

    const typeIcon = (type: string) => {
        if (type === 'url') return <Link size={14} />
        if (type === 'image') return <ImageIcon size={14} />
        return <FileText size={14} />
    }

    return (
        <div className="flex min-h-0 flex-1 flex-col" onKeyDown={handleKeyDown}>
            <div className="border-b border-border px-4 py-3">
                <Input
                    ref={inputRef}
                    className="h-8 border-input bg-secondary/30 text-[14px] text-foreground placeholder:text-muted-foreground/60 focus-visible:ring-indigo-400/30"
                    placeholder="Search clipboard history…"
                    value={searchQuery}
                    onChange={e => setSearchQuery(e.target.value)}
                />
            </div>

            <div className="flex items-center gap-2 border-b border-border px-3 py-2">
                {FILTER_TYPES.map(t => (
                    <button
                        key={t}
                        className={cn(
                            "rounded-md px-2.5 py-1 text-xs font-medium transition",
                            filter === t
                                ? "bg-accent text-foreground"
                                : "text-muted-foreground hover:bg-accent/50 hover:text-foreground/80"
                        )}
                        onClick={() => setFilter(t)}
                    >
                        {t.charAt(0).toUpperCase() + t.slice(1)}
                    </button>
                ))}
            </div>

            <div className="flex min-h-0 flex-1">
                <div className="w-[40%] overflow-y-auto border-r border-border p-2">
                    {loading ? (
                        <div className="flex items-center justify-center px-4 py-8 text-xs text-muted-foreground">Loading…</div>
                    ) : filtered.length === 0 ? (
                        <div className="flex items-center justify-center px-4 py-8 text-xs text-muted-foreground">No items</div>
                    ) : (
                        filtered.map((item, idx) => (
                            <div
                                key={item.id}
                                className={cn(
                                    "flex cursor-pointer items-start gap-2 rounded-md border border-transparent px-2 py-2 text-sm transition",
                                    idx === selectedIndex ? "border-indigo-400/40 bg-indigo-500/20" : "hover:bg-accent"
                                )}
                                onClick={() => setSelectedIndex(idx)}
                                onDoubleClick={() => pasteItem(item)}
                            >
                                <span className="mt-0.5 text-muted-foreground">{typeIcon(item.type)}</span>
                                <span className="truncate text-[13px] text-foreground/80">{item.preview || item.content}</span>
                            </div>
                        ))
                    )}
                </div>

                <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-4">
                    {selectedItem ? (
                        <>
                            {selectedItem.type === 'image' ? (
                                <img
                                    src={`file://${selectedItem.content}`}
                                    alt="Clipboard"
                                    className="w-full rounded-lg border border-border"
                                />
                            ) : (
                                <pre className="whitespace-pre-wrap break-all rounded-lg border border-border bg-secondary/30 p-3 text-xs text-foreground/80">
                                    {selectedItem.content}
                                </pre>
                            )}
                            <div className="border-t border-border pt-2 text-xs text-muted-foreground">
                                {formatDate(selectedItem.timestamp)}
                            </div>
                        </>
                    ) : (
                        <div className="flex flex-1 items-center justify-center text-xs text-muted-foreground">Select an item to preview</div>
                    )}
                </div>
            </div>

            <div className="flex items-center justify-between gap-2 border-t border-border px-3 py-2">
                <span className="text-xs text-muted-foreground/60">{filtered.length} items</span>
                <div className="flex items-center gap-2">
                    {selectedItem && (
                        <>
                            <Button
                                variant="secondary"
                                size="sm"
                                className="bg-secondary text-secondary-foreground hover:bg-accent"
                                onClick={() => copyItem()}
                                title="Copy"
                            >
                                <Copy size={13} />
                                Copy
                                <kbd className="ml-1 inline-flex h-5 min-w-5 items-center justify-center rounded bg-secondary px-1 text-[11px] text-muted-foreground">⌘↵</kbd>
                            </Button>
                            <Button
                                variant="secondary"
                                size="sm"
                                className="bg-secondary text-red-600 dark:text-red-300 hover:bg-accent"
                                onClick={() => deleteItem()}
                                title="Delete"
                            >
                                <Trash2 size={13} />
                            </Button>
                            <Button
                                size="sm"
                                className="bg-indigo-500/30 text-indigo-600 dark:text-indigo-100 hover:bg-indigo-500/45"
                                onClick={() => pasteItem()}
                            >
                                <Clipboard size={13} />
                                Paste
                                <kbd className="ml-1 inline-flex h-5 min-w-5 items-center justify-center rounded bg-indigo-500/30 px-1 text-[11px] text-indigo-600 dark:text-indigo-100">↵</kbd>
                            </Button>
                        </>
                    )}
                </div>
            </div>
        </div>
    )
}

export default ClipboardManager
