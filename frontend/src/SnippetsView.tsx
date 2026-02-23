import React, { useState, useEffect, useRef } from 'react'
import { Plus, Edit2, Trash2, Copy, Search, X } from 'lucide-react'
import * as SnippetsService from '../bindings/github.com/dyike/Void/internal/service/snippets/service.js'

interface Snippet {
    id: string
    name: string
    content: string
    prefix: string
    tags: string[]
    createdAt: number
    updatedAt: number
}

const SnippetsView: React.FC = () => {
    const [snippets, setSnippets] = useState<Snippet[]>([])
    const [query, setQuery] = useState('')
    const [selected, setSelected] = useState<Snippet | null>(null)
    const [editing, setEditing] = useState(false)
    const [form, setForm] = useState({ name: '', content: '', prefix: '', tags: '' })
    const [copied, setCopied] = useState(false)
    const searchRef = useRef<HTMLInputElement>(null)

    const load = async (q = query) => {
        try {
            const result = await SnippetsService.GetSnippets(q)
            setSnippets((result || []) as Snippet[])
        } catch (e) {
            console.error(e)
        }
    }

    useEffect(() => {
        load('')
        searchRef.current?.focus()
    }, [])

    useEffect(() => {
        const t = setTimeout(() => load(), 200)
        return () => clearTimeout(t)
    }, [query])

    const startCreate = () => {
        setSelected(null)
        setForm({ name: '', content: '', prefix: '', tags: '' })
        setEditing(true)
    }

    const startEdit = (sn: Snippet) => {
        setSelected(sn)
        setForm({
            name: sn.name,
            content: sn.content,
            prefix: sn.prefix || '',
            tags: sn.tags?.join(', ') || '',
        })
        setEditing(true)
    }

    const save = async () => {
        const tags = form.tags ? form.tags.split(',').map(t => t.trim()).filter(Boolean) : []
        try {
            if (selected) {
                await SnippetsService.UpdateSnippet(selected.id, form.name, form.content, form.prefix, tags)
            } else {
                await SnippetsService.CreateSnippet(form.name, form.content, form.prefix, tags)
            }
            setEditing(false)
            load()
        } catch (e) {
            console.error(e)
        }
    }

    const deleteSnippet = async (sn: Snippet) => {
        try {
            await SnippetsService.DeleteSnippet(sn.id)
            if (selected?.id === sn.id) setSelected(null)
            load()
        } catch (e) {
            console.error(e)
        }
    }

    const copy = async (sn: Snippet) => {
        try {
            await SnippetsService.CopyToClipboard(sn.id)
            setCopied(true)
            setTimeout(() => setCopied(false), 1500)
        } catch (e) {
            console.error(e)
        }
    }

    if (editing) {
        return (
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', padding: 20, gap: 14 }}>
                <div style={{ fontWeight: 600, fontSize: 14, color: 'rgba(255,255,255,0.8)' }}>
                    {selected ? 'Edit Snippet' : 'New Snippet'}
                </div>
                <input
                    className="settings-input"
                    style={{ width: '100%' }}
                    placeholder="Name"
                    value={form.name}
                    onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                    autoFocus
                />
                <input
                    className="settings-input"
                    style={{ width: '100%' }}
                    placeholder="Prefix (e.g. ;addr)"
                    value={form.prefix}
                    onChange={e => setForm(f => ({ ...f, prefix: e.target.value }))}
                />
                <textarea
                    className="settings-input"
                    style={{ width: '100%', minHeight: 120, resize: 'vertical', lineHeight: 1.6 }}
                    placeholder="Snippet content…"
                    value={form.content}
                    onChange={e => setForm(f => ({ ...f, content: e.target.value }))}
                />
                <input
                    className="settings-input"
                    style={{ width: '100%' }}
                    placeholder="Tags (comma-separated)"
                    value={form.tags}
                    onChange={e => setForm(f => ({ ...f, tags: e.target.value }))}
                />
                <div style={{ display: 'flex', gap: 8 }}>
                    <button className="settings-save-btn" onClick={save}>Save</button>
                    <button
                        className="settings-save-btn"
                        style={{ background: 'rgba(255,255,255,0.07)', color: 'rgba(255,255,255,0.6)' }}
                        onClick={() => setEditing(false)}
                    >
                        Cancel
                    </button>
                </div>
            </div>
        )
    }

    return (
        <div className="view-overlay" style={{ display: 'flex', flexDirection: 'column' }}>
            {/* Search */}
            <div className="search-bar">
                <Search size={14} style={{ color: 'rgba(255,255,255,0.35)', flexShrink: 0 }} />
                <input
                    ref={searchRef}
                    className="search-input"
                    placeholder="Search snippets…"
                    value={query}
                    onChange={e => setQuery(e.target.value)}
                />
                {query && (
                    <button className="search-clear" onClick={() => setQuery('')}><X size={13} /></button>
                )}
                <button
                    className="footer-btn primary"
                    style={{ flexShrink: 0 }}
                    onClick={startCreate}
                >
                    <Plus size={13} /> New
                </button>
            </div>

            {/* List + Preview */}
            <div className="clipboard-split" style={{ flex: 1, minHeight: 0 }}>
                <div className="clipboard-list">
                    {snippets.length === 0
                        ? <div className="empty-state" style={{ padding: 20 }}>No snippets yet</div>
                        : snippets.map(sn => (
                            <div
                                key={sn.id}
                                className={`clip-item${selected?.id === sn.id ? ' selected' : ''}`}
                                onClick={() => setSelected(sn)}
                                onDoubleClick={() => copy(sn)}
                            >
                                <span className="clip-text">
                                    {sn.prefix && (
                                        <span style={{ marginRight: 5, opacity: 0.5, fontSize: 11, fontFamily: 'monospace' }}>
                                            {sn.prefix}
                                        </span>
                                    )}
                                    {sn.name}
                                </span>
                            </div>
                        ))
                    }
                </div>

                <div className="clipboard-preview">
                    {selected
                        ? <>
                            <div style={{ fontSize: 13, fontWeight: 600, color: 'rgba(255,255,255,0.85)', marginBottom: 8 }}>
                                {selected.name}
                            </div>
                            {selected.prefix && (
                                <div style={{ fontSize: 12, fontFamily: 'monospace', color: 'rgba(255,255,255,0.4)', marginBottom: 10 }}>
                                    prefix: {selected.prefix}
                                </div>
                            )}
                            <pre className="preview-content">{selected.content}</pre>
                            {selected.tags?.length > 0 && (
                                <div style={{ marginTop: 10, display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                                    {selected.tags.map(t => (
                                        <span key={t} style={{ fontSize: 11, padding: '2px 8px', borderRadius: 4, background: 'rgba(99,102,241,0.18)', color: 'rgba(180,183,255,0.7)' }}>
                                            {t}
                                        </span>
                                    ))}
                                </div>
                            )}
                        </>
                        : <div className="empty-state">Select a snippet to preview</div>
                    }
                </div>
            </div>

            {/* Footer */}
            <div className="footer">
                <span className="footer-left">{snippets.length} snippets</span>
                <div className="footer-actions">
                    {selected && (
                        <>
                            <button className="footer-btn" onClick={() => startEdit(selected)}>
                                <Edit2 size={13} /> Edit
                            </button>
                            <button
                                className="footer-btn"
                                style={{ color: 'rgba(248,113,113,0.85)' }}
                                onClick={() => deleteSnippet(selected)}
                            >
                                <Trash2 size={13} />
                            </button>
                            <button className="footer-btn primary" onClick={() => copy(selected)}>
                                <Copy size={13} /> {copied ? 'Copied!' : 'Copy'} <kbd>↵</kbd>
                            </button>
                        </>
                    )}
                </div>
            </div>
        </div>
    )
}

export default SnippetsView
