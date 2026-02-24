import React, { useState, useEffect, useRef, useCallback } from 'react'
import * as SettingsService from '../bindings/github.com/dyike/Void/internal/service/settings/service.js'
import { AppSettings } from '../bindings/github.com/dyike/Void/internal/service/settings/models.js'

// ─── Icons ─────────────────────────────────────────────────────────────────

const Icons = {
    general: <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" /></svg>,
    extensions: <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><rect x="7" y="7" width="10" height="10" rx="2" ry="2" /><path d="M7 12H3M21 12h-4M12 7V3M12 21v-4M8.5 7v10M15.5 7v10M7 8.5h10M7 15.5h10" /></svg>,
    ai: <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><path d="m12 3-1.912 5.813a2 2 0 0 1-1.275 1.275L3 12l5.813 1.912a2 2 0 0 1 1.275 1.275L12 21l1.912-5.813a2 2 0 0 1 1.275-1.275L21 12l-5.813-1.912a2 2 0 0 1-1.275-1.275L12 3Z" /><path d="M5 3v4M3 5h4" /></svg>,
    advanced: <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><path d="M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z" /></svg>,
    about: <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10" /><line x1="12" y1="16" x2="12" y2="12" /><line x1="12" y1="8" x2="12.01" y2="8" /></svg>,
}

// ─── Hotkey Recorder ─────────────────────────────────────────────────────

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
            className={`rc-hotkey-btn${recording ? ' recording' : ''}`}
            onClick={() => { if (recording) { setRecording(false); setPending('') } else { setRecording(true); setPending('Record shortcut…'); ref.current?.focus() } }}
            onKeyDown={handleKeyDown}
            onBlur={() => { setRecording(false); setPending('') }}
            tabIndex={0}
        >
            {recording ? (pending || 'Record shortcut…') : (display || value)}
        </button>
    )
}

// ─── Tabs ─────────────────────────────────────────────────────────────────

type Tab = 'general' | 'extensions' | 'ai' | 'advanced' | 'about'

const TABS: { id: Tab; label: string; icon: React.ReactNode }[] = [
    { id: 'general', label: 'General', icon: Icons.general },
    { id: 'extensions', label: 'Extensions', icon: Icons.extensions },
    { id: 'ai', label: 'AI', icon: Icons.ai },
    { id: 'advanced', label: 'Advanced', icon: Icons.advanced },
    { id: 'about', label: 'About', icon: Icons.about },
]

// ─── Settings View ────────────────────────────────────────────────────────

