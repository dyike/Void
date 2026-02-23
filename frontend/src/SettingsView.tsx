import React, { useState, useEffect, useRef, useCallback } from 'react'
import * as SettingsService from '../bindings/github.com/dyike/Void/internal/service/settings/service.js'

interface AppSettings {
    globalShortcut: string
    fontSize: 'small' | 'medium' | 'large'
    baseColor: string
    ai: {
        provider: 'openai' | 'anthropic' | 'ollama' | 'openai-compatible'
        openaiApiKey: string
        ollamaBaseUrl: string
        defaultModel: string
        openaiCompatibleBaseUrl: string
        openaiCompatibleApiKey: string
        openaiCompatibleModel: string
        enabled: boolean
    }
}

// ─── Hotkey Recorder ─────────────────────────────────────────────────────

interface HotkeyRecorderProps {
    value: string            // current spec, e.g. "option+space"
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
        return keyDisplay[p] || p.toUpperCase()
    }).join('')
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
        e.preventDefault()
        e.stopPropagation()

        if (MODIFIER_KEYS.has(e.key)) {
            // Show pending modifier
            const mods: string[] = []
            if (e.altKey) mods.push('⌥')
            if (e.metaKey) mods.push('⌘')
            if (e.ctrlKey) mods.push('⌃')
            if (e.shiftKey) mods.push('⇧')
            setPending(mods.join('') + '…')
            return
        }

        // Build spec string
        const mods: string[] = []
        if (e.altKey) mods.push('option')
        if (e.metaKey) mods.push('cmd')
        if (e.ctrlKey) mods.push('ctrl')
        if (e.shiftKey) mods.push('shift')

        let key = e.key.toLowerCase()
        // Normalize
        if (key === ' ') key = 'space'
        else if (key === 'enter') key = 'return'
        else if (key === 'backspace') key = 'delete'

        const spec = [...mods, key].join('+')
        const disp = specToDisplay(spec)
        setDisplay(disp)
        setPending('')
        setRecording(false)
        onChange(spec, disp)
    }, [recording, onChange])

    const start = () => {
        setRecording(true)
        setPending('Press shortcut…')
        ref.current?.focus()
    }

    const cancel = () => {
        setRecording(false)
        setPending('')
    }

    return (
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <button
                ref={ref}
                className={`hotkey-recorder${recording ? ' recording' : ''}`}
                onClick={recording ? cancel : start}
                onKeyDown={handleKeyDown}
                onBlur={() => { setRecording(false); setPending('') }}
                tabIndex={0}
                title={recording ? 'Press your shortcut, or click to cancel' : 'Click to record shortcut'}
            >
                {recording ? (pending || 'Press shortcut…') : (display || value)}
            </button>
            {!recording && (
                <span style={{ fontSize: 11, color: 'rgba(255,255,255,0.3)' }}>{value}</span>
            )}
        </div>
    )
}

// ─── Settings View ────────────────────────────────────────────────────────

