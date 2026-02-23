import React from 'react'
import { X, Copy, ExternalLink, Clipboard, Sparkles } from 'lucide-react'

export interface Action {
    id: string
    label: string
    icon?: React.ReactNode
    shortcut?: string
    danger?: boolean
}

interface Props {
    title: string
    subtitle?: string
    actions: Action[]
    onAction: (id: string) => void
    onClose: () => void
}

/** Action Panel overlay — shown on Cmd+K over the selected command */
const ActionPanel: React.FC<Props> = ({ title, subtitle, actions, onAction, onClose }) => {
    const handleKey = (e: React.KeyboardEvent) => {
        if (e.key === 'Escape') { e.preventDefault(); onClose() }
    }

    return (
        <div
            className="action-panel-backdrop"
            onClick={onClose}
            onKeyDown={handleKey}
        >
            <div
                className="action-panel"
                onClick={e => e.stopPropagation()}
                onKeyDown={handleKey}
                tabIndex={-1}
                autoFocus
            >
                {/* Header */}
                <div className="action-panel-header">
                    <div>
                        <div className="action-panel-title">{title}</div>
                        {subtitle && <div className="action-panel-subtitle">{subtitle}</div>}
                    </div>
                    <button className="action-panel-close" onClick={onClose}>
                        <X size={14} />
                    </button>
                </div>

                {/* Actions */}
                <div className="action-panel-list">
                    {actions.map(action => (
                        <button
                            key={action.id}
                            className={`action-panel-item${action.danger ? ' danger' : ''}`}
                            onClick={() => { onAction(action.id); onClose() }}
                        >
                            {action.icon && (
                                <span className="action-panel-item-icon">{action.icon}</span>
                            )}
                            <span className="action-panel-item-label">{action.label}</span>
                            {action.shortcut && (
                                <span className="action-panel-item-shortcut">{action.shortcut}</span>
                            )}
                        </button>
                    ))}
                </div>
            </div>

            <style>{`
        .action-panel-backdrop {
          position: fixed;
          inset: 0;
          background: rgba(0,0,0,0.3);
          display: flex;
          align-items: flex-end;
          z-index: 100;
          animation: panel-backdrop-in .12s ease;
        }
        @keyframes panel-backdrop-in {
          from { opacity: 0 }
          to   { opacity: 1 }
        }

        .action-panel {
          width: 100%;
          background: rgba(20, 20, 26, 0.96);
          backdrop-filter: blur(40px);
          -webkit-backdrop-filter: blur(40px);
          border-top: 1px solid rgba(255,255,255,0.09);
          border-radius: 0 0 14px 14px;
          padding: 6px;
          animation: panel-slide-in .18s cubic-bezier(0.23,1,0.32,1);
        }
        @keyframes panel-slide-in {
          from { transform: translateY(100%); opacity: 0 }
          to   { transform: translateY(0);    opacity: 1 }
        }

        .action-panel-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          padding: 8px 10px 10px;
          border-bottom: 1px solid rgba(255,255,255,0.06);
          margin-bottom: 4px;
        }
        .action-panel-title {
          font-size: 13px;
          font-weight: 600;
          color: rgba(255,255,255,0.85);
        }
        .action-panel-subtitle {
          font-size: 11px;
          color: rgba(255,255,255,0.35);
          margin-top: 1px;
        }
        .action-panel-close {
          background: none;
          border: none;
          cursor: pointer;
          color: rgba(255,255,255,0.3);
          padding: 4px;
          display: flex;
          align-items: center;
          transition: color .15s;
        }
        .action-panel-close:hover { color: rgba(255,255,255,0.6); }

        .action-panel-list {
          display: flex;
          flex-direction: column;
          gap: 2px;
        }
        .action-panel-item {
          display: flex;
          align-items: center;
          gap: 10px;
          padding: 9px 12px;
          border-radius: 7px;
          border: none;
          background: transparent;
          cursor: pointer;
          text-align: left;
          font-family: inherit;
          font-size: 13px;
          color: rgba(255,255,255,0.80);
          transition: background .1s;
        }
        .action-panel-item:hover { background: rgba(255,255,255,0.07); }
        .action-panel-item.danger { color: rgba(248,113,113,0.85); }
        .action-panel-item.danger:hover { background: rgba(248,113,113,0.08); }
        .action-panel-item-icon { color: rgba(255,255,255,0.40); display: flex; }
        .action-panel-item .action-panel-item-icon { flex-shrink: 0; }
        .action-panel-item-label { flex: 1; }
        .action-panel-item-shortcut {
          font-size: 11px;
          color: rgba(255,255,255,0.30);
          font-family: -apple-system, monospace;
        }
      `}</style>
        </div>
    )
}

/** Build the default action list for a given command category */
export function buildActions(cmd: { id: string; category: string; title: string; path?: string }): Action[] {
    const actions: Action[] = []

    if (cmd.category === 'app') {
        actions.push({ id: 'open', label: 'Open', icon: <ExternalLink size={14} />, shortcut: '↵' })
        actions.push({ id: 'reveal', label: 'Reveal in Finder', icon: <ExternalLink size={14} /> })
        actions.push({ id: 'copy-path', label: 'Copy Path', icon: <Copy size={14} /> })
    } else if (cmd.category === 'file') {
        actions.push({ id: 'open', label: 'Open', icon: <ExternalLink size={14} />, shortcut: '↵' })
        actions.push({ id: 'reveal', label: 'Reveal in Finder', icon: <ExternalLink size={14} /> })
        actions.push({ id: 'copy-path', label: 'Copy Path', icon: <Copy size={14} /> })
    } else if (cmd.category === 'system') {
        if (cmd.id === 'system-clipboard-manager') {
            actions.push({ id: 'open', label: 'Open Clipboard History', icon: <Clipboard size={14} />, shortcut: '↵' })
        } else if (cmd.id === 'system-ai-chat') {
            actions.push({ id: 'open', label: 'Open AI Chat', icon: <Sparkles size={14} />, shortcut: '↵' })
        }
        actions.push({ id: 'copy-name', label: 'Copy Name', icon: <Copy size={14} /> })
    }

    return actions
}

export default ActionPanel