const SettingsView: React.FC = () => {
    const [settings, setSettings] = useState<AppSettings | null>(null)
    const [saving, setSaving] = useState(false)
    const [saved, setSaved] = useState(false)
    const [hotkeyStatus, setHotkeyStatus] = useState<'idle' | 'saved' | 'error'>('idle')
    const [activeTab, setActiveTab] = useState<Tab>('general')

    useEffect(() => {
        SettingsService.GetSettings()
            .then(s => setSettings(s as AppSettings))
            .catch(console.error)
    }, [])

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

    // Auto-save when settings change (debounced or on blur conceptually, but explicit save is okay for now,
    // though Raycast auto-saves. Let's just keep the explicit button for non-hotkey fields or auto-save.)
    // For simplicity, we'll keep the button.

    if (!settings) return <div className="rc-loading">Loading…</div>

    return (
        <div className="rc-root">
            {/* ── Header Toolbar ── */}
            <div className="rc-toolbar-title">Void Settings</div>

            <div className="rc-tabbar">
                {TABS.map(t => (
                    <button
                        key={t.id}
                        className={`rc-tab${activeTab === t.id ? ' active' : ''}`}
                        onClick={() => setActiveTab(t.id)}
                    >
                        <div className="rc-tab-icon">{t.icon}</div>
                        <div className="rc-tab-label">{t.label}</div>
                    </button>
                ))}
            </div>

            {/* ── Content Area ── */}
            <div className="rc-content">
                <div className="rc-scroll-view">
                    {activeTab === 'general' && (
                        <div className="rc-section">
                            <Row label="Startup">
                                <label className="rc-checkbox">
                                    <input type="checkbox" disabled />
                                    Launch Void at login
                                </label>
                            </Row>
                            <Row label="Void Hotkey">
                                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                    <HotkeyRecorder value={settings.globalShortcut} onChange={(spec) => handleHotkeyChange(spec)} />
                                    {hotkeyStatus === 'saved' && <span style={{ color: '#10b981', fontSize: 13 }}>✓</span>}
                                    {hotkeyStatus === 'error' && <span style={{ color: '#ef4444', fontSize: 13 }}>✗</span>}
                                </div>
                            </Row>
                            <Row label="Menu Bar Icon">
                                <label className="rc-checkbox">
                                    <input type="checkbox" defaultChecked disabled />
                                    Show Void in menu bar
                                </label>
                            </Row>

                            <div className="rc-divider" />

                            <Row label="Text Size">
                                <div className="rc-segmented-control">
                                    <button
                                        className={`rc-segment ${settings.fontSize === 'small' ? 'active' : ''}`}
                                        onClick={() => update('fontSize', 'small')}
                                    >
                                        <span style={{ fontSize: '13px', fontWeight: 600 }}>Aa</span>
                                    </button>
                                    <button
                                        className={`rc-segment ${settings.fontSize === 'medium' ? 'active' : ''}`}
                                        onClick={() => update('fontSize', 'medium')}
                                    >
                                        <span style={{ fontSize: '15px', fontWeight: 600 }}>Aa</span>
                                    </button>
                                </div>
                            </Row>

                            <Row label="Appearance">
                                <div className="rc-appearance-group">
                                    <button className="rc-appearance-btn">
                                        <div className="rc-appearance-icon light">
                                            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="5" /><line x1="12" y1="1" x2="12" y2="3" /><line x1="12" y1="21" x2="12" y2="23" /><line x1="4.22" y1="4.22" x2="5.64" y2="5.64" /><line x1="18.36" y1="18.36" x2="19.78" y2="19.78" /><line x1="1" y1="12" x2="3" y2="12" /><line x1="21" y1="12" x2="23" y2="12" /><line x1="4.22" y1="19.78" x2="5.64" y2="18.36" /><line x1="18.36" y1="4.22" x2="19.78" y2="5.64" /></svg>
                                        </div>
                                        <span>Light</span>
                                    </button>
                                    <button className="rc-appearance-btn">
                                        <div className="rc-appearance-icon dark">
                                            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" /></svg>
                                        </div>
                                        <span>Dark</span>
                                    </button>
                                    <button className="rc-appearance-btn active">
                                        <div className="rc-appearance-icon system">
                                            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="10" /><path d="M12 2a10 10 0 0 0 0 20Z" fill="currentColor" /></svg>
                                        </div>
                                        <span>System</span>
                                    </button>
                                </div>
                            </Row>
                        </div>
                    )}

                    {activeTab === 'ai' && (
                        <div className="rc-section">
                            <Row label="Provider">
                                <select className="rc-select" value={settings.ai.provider} onChange={e => updateAI('provider', e.target.value as any)}>
                                    <option value="openai">OpenAI</option>
                                    <option value="ollama">Ollama</option>
                                    <option value="openai-compatible">OpenAI-Compatible</option>
                                </select>
                            </Row>
                            {settings.ai.provider === 'openai' && (<>
                                <Row label="API Key"><input className="rc-input" type="password" value={settings.ai.openaiApiKey} onChange={e => updateAI('openaiApiKey', e.target.value)} placeholder="sk-..." /></Row>
                                <Row label="Model"><input className="rc-input" value={settings.ai.defaultModel} onChange={e => updateAI('defaultModel', e.target.value)} placeholder="gpt-4o-mini" /></Row>
                            </>)}
                            {settings.ai.provider === 'ollama' && (<>
                                <Row label="URL"><input className="rc-input" value={settings.ai.ollamaBaseUrl} onChange={e => updateAI('ollamaBaseUrl', e.target.value)} placeholder="http://localhost:11434" /></Row>
                                <Row label="Model"><input className="rc-input" value={settings.ai.defaultModel} onChange={e => updateAI('defaultModel', e.target.value)} placeholder="llama3" /></Row>
                            </>)}
                            {settings.ai.provider === 'openai-compatible' && (<>
                                <Row label="Base URL"><input className="rc-input" value={settings.ai.openaiCompatibleBaseUrl} onChange={e => updateAI('openaiCompatibleBaseUrl', e.target.value)} placeholder="https://api.example.com/v1" /></Row>
                                <Row label="API Key"><input className="rc-input" type="password" value={settings.ai.openaiCompatibleApiKey} onChange={e => updateAI('openaiCompatibleApiKey', e.target.value)} /></Row>
                                <Row label="Model"><input className="rc-input" value={settings.ai.openaiCompatibleModel} onChange={e => updateAI('openaiCompatibleModel', e.target.value)} /></Row>
                            </>)}
                        </div>
                    )}

                    {activeTab === 'extensions' && (
                        <div className="rc-empty">Manage your extensions here (Coming Soon)</div>
                    )}

                    {activeTab === 'advanced' && (
                        <div className="rc-section">
                            <Row label="Data Directory">
                                <span style={{ fontSize: 13, color: '#404040', fontFamily: 'monospace' }}>~/Library/Application Support/Void</span>
                            </Row>
                        </div>
                    )}

                    {activeTab === 'about' && (
                        <div className="rc-empty">Void Launcher<br />Version 1.0.0</div>
                    )}
                </div>
            </div>

            {/* Save Banner */}
            {(activeTab === 'general' || activeTab === 'ai') && (
                <div className="rc-footer">
                    <button className="rc-save-btn" onClick={save} disabled={saving}>
                        {saving ? 'Saving...' : saved ? 'Settings Saved' : 'Save Changes'}
                    </button>
                    <span className="rc-footer-hint">Some changes require a restart</span>
                </div>
            )}

            <style>{STYLES}</style>
        </div>
    )
}

