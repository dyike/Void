import React, { useState, useEffect, useRef } from 'react'
import { Plus, Edit2, Trash2, Copy, Search, X } from 'lucide-react'
import * as SnippetsService from '../bindings/github.com/dyike/Void/internal/service/snippets/service.js'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

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
            <div className="flex flex-1 flex-col gap-3 px-5 py-4">
                <div className="text-sm font-semibold text-foreground/80">
                    {selected ? 'Edit Snippet' : 'New Snippet'}
                </div>
                <Input
                    className="h-9 border-input bg-secondary/30 text-foreground placeholder:text-muted-foreground/60 focus-visible:ring-indigo-400/30"
                    placeholder="Name"
                    value={form.name}
                    onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                    autoFocus
                />
                <Input
                    className="h-9 border-input bg-secondary/30 text-foreground placeholder:text-muted-foreground/60 focus-visible:ring-indigo-400/30"
                    placeholder="Prefix (e.g. ;addr)"
                    value={form.prefix}
                    onChange={e => setForm(f => ({ ...f, prefix: e.target.value }))}
                />
                <textarea
                    className="min-h-[120px] w-full resize-y rounded-md border border-input bg-secondary/30 px-3 py-2 text-sm text-foreground outline-none placeholder:text-muted-foreground/60 focus-visible:ring-1 focus-visible:ring-indigo-400/30"
                    placeholder="Snippet content…"
                    value={form.content}
                    onChange={e => setForm(f => ({ ...f, content: e.target.value }))}
                />
                <Input
                    className="h-9 border-input bg-secondary/30 text-foreground placeholder:text-muted-foreground/60 focus-visible:ring-indigo-400/30"
                    placeholder="Tags (comma-separated)"
                    value={form.tags}
                    onChange={e => setForm(f => ({ ...f, tags: e.target.value }))}
                />
                <div className="flex items-center gap-2">
                    <Button className="bg-indigo-500/30 text-indigo-600 dark:text-indigo-100 hover:bg-indigo-500/45" onClick={save}>
                        Save
                    </Button>
                    <Button
                        variant="secondary"
                        className="bg-secondary text-secondary-foreground hover:bg-accent"
                        onClick={() => setEditing(false)}
                    >
                        Cancel
                    </Button>
                </div>
            </div>
        )
    }

    return (
        <div className="flex min-h-0 flex-1 flex-col">
            <div className="flex items-center gap-2 border-b border-border px-4 py-3">
                <Search size={14} className="text-muted-foreground" />
                <Input
                    ref={searchRef}
                    className="h-8 border-0 bg-transparent px-0 text-[14px] text-foreground placeholder:text-muted-foreground/60 shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
                    placeholder="Search snippets…"
                    value={query}
                    onChange={e => setQuery(e.target.value)}
                />
                {query && (
                    <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7 text-muted-foreground hover:text-foreground"
                        onClick={() => setQuery('')}
                    >
                        <X size={13} />
                    </Button>
                )}
                <Button
                    size="sm"
                    className="ml-auto bg-indigo-500/30 text-indigo-600 dark:text-indigo-100 hover:bg-indigo-500/45"
                    onClick={startCreate}
                >
                    <Plus size={13} />
                    New
                </Button>
            </div>

            <div className="flex min-h-0 flex-1">
                <div className="w-[40%] overflow-y-auto border-r border-border p-2">
                    {snippets.length === 0
                        ? <div className="flex items-center justify-center px-4 py-8 text-xs text-muted-foreground">No snippets yet</div>
                        : snippets.map(sn => (
                            <div
                                key={sn.id}
                                className={cn(
                                    "flex cursor-pointer items-center gap-2 rounded-md border border-transparent px-2 py-2 text-sm transition",
                                    selected?.id === sn.id ? "border-indigo-400/40 bg-indigo-500/20" : "hover:bg-accent"
                                )}
                                onClick={() => setSelected(sn)}
                                onDoubleClick={() => copy(sn)}
                            >
                                <span className="truncate text-[13px] text-foreground/80">
                                    {sn.prefix && (
                                        <span className="mr-1.5 font-mono text-[11px] text-muted-foreground">
                                            {sn.prefix}
                                        </span>
                                    )}
                                    {sn.name}
                                </span>
                            </div>
                        ))
                    }
                </div>

                <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto p-4">
                    {selected
                        ? <>
                            <div className="text-sm font-semibold text-foreground/85">
                                {selected.name}
                            </div>
                            {selected.prefix && (
                                <div className="font-mono text-xs text-muted-foreground">
                                    prefix: {selected.prefix}
                                </div>
                            )}
                            <pre className="whitespace-pre-wrap break-all rounded-lg border border-border bg-secondary/30 p-3 text-xs text-foreground/80">
                                {selected.content}
                            </pre>
                            {selected.tags?.length > 0 && (
                                <div className="mt-1 flex flex-wrap gap-1.5">
                                    {selected.tags.map(t => (
                                        <span
                                            key={t}
                                            className="rounded-full bg-indigo-500/20 px-2 py-0.5 text-[11px] text-indigo-600 dark:text-indigo-200/80"
                                        >
                                            {t}
                                        </span>
                                    ))}
                                </div>
                            )}
                        </>
                        : <div className="flex flex-1 items-center justify-center text-xs text-muted-foreground">Select a snippet to preview</div>
                    }
                </div>
            </div>

            <div className="flex items-center justify-between gap-2 border-t border-border px-3 py-2">
                <span className="text-xs text-muted-foreground/60">{snippets.length} snippets</span>
                <div className="flex items-center gap-2">
                    {selected && (
                        <>
                            <Button
                                variant="secondary"
                                size="sm"
                                className="bg-secondary text-secondary-foreground hover:bg-accent"
                                onClick={() => startEdit(selected)}
                            >
                                <Edit2 size={13} />
                                Edit
                            </Button>
                            <Button
                                variant="secondary"
                                size="sm"
                                className="bg-secondary text-red-600 dark:text-red-300 hover:bg-accent"
                                onClick={() => deleteSnippet(selected)}
                            >
                                <Trash2 size={13} />
                            </Button>
                            <Button
                                size="sm"
                                className="bg-indigo-500/30 text-indigo-600 dark:text-indigo-100 hover:bg-indigo-500/45"
                                onClick={() => copy(selected)}
                            >
                                <Copy size={13} />
                                {copied ? 'Copied!' : 'Copy'}
                                <kbd className="ml-1 inline-flex h-5 min-w-5 items-center justify-center rounded bg-indigo-500/30 px-1 text-[11px] text-indigo-600 dark:text-indigo-100">↵</kbd>
                            </Button>
                        </>
                    )}
                </div>
            </div>
        </div>
    )
}

export default SnippetsView
