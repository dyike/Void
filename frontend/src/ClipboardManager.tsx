import React, { useState, useEffect, useRef, useCallback } from 'react'
import { FileText, Link, Image as ImageIcon, Trash2, Copy, Clipboard } from 'lucide-react'
import * as ClipboardService from '../bindings/github.com/dyike/Void/internal/service/clipboard/service.js'

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
        <div className="view-overlay" onKeyDown={handleKeyDown}>
            {/* Search */}
            <div className="search-bar" style={{ borderTop: 'none' }}>
                <input
                    ref={inputRef}
                    className="search-input"
                    placeholder="Search clipboard history…"
                    value={searchQuery}
                    onChange={e => setSearchQuery(e.target.value)}
                />
            </div>

            {/* Filter tabs */}
            <div className="filter-tabs">
                {FILTER_TYPES.map(t => (
                    <button
                        key={t}
                        className={`filter-tab${filter === t ? ' active' : ''}`}
                        onClick={() => setFilter(t)}
                    >
                        {t.charAt(0).toUpperCase() + t.slice(1)}
                    </button>
                ))}
            </div>

            {/* Split */}
            <div className="clipboard-split">
                {/* List */}
                <div className="clipboard-list">
                    {loading ? (
                        <div className="loading-spinner">Loading…</div>
                    ) : filtered.length === 0 ? (
                        <div className="empty-state" style={{ padding: 20 }}>No items</div>
                    ) : (
                        filtered.map((item, idx) => (
                            <div
                                key={item.id}
                                className={`clip-item${idx === selectedIndex ? ' selected' : ''}`}
                                onClick={() => setSelectedIndex(idx)}
                                onDoubleClick={() => pasteItem(item)}
                            >
                                <span className="clip-icon">{typeIcon(item.type)}</span>
                                <span className="clip-text">{item.preview || item.content}</span>
                            </div>
                        ))
                    )}
                </div>

                {/* Preview */}
                <div className="clipboard-preview">
                    {selectedItem ? (
                        <>
                            {selectedItem.type === 'image' ? (
                                <img
                                    src={`file://${selectedItem.content}`}
                                    alt="Clipboard"
                                    style={{ width: '100%', borderRadius: 8, border: '1px solid rgba(255,255,255,0.08)' }}
                                />
                            ) : (
                                <pre className="preview-content">{selectedItem.content}</pre>
                            )}
                            <div style={{ marginTop: 12, paddingTop: 10, borderTop: '1px solid rgba(255,255,255,0.06)', fontSize: 12, color: 'rgba(255,255,255,0.35)' }}>
                                {formatDate(selectedItem.timestamp)}
                            </div>
                        </>
                    ) : (
                        <div className="empty-state">Select an item to preview</div>
                    )}
                </div>
            </div>

            {/* Footer */}
            <div className="footer">
                <span className="footer-left">{filtered.length} items</span>
                <div className="footer-actions">
                    {selectedItem && (
                        <>
                            <button className="footer-btn" onClick={() => copyItem()} title="Copy">
                                <Copy size={13} /> Copy <kbd>⌘↵</kbd>
                            </button>
                            <button className="footer-btn" style={{ color: 'rgba(248,113,113,0.85)' }} onClick={() => deleteItem()} title="Delete">
                                <Trash2 size={13} />
                            </button>
                            <button className="footer-btn primary" onClick={() => pasteItem()}>
                                <Clipboard size={13} /> Paste <kbd>↵</kbd>
                            </button>
                        </>
                    )}
                </div>
            </div>
        </div>
    )
}

export default ClipboardManager
