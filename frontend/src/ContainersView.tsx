import React, { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Box,
  ChevronDown,
  CheckCircle2,
  Download,
  ListRestart,
  Loader2,
  Network,
  Play,
  Plus,
  RefreshCw,
  SlidersHorizontal,
  Square,
  Terminal,
  Trash2,
} from 'lucide-react'
import * as ContainerService from '../bindings/github.com/dyike/Void/internal/service/containerservice.js'
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { cn } from "@/lib/utils"

type DetailTab = 'summary' | 'logs' | 'inspect'
type WorkspaceMode = 'containers' | 'images' | 'networks' | 'run'
type InitialWorkspaceMode = WorkspaceMode | 'pull'

interface ContainerInfo {
  id: string
  name: string
  image: string
  status: string
  ip?: string
  cpus?: string
  memory?: string
  os?: string
  arch?: string
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

interface NetworkInfo {
  id: string
  name: string
  mode?: string
  plugin?: string
  ipv4Subnet?: string
  ipv4Gateway?: string
  ipv6Subnet?: string
  labels?: string[]
  options?: string[]
  createdAt?: string
}

interface ContainerCLIStatus {
  available: boolean
  supported?: boolean
  path?: string
  version?: string
  system?: string
  osVersion?: string
  error?: string
}

interface HostResourceInfo {
  cpuCores: number
  memoryBytes: number
  memoryMiB: number
}

interface Props {
  initialContainerId?: string
  initialTab?: DetailTab
  initialMode?: InitialWorkspaceMode
}

const emptyRunForm = {
  image: '',
  name: '',
  ports: '',
  volumes: '',
  envs: '',
  command: '',
  detach: true,
  remove: false,
  network: '',
  dns: '',
  noDns: false,
  platform: '',
  cpus: '',
  memory: '',
  workdir: '',
  user: '',
  entrypoint: '',
  labels: '',
  init: false,
  readOnly: false,
  ssh: false,
}

const emptyNetworkForm = {
  name: '',
  internal: false,
  subnet: '',
  subnetV6: '',
  plugin: 'container-network-vmnet',
  labels: '',
  options: '',
}

const ContainersView: React.FC<Props> = ({ initialContainerId, initialTab = 'summary', initialMode = 'containers' }) => {
  const initialWorkspaceMode: WorkspaceMode = initialMode === 'pull' ? 'images' : initialMode
  const [status, setStatus] = useState<ContainerCLIStatus | null>(null)
  const [containers, setContainers] = useState<ContainerInfo[]>([])
  const [images, setImages] = useState<ImageInfo[]>([])
  const [networks, setNetworks] = useState<NetworkInfo[]>([])
  const [hostResources, setHostResources] = useState<HostResourceInfo | null>(null)
  const [selectedId, setSelectedId] = useState(initialContainerId || '')
  const [selectedImageKey, setSelectedImageKey] = useState('')
  const [selectedNetworkKey, setSelectedNetworkKey] = useState('')
  const [detailTab, setDetailTab] = useState<DetailTab>(initialTab)
  const [mode, setMode] = useState<WorkspaceMode>(initialWorkspaceMode)
  const [query, setQuery] = useState('')
  const [logs, setLogs] = useState('')
  const [inspect, setInspect] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [pullRef, setPullRef] = useState('alpine:latest')
  const [runForm, setRunForm] = useState(emptyRunForm)
  const [lastCommand, setLastCommand] = useState('')
  const [pendingImageDeleteKey, setPendingImageDeleteKey] = useState('')
  const [imageDeleteConfirmStep, setImageDeleteConfirmStep] = useState<0 | 1 | 2>(0)
  const [pendingNetworkDeleteKey, setPendingNetworkDeleteKey] = useState('')
  const [networkDeleteConfirm, setNetworkDeleteConfirm] = useState(false)
  const [showPullPanel, setShowPullPanel] = useState(initialMode === 'pull')
  const [showAdvancedRun, setShowAdvancedRun] = useState(false)
  const [showNetworkCreate, setShowNetworkCreate] = useState(false)
  const [networkForm, setNetworkForm] = useState(emptyNetworkForm)
  const supported = status?.supported !== false
  const ready = Boolean(supported && status?.available && !status?.error)
  const showDetailPane = mode === 'containers' || mode === 'images' || mode === 'networks'

  const selected = containers.find(c => c.id === selectedId) || containers[0]
  const selectedImage = images.find(image => imageKey(image) === selectedImageKey) || images[0]
  const selectedNetwork = networks.find(network => networkKey(network) === selectedNetworkKey) || networks[0]

  const refresh = useCallback(async (options: { resetImageSelection?: boolean } = {}) => {
    setBusy(true)
    setError('')
    try {
      const [cliStatus, nextContainers, nextImages, nextNetworks, nextHostResources] = await Promise.all([
        ContainerService.GetStatus(),
        ContainerService.ListContainers().catch(() => []),
        ContainerService.ListImages().catch(() => []),
        ContainerService.ListNetworks().catch(() => []),
        ContainerService.GetHostResources().catch(() => null),
      ])
      const nextContainerList = (nextContainers || []) as ContainerInfo[]
      const nextImageList = (nextImages || []) as ImageInfo[]
      const nextNetworkList = (nextNetworks || []) as NetworkInfo[]
      setStatus(cliStatus as ContainerCLIStatus)
      setContainers(nextContainerList)
      setImages(nextImageList)
      setNetworks(nextNetworkList)
      if (nextHostResources) setHostResources(nextHostResources as HostResourceInfo)
      if (!selectedId && nextContainerList[0]?.id) setSelectedId(nextContainerList[0].id)
      if (options.resetImageSelection) {
        setSelectedImageKey(nextImageList[0] ? imageKey(nextImageList[0]) : '')
      } else if (!selectedImageKey && nextImageList[0]) {
        setSelectedImageKey(imageKey(nextImageList[0]))
      } else if (selectedImageKey && !nextImageList.some(image => imageKey(image) === selectedImageKey)) {
        setSelectedImageKey(nextImageList[0] ? imageKey(nextImageList[0]) : '')
      }
      if (!selectedNetworkKey && nextNetworkList[0]) {
        setSelectedNetworkKey(networkKey(nextNetworkList[0]))
      } else if (selectedNetworkKey && !nextNetworkList.some(network => networkKey(network) === selectedNetworkKey)) {
        setSelectedNetworkKey(nextNetworkList[0] ? networkKey(nextNetworkList[0]) : '')
      }
    } catch (e: any) {
      setError(String(e?.message || e))
    } finally {
      setBusy(false)
    }
  }, [selectedId, selectedImageKey, selectedNetworkKey])

  useEffect(() => {
    refresh()
  }, [refresh])

  useEffect(() => {
    setPendingImageDeleteKey('')
    setImageDeleteConfirmStep(0)
    setPendingNetworkDeleteKey('')
    setNetworkDeleteConfirm(false)
  }, [selectedImageKey, mode])

  useEffect(() => {
    setPendingNetworkDeleteKey('')
    setNetworkDeleteConfirm(false)
  }, [selectedNetworkKey])

  useEffect(() => {
    if (initialContainerId) {
      setSelectedId(initialContainerId)
      setMode('containers')
    }
    setDetailTab(initialTab)
    if (initialMode === 'pull') {
      setMode('images')
      setShowPullPanel(true)
    } else if (initialMode) {
      setMode(initialMode)
    }
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

  const filteredNetworks = useMemo(() => {
    const q = query.toLowerCase().trim()
    if (!q) return networks
    return networks.filter(network =>
      [
        network.name,
        network.id,
        network.mode,
        network.plugin,
        network.ipv4Subnet,
        network.ipv4Gateway,
        network.ipv6Subnet,
        ...(network.labels || []),
        ...(network.options || []),
      ].join(' ').toLowerCase().includes(q)
    )
  }, [networks, query])

  const runCommandPreview = useMemo(() => {
    const args = ['container', 'run']
    const command = effectiveRunCommand(runForm)
    if (runForm.detach) args.push('--detach')
    if (runForm.remove && !runForm.detach) args.push('--rm')
    if (runForm.name.trim()) args.push('--name', quote(runForm.name))
    if (runForm.network.trim()) args.push('--network', quote(runForm.network))
    splitLines(runForm.dns).forEach(dns => args.push('--dns', quote(dns)))
    if (runForm.noDns) args.push('--no-dns')
    if (runForm.platform.trim()) args.push('--platform', quote(runForm.platform))
    if (runResourceValue(runForm.cpus)) args.push('--cpus', quote(runResourceValue(runForm.cpus)))
    if (runResourceValue(runForm.memory)) args.push('--memory', quote(runResourceValue(runForm.memory)))
    if (runForm.workdir.trim()) args.push('--workdir', quote(runForm.workdir))
    if (runForm.user.trim()) args.push('--user', quote(runForm.user))
    if (runForm.entrypoint.trim()) args.push('--entrypoint', quote(runForm.entrypoint))
    splitLines(runForm.labels).forEach(label => args.push('--label', quote(label)))
    if (runForm.init) args.push('--init')
    if (runForm.readOnly) args.push('--read-only')
    if (runForm.ssh) args.push('--ssh')
    splitLines(runForm.ports).forEach(port => args.push('--publish', quote(port)))
    splitLines(runForm.envs).forEach(env => args.push('--env', quote(env)))
    splitLines(runForm.volumes).forEach(volume => args.push('--volume', quote(volume)))
    if (runForm.image.trim()) args.push(quote(runForm.image))
    if (command) args.push('/bin/sh', '-lc', quote(command))
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

  const removeImage = async (image = selectedImage) => {
    if (!image) return
    const reference = imageReference(image)
    setBusy(true)
    setError('')
    try {
      const result = await ContainerService.RemoveImage(reference)
      const pruneResult = await ContainerService.PruneImages(false).catch(() => null)
      setLastCommand(
        [result?.command || `container image delete --force ${quote(reference)}`, pruneResult?.command]
          .filter(Boolean)
          .join(' && ')
      )
      setPendingImageDeleteKey('')
      setImageDeleteConfirmStep(0)
      await refresh({ resetImageSelection: true })
      setMode('images')
    } catch (e: any) {
      setError(String(e?.message || e))
    } finally {
      setBusy(false)
    }
  }

  const pruneImages = async () => {
    const confirmed = window.confirm('Remove unused images?')
    if (!confirmed) return
    setBusy(true)
    setError('')
    try {
      const result = await ContainerService.PruneImages(false)
      setLastCommand(result?.command || 'container image prune')
      await refresh({ resetImageSelection: true })
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
        command: effectiveRunCommand(runForm),
        detach: runForm.detach,
        remove: runForm.detach ? false : runForm.remove,
        network: runForm.network,
        dns: splitLines(runForm.dns),
        noDns: runForm.noDns,
        platform: runForm.platform,
        cpus: runResourceValue(runForm.cpus),
        memory: runResourceValue(runForm.memory),
        workdir: runForm.workdir,
        user: runForm.user,
        entrypoint: runForm.entrypoint,
        labels: splitLines(runForm.labels),
        init: runForm.init,
        readOnly: runForm.readOnly,
        ssh: runForm.ssh,
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

  const createNetwork = async () => {
    const name = networkForm.name.trim()
    if (!name) {
      setError('network name is required')
      return
    }
    setBusy(true)
    setError('')
    try {
      const result = await ContainerService.CreateNetwork({
        name,
        internal: networkForm.internal,
        subnet: networkForm.subnet,
        subnetV6: networkForm.subnetV6,
        plugin: networkForm.plugin,
        labels: splitLines(networkForm.labels),
        options: splitLines(networkForm.options),
      })
      setLastCommand(result?.command || `container network create ${quote(name)}`)
      await refresh()
      setSelectedNetworkKey(name)
      setShowNetworkCreate(false)
      setNetworkForm(emptyNetworkForm)
      setMode('networks')
    } catch (e: any) {
      setError(String(e?.message || e))
    } finally {
      setBusy(false)
    }
  }

  const removeNetwork = async (network = selectedNetwork) => {
    if (!network || isBuiltinNetwork(network)) return
    const name = network.name || network.id
    setBusy(true)
    setError('')
    try {
      const result = await ContainerService.RemoveNetwork(name)
      setLastCommand(result?.command || `container network delete ${quote(name)}`)
      setPendingNetworkDeleteKey('')
      setNetworkDeleteConfirm(false)
      await refresh()
      setMode('networks')
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
          !supported
            ? "border-red-500/25 bg-red-500/10 text-red-600 dark:text-red-300"
            : ready
            ? "border-emerald-500/25 bg-emerald-500/10 text-emerald-600 dark:text-emerald-300"
            : "border-amber-500/25 bg-amber-500/10 text-amber-600 dark:text-amber-300"
        )}>
          <CheckCircle2 size={13} />
          <span>{containerStatusLabel(status, ready)}</span>
        </div>
        <Input
          value={query}
          onChange={e => setQuery(e.target.value)}
          placeholder="Search containers, images, or networks"
          className="h-8 flex-1 border-border bg-secondary/40 text-[13px] focus-visible:ring-0"
        />
        <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => refresh()} disabled={busy} title="Refresh">
          {busy ? <Loader2 size={15} className="animate-spin" /> : <RefreshCw size={15} />}
        </Button>
        {status && supported && !status.available ? (
          <Button variant="secondary" size="sm" className="h-8 gap-1.5" onClick={installContainer}>
            <Download size={14} /> Install
          </Button>
        ) : status && supported && status.error ? (
          <Button variant="secondary" size="sm" className="h-8 gap-1.5" onClick={startSystem}>
            <Play size={14} /> Start Service
          </Button>
        ) : null}
        {mode === 'images' && (
          <Button variant="secondary" size="sm" className="h-8 gap-1.5" onClick={pruneImages} disabled={busy || !ready}>
            <Trash2 size={14} /> Prune Unused
          </Button>
        )}
        {mode === 'networks' && (
          <Button variant="secondary" size="sm" className="h-8 gap-1.5" disabled={!ready} onClick={() => {
            setNetworkForm(emptyNetworkForm)
            setShowNetworkCreate(true)
          }}>
            <Plus size={14} /> Create Network
          </Button>
        )}
        <Button variant="secondary" size="sm" className="h-8 gap-1.5" onClick={() => {
          setMode('images')
          setShowPullPanel(true)
        }} disabled={!ready}>
          <Download size={14} /> Pull Image
        </Button>
        <Button size="sm" className="h-8 gap-1.5 bg-indigo-500/30 text-indigo-700 hover:bg-indigo-500/40 dark:text-indigo-100" onClick={() => {
          setRunForm(emptyRunForm)
          setMode('run')
        }} disabled={!ready}>
          <Play size={14} /> Run Container
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

      <div className={cn(
        "grid min-h-0 flex-1",
        showDetailPane
          ? "grid-cols-[136px_minmax(0,1fr)_minmax(280px,360px)]"
          : "grid-cols-[136px_minmax(0,1fr)]"
      )}>
        <aside className="border-r border-launcher-border p-2">
          {[
            ['containers', 'Containers', Box],
            ['images', 'Images', Download],
            ['networks', 'Networks', Network],
            ['run', 'Run Flow', Play],
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

        <main className={cn("min-h-0 min-w-0 overflow-hidden", showDetailPane && "border-r border-launcher-border")}>
          {mode === 'containers' && (
            <div className="flex h-full min-h-0 flex-col overflow-hidden">
              <div className="grid grid-cols-[82px_minmax(0,1fr)_minmax(112px,0.7fr)] gap-3 border-b border-launcher-border px-3 py-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground/70">
                <span>Status</span>
                <span>Container</span>
                <span>Network</span>
              </div>
              <div className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden">
                {filteredContainers.map(container => (
                  <button
                    key={container.id}
                    className={cn(
                      "grid w-full grid-cols-[82px_minmax(0,1fr)_minmax(112px,0.7fr)] items-center gap-3 border-b border-border/60 px-3 py-2 text-left transition hover:bg-accent/70",
                      selected?.id === container.id && "bg-indigo-500/15"
                    )}
                    onClick={() => setSelectedId(container.id)}
                  >
                    <StatusPill status={container.status} />
                    <Cell
                      title={container.name || container.id}
                      sub={[shortId(container.id), container.image || 'unknown image', platformLabel(container)].filter(Boolean).join(' • ')}
                    />
                    <Cell
                      title={container.ip || '-'}
                      sub={[container.cpus ? `${container.cpus} CPU` : '', container.memory, (container.ports || []).join(', ')].filter(Boolean).join(' • ')}
                    />
                  </button>
                ))}
                {filteredContainers.length === 0 && <EmptyState text="No containers found." />}
              </div>
            </div>
          )}

          {mode === 'images' && (
            <div className="flex h-full min-h-0 flex-col overflow-hidden">
              <div className="border-b border-launcher-border p-3">
                <div className="flex items-center justify-between gap-3">
                  <div className="min-w-0">
                    <div className="text-[14px] font-semibold">Images</div>
                    <div className="text-[11px] text-muted-foreground">Pull, run, inspect, and remove local images.</div>
                  </div>
                  <Button
                    variant="secondary"
                    size="sm"
                    className="h-8 shrink-0 gap-1.5"
                    onClick={() => setShowPullPanel(value => !value)}
                  >
                    <Download size={14} />
                    Pull Image
                  </Button>
                </div>
                {showPullPanel && (
                  <div className="mt-3 rounded-md border border-border bg-secondary/20 p-3">
                    <div className="grid grid-cols-[78px_minmax(0,1fr)_94px] items-center gap-2">
                      <div className="text-right text-[12px] font-medium text-muted-foreground">Reference</div>
                      <Input
                        value={pullRef}
                        onChange={e => setPullRef(e.target.value)}
                        placeholder="alpine:latest or docker.io/library/ubuntu:latest"
                        className="h-8 bg-background/70"
                      />
                      <Button
                        size="sm"
                        className="h-8 gap-1.5 bg-indigo-500/30 text-indigo-700 hover:bg-indigo-500/40 dark:text-indigo-100"
                        onClick={pullImage}
                        disabled={busy || !ready || !pullRef.trim()}
                      >
                        {busy ? <Loader2 size={14} className="animate-spin" /> : <Download size={14} />}
                        Pull
                      </Button>
                    </div>
                    <div className="mt-2 pl-[86px]">
                      <CommandPreview command={`container image pull --progress plain ${quote(pullRef)}`} />
                    </div>
                  </div>
                )}
              </div>
              <div className="grid grid-cols-[minmax(0,1fr)_104px] gap-3 border-b border-launcher-border px-3 py-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground/70">
                <span>Image</span>
                <span className="text-right">Size</span>
              </div>
              <div className="min-h-0 flex-1 overflow-y-auto">
                {filteredImages.map((image, index) => (
                  <button
                    key={`${image.name}-${index}`}
                    className={cn(
                      "grid w-full grid-cols-[minmax(0,1fr)_104px] items-center gap-3 border-b border-border/60 px-3 py-2 text-left transition hover:bg-accent/70",
                      imageKey(image) === imageKey(selectedImage) && "bg-indigo-500/15"
                    )}
                    onClick={() => setSelectedImageKey(imageKey(image))}
                  >
                    <Cell title={image.name} sub={[image.tag || 'untagged', image.id ? shortId(image.id) : image.digest].filter(Boolean).join(' • ')} />
                    <div className="min-w-0 text-right">
                      <div className="truncate text-[13px] text-foreground/90">{image.size || '-'}</div>
                      <div className="truncate text-[11px] text-muted-foreground">{compactDate(image.createdAt)}</div>
                    </div>
                  </button>
                ))}
                {filteredImages.length === 0 && <EmptyState text="No images found." />}
              </div>
            </div>
          )}

          {mode === 'networks' && (
            <div className="flex h-full flex-col">
              <div className="grid grid-cols-[minmax(120px,0.9fr)_76px_minmax(130px,1fr)_minmax(120px,0.9fr)] gap-3 border-b border-launcher-border px-3 py-2 text-[11px] font-semibold uppercase tracking-[0.14em] text-muted-foreground/70">
                <span>Name</span>
                <span>Mode</span>
                <span>Subnet</span>
                <span>Gateway</span>
              </div>
              <div className="min-h-0 flex-1 overflow-y-auto">
                {filteredNetworks.map(network => (
                  <button
                    key={networkKey(network)}
                    className={cn(
                      "grid w-full grid-cols-[minmax(120px,0.9fr)_76px_minmax(130px,1fr)_minmax(120px,0.9fr)] items-center gap-3 border-b border-border/60 px-3 py-2 text-left transition hover:bg-accent/70",
                      networkKey(network) === networkKey(selectedNetwork) && "bg-indigo-500/15"
                    )}
                    onClick={() => {
                      setSelectedNetworkKey(networkKey(network))
                      setShowNetworkCreate(false)
                    }}
                  >
                    <Cell title={network.name || network.id} sub={network.plugin || network.id} />
                    <Cell title={network.mode || '-'} />
                    <Cell title={network.ipv4Subnet || '-'} sub={network.ipv6Subnet || ''} />
                    <Cell title={network.ipv4Gateway || '-'} />
                  </button>
                ))}
                {filteredNetworks.length === 0 && <EmptyState text="No networks found." />}
              </div>
            </div>
          )}

          {mode === 'run' && (
            <FlowPanel title="Run Container" actionLabel="Run Container" onAction={runContainer} busy={busy} disabled={!ready}>
              <Field label="Image">
                <div className="grid grid-cols-[220px_minmax(0,1fr)] gap-2">
                  <Select
                    value={imageSelectValue(runForm.image, images)}
                    onValueChange={value => {
                      if (value === '__custom__') return
                      setRunForm({ ...runForm, image: value })
                    }}
                  >
                    <SelectTrigger className="h-8 border-border bg-secondary/50 text-[13px] focus:ring-0">
                      <SelectValue placeholder="Select image" />
                    </SelectTrigger>
                    <SelectContent>
                      {localImageReferences(images).map(reference => (
                        <SelectItem key={reference} value={reference}>
                          {reference}
                        </SelectItem>
                      ))}
                      <SelectItem value="__custom__">Remote / Custom</SelectItem>
                    </SelectContent>
                  </Select>
                  <Input
                    value={runForm.image}
                    onChange={e => setRunForm({ ...runForm, image: e.target.value })}
                    placeholder="nginx:latest or ghcr.io/org/image:tag"
                    className="h-8 bg-secondary/50"
                  />
                </div>
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
                <Input value={runForm.command} onChange={e => setRunForm({ ...runForm, command: e.target.value })} placeholder={defaultDetachedCommand(runForm.image) || 'echo hello'} className="h-8 bg-secondary/50" />
              </Field>
              <button
                type="button"
                className="ml-[92px] flex h-8 w-fit items-center gap-1.5 rounded-md px-2 text-[12px] text-muted-foreground transition hover:bg-accent hover:text-foreground"
                onClick={() => setShowAdvancedRun(value => !value)}
              >
                <SlidersHorizontal size={13} />
                Advanced
                <ChevronDown size={13} className={cn("transition", showAdvancedRun && "rotate-180")} />
              </button>
              {showAdvancedRun && (
	                <div className="space-y-3 border-t border-border/70 pt-3">
	                  <Field label="Network">
	                    <div className="grid grid-cols-[190px_minmax(0,1fr)] gap-2">
	                      <Select
	                        value={networkSelectValue(runForm.network, networks)}
	                        onValueChange={value => {
	                          if (value === '__omit__') {
	                            setRunForm({ ...runForm, network: '' })
	                            return
	                          }
	                          if (value === '__custom__') return
	                          setRunForm({ ...runForm, network: value })
	                        }}
	                      >
	                        <SelectTrigger className="h-8 border-border bg-secondary/50 text-[13px] focus:ring-0">
	                          <SelectValue placeholder="Select network" />
	                        </SelectTrigger>
	                        <SelectContent>
	                          <SelectItem value="__omit__">Default</SelectItem>
	                          {networks.map(network => {
	                            const value = network.name || network.id
	                            return (
	                              <SelectItem key={networkKey(network)} value={value}>
	                                {value}
	                              </SelectItem>
	                            )
	                          })}
	                          <SelectItem value="__custom__">Custom</SelectItem>
	                        </SelectContent>
	                      </Select>
	                      <Input
	                        value={runForm.network}
	                        onChange={e => setRunForm({ ...runForm, network: e.target.value })}
	                        placeholder="mynet or mynet,mtu=1500"
	                        className="h-8 bg-secondary/50"
	                      />
	                    </div>
	                  </Field>
                  <Field label="DNS">
                    <Input value={runForm.dns} onChange={e => setRunForm({ ...runForm, dns: e.target.value })} placeholder="1.1.1.1, 8.8.8.8" className="h-8 bg-secondary/50" />
                  </Field>
                  <Field label="Platform">
                    <Input value={runForm.platform} onChange={e => setRunForm({ ...runForm, platform: e.target.value })} placeholder="linux/arm64, linux/amd64" className="h-8 bg-secondary/50" />
                  </Field>
                  <div className="grid grid-cols-[80px_1fr] items-center gap-3">
                    <div className="text-right text-[12px] font-medium text-muted-foreground">Resources</div>
                    <ResourceSliders
                      cpus={runForm.cpus}
                      memory={runForm.memory}
                      hostResources={hostResources}
                      onChange={(cpus, memory) => setRunForm({ ...runForm, cpus, memory })}
                    />
                  </div>
                  <div className="grid grid-cols-[80px_1fr] items-center gap-3">
                    <div className="text-right text-[12px] font-medium text-muted-foreground">Process</div>
                    <div className="grid grid-cols-2 gap-2">
                      <Input value={runForm.workdir} onChange={e => setRunForm({ ...runForm, workdir: e.target.value })} placeholder="Workdir, e.g. /app" className="h-8 bg-secondary/50" />
                      <Input value={runForm.user} onChange={e => setRunForm({ ...runForm, user: e.target.value })} placeholder="User, e.g. 1000:1000" className="h-8 bg-secondary/50" />
                    </div>
                  </div>
                  <Field label="Entrypoint">
                    <Input value={runForm.entrypoint} onChange={e => setRunForm({ ...runForm, entrypoint: e.target.value })} placeholder="/bin/sh" className="h-8 bg-secondary/50" />
                  </Field>
                  <Field label="Labels">
                    <Input value={runForm.labels} onChange={e => setRunForm({ ...runForm, labels: e.target.value })} placeholder="env=dev, owner=void" className="h-8 bg-secondary/50" />
                  </Field>
                  <div className="flex flex-wrap gap-2 pl-[92px]">
                    <Toggle label="No DNS" checked={runForm.noDns} onClick={() => setRunForm({ ...runForm, noDns: !runForm.noDns })} />
                    <Toggle label="Init" checked={runForm.init} onClick={() => setRunForm({ ...runForm, init: !runForm.init })} />
                    <Toggle label="Read only" checked={runForm.readOnly} onClick={() => setRunForm({ ...runForm, readOnly: !runForm.readOnly })} />
                    <Toggle label="SSH agent" checked={runForm.ssh} onClick={() => setRunForm({ ...runForm, ssh: !runForm.ssh })} />
                  </div>
                </div>
              )}
              <div className="flex gap-2 pl-[92px]">
                <Toggle label="Detach" checked={runForm.detach} onClick={() => setRunForm({ ...runForm, detach: !runForm.detach, remove: runForm.detach ? runForm.remove : false })} />
                <Toggle label="Remove on exit" checked={runForm.remove && !runForm.detach} disabled={runForm.detach} onClick={() => setRunForm({ ...runForm, remove: !runForm.remove })} />
              </div>
              <CommandPreview command={runCommandPreview} />
            </FlowPanel>
          )}
        </main>

        {showDetailPane && (
          <section className="min-h-0 min-w-0 overflow-hidden">
            {mode === 'images' ? (
              selectedImage ? (
                <ImageDetails
                  image={selectedImage}
                  busy={busy}
                  onRun={() => {
                    setRunForm({ ...emptyRunForm, image: imageReference(selectedImage) })
                    setMode('run')
                  }}
                  onRequestRemove={() => {
                    setPendingImageDeleteKey(imageKey(selectedImage))
                    setImageDeleteConfirmStep(1)
                  }}
                  onCancelRemove={() => {
                    setPendingImageDeleteKey('')
                    setImageDeleteConfirmStep(0)
                  }}
                  onAdvanceRemove={() => setImageDeleteConfirmStep(2)}
                  onConfirmRemove={() => removeImage(selectedImage)}
                  deleteConfirmStep={pendingImageDeleteKey === imageKey(selectedImage) ? imageDeleteConfirmStep : 0}
                />
              ) : (
                <EmptyState text="Select an image to inspect it." />
              )
            ) : mode === 'networks' ? (
              showNetworkCreate ? (
                <NetworkCreatePanel
                  form={networkForm}
                  busy={busy || !ready}
                  onChange={setNetworkForm}
                  onCancel={() => setShowNetworkCreate(false)}
                  onCreate={createNetwork}
                />
              ) : selectedNetwork ? (
                <NetworkDetails
                  network={selectedNetwork}
                  busy={busy}
                  deleteConfirm={pendingNetworkDeleteKey === networkKey(selectedNetwork) && networkDeleteConfirm}
                  onUseInRun={() => {
                    setRunForm({ ...emptyRunForm, network: selectedNetwork.name || selectedNetwork.id })
                    setMode('run')
                    setShowAdvancedRun(true)
                  }}
                  onRequestRemove={() => {
                    setPendingNetworkDeleteKey(networkKey(selectedNetwork))
                    setNetworkDeleteConfirm(true)
                  }}
                  onCancelRemove={() => {
                    setPendingNetworkDeleteKey('')
                    setNetworkDeleteConfirm(false)
                  }}
                  onConfirmRemove={() => removeNetwork(selectedNetwork)}
                />
              ) : (
                <EmptyState text="Select a network to inspect it." />
              )
            ) : selected ? (
              <div className="flex h-full min-h-0 flex-col overflow-hidden">
                <div className="border-b border-launcher-border px-3 py-3">
                  <div className="flex items-center justify-between gap-3">
                    <div className="min-w-0">
                      <div className="truncate text-[14px] font-semibold">{selected.name || selected.id}</div>
                      <div className="truncate text-[11px] text-muted-foreground">{selected.image || selected.id}</div>
                    </div>
                    <StatusPill status={selected.status} />
                  </div>
                  <div className="mt-3 flex flex-wrap items-center gap-1.5">
                    {isContainerRunning(selected) ? (
                      <IconAction title="Stop" icon={<Square size={14} />} onClick={() => performContainerAction('stop')} />
                    ) : (
                      <IconAction title="Start" icon={<Play size={14} />} onClick={() => performContainerAction('start')} />
                    )}
                    <IconAction title="Restart" icon={<ListRestart size={14} />} onClick={() => performContainerAction('restart')} />
                    {isContainerRunning(selected) && (
                      <IconAction title="Open Shell" icon={<Terminal size={14} />} onClick={() => performContainerAction('shell')} />
                    )}
                    <IconAction title="Delete Container" icon={<Trash2 size={14} />} danger onClick={() => performContainerAction('remove')} />
                  </div>
                </div>

                <Tabs value={detailTab} onValueChange={value => setDetailTab(value as DetailTab)} className="flex min-h-0 flex-1 flex-col">
                  <TabsList className="mx-3 mt-3 grid h-8 shrink-0 grid-cols-3 bg-secondary/40">
                    <TabsTrigger value="summary" className="text-[12px]">Summary</TabsTrigger>
                    <TabsTrigger value="logs" className="text-[12px]">Logs</TabsTrigger>
                    <TabsTrigger value="inspect" className="text-[12px]">Inspect</TabsTrigger>
                  </TabsList>
                  <div className="min-h-0 flex-1 overflow-hidden p-3">
                    {detailTab === 'summary' && (
                      <div className="space-y-3 text-[12px]">
                        <SummaryRow label="ID" value={selected.id} />
                        <SummaryRow label="Image" value={selected.image || '-'} />
                        <SummaryRow label="IP" value={selected.ip || '-'} />
                        <SummaryRow label="Platform" value={platformLabel(selected) || '-'} />
                        <SummaryRow label="Resources" value={[selected.cpus ? `${selected.cpus} CPU` : '', selected.memory].filter(Boolean).join(' / ') || '-'} />
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
        )}
      </div>
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
  const running = isRunningStatus(normalized)
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

function containerStatusLabel(status: ContainerCLIStatus | null, ready: boolean) {
  if (!status) return 'checking container'
  if (status.supported === false) return 'macOS unsupported'
  if (ready) return 'container ready'
  if (!status.available) return 'container missing'
  return 'system stopped'
}

function isContainerRunning(container?: ContainerInfo) {
  return isRunningStatus(container?.status || '')
}

function isRunningStatus(status: string) {
  return status.toLowerCase().includes('running')
}

function IconAction({ title, icon, danger, onClick }: { title: string; icon: React.ReactNode; danger?: boolean; onClick: () => void }) {
  return (
    <Button
      variant="ghost"
      size="sm"
      title={title}
      onClick={onClick}
      className={cn("h-7 gap-1.5 px-2 text-muted-foreground hover:text-foreground", danger && "hover:bg-red-500/10 hover:text-red-500")}
    >
      {icon}
      <span>{title}</span>
    </Button>
  )
}

function ImageDetails({
  image,
  busy,
  deleteConfirmStep,
  onRun,
  onRequestRemove,
  onCancelRemove,
  onAdvanceRemove,
  onConfirmRemove,
}: {
  image: ImageInfo
  busy: boolean
  deleteConfirmStep: 0 | 1 | 2
  onRun: () => void
  onRequestRemove: () => void
  onCancelRemove: () => void
  onAdvanceRemove: () => void
  onConfirmRemove: () => void
}) {
  const reference = imageReference(image)
  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden">
      <div className="border-b border-launcher-border px-3 py-3">
        <div className="min-w-0">
          <div className="truncate text-[14px] font-semibold">{image.name}</div>
          <div className="truncate text-[11px] text-muted-foreground">{image.tag || image.id || image.digest || 'local image'}</div>
        </div>
        <div className="mt-3 flex flex-wrap items-center gap-1.5">
          <Button variant="ghost" size="sm" className="h-7 gap-1.5 text-muted-foreground hover:text-foreground" onClick={onRun} disabled={busy}>
            <Play size={14} /> Run Image
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-7 gap-1.5 text-muted-foreground hover:text-foreground"
            onClick={() => navigator.clipboard.writeText(reference).catch(() => { })}
            disabled={busy}
          >
            Copy Reference
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-7 gap-1.5 text-muted-foreground hover:bg-red-500/10 hover:text-red-500"
            onClick={onRequestRemove}
            disabled={busy}
          >
            <Trash2 size={14} /> Delete Image
          </Button>
        </div>
      </div>
      {deleteConfirmStep > 0 && (
        <div className="border-b border-red-500/15 bg-red-500/8 px-3 py-3">
          <div className="text-[12px] font-medium text-red-600 dark:text-red-300">
            {deleteConfirmStep === 1 ? 'Delete this image?' : 'Confirm permanent deletion'}
          </div>
          <div className="mt-1 break-all text-[11px] leading-relaxed text-muted-foreground">{reference}</div>
          <div className="mt-3 flex items-center gap-2">
            <Button variant="ghost" size="sm" className="h-7 text-muted-foreground" onClick={onCancelRemove} disabled={busy}>
              Cancel
            </Button>
            {deleteConfirmStep === 1 ? (
              <Button variant="secondary" size="sm" className="h-7 gap-1.5 text-red-600 hover:text-red-600" onClick={onAdvanceRemove} disabled={busy}>
                <Trash2 size={14} /> Confirm
              </Button>
            ) : (
              <Button variant="secondary" size="sm" className="h-7 gap-1.5 bg-red-500/15 text-red-600 hover:bg-red-500/20 hover:text-red-600" onClick={onConfirmRemove} disabled={busy}>
                {busy ? <Loader2 size={14} className="animate-spin" /> : <Trash2 size={14} />}
                Delete
              </Button>
            )}
          </div>
        </div>
      )}
      <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-3 text-[12px]">
        <DetailRow label="Ref" value={reference} />
        <DetailRow label="ID" value={image.id || '-'} />
        <DetailRow label="Digest" value={image.digest || '-'} />
        <DetailRow label="Size" value={image.size || '-'} />
        <DetailRow label="Created" value={formatDate(image.createdAt)} />
      </div>
    </div>
  )
}

function NetworkCreatePanel({
  form,
  busy,
  onChange,
  onCancel,
  onCreate,
}: {
  form: typeof emptyNetworkForm
  busy: boolean
  onChange: (form: typeof emptyNetworkForm) => void
  onCancel: () => void
  onCreate: () => void
}) {
  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden">
      <div className="border-b border-launcher-border px-3 py-3">
        <div className="text-[14px] font-semibold">Create Network</div>
        <div className="text-[11px] text-muted-foreground">container network create</div>
      </div>
      <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-3 text-[12px]">
        <Field label="Name">
          <Input value={form.name} onChange={e => onChange({ ...form, name: e.target.value })} placeholder="mynet" className="h-8 bg-secondary/50" />
        </Field>
        <Field label="Mode">
          <Toggle label="Internal host-only" checked={form.internal} onClick={() => onChange({ ...form, internal: !form.internal })} />
        </Field>
        <Field label="Subnet">
          <Input value={form.subnet} onChange={e => onChange({ ...form, subnet: e.target.value })} placeholder="192.168.128.0/24" className="h-8 bg-secondary/50" />
        </Field>
        <Field label="IPv6">
          <Input value={form.subnetV6} onChange={e => onChange({ ...form, subnetV6: e.target.value })} placeholder="fd00:128::/64" className="h-8 bg-secondary/50" />
        </Field>
        <Field label="Plugin">
          <Input value={form.plugin} onChange={e => onChange({ ...form, plugin: e.target.value })} placeholder="container-network-vmnet" className="h-8 bg-secondary/50" />
        </Field>
        <Field label="Labels">
          <Input value={form.labels} onChange={e => onChange({ ...form, labels: e.target.value })} placeholder="env=dev, owner=void" className="h-8 bg-secondary/50" />
        </Field>
        <Field label="Options">
          <Input value={form.options} onChange={e => onChange({ ...form, options: e.target.value })} placeholder="key=value" className="h-8 bg-secondary/50" />
        </Field>
      </div>
      <div className="flex shrink-0 justify-end gap-2 border-t border-launcher-border p-3">
        <Button variant="ghost" size="sm" className="h-8" onClick={onCancel} disabled={busy}>
          Cancel
        </Button>
        <Button size="sm" className="h-8 gap-1.5 bg-indigo-500/30 text-indigo-700 hover:bg-indigo-500/40 dark:text-indigo-100" onClick={onCreate} disabled={busy}>
          {busy ? <Loader2 size={14} className="animate-spin" /> : <Plus size={14} />}
          Create
        </Button>
      </div>
    </div>
  )
}

function NetworkDetails({
  network,
  busy,
  deleteConfirm,
  onUseInRun,
  onRequestRemove,
  onCancelRemove,
  onConfirmRemove,
}: {
  network: NetworkInfo
  busy: boolean
  deleteConfirm: boolean
  onUseInRun: () => void
  onRequestRemove: () => void
  onCancelRemove: () => void
  onConfirmRemove: () => void
}) {
  const builtin = isBuiltinNetwork(network)
  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden">
      <div className="border-b border-launcher-border px-3 py-3">
        <div className="min-w-0">
          <div className="truncate text-[14px] font-semibold">{network.name || network.id}</div>
          <div className="truncate text-[11px] text-muted-foreground">{network.plugin || 'container network'}</div>
        </div>
        <div className="mt-3 flex flex-wrap items-center gap-1.5">
          <Button variant="ghost" size="sm" className="h-7 gap-1.5 text-muted-foreground hover:text-foreground" onClick={onUseInRun}>
            <Play size={14} /> Use Network
          </Button>
          <Button
            variant="ghost"
            size="sm"
            className="h-7 gap-1.5 text-muted-foreground hover:bg-red-500/10 hover:text-red-500"
            onClick={onRequestRemove}
            disabled={busy || builtin}
            title={builtin ? 'Built-in network cannot be deleted' : 'Delete network'}
          >
            <Trash2 size={14} /> Delete Network
          </Button>
        </div>
      </div>
      {deleteConfirm && (
        <div className="border-b border-red-500/15 bg-red-500/8 px-3 py-3">
          <div className="text-[12px] font-medium text-red-600 dark:text-red-300">Delete this network?</div>
          <div className="mt-1 break-all text-[11px] leading-relaxed text-muted-foreground">{network.name || network.id}</div>
          <div className="mt-3 flex items-center gap-2">
            <Button variant="ghost" size="sm" className="h-7 text-muted-foreground" onClick={onCancelRemove} disabled={busy}>
              Cancel
            </Button>
            <Button variant="secondary" size="sm" className="h-7 gap-1.5 bg-red-500/15 text-red-600 hover:bg-red-500/20 hover:text-red-600" onClick={onConfirmRemove} disabled={busy}>
              {busy ? <Loader2 size={14} className="animate-spin" /> : <Trash2 size={14} />}
              Delete
            </Button>
          </div>
        </div>
      )}
      <div className="min-h-0 flex-1 space-y-3 overflow-y-auto p-3 text-[12px]">
        <DetailRow label="ID" value={network.id || '-'} />
        <DetailRow label="Mode" value={network.mode || '-'} />
        <DetailRow label="Plugin" value={network.plugin || '-'} />
        <DetailRow label="IPv4" value={network.ipv4Subnet || '-'} />
        <DetailRow label="Gateway" value={network.ipv4Gateway || '-'} />
        <DetailRow label="IPv6" value={network.ipv6Subnet || '-'} />
        <DetailRow label="Labels" value={(network.labels || []).join(', ') || '-'} />
        <DetailRow label="Options" value={(network.options || []).join(', ') || '-'} />
        <DetailRow label="Created" value={formatDate(network.createdAt)} />
      </div>
    </div>
  )
}

function ResourceSliders({
  cpus,
  memory,
  hostResources,
  onChange,
}: {
  cpus: string
  memory: string
  hostResources: HostResourceInfo | null
  onChange: (cpus: string, memory: string) => void
}) {
  const cpuMax = Math.max(1, hostResources?.cpuCores || 8)
  const memoryMax = Math.max(256, roundDownToStep(hostResources?.memoryMiB || 8192, 256))
  const defaultCPUs = Math.min(4, cpuMax)
  const defaultMemory = Math.min(1024, memoryMax)
  const cpuCustom = runResourceValue(cpus) !== ''
  const memoryCustom = runResourceValue(memory) !== ''
  const cpuValue = clampNumber(parseInt(runResourceValue(cpus), 10) || defaultCPUs, 1, cpuMax)
  const memoryValue = clampNumber(parseMemoryMiB(runResourceValue(memory)) || defaultMemory, 256, memoryMax)

  return (
    <div className="space-y-3 rounded-md border border-border/70 bg-secondary/20 p-3">
      <div className="flex items-center justify-between gap-3">
        <div className="text-[11px] text-muted-foreground">
          Host: {cpuMax} CPU / {formatMemoryMiB(memoryMax)}
        </div>
        <Button
          variant="ghost"
          size="sm"
          className="h-7 px-2 text-[12px] text-muted-foreground"
          onClick={() => onChange('', '')}
          disabled={!cpuCustom && !memoryCustom}
        >
          Use Defaults
        </Button>
      </div>
      <ResourceSlider
        label="CPU"
        value={cpuValue}
        min={1}
        max={cpuMax}
        step={1}
        valueLabel={`${cpuCustom ? cpuValue : defaultCPUs} cores${cpuCustom ? '' : ' default'}`}
        onChange={value => onChange(String(value), memory)}
      />
      <ResourceSlider
        label="Memory"
        value={memoryValue}
        min={256}
        max={memoryMax}
        step={256}
        valueLabel={`${formatMemoryMiB(memoryCustom ? memoryValue : defaultMemory)}${memoryCustom ? '' : ' default'}`}
        onChange={value => onChange(cpus, `${value}M`)}
      />
    </div>
  )
}

function ResourceSlider({
  label,
  value,
  min,
  max,
  step,
  valueLabel,
  onChange,
}: {
  label: string
  value: number
  min: number
  max: number
  step: number
  valueLabel: string
  onChange: (value: number) => void
}) {
  return (
    <div className="grid grid-cols-[70px_minmax(0,1fr)_118px] items-center gap-3">
      <div className="text-[12px] font-medium text-muted-foreground">{label}</div>
      <input
        type="range"
        min={min}
        max={max}
        step={step}
        value={value}
        onChange={event => onChange(Number(event.target.value))}
        className="h-2 w-full cursor-pointer accent-indigo-500"
      />
      <div className="text-right font-mono text-[11px] text-foreground/80">{valueLabel}</div>
    </div>
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

function FlowPanel({ title, actionLabel, busy, disabled, children, onAction }: { title: string; actionLabel: string; busy: boolean; disabled?: boolean; children: React.ReactNode; onAction: () => void }) {
  return (
    <div className="h-full min-h-0 overflow-y-auto">
      <div className="mx-auto flex max-w-[720px] flex-col gap-3 px-6 py-6">
        <div>
          <div className="text-[15px] font-semibold">{title}</div>
          <div className="text-[12px] text-muted-foreground">Keyboard-first Apple Container command flow.</div>
        </div>
        {children}
        <div className="flex justify-end pb-2">
          <Button onClick={onAction} disabled={busy || disabled} className="h-8 gap-1.5 bg-indigo-500/30 text-indigo-700 hover:bg-indigo-500/40 dark:text-indigo-100">
            {busy ? <Loader2 size={14} className="animate-spin" /> : <Play size={14} />}
            {actionLabel}
          </Button>
        </div>
      </div>
    </div>
  )
}

function Toggle({ label, checked, disabled, onClick }: { label: string; checked: boolean; disabled?: boolean; onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "rounded-md border px-2 py-1 text-[12px] transition",
        checked ? "border-indigo-500/30 bg-indigo-500/15 text-foreground" : "border-border bg-secondary/40 text-muted-foreground",
        disabled && "cursor-not-allowed opacity-45"
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

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid grid-cols-[68px_minmax(0,1fr)] gap-3">
      <div className="text-right text-muted-foreground">{label}</div>
      <div className="min-w-0 break-all font-mono text-[11px] leading-relaxed text-foreground/80">{value}</div>
    </div>
  )
}

function CodeBlock({ text }: { text: string }) {
  return (
    <pre className="h-full min-h-0 overflow-auto whitespace-pre-wrap break-words rounded-md border border-border bg-black/85 p-3 font-mono text-[11px] leading-relaxed text-zinc-200">
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

function imageKey(image: ImageInfo) {
  return image.id || image.digest || imageReference(image)
}

function networkKey(network?: NetworkInfo) {
  return network?.id || network?.name || ''
}

function networkSelectValue(value: string, networks: NetworkInfo[]) {
  const trimmed = value.trim()
  if (!trimmed) return '__omit__'
  return networks.some(network => (network.name || network.id) === trimmed) ? trimmed : '__custom__'
}

function runResourceValue(value: string) {
  const trimmed = value.trim()
  if (!trimmed || isZeroResourceValue(trimmed)) return ''
  return trimmed
}

function parseMemoryMiB(value: string) {
  const trimmed = value.trim().toLowerCase()
  if (!trimmed) return 0
  const match = trimmed.match(/^(\d+(?:\.\d+)?)([kmgtp]?i?b?|bytes?)?$/)
  if (!match) return 0
  const amount = Number(match[1])
  if (!Number.isFinite(amount)) return 0
  const unit = match[2] || 'm'
  if (unit.startsWith('k')) return Math.max(1, Math.round(amount / 1024))
  if (unit.startsWith('g')) return Math.round(amount * 1024)
  if (unit.startsWith('t')) return Math.round(amount * 1024 * 1024)
  if (unit.startsWith('p')) return Math.round(amount * 1024 * 1024 * 1024)
  if (unit.startsWith('b') || unit === 'byte' || unit === 'bytes') return Math.max(1, Math.round(amount / 1024 / 1024))
  return Math.round(amount)
}

function formatMemoryMiB(value: number) {
  if (value >= 1024) {
    const gib = value / 1024
    return `${Number.isInteger(gib) ? gib.toFixed(0) : gib.toFixed(1)} GiB`
  }
  return `${value} MiB`
}

function clampNumber(value: number, min: number, max: number) {
  if (!Number.isFinite(value)) return min
  return Math.min(max, Math.max(min, value))
}

function roundDownToStep(value: number, step: number) {
  return Math.max(step, Math.floor(value / step) * step)
}

function isZeroResourceValue(value: string) {
  let cleaned = value.trim().toLowerCase().replace(/\s+/g, '')
  for (const suffix of ['bytes', 'byte', 'mib', 'gib', 'tib', 'pib', 'kib', 'mb', 'gb', 'tb', 'pb', 'kb', 'mi', 'gi', 'ti', 'pi', 'ki', 'b', 'm', 'g', 't', 'p', 'k']) {
    if (cleaned.endsWith(suffix)) {
      cleaned = cleaned.slice(0, -suffix.length)
      break
    }
  }
  return Number(cleaned) === 0
}

function imageSelectValue(value: string, images: ImageInfo[]) {
  const trimmed = value.trim()
  if (!trimmed) return '__custom__'
  return images.some(image => imageReference(image) === trimmed) ? trimmed : '__custom__'
}

function localImageReferences(images: ImageInfo[]) {
  const references: string[] = []
  const seen = new Set<string>()
  for (const image of images) {
    const reference = imageReference(image)
    if (!reference || seen.has(reference)) continue
    references.push(reference)
    seen.add(reference)
  }
  return references
}

function isBuiltinNetwork(network?: NetworkInfo) {
  if (!network) return false
  return network.name === 'default' || network.id === 'default' || (network.labels || []).some(label => label === 'com.apple.container.resource.role=builtin')
}

function imageReference(image: ImageInfo) {
  return image.tag ? `${image.name}:${image.tag}` : image.name
}

function effectiveRunCommand(form: typeof emptyRunForm) {
  const command = form.command.trim()
  if (command) return command
  if (!form.detach) return ''
  return defaultDetachedCommand(form.image)
}

function defaultDetachedCommand(image: string) {
  const name = normalizedImageName(image)
  return ['alpine', 'ubuntu', 'debian', 'busybox', 'fedora', 'centos', 'rockylinux', 'archlinux', 'oraclelinux'].includes(name)
    ? 'sleep infinity'
    : ''
}

function normalizedImageName(image: string) {
  let value = image.trim()
  if (!value) return ''
  value = value.split('@')[0]
  const lastSlash = value.lastIndexOf('/')
  const lastColon = value.lastIndexOf(':')
  if (lastColon > lastSlash) value = value.slice(0, lastColon)
  if (lastSlash >= 0) value = value.slice(lastSlash + 1)
  return value.toLowerCase()
}

function platformLabel(container: ContainerInfo) {
  return [container.os, container.arch].filter(Boolean).join('/')
}

function formatDate(value?: string) {
  if (!value) return '-'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return value
  return parsed.toLocaleString(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function compactDate(value?: string) {
  if (!value) return '-'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return value
  return parsed.toLocaleDateString(undefined, {
    month: '2-digit',
    day: '2-digit',
  })
}

export default ContainersView
