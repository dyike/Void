import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  ArrowRight,
  Box,
  CheckCircle2,
  Download,
  ListRestart,
  Loader2,
  Play,
  RefreshCw,
  Square,
  Terminal,
  Trash2,
} from 'lucide-react'
import * as ContainerService from '../bindings/github.com/dyike/Void/internal/service/containerservice.js'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { cn } from "@/lib/utils"

type DetailTab = 'summary' | 'logs' | 'inspect'
type WorkspaceMode = 'containers' | 'images' | 'run' | 'pull'

interface ContainerInfo {
  id: string
  name: string
  image: string
  status: string
  ports?: string[]
  createdAt?: string
  age?: string
}

interface ImageInfo {
  id?: string
  name: string
  tag?: string
  digest?: string
  size?: string
  createdAt?: string
}

interface ContainerCLIStatus {
  available: boolean
  path?: string
  version?: string
  system?: string
  error?: string
}

interface Props {
  initialContainerId?: string
  initialTab?: DetailTab
  initialMode?: WorkspaceMode
}

const emptyRunForm = {
  image: 'postgres:16',
  name: 'pg-dev',
  ports: '5432:5432',
  volumes: 'pg-data:/var/lib/postgresql/data',
  envs: 'POSTGRES_PASSWORD=postgres',
  command: '',
  detach: true,
  remove: false,
}

