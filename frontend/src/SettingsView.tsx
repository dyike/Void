import React, { useState, useEffect, useRef, useCallback } from 'react'
import * as SettingsService from '../bindings/github.com/dyike/Void/internal/service/settings/service.js'
import { AppSettings, Skill } from '../bindings/github.com/dyike/Void/internal/service/settings/models.js'
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Checkbox } from "@/components/ui/checkbox"
import { Button } from "@/components/ui/button"
import { Settings, Puzzle, Cpu, Wrench, Info, FolderGit2, Sun, Moon, Monitor } from 'lucide-react'

// Hotkey Recorder
interface HotkeyRecorderProps {
    value: string
    onChange: (spec: string, display: string) => void
}

function specToDisplay(spec: string): string {
    const modSymbols: Record<string, string> = {
        option: '⌥', alt: '⌥', cmd: '⌘', command: '⌘',
        ctrl: '⌃', control: '⌃', shift: '⇧',
    }
    const keyDisplay: Record<string, string> = {
        space: 'Space', return: '↩', enter: '↩', tab: '⇥',
        delete: '⌫', backspace: '⌫', escape: '⎋', esc: '⎋',
        up: '↑', down: '↓', left: '←', right: '→',
    }
    const parts = spec.toLowerCase().split('+')
    return parts.map((p, i) => {
        if (i < parts.length - 1) return modSymbols[p] || p.toUpperCase()
        return keyDisplay[p] || (p.charAt(0).toUpperCase() + p.slice(1))
    }).join(' ')
}

const MODIFIER_KEYS = new Set(['Meta', 'Alt', 'Control', 'Shift'])

const HotkeyRecorder: React.FC<HotkeyRecorderProps> = ({ value, onChange }) => {
    const [recording, setRecording] = useState(false)
    const [display, setDisplay] = useState(specToDisplay(value))
    const [pending, setPending] = useState('')
    const ref = useRef<HTMLButtonElement>(null)

    useEffect(() => { setDisplay(specToDisplay(value)) }, [value])

    const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
        if (!recording) return
        e.preventDefault(); e.stopPropagation()
        if (MODIFIER_KEYS.has(e.key)) {
            const mods: string[] = []
            if (e.metaKey) mods.push('⌘')
            if (e.altKey) mods.push('⌥')
            if (e.ctrlKey) mods.push('⌃')
            if (e.shiftKey) mods.push('⇧')
            setPending(mods.join(' ') + ' …'); return
        }
        const mods: string[] = []
        if (e.metaKey) mods.push('cmd')
        if (e.altKey) mods.push('option')
        if (e.ctrlKey) mods.push('ctrl')
        if (e.shiftKey) mods.push('shift')
        let key = e.key.toLowerCase()
        if (key === ' ' || key === '\u00a0') key = 'space'
        else if (key === 'enter') key = 'return'
        else if (key === 'backspace') key = 'delete'
        const spec = [...mods, key].join('+')
        setDisplay(specToDisplay(spec)); setPending(''); setRecording(false)
        onChange(spec, specToDisplay(spec))
    }, [recording, onChange])

    return (
        <button
            ref={ref}
            className={`px-6 py-2 rounded-lg border flex-1 text-sm font-medium transition-all min-w-[280px]
                ${recording
                    ? 'bg-indigo-500/15 border-indigo-500/40 text-indigo-300 ring-2 ring-indigo-500/20 shadow-none'
                    : 'bg-white/5 border-white/10 text-white/90 hover:bg-white/10'}`}
            onClick={() => { if (recording) { setRecording(false); setPending('') } else { setRecording(true); setPending('Record shortcut…'); ref.current?.focus() } }}
            onKeyDown={handleKeyDown}
            onBlur={() => { setRecording(false); setPending('') }}
            tabIndex={0}
        >
            {recording ? (pending || 'Record shortcut…') : (display || value)}
        </button>
    )
}

function Row({ label, children, alignTop = false }: { label: string; children?: React.ReactNode, alignTop?: boolean }) {
    return (
        <div className={`grid grid-cols-[160px_1fr] gap-6 py-2.5 px-8 items-${alignTop ? 'start' : 'center'}`}>
            <div className="text-right text-[13px] font-medium text-white/50 tracking-wide mt-1">
                {label}
            </div>
            <div className="flex flex-col items-start gap-2 w-full">
                {children}
            </div>
        </div>
    )
}