// ─── Shared Components ────────────────────────────────────────────────────

function Row({ label, children, ...props }: { label: string; children?: React.ReactNode;[key: string]: any }) {
    return (
        <div className="rc-row" {...props}>
            <div className="rc-row-label">{label}</div>
            <div className="rc-row-control">{children}</div>
        </div>
    )
}

// ─── Styles ───────────────────────────────────────────────────────────────

const STYLES = `
html, body, #root { margin:0; padding:0; height:100%; }

.rc-root {
    display: flex;
    flex-direction: column;
    height: 100vh;
    background: #282828; /* Less black, more dark gray */
    color: #e0e0e0;
    font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", "Segoe UI", sans-serif;
    -webkit-font-smoothing: antialiased;
    user-select: none;
}

.rc-loading { display:flex; align-items:center; justify-content:center; height:100vh; color:#999; font-size:14px; }

/* ── Toolbar & Tab Bar ── */
.rc-toolbar-title {
    position: absolute;
    top: 0; left: 0; right: 0;
    height: 38px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 13px;
    font-weight: 600;
    color: #b0b0b0;
    -webkit-app-region: drag;
    pointer-events: none;
}

.rc-tabbar {
    display: flex;
    justify-content: center;
    gap: 16px;
    padding: 38px 24px 0;
    border-bottom: 1px solid #3e3e3e;
    -webkit-app-region: drag;
}

.rc-tab {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 6px;
    padding: 10px 18px 12px;
    border: none;
    background: transparent;
    cursor: pointer;
    color: #a0a0a0;
    border-radius: 8px 8px 0 0;
    transition: all 0.1s ease;
    -webkit-app-region: no-drag;
}
.rc-tab:hover { background: #383838; color: #e0e0e0; }
.rc-tab.active { background: #383838; color: #ffffff; }

.rc-tab-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    color: inherit;
}

.rc-tab-label {
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.2px;
}

/* ── Content ── */
.rc-content {
    flex: 1;
    overflow: hidden;
    position: relative;
    background: #282828;
}
.rc-scroll-view {
    position: absolute;
    top: 0; left: 0; right: 0; bottom: 0;
    overflow-y: auto;
    padding: 32px 0 64px;
}
.rc-section {
    max-width: 600px;
    margin: 0 auto;
}
.rc-divider {
    height: 1px;
    background: #3e3e3e;
    margin: 32px auto;
    width: calc(100% - 64px);
    max-width: 536px;
}

/* ── Layout Rows ── */
.rc-row {
    display: grid;
    grid-template-columns: 160px 1fr;
    gap: 24px;
    padding: 12px 32px;
    align-items: center;
}
.rc-row-label {
    text-align: right;
    font-size: 13px;
    font-weight: 600;
    color: #a0a0a0;
}
.rc-row-control {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
}

/* ── Controls ── */
.rc-checkbox {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: #e0e0e0;
    cursor: pointer;
}
.rc-checkbox input[type="checkbox"] {
    margin: 0;
    width: 14px;
    height: 14px;
    accent-color: #ffffff;
}

.rc-hotkey-btn {
    background: #383838;
    border: 1px solid #484848;
    border-radius: 8px;
    padding: 8px 24px;
    color: #e0e0e0;
    font-size: 15px;
    font-weight: 600;
    font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", sans-serif;
    cursor: pointer;
    min-width: 280px;
    text-align: center;
    transition: all 0.1s;
}
.rc-hotkey-btn:hover { background: #484848; border-color: #5a5a5a; }
.rc-hotkey-btn.recording {
    background: #23355c;
    border-color: #3b82f6;
    color: #93c5fd;
    box-shadow: 0 0 0 3px rgba(59,130,246,0.3);
}

.rc-segmented-control {
    display: inline-flex;
    background: #383838;
    padding: 3px;
    border-radius: 8px;
    border: 1px solid #484848;
}
.rc-segment {
    background: transparent;
    border: none;
    border-radius: 6px;
    padding: 4px 12px;
    color: #b0b0b0;
    font-size: 14px;
    cursor: pointer;
    transition: all 0.1s;
    min-width: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
}
.rc-segment:hover { color: #ffffff; }
.rc-segment.active {
    background: #5a5a5a;
    color: #ffffff;
    box-shadow: 0 1px 3px rgba(0,0,0,0.3);
}

/* ── Appearance Group ── */
.rc-appearance-group {
    display: flex;
    gap: 24px;
    margin-top: 4px;
}
.rc-appearance-btn {
    background: none;
    border: none;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    color: #b0b0b0;
    font-size: 12px;
    font-weight: 500;
}
.rc-appearance-btn:hover { color: #ffffff; }
.rc-appearance-btn.active { color: #ffffff; font-weight: 600; }
.rc-appearance-icon {
    width: 36px;
    height: 36px;
    border-radius: 20px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #a0a0a0;
    border: 1px solid #5a5a5a;
    background: #383838;
    transition: all 0.1s;
}
.rc-appearance-btn:hover .rc-appearance-icon { border-color: #7a7a7a; }
.rc-appearance-btn.active .rc-appearance-icon {
    background: #484848;
    border-color: #ffffff;
    border-width: 2px;
    color: #ffffff;
}

/* ── Inputs ── */
.rc-input, .rc-select {
    background: #282828;
    border: 1px solid #484848;
    border-radius: 6px;
    padding: 6px 12px;
    color: #e0e0e0;
    font-size: 13px;
    font-family: inherit;
    width: 280px;
    outline: none;
    transition: all 0.1s;
    box-sizing: border-box;
}
.rc-input:focus, .rc-select:focus {
    border-color: #3b82f6;
    background: #333333;
    box-shadow: 0 0 0 3px rgba(59,130,246,0.25);
}
.rc-input::placeholder { color: #808080; }

.rc-empty { text-align: center; padding: 64px 0; color: #808080; font-size: 14px; }

/* ── Footer ── */
.rc-footer {
    padding: 16px 32px;
    border-top: 1px solid #3e3e3e;
    background: #2f2f2f;
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 16px;
}
.rc-save-btn {
    background: #e0e0e0;
    border: none;
    border-radius: 6px;
    padding: 6px 16px;
    color: #1c1c1e;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
}
.rc-save-btn:hover { background: #ffffff; }
.rc-save-btn:disabled { opacity: 0.5; }
.rc-footer-hint { font-size: 12px; color: #a0a0a0; }
`

export default SettingsView