const ContainersView: React.FC<Props> = ({ initialContainerId, initialTab = 'summary', initialMode = 'containers' }) => {
  const [status, setStatus] = useState<ContainerCLIStatus | null>(null)
  const [containers, setContainers] = useState<ContainerInfo[]>([])
  const [images, setImages] = useState<ImageInfo[]>([])
  const [selectedId, setSelectedId] = useState(initialContainerId || '')
  const [detailTab, setDetailTab] = useState<DetailTab>(initialTab)
  const [mode, setMode] = useState<WorkspaceMode>(initialMode)
  const [query, setQuery] = useState('')
  const [logs, setLogs] = useState('')
  const [inspect, setInspect] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [pullRef, setPullRef] = useState('alpine:latest')
  const [runForm, setRunForm] = useState(emptyRunForm)
  const [lastCommand, setLastCommand] = useState('')
  const ready = Boolean(status?.available && !status?.error)

  const selected = containers.find(c => c.id === selectedId) || containers[0]

  const refresh = useCallback(async () => {
    setBusy(true)
    setError('')
    try {
      const [cliStatus, nextContainers, nextImages] = await Promise.all([
        ContainerService.GetStatus(),
        ContainerService.ListContainers().catch(() => []),
        ContainerService.ListImages().catch(() => []),
      ])
      setStatus(cliStatus as ContainerCLIStatus)
      setContainers((nextContainers || []) as ContainerInfo[])
      setImages((nextImages || []) as ImageInfo[])
      if (!selectedId && nextContainers?.[0]?.id) setSelectedId(nextContainers[0].id)
    } catch (e: any) {
      setError(String(e?.message || e))
    } finally {
      setBusy(false)
    }
  }, [selectedId])

  useEffect(() => {
    refresh()
  }, [refresh])

  useEffect(() => {
    if (initialContainerId) {
      setSelectedId(initialContainerId)
      setMode('containers')
    }
    setDetailTab(initialTab)
    if (initialMode) setMode(initialMode)
  }, [initialContainerId, initialMode, initialTab])

  useEffect(() => {
    if (!selected) return
    if (detailTab === 'logs') {
      ContainerService.GetLogs(selected.id, 200)
        .then(text => setLogs(text || 'No logs yet.'))
        .catch(e => setLogs(String(e?.message || e)))
    }
    if (detailTab === 'inspect') {
      ContainerService.InspectContainer(selected.id)
        .then(text => setInspect(text || '{}'))
        .catch(e => setInspect(String(e?.message || e)))
    }
  }, [selected, detailTab])

  const filteredContainers = useMemo(() => {
    const q = query.toLowerCase().trim()
    if (!q) return containers
    return containers.filter(c =>
      [c.name, c.id, c.image, c.status, ...(c.ports || [])].join(' ').toLowerCase().includes(q)
    )
  }, [containers, query])

  const filteredImages = useMemo(() => {
    const q = query.toLowerCase().trim()
    if (!q) return images
    return images.filter(img => [img.name, img.tag, img.id, img.size].join(' ').toLowerCase().includes(q))
  }, [images, query])

  const runCommandPreview = useMemo(() => {
    const args = ['container', 'run']
    if (runForm.detach) args.push('--detach')
    if (runForm.remove) args.push('--rm')
    if (runForm.name.trim()) args.push('--name', quote(runForm.name))
    splitLines(runForm.ports).forEach(port => args.push('--publish', quote(port)))
    splitLines(runForm.envs).forEach(env => args.push('--env', quote(env)))
    splitLines(runForm.volumes).forEach(volume => args.push('--volume', quote(volume)))
    if (runForm.image.trim()) args.push(quote(runForm.image))
    if (runForm.command.trim()) args.push(runForm.command.trim())
    return args.join(' ')
  }, [runForm])

  const performContainerAction = async (action: 'start' | 'stop' | 'restart' | 'remove' | 'shell', id = selected?.id) => {
    if (!id) return
    setBusy(true)
    setError('')
    try {
      if (action === 'start') await ContainerService.StartContainer(id)
      if (action === 'stop') await ContainerService.StopContainer(id)
      if (action === 'restart') await ContainerService.RestartContainer(id)
      if (action === 'remove') await ContainerService.RemoveContainer(id)
      if (action === 'shell') await ContainerService.OpenShell(id)
      await refresh()
    } catch (e: any) {
      setError(String(e?.message || e))
    } finally {
      setBusy(false)
    }
  }

  const pullImage = async () => {
    setBusy(true)
    setError('')
    try {
      const result = await ContainerService.PullImage(pullRef)
      setLastCommand(result?.command || '')
      await refresh()
      setMode('images')
    } catch (e: any) {
      setError(String(e?.message || e))
    } finally {
      setBusy(false)
    }
  }

  const installContainer = async () => {
    setError('')
    try {
      await ContainerService.InstallContainer()
    } catch (e: any) {
      setError(String(e?.message || e))
    }
  }

  const startSystem = async () => {
    setError('')
    try {
      await ContainerService.StartSystem()
      setTimeout(() => refresh(), 1800)
    } catch (e: any) {
      setError(String(e?.message || e))
    }
  }

  const runContainer = async () => {
    setBusy(true)
    setError('')
    try {
      const result = await ContainerService.RunContainer({
        image: runForm.image,
        name: runForm.name,
        ports: splitLines(runForm.ports),
        envs: splitLines(runForm.envs),
        volumes: splitLines(runForm.volumes),
        command: runForm.command,
        detach: runForm.detach,
        remove: runForm.remove,
      })
      setLastCommand(result?.command || runCommandPreview)
      await refresh()
      setMode('containers')
    } catch (e: any) {
      setError(String(e?.message || e))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden bg-background/30 text-[13px]">
      <div className="flex h-12 shrink-0 items-center gap-2 border-b border-launcher-border px-3">
        <div className={cn(
          "flex items-center gap-1.5 rounded-md border px-2 py-1 text-[11px]",
          ready
            ? "border-emerald-500/25 bg-emerald-500/10 text-emerald-600 dark:text-emerald-300"
            : "border-amber-500/25 bg-amber-500/10 text-amber-600 dark:text-amber-300"
        )}>
          <CheckCircle2 size={13} />
          <span>{ready ? 'container ready' : status?.available ? 'system stopped' : 'container missing'}</span>
        </div>
        <Input
          value={query}
          onChange={e => setQuery(e.target.value)}
          placeholder="Search containers or images"
          className="h-8 flex-1 border-border bg-secondary/40 text-[13px] focus-visible:ring-0"
        />
        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={refresh} disabled={busy} title="Refresh">
          {busy ? <Loader2 size={15} className="animate-spin" /> : <RefreshCw size={15} />}
        </Button>
        {!status?.available ? (
          <Button variant="secondary" size="sm" className="h-8 gap-1.5" onClick={installContainer}>
            <Download size={14} /> Install
          </Button>
        ) : status.error ? (
          <Button variant="secondary" size="sm" className="h-8 gap-1.5" onClick={startSystem}>
            <Play size={14} /> Start Service
          </Button>
        ) : null}
        <Button variant="secondary" size="sm" className="h-8 gap-1.5" onClick={() => setMode('pull')}>
          <Download size={14} /> Pull
        </Button>
        <Button size="sm" className="h-8 gap-1.5 bg-indigo-500/30 text-indigo-700 hover:bg-indigo-500/40 dark:text-indigo-100" onClick={() => setMode('run')}>
          <Play size={14} /> Run
        </Button>
      </div>

      {(error || status?.error || lastCommand) && (
        <div className="shrink-0 border-b border-launcher-border px-4 py-2 text-[12px]">
          {error || status?.error ? (
            <div className="text-red-600 dark:text-red-300">{error || status?.error}</div>
          ) : (
            <div className="truncate font-mono text-muted-foreground">{lastCommand}</div>
          )}
        </div>
      )}

      <div className="grid min-h-0 flex-1 grid-cols-[148px_minmax(360px,1fr)_minmax(320px,420px)]">
        <aside className="border-r border-launcher-border p-2">
          {[
            ['containers', 'Containers', Box],
            ['images', 'Images', Download],
            ['run', 'Run Flow', Play],
            ['pull', 'Pull Image', ArrowRight],
          ].map(([id, label, Icon]) => (
            <button
              key={id as string}
              className={cn(
                "flex w-full items-center gap-2 rounded-md px-2 py-2 text-left text-[13px] transition",
                mode === id ? "bg-indigo-500/18 text-foreground" : "text-muted-foreground hover:bg-accent hover:text-foreground"
              )}
              onClick={() => setMode(id as WorkspaceMode)}
            >
              {React.createElement(Icon as any, { size: 15 })}
              <span>{label as string}</span>
            </button>
          ))}
        </aside>

        <main className="min-w-0 border-r border-launcher-border">
          {mode === 'containers' && (
            <div className="flex h-full flex-col">
              <TableHeader columns={['Status', 'Name', 'Image', 'Ports', 'Age']} />
              <div className="min-h-0 flex-1 overflow-y-auto">
                {filteredContainers.map(container => (
                  <button
                    key={container.id}
                    className={cn(
                      "grid w-full grid-cols-[82px_1.1fr_1.2fr_1fr_70px] items-center gap-3 border-b border-border/60 px-3 py-2 text-left transition hover:bg-accent/70",
                      selected?.id === container.id && "bg-indigo-500/15"
                    )}
                    onClick={() => setSelectedId(container.id)}
                  >
                    <StatusPill status={container.status} />
                    <Cell title={container.name || container.id} sub={shortId(container.id)} />
                    <Cell title={container.image || 'unknown image'} />
                    <Cell title={(container.ports || []).join(', ') || '-'} />
                    <Cell title={container.age || container.createdAt || '-'} />
                  </button>
                ))}
                {filteredContainers.length === 0 && <EmptyState text="No containers found." />}
              </div>
            </div>
          )}

          {mode === 'images' && (
            <div className="flex h-full flex-col">
              <TableHeader columns={['Image', 'Tag', 'Size', 'Created']} />
              <div className="min-h-0 flex-1 overflow-y-auto">
                {filteredImages.map((image, index) => (
                  <div key={`${image.name}-${index}`} className="grid grid-cols-[1.4fr_90px_90px_120px] items-center gap-3 border-b border-border/60 px-3 py-2">
                    <Cell title={image.name} sub={image.id ? shortId(image.id) : image.digest} />
                    <Cell title={image.tag || '-'} />
                    <Cell title={image.size || '-'} />
                    <Cell title={image.createdAt || '-'} />
                  </div>
                ))}
                {filteredImages.length === 0 && <EmptyState text="No images found." />}
              </div>
            </div>
          )}

          {mode === 'pull' && (
            <FlowPanel title="Pull Image" actionLabel="Pull Image" onAction={pullImage} busy={busy}>
              <Field label="Reference">
                <Input value={pullRef} onChange={e => setPullRef(e.target.value)} className="h-8 bg-secondary/50" />
              </Field>
              <CommandPreview command={`container image pull --progress plain ${quote(pullRef)}`} />
            </FlowPanel>
          )}

          {mode === 'run' && (
            <FlowPanel title="Run Container" actionLabel="Run Container" onAction={runContainer} busy={busy}>
              <Field label="Image">
                <Input value={runForm.image} onChange={e => setRunForm({ ...runForm, image: e.target.value })} className="h-8 bg-secondary/50" />
              </Field>
              <Field label="Name">
                <Input value={runForm.name} onChange={e => setRunForm({ ...runForm, name: e.target.value })} className="h-8 bg-secondary/50" />
              </Field>
              <Field label="Ports">
                <Input value={runForm.ports} onChange={e => setRunForm({ ...runForm, ports: e.target.value })} placeholder="8080:80, 5432:5432" className="h-8 bg-secondary/50" />
              </Field>
              <Field label="Volumes">
                <Input value={runForm.volumes} onChange={e => setRunForm({ ...runForm, volumes: e.target.value })} placeholder="data:/var/lib/data" className="h-8 bg-secondary/50" />
              </Field>
              <Field label="Env">
                <Input value={runForm.envs} onChange={e => setRunForm({ ...runForm, envs: e.target.value })} placeholder="KEY=value" className="h-8 bg-secondary/50" />
              </Field>
              <Field label="Command">
                <Input value={runForm.command} onChange={e => setRunForm({ ...runForm, command: e.target.value })} placeholder="/bin/sh" className="h-8 bg-secondary/50" />
              </Field>
              <div className="flex gap-2 pl-[92px]">
                <Toggle label="Detach" checked={runForm.detach} onClick={() => setRunForm({ ...runForm, detach: !runForm.detach })} />
                <Toggle label="Remove on exit" checked={runForm.remove} onClick={() => setRunForm({ ...runForm, remove: !runForm.remove })} />
              </div>
              <CommandPreview command={runCommandPreview} />
            </FlowPanel>
          )}
        </main>

        <section className="min-w-0">
          {selected ? (
            <div className="flex h-full flex-col">
              <div className="border-b border-launcher-border px-3 py-3">
                <div className="flex items-center justify-between gap-3">
                  <div className="min-w-0">
                    <div className="truncate text-[14px] font-semibold">{selected.name || selected.id}</div>
                    <div className="truncate text-[11px] text-muted-foreground">{selected.image || selected.id}</div>
                  </div>
                  <StatusPill status={selected.status} />
                </div>
                <div className="mt-3 flex items-center gap-1">
                  <IconAction title="Start" icon={<Play size={14} />} onClick={() => performContainerAction('start')} />
                  <IconAction title="Stop" icon={<Square size={14} />} onClick={() => performContainerAction('stop')} />
                  <IconAction title="Restart" icon={<ListRestart size={14} />} onClick={() => performContainerAction('restart')} />
                  <IconAction title="Shell" icon={<Terminal size={14} />} onClick={() => performContainerAction('shell')} />
                  <IconAction title="Remove" icon={<Trash2 size={14} />} danger onClick={() => performContainerAction('remove')} />
                </div>
              </div>

              <Tabs value={detailTab} onValueChange={value => setDetailTab(value as DetailTab)} className="min-h-0 flex-1">
                <TabsList className="mx-3 mt-3 grid h-8 grid-cols-3 bg-secondary/40">
                  <TabsTrigger value="summary" className="text-[12px]">Summary</TabsTrigger>
                  <TabsTrigger value="logs" className="text-[12px]">Logs</TabsTrigger>
                  <TabsTrigger value="inspect" className="text-[12px]">Inspect</TabsTrigger>
                </TabsList>
                <div className="min-h-0 flex-1 p-3">
                  {detailTab === 'summary' && (
                    <div className="space-y-3 text-[12px]">
                      <SummaryRow label="ID" value={selected.id} />
                      <SummaryRow label="Image" value={selected.image || '-'} />
                      <SummaryRow label="Ports" value={(selected.ports || []).join(', ') || '-'} />
                      <SummaryRow label="Created" value={selected.createdAt || selected.age || '-'} />
                    </div>
                  )}
                  {detailTab === 'logs' && <CodeBlock text={logs || 'Select a container to load logs.'} />}
                  {detailTab === 'inspect' && <CodeBlock text={inspect || '{}'} />}
                </div>
              </Tabs>
            </div>
          ) : (
            <EmptyState text="Select a container to inspect it." />
          )}
        </section>
      </div>
    </div>
  )
}

function TableHeader({ columns }: { columns: string[] }) {
  return (
    <div className="grid grid-flow-col auto-cols-fr gap-3 border-b border-launcher-border px-3 py-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground/70">
      {columns.map(column => <span key={column}>{column}</span>)}
    </div>
  )
}

function Cell({ title, sub }: { title: string; sub?: string }) {
  return (
    <div className="min-w-0">
      <div className="truncate text-[13px] text-foreground/90">{title}</div>
      {sub && <div className="truncate text-[11px] text-muted-foreground">{sub}</div>}
    </div>
  )
}

function StatusPill({ status }: { status: string }) {
  const normalized = status?.toLowerCase() || 'unknown'
  const running = normalized.includes('running')
  const stopped = normalized.includes('stopped') || normalized.includes('exited')
  return (
    <span className={cn(
      "inline-flex w-fit items-center rounded px-1.5 py-0.5 text-[11px] font-medium",
      running && "bg-emerald-500/12 text-emerald-600 dark:text-emerald-300",
      stopped && "bg-muted text-muted-foreground",
      !running && !stopped && "bg-amber-500/12 text-amber-600 dark:text-amber-300"
    )}>
      {status || 'unknown'}
    </span>
  )
}

function IconAction({ title, icon, danger, onClick }: { title: string; icon: React.ReactNode; danger?: boolean; onClick: () => void }) {
  return (
    <Button
      variant="ghost"
      size="icon"
      title={title}
      onClick={onClick}
      className={cn("h-7 w-7 text-muted-foreground hover:text-foreground", danger && "hover:bg-red-500/10 hover:text-red-500")}
    >
      {icon}
    </Button>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[80px_1fr] items-center gap-3">
      <div className="text-right text-[12px] font-medium text-muted-foreground">{label}</div>
      {children}
    </div>
  )
}

function FlowPanel({ title, actionLabel, busy, children, onAction }: { title: string; actionLabel: string; busy: boolean; children: React.ReactNode; onAction: () => void }) {
  return (
    <div className="mx-auto flex max-w-[560px] flex-col gap-3 px-6 py-6">
      <div>
        <div className="text-[15px] font-semibold">{title}</div>
        <div className="text-[12px] text-muted-foreground">Keyboard-first Apple Container command flow.</div>
      </div>
      {children}
      <div className="flex justify-end">
        <Button onClick={onAction} disabled={busy} className="h-8 gap-1.5 bg-indigo-500/30 text-indigo-700 hover:bg-indigo-500/40 dark:text-indigo-100">
          {busy ? <Loader2 size={14} className="animate-spin" /> : <Play size={14} />}
          {actionLabel}
        </Button>
      </div>
    </div>
  )
}

function Toggle({ label, checked, onClick }: { label: string; checked: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "rounded-md border px-2 py-1 text-[12px] transition",
        checked ? "border-indigo-500/30 bg-indigo-500/15 text-foreground" : "border-border bg-secondary/40 text-muted-foreground"
      )}
    >
      {label}
    </button>
  )
}

