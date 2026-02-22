import React, { useState, useEffect } from 'react'
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

const SettingsView: React.FC = () => {
    const [settings, setSettings] = useState<AppSettings | null>(null)
    const [saving, setSaving] = useState(false)
    const [saved, setSaved] = useState(false)

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
                    <span className="settings-label">Global Shortcut</span>
                    <input
                        className="settings-input"
                        value={settings.globalShortcut}
                        onChange={e => update('globalShortcut', e.target.value)}
                        placeholder="Alt+Space"
                    />
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