const SettingsView: React.FC = () => {
    const [settings, setSettings] = useState<AppSettings | null>(null)
    const [saving, setSaving] = useState(false)
    const [saved, setSaved] = useState(false)
    const [hotkeyStatus, setHotkeyStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle')

    useEffect(() => {
        SettingsService.GetSettings()
            .then(s => setSettings(s as AppSettings))
            .catch(console.error)
    }, [])

    const update = <K extends keyof AppSettings>(key: K, value: AppSettings[K]) => {
        setSettings(prev => prev ? { ...prev, [key]: value } : prev)
    }

    const updateAI = <K extends keyof AppSettings['ai']>(key: K, value: AppSettings['ai'][K]) => {
        setSettings(prev => prev ? { ...prev, ai: { ...prev.ai, [key]: value } } : prev)
    }

    const handleHotkeyChange = async (spec: string) => {
        update('globalShortcut', spec)
        setHotkeyStatus('saving')
        try {
            await SettingsService.SetGlobalHotkey(spec)
            setHotkeyStatus('saved')
            setTimeout(() => setHotkeyStatus('idle'), 2000)
        } catch {
            setHotkeyStatus('error')
            setTimeout(() => setHotkeyStatus('idle'), 2000)
        }
    }

    const save = async () => {
        if (!settings) return
        setSaving(true)
        try {
            await SettingsService.SaveSettings(settings as any)
            setSaved(true)
            setTimeout(() => setSaved(false), 2000)
        } catch (e) {
            console.error('Failed to save settings:', e)
        } finally {
            setSaving(false)
        }
    }

    if (!settings) {
        return <div className="loading-spinner">Loading settings…</div>
    }

    return (
        <div className="settings-view">
            {/* General */}
            <div>
                <div className="settings-section-title">General</div>
                <div className="settings-row">
                    <span className="settings-label">
                        Global Shortcut
                        {hotkeyStatus === 'saved' && <span style={{ color: 'rgba(52,211,153,0.8)', marginLeft: 6, fontSize: 11 }}>✓ Applied</span>}
                        {hotkeyStatus === 'error' && <span style={{ color: 'rgba(248,113,113,0.8)', marginLeft: 6, fontSize: 11 }}>Error</span>}
                    </span>
                    <HotkeyRecorder
                        value={settings.globalShortcut}
                        onChange={(spec) => handleHotkeyChange(spec)}
                    />
                </div>
                <div className="settings-row">
                    <span className="settings-label" style={{ fontSize: 11, opacity: 0.45 }}>
                        Restart app to apply. Requires Accessibility in System Settings.
                    </span>
                </div>
                <div className="settings-row">
                    <span className="settings-label">Font Size</span>
                    <select
                        className="settings-select"
                        value={settings.fontSize}
                        onChange={e => update('fontSize', e.target.value as 'small' | 'medium' | 'large')}
                    >
                        <option value="small">Small</option>
                        <option value="medium">Medium</option>
                        <option value="large">Large</option>
                    </select>
                </div>
            </div>

            {/* AI */}
            <div>
                <div className="settings-section-title">AI</div>
                <div className="settings-row">
                    <span className="settings-label">Provider</span>
                    <select
                        className="settings-select"
                        value={settings.ai.provider}
                        onChange={e => updateAI('provider', e.target.value as any)}
                    >
                        <option value="openai">OpenAI</option>
                        <option value="ollama">Ollama</option>
                        <option value="openai-compatible">OpenAI-Compatible</option>
                    </select>
                </div>

                {settings.ai.provider === 'openai' && (
                    <>
                        <div className="settings-row">
                            <span className="settings-label">OpenAI API Key</span>
                            <input
                                className="settings-input"
                                type="password"
                                value={settings.ai.openaiApiKey}
                                onChange={e => updateAI('openaiApiKey', e.target.value)}
                                placeholder="sk-..."
                            />
                        </div>
                        <div className="settings-row">
                            <span className="settings-label">Model</span>
                            <input
                                className="settings-input"
                                value={settings.ai.defaultModel}
                                onChange={e => updateAI('defaultModel', e.target.value)}
                                placeholder="gpt-4o-mini"
                            />
                        </div>
                    </>
                )}

                {settings.ai.provider === 'ollama' && (
                    <>
                        <div className="settings-row">
                            <span className="settings-label">Ollama URL</span>
                            <input
                                className="settings-input"
                                value={settings.ai.ollamaBaseUrl}
                                onChange={e => updateAI('ollamaBaseUrl', e.target.value)}
                                placeholder="http://localhost:11434"
                            />
                        </div>
                        <div className="settings-row">
                            <span className="settings-label">Model</span>
                            <input
                                className="settings-input"
                                value={settings.ai.defaultModel}
                                onChange={e => updateAI('defaultModel', e.target.value)}
                                placeholder="llama3"
                            />
                        </div>
                    </>
                )}

                {settings.ai.provider === 'openai-compatible' && (
                    <>
                        <div className="settings-row">
                            <span className="settings-label">Base URL</span>
                            <input
                                className="settings-input"
                                value={settings.ai.openaiCompatibleBaseUrl}
                                onChange={e => updateAI('openaiCompatibleBaseUrl', e.target.value)}
                                placeholder="https://api.example.com/v1"
                            />
                        </div>
                        <div className="settings-row">
                            <span className="settings-label">API Key</span>
                            <input
                                className="settings-input"
                                type="password"
                                value={settings.ai.openaiCompatibleApiKey}
                                onChange={e => updateAI('openaiCompatibleApiKey', e.target.value)}
                            />
                        </div>
                        <div className="settings-row">
                            <span className="settings-label">Model</span>
                            <input
                                className="settings-input"
                                value={settings.ai.openaiCompatibleModel}
                                onChange={e => updateAI('openaiCompatibleModel', e.target.value)}
                            />
                        </div>
                    </>
                )}
            </div>

            <button className="settings-save-btn" onClick={save} disabled={saving}>
                {saving ? 'Saving…' : saved ? '✓ Saved' : 'Save Settings'}
            </button>
        </div>
    )
}

export default SettingsView