function CommandPreview({ command }: { command: string }) {
  return (
    <div className="rounded-md border border-border bg-secondary/35 p-3 font-mono text-[12px] leading-relaxed text-muted-foreground">
      {command}
    </div>
  )
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid grid-cols-[72px_1fr] gap-3">
      <div className="text-right text-muted-foreground">{label}</div>
      <div className="min-w-0 truncate font-mono text-foreground/80">{value}</div>
    </div>
  )
}

function CodeBlock({ text }: { text: string }) {
  return (
    <pre className="h-full min-h-[320px] overflow-auto rounded-md border border-border bg-black/85 p-3 font-mono text-[11px] leading-relaxed text-zinc-200">
      {text}
    </pre>
  )
}

function EmptyState({ text }: { text: string }) {
  return <div className="flex h-full items-center justify-center p-8 text-sm text-muted-foreground">{text}</div>
}

function splitLines(value: string) {
  return value.split(/[,\n]/).map(v => v.trim()).filter(Boolean)
}

function quote(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return "''"
  return /[\s'"\\$`!*?[\]{}()&;|<>]/.test(trimmed)
    ? `'${trimmed.replace(/'/g, `'\\''`)}'`
    : trimmed
}

function shortId(id: string) {
  return id.length > 12 ? id.slice(0, 12) : id
}

export default ContainersView