const SettingsView: React.FC = () => {
    const [settings, setSettings] = useState<AppSettings | null>(null)
    const [saving, setSaving] = useState(false)
    const [saved, setSaved] = useState(false)
    const [hotkeyStatus, setHotkeyStatus] = useState<'idle' | 'saved' | 'error'>('idle')
    const [activeTab, setActiveTab] = useState('general')

    const [skillsList, setSkillsList] = useState<Skill[]>([])

    useEffect(() => {
        SettingsService.NotifySettingsReady().catch(() => {})
    }, [])

    useEffect(() => {
        SettingsService.GetSettings()
            .then(s => setSettings(s as AppSettings))
            .catch(console.error)
    }, [])

    useEffect(() => {
        if (activeTab === 'skills') {
            SettingsService.GetSkills()
                .then(s => setSkillsList(s || []))
                .catch(console.error)
        }
    }, [activeTab, settings?.skillsPath])

    useEffect(() => {
        if (activeTab === 'about') {
            SettingsService.ResizeSettingsWindow(780, 320).catch(console.error)
        } else {
            SettingsService.ResizeSettingsWindow(780, 520).catch(console.error)
        }
    }, [activeTab])

    const update = <K extends keyof AppSettings>(key: K, value: any) => {
        setSettings(prev => prev ? { ...prev, [key]: value } as AppSettings : prev)
    }
    const updateAI = <K extends keyof AppSettings['ai']>(key: K, value: any) => {
        setSettings(prev => prev ? { ...prev, ai: { ...prev.ai, [key]: value } } as AppSettings : prev)
    }

    const handleHotkeyChange = async (spec: string) => {
        update('globalShortcut', spec)
        try {
            await SettingsService.SetGlobalHotkey(spec)
            setHotkeyStatus('saved'); setTimeout(() => setHotkeyStatus('idle'), 2000)
        } catch { setHotkeyStatus('error'); setTimeout(() => setHotkeyStatus('idle'), 2000) }
    }

    const save = async () => {
        if (!settings) return
        setSaving(true)
        try {
            const u = AppSettings.createFrom(settings)
            await SettingsService.SaveSettings(u)
            setSaved(true); setTimeout(() => setSaved(false), 2000)
        } catch (e) { console.error(e) } finally { setSaving(false) }
    }

    if (!settings) return <div className="flex items-center justify-center h-screen text-white/40 text-sm">Loading…</div>

    return (
        <div className="flex flex-col h-screen text-white/90 font-sans select-none overflow-hidden bg-[#0e0e12]/80 backdrop-blur-3xl" style={{ WebkitAppRegion: 'drag' } as any}>
            <div className="absolute top-0 left-0 right-0 h-[38px] flex items-center justify-center text-[13px] font-semibold text-white/50 pointer-events-none">
                Void Settings
            </div>

            <Tabs value={activeTab} onValueChange={setActiveTab} className="flex flex-col flex-1 overflow-hidden" defaultValue="general">
                <div className="flex justify-center pt-[38px]">
                    <TabsList className="bg-transparent border-b border-white/10 rounded-none h-auto p-0 gap-4 flex w-full justify-center px-6">
                        <TabsTrigger value="general" className="w-[72px] flex flex-col gap-1.5 px-2 py-3 rounded-t-lg border-b-2 border-transparent data-[state=active]:border-white/90 data-[state=active]:bg-white/5 data-[state=active]:text-white data-[state=inactive]:text-white/40 shadow-none -mb-[1px]" style={{ WebkitAppRegion: 'no-drag' } as any}>
                            <Settings className="w-[22px] h-[22px]" />
                            <span className="text-xs font-semibold tracking-wide">General</span>
                        </TabsTrigger>
                        <TabsTrigger value="extensions" className="w-[72px] flex flex-col gap-1.5 px-2 py-3 rounded-t-lg border-b-2 border-transparent data-[state=active]:border-white/90 data-[state=active]:bg-white/5 data-[state=active]:text-white data-[state=inactive]:text-white/40 shadow-none -mb-[1px]" style={{ WebkitAppRegion: 'no-drag' } as any}>
                            <Puzzle className="w-[22px] h-[22px]" />
                            <span className="text-xs font-semibold tracking-wide">Extensions</span>
                        </TabsTrigger>
                        <TabsTrigger value="skills" className="w-[72px] flex flex-col gap-1.5 px-2 py-3 rounded-t-lg border-b-2 border-transparent data-[state=active]:border-white/90 data-[state=active]:bg-white/5 data-[state=active]:text-white data-[state=inactive]:text-white/40 shadow-none -mb-[1px]" style={{ WebkitAppRegion: 'no-drag' } as any}>
                            <FolderGit2 className="w-[22px] h-[22px]" />
                            <span className="text-xs font-semibold tracking-wide">Skills</span>
                        </TabsTrigger>
                        <TabsTrigger value="ai" className="w-[72px] flex flex-col gap-1.5 px-2 py-3 rounded-t-lg border-b-2 border-transparent data-[state=active]:border-white/90 data-[state=active]:bg-white/5 data-[state=active]:text-white data-[state=inactive]:text-white/40 shadow-none -mb-[1px]" style={{ WebkitAppRegion: 'no-drag' } as any}>
                            <Cpu className="w-[22px] h-[22px]" />
                            <span className="text-xs font-semibold tracking-wide">AI</span>
                        </TabsTrigger>
                        <TabsTrigger value="advanced" className="w-[72px] flex flex-col gap-1.5 px-2 py-3 rounded-t-lg border-b-2 border-transparent data-[state=active]:border-white/90 data-[state=active]:bg-white/5 data-[state=active]:text-white data-[state=inactive]:text-white/40 shadow-none -mb-[1px]" style={{ WebkitAppRegion: 'no-drag' } as any}>
                            <Wrench className="w-[22px] h-[22px]" />
                            <span className="text-xs font-semibold tracking-wide">Advanced</span>
                        </TabsTrigger>
                        <TabsTrigger value="about" className="w-[72px] flex flex-col gap-1.5 px-2 py-3 rounded-t-lg border-b-2 border-transparent data-[state=active]:border-white/90 data-[state=active]:bg-white/5 data-[state=active]:text-white data-[state=inactive]:text-white/40 shadow-none -mb-[1px]" style={{ WebkitAppRegion: 'no-drag' } as any}>
                            <Info className="w-[22px] h-[22px]" />
                            <span className="text-xs font-semibold tracking-wide">About</span>
                        </TabsTrigger>
                    </TabsList>
                </div>

                <div className="flex-1 overflow-y-auto w-full" style={{ WebkitAppRegion: 'no-drag' } as any}>
                    {activeTab === 'about' ? (
                        <TabsContent value="about" className="mt-0 outline-none h-full flex items-center justify-center" style={{ minHeight: 180 }}>
                            <div className="flex flex-col items-center gap-2">
                                <span className="text-white/80 font-semibold tracking-wide">Void Launcher</span>
                                <span className="text-white/40 text-sm">Version 1.0.0</span>
                            </div>
                        </TabsContent>
                    ) : (
                        <div className="max-w-[600px] mx-auto py-8">

                            <TabsContent value="general" className="mt-0 outline-none">
                                <Row label="Startup">
                                    <div className="flex items-center space-x-2 opacity-60">
                                        <Checkbox id="startup" disabled checked={false} />
                                        <label htmlFor="startup" className="text-[13px] font-medium leading-none cursor-not-allowed text-white/80">Launch Void at login</label>
                                    </div>
                                </Row>

                                <Row label="Void Hotkey">
                                    <div className="flex items-center gap-2">
                                        <HotkeyRecorder value={settings.globalShortcut} onChange={(s) => handleHotkeyChange(s)} />
                                        {hotkeyStatus === 'saved' && <span className="text-emerald-500 text-[13px]">✓</span>}
                                        {hotkeyStatus === 'error' && <span className="text-red-500 text-[13px]">✗</span>}
                                    </div>
                                </Row>

                                <Row label="Menu Bar Icon">
                                    <div className="flex items-center space-x-2 opacity-60">
                                        <Checkbox id="menubar" disabled checked={true} />
                                        <label htmlFor="menubar" className="text-[13px] font-medium leading-none cursor-not-allowed text-white/80">Show Void in menu bar</label>
                                    </div>
                                </Row>

                                <div className="h-[1px] bg-white/10 my-6 w-full max-w-[536px] mx-auto" />

                                <Row label="Terminal App">
                                    <Select value={settings.terminalApp || 'Terminal'} onValueChange={v => update('terminalApp', v)}>
                                        <SelectTrigger className="w-[280px]">
                                            <SelectValue placeholder="Select terminal" />
                                        </SelectTrigger>
                                        <SelectContent>
                                            <SelectItem value="Terminal">Terminal</SelectItem>
                                            <SelectItem value="iTerm">iTerm</SelectItem>
                                            <SelectItem value="Warp">Warp</SelectItem>
                                            <SelectItem value="Alacritty">Alacritty</SelectItem>
                                            <SelectItem value="WezTerm">WezTerm</SelectItem>
                                            <SelectItem value="Ghostty">Ghostty</SelectItem>
                                        </SelectContent>
                                    </Select>
                                </Row>

                                <Row label="Text Size">
                                    <div className="inline-flex bg-white/5 p-1 rounded-lg border border-white/5">
                                        <button
                                            className={`px-3 py-1 text-sm rounded-md transition-all ${settings.fontSize === 'small' ? 'bg-white/15 text-white/95 shadow-sm' : 'text-white/40 hover:text-white/80'}`}
                                            onClick={() => update('fontSize', 'small')}
                                        >
                                            <span className="text-[13px] font-semibold">Aa</span>
                                        </button>
                                        <button
                                            className={`px-3 py-1 text-sm rounded-md transition-all ${settings.fontSize === 'medium' ? 'bg-white/15 text-white/95 shadow-sm' : 'text-white/40 hover:text-white/80'}`}
                                            onClick={() => update('fontSize', 'medium')}
                                        >
                                            <span className="text-[15px] font-semibold">Aa</span>
                                        </button>
                                    </div>
                                </Row>

                                <Row label="Appearance">
                                    <div className="flex gap-6 mt-1">
                                        <button className="flex flex-col items-center gap-2 group text-white/40 hover:text-white/80">
                                            <div className="w-9 h-9 flex items-center justify-center rounded-full border border-white/10 bg-white/5 group-hover:border-white/30 transition-all">
                                                <Sun className="w-5 h-5" />
                                            </div>
                                            <span className="text-xs font-medium">Light</span>
                                        </button>
                                        <button className="flex flex-col items-center gap-2 group text-white/40 hover:text-white/80">
                                            <div className="w-9 h-9 flex items-center justify-center rounded-full border border-white/10 bg-white/5 group-hover:border-white/30 transition-all">
                                                <Moon className="w-5 h-5" />
                                            </div>
                                            <span className="text-xs font-medium">Dark</span>
                                        </button>
                                        <button className="flex flex-col items-center gap-2 text-white/95 font-semibold">
                                            <div className="w-9 h-9 flex items-center justify-center rounded-full border-2 border-indigo-500/50 bg-indigo-500/15 text-indigo-200 transition-all">
                                                <Monitor className="w-5 h-5" />
                                            </div>
                                            <span className="text-xs">System</span>
                                        </button>
                                    </div>
                                </Row>
                            </TabsContent>

                            <TabsContent value="skills" className="mt-0 outline-none">
                                <Row label="Skills Path">
                                    <Input
                                        className="w-[280px]"
                                        placeholder="e.g. ~/my-skills"
                                        value={settings.skillsPath || ''}
                                        onChange={e => update('skillsPath', e.target.value)}
                                    />
                                </Row>
                                <div className="h-[1px] bg-border my-6 w-full max-w-[536px] mx-auto" />
                                <Row label="Discovered Skills" alignTop>
                                    <div className="grid grid-cols-2 gap-3 w-full">
                                        {skillsList.length === 0 ? (
                                            <div className="text-white/40 text-[13px] col-span-2">
                                                No skills found in this directory. Add a valid path above. <br />
                                                <span className="opacity-80 mt-1 block">(Tip: Clone https://github.com/dyike/eino-skills)</span>
                                            </div>
                                        ) : (
                                            skillsList.map((skill, idx) => (
                                                <div
                                                    key={idx}
                                                    title={skill.description || 'No description available in SKILL.md'}
                                                    className="group flex flex-col justify-start bg-white/5 p-3 rounded-lg border border-white/10 cursor-pointer transition-all hover:bg-white/10 hover:-translate-y-0.5"
                                                >
                                                    <div className="flex items-center gap-2 mb-1.5">
                                                        <FolderGit2 className="w-4 h-4 text-indigo-400 group-hover:text-indigo-300 transition-colors" />
                                                        <span className="font-semibold text-[14px] text-white/90 truncate">
                                                            {skill.name}
                                                        </span>
                                                    </div>
                                                    <span className="text-xs text-white/40 break-all line-clamp-2 leading-relaxed">
                                                        {skill.description || skill.path}
                                                    </span>
                                                </div>
                                            ))
                                        )}
                                    </div>
                                </Row>
                            </TabsContent>

                            <TabsContent value="ai" className="mt-0 outline-none">
                                <Row label="Provider">
                                    <Select value={settings.ai.provider} onValueChange={(v: any) => updateAI('provider', v)}>
                                        <SelectTrigger className="w-[280px] bg-white/5 border-white/10 text-white/90 focus:ring-indigo-500/20 focus:border-indigo-500">
                                            <SelectValue placeholder="Select AI provider" />
                                        </SelectTrigger>
                                        <SelectContent className="bg-zinc-900 border-zinc-800 text-white/90">
                                            <SelectItem value="openai">OpenAI</SelectItem>
                                            <SelectItem value="ollama">Ollama</SelectItem>
                                            <SelectItem value="openai-compatible">OpenAI-Compatible</SelectItem>
                                        </SelectContent>
                                    </Select>
                                </Row>

                                {settings.ai.provider === 'openai' && (<>
                                    <Row label="API Key">
                                        <Input type="password" placeholder="sk-..." className="w-[280px] bg-white/5 border-white/10 text-white/90 focus:ring-indigo-500/20 focus:border-indigo-500" value={settings.ai.openaiApiKey} onChange={e => updateAI('openaiApiKey', e.target.value)} />
                                    </Row>
                                    <Row label="Model">
                                        <Input placeholder="gpt-4o-mini" className="w-[280px] bg-white/5 border-white/10 text-white/90 focus:ring-indigo-500/20 focus:border-indigo-500" value={settings.ai.defaultModel} onChange={e => updateAI('defaultModel', e.target.value)} />
                                    </Row>
                                </>)}

                                {settings.ai.provider === 'ollama' && (<>
                                    <Row label="URL">
                                        <Input placeholder="http://localhost:11434" className="w-[280px] bg-white/5 border-white/10 text-white/90 focus:ring-indigo-500/20 focus:border-indigo-500" value={settings.ai.ollamaBaseUrl} onChange={e => updateAI('ollamaBaseUrl', e.target.value)} />
                                    </Row>
                                    <Row label="Model">
                                        <Input placeholder="llama3" className="w-[280px] bg-white/5 border-white/10 text-white/90 focus:ring-indigo-500/20 focus:border-indigo-500" value={settings.ai.defaultModel} onChange={e => updateAI('defaultModel', e.target.value)} />
                                    </Row>
                                </>)}

                                {settings.ai.provider === 'openai-compatible' && (<>
                                    <Row label="Base URL">
                                        <Input placeholder="https://api.example.com/v1" className="w-[280px] bg-white/5 border-white/10 text-white/90 focus:ring-indigo-500/20 focus:border-indigo-500" value={settings.ai.openaiCompatibleBaseUrl} onChange={e => updateAI('openaiCompatibleBaseUrl', e.target.value)} />
                                    </Row>
                                    <Row label="API Key">
                                        <Input type="password" placeholder="sk-..." className="w-[280px] bg-white/5 border-white/10 text-white/90 focus:ring-indigo-500/20 focus:border-indigo-500" value={settings.ai.openaiCompatibleApiKey} onChange={e => updateAI('openaiCompatibleApiKey', e.target.value)} />
                                    </Row>
                                    <Row label="Model">
                                        <Input placeholder="deepseek-chat" className="w-[280px] bg-white/5 border-white/10 text-white/90 focus:ring-indigo-500/20 focus:border-indigo-500" value={settings.ai.openaiCompatibleModel} onChange={e => updateAI('openaiCompatibleModel', e.target.value)} />
                                    </Row>
                                </>)}
                            </TabsContent>

                            <TabsContent value="extensions" className="mt-0 outline-none flex items-center justify-center p-16">
                                <span className="text-white/30 text-sm">Manage your extensions here (Coming Soon)</span>
                            </TabsContent>

                            <TabsContent value="advanced" className="mt-0 outline-none">
                                <Row label="Data Directory">
                                    <span className="text-[13px] text-white/60 font-mono tracking-tight bg-white/5 px-3 py-1.5 rounded border border-white/5">~/Library/Application Support/Void</span>
                                </Row>
                            </TabsContent>

                        </div>
                    )}
                </div>

                {/* Save Banner */}
                {(activeTab === 'general' || activeTab === 'ai') && (
                    <div className="border-t border-white/5 bg-white/[0.015] px-8 py-4 flex items-center justify-end gap-4" style={{ WebkitAppRegion: 'no-drag' } as any}>
                        <span className="text-xs text-white/30">Some changes require a restart</span>
                        <Button
                            variant="secondary"
                            size="sm"
                            className="bg-indigo-500/20 hover:bg-indigo-500/30 text-indigo-200 border-none"
                            onClick={save}
                            disabled={saving}
                        >
                            {saving ? 'Saving...' : saved ? 'Settings Saved' : 'Save Changes'}
                        </Button>
                    </div>
                )}
            </Tabs>
        </div>
    )
}

export default SettingsView
