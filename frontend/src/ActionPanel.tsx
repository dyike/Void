import React from 'react'
import { X, Copy, ExternalLink, Clipboard, Sparkles, Eye, FileText, ListRestart, Play, Square, Terminal, Trash2 } from 'lucide-react'
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

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
      className="fixed inset-0 z-[100] flex items-end bg-foreground/10 dark:bg-black/30"
      onClick={onClose}
      onKeyDown={handleKey}
    >
      <div
        className="w-full rounded-b-[14px] border-t border-border bg-background/95 p-2 backdrop-blur-3xl"
        onClick={e => e.stopPropagation()}
        onKeyDown={handleKey}
        tabIndex={-1}
        autoFocus
      >
        <div className="flex items-center justify-between border-b border-border px-2 pb-2">
          <div>
            <div className="text-sm font-semibold text-foreground/85">{title}</div>
            {subtitle && <div className="text-[11px] text-muted-foreground">{subtitle}</div>}
          </div>
          <Button
            variant="ghost"
            size="icon"
            className="h-7 w-7 text-muted-foreground hover:text-foreground/70"
            onClick={onClose}
          >
            <X size={14} />
          </Button>
        </div>

        <div className="flex flex-col gap-1 pt-2">
          {actions.map(action => (
            <button
              key={action.id}
              className={cn(
                "flex items-center gap-2 rounded-md px-3 py-2 text-sm text-foreground/80 transition hover:bg-accent",
                action.danger && "text-red-600 dark:text-red-300 hover:bg-red-500/10"
              )}
              onClick={() => { onAction(action.id); onClose() }}
            >
              {action.icon && (
                <span className="flex text-muted-foreground">{action.icon}</span>
              )}
              <span className="flex-1 text-left">{action.label}</span>
              {action.shortcut && (
                <span className="text-[11px] text-muted-foreground/60">{action.shortcut}</span>
              )}
            </button>
          ))}
        </div>
      </div>
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
    actions.push({ id: 'quicklook', label: 'Quick Look', icon: <Eye size={14} />, shortcut: '⌘Y' })
    actions.push({ id: 'reveal', label: 'Reveal in Finder', icon: <ExternalLink size={14} /> })
    actions.push({ id: 'copy-path', label: 'Copy Path', icon: <Copy size={14} /> })
  } else if (cmd.category === 'system') {
    if (cmd.id === 'system-clipboard-manager') {
      actions.push({ id: 'open', label: 'Open Clipboard History', icon: <Clipboard size={14} />, shortcut: '↵' })
    } else if (cmd.id === 'system-ai-chat') {
      actions.push({ id: 'open', label: 'Open AI Chat', icon: <Sparkles size={14} />, shortcut: '↵' })
    }
    actions.push({ id: 'copy-name', label: 'Copy Name', icon: <Copy size={14} /> })
  } else if (cmd.category === 'snippet') {
    actions.push({ id: 'copy-snippet', label: 'Copy Snippet', icon: <Copy size={14} />, shortcut: '↵' })
    actions.push({ id: 'copy-name', label: 'Copy Name', icon: <Copy size={14} /> })
  } else if (cmd.category === 'container') {
    actions.push({ id: 'container-open', label: 'Open in Workspace', icon: <ExternalLink size={14} />, shortcut: '↵' })
    actions.push({ id: 'container-logs', label: 'Open Logs', icon: <FileText size={14} /> })
    actions.push({ id: 'container-shell', label: 'Exec Shell', icon: <Terminal size={14} /> })
    actions.push({ id: 'container-restart', label: 'Restart', icon: <ListRestart size={14} /> })
    actions.push({ id: 'container-start', label: 'Start', icon: <Play size={14} /> })
    actions.push({ id: 'container-stop', label: 'Stop', icon: <Square size={14} /> })
    actions.push({ id: 'container-inspect', label: 'Inspect JSON', icon: <Eye size={14} /> })
    actions.push({ id: 'container-remove', label: 'Remove', icon: <Trash2 size={14} />, danger: true })
  }

  return actions
}

export default ActionPanel
