# Void

Void 是基于 Go 和 [Keel](https://github.com/dyike/keel) 的 macOS 启动器，提供应用搜索、剪贴板历史、文本片段、脚本命令、文件搜索和 Apple Containers 管理。界面由 Keel/Gio 绘制，使用 macOS 原生窗口材质，目前界面语言为英文。

## 功能

| 功能 | 用法 |
| --- | --- |
| 应用与系统命令 | 输入名称搜索应用、系统设置或系统命令；支持置顶，排序结合搜索匹配、使用频率和最近使用时间。 |
| 计算器 | 直接输入算式或单位换算，例如 `2+3`、`10 km to m`，选中结果后按 Enter 复制。 |
| 网页搜索 | 输入 `@关键词`，选择 Google、Baidu、Bing、GitHub 或 YouTube 后打开搜索。 |
| 剪贴板历史 | 搜索、复制、预览和删除文本、图片或文件记录；支持清空历史。图片和单个图片文件可显示缩略图，选中后悬停可放大预览。 |
| 文本片段 | 创建、编辑、删除和复制片段，配置展开前缀及标签；在其他应用中输入前缀可触发展开。 |
| 脚本命令 | 从数据目录中的 `script-commands/` 发现带 Raycast 元数据的可执行脚本，填写参数、运行并查看输出。 |
| 文件搜索 | 按文件名模糊搜索，或搜索文本文件内容；支持打开文件、在 Finder 中显示、Quick Look 和复制路径。 |
| Apple Containers | 管理容器、镜像和网络，查看日志与详情，打开 shell，拉取镜像，以及创建容器或网络。 |

应用索引从 `/Applications`、`/System/Applications` 和 `~/Applications` 收集应用，过滤备份目录、失效链接和内嵌辅助程序，并按 Bundle ID 与真实路径合并副本。Nix / Home Manager 的有效应用链接会保留。

文件名和文件内容搜索至少需要两个字符，从用户主目录开始扫描，跳过隐藏目录、`Library`、`node_modules` 等目录，最多显示 30 条结果。内容搜索仅检查支持的文本文件类型，单个文件不超过 1 MiB。

## 使用

默认按 `Option+Space` 显示或收起 Void，也可从菜单栏打开。搜索并执行 `Clipboard History`、`Snippets`、`Script Commands`、`File Search`、`Search File Contents` 或 `Apple Containers` 进入对应功能。

| 快捷键 | 操作 |
| --- | --- |
| `↑` / `↓` | 选择列表条目 |
| `Enter` | 执行当前条目 |
| `Cmd+K` | 打开或关闭当前条目的操作菜单 |
| `Cmd+,` | 打开 Settings |
| `Cmd+0` | 返回主搜索 |
| `Cmd+1` / `Cmd+2` / `Cmd+3` | 打开剪贴板、片段、脚本 |
| `Cmd+4` / `Cmd+5` | 打开文件名搜索、容器管理 |
| `Esc` | 关闭预览、表单或操作菜单，清空搜索，返回主搜索，最后收起窗口 |
| `Cmd+Q` | 在启动器中退出 Void |

启动器失去焦点后会收起。关闭启动器或设置窗口后，应用继续常驻；菜单栏中的 `Quit Void` 可退出应用。

### 设置与快捷键录制

Settings 分为 General、Appearance 和 Advanced，左侧的 Search 可搜索设置项。

- **General**：登录启动、菜单栏图标、全局快捷键。
- **Appearance**：System / Light / Dark 外观、文字大小和窗口材质。
- **Advanced**：打开终端命令时使用的终端应用。

修改全局快捷键时，点击 `Global shortcut` 右侧的控件，然后直接按组合键，例如 `Option+Space` 或 `Cmd+Shift+Space`。组合键需要包含 Command、Option 或 Control；按 Esc 可取消录制。录制期间暂停全局快捷键和片段展开，录制结束或离开窗口后恢复。

点击 `Save settings` 或按 `Cmd+S` 保存修改；`Revert` 放弃尚未保存的修改。录制快捷键期间，`Cmd+S` 会被录为快捷键，结束录制后再点击保存。

默认材质为 Frosted。`Launcher material` 同时影响启动器与设置窗口，可选 Clear glass、Soft glass 或 Frosted，保存后立即应用。Clear glass 和 Soft glass 在 macOS 26+ 使用 Liquid Glass，旧系统回退到毛玻璃。实际透光效果也受 macOS「降低透明度」设置影响。

## 从源码运行

需要 macOS 14+、Go 1.26.1+ 和 Xcode Command Line Tools。项目的剪贴板和全局键盘集成使用 cgo，构建时需要启用它。Python 3 用于计算器的回退求值和界面冒烟测试；Apple Containers 功能还需要 macOS 26+，以及已安装并启动的 Apple `container` CLI。

Void 使用已发布的 Keel v0.3.4，依赖由 Go 模块下载。首次获取源码并启动：

```sh
git clone https://github.com/dyike/Void.git
cd Void
go mod download
go tool keel run
```

需要同时修改 Keel 时，可将其放在 Void 的同级目录，并在 Void 目录中创建本地 workspace：

```sh
go work init . ../keel
```

`go tool keel run` 会监听文件修改并重新编译，也会监听 workspace 中的本地依赖；`go run .` 可直接运行。Keel CLI 已通过 Go 的 tool 指令声明。`go.work` 和 `go.work.sum` 不提交到仓库，发布构建使用 `GOWORK=off` 解析正式依赖。

首次运行后，在「系统设置 → 隐私与安全性」中为实际运行的 Void 应用授予辅助功能和输入监控权限，以启用全局快捷键和片段展开。用于日常使用时，先将打包应用放到固定位置，再启用 `Launch at login`；登录启动记录的是当前可执行文件的路径。

### 构建

```sh
GOWORK=off go tool keel build       # 生成 dist/Void.app
open dist/Void.app
```

`./release.sh` 先运行 `go test ./...`，再调用 Keel 打包，并将收到的参数传给 `keel build`。默认构建使用本机 ad hoc 签名；分发给其他 Mac 时需要另行签名和公证。应用 ID、版本和图标路径配置在 [keel.json](keel.json)。

### 独立预览与截图

使用临时数据打开预览，不读取日常配置，也不启用全局快捷键、剪贴板轮询、菜单栏或登录启动集成。预览窗口失去焦点后仍保持显示：

```sh
preview_dir=$(mktemp -d /tmp/void-preview.XXXXXX)
go run . -preview -data-dir "$preview_dir" -material frosted -preview-backdrop
```

`-material` 可选 `clear`、`regular` 或 `frosted`。`-preview-backdrop` 在窗口后显示彩色参照，用于观察原生材质效果。

生成启动器的离屏 PNG 并退出：

```sh
screenshot_dir=$(mktemp -d /tmp/void-screenshot.XXXXXX)
go run . -data-dir "$screenshot_dir" -screenshot "$screenshot_dir/launcher.png"
```

离屏渲染使用实色背景。离屏截图或独立窗口截图均不能作为桌面背景透光、模糊和折射效果的判断依据。

### 验证

```sh
go test ./...
go vet ./...
python3 scripts/smoke.py
```

[scripts/smoke.py](scripts/smoke.py) 构建带竞态检测的应用，以临时数据运行离屏交互检查，覆盖剪贴板图片预览、计算器、片段、设置搜索与保存、快捷键录制、材质切换、容器表单和常驻生命周期。

在 macOS 图形会话中运行原生焦点回归：

```sh
KEEL_DESKTOP=1 go test -tags keelnativeqa -run '^TestNativeLauncherActivation$' -count=1 ./internal/window
```

该测试使用临时应用和数据，打开真实窗口，覆盖三种材质下的输入、`Cmd+,`、设置窗口层级和应用退到后台后的连续唤起。测试向应用自身发送键盘事件，不需要辅助功能或输入监控权限。

## 数据

默认数据目录为 `~/Library/Application Support/Void/`，可通过 `-data-dir` 指定其他目录。

```text
Void/
├── settings.json          # 设置、置顶和命令配置
├── clipboard-history.json # 剪贴板历史
├── clipboard-assets/      # 保存的剪贴板图片
├── snippets.json          # 文本片段
├── script-commands/       # 可执行脚本
├── frecency.json          # 命令使用记录
└── icon-cache/            # 应用图标缓存
```

登录启动另使用 `~/Library/LaunchAgents/com.dyike.void.plist`。修改数据目录前请退出正在运行的实例，备份时同时保留历史文件和 `clipboard-assets/`。

## 代码与图标

```text
main.go             # 入口、常驻生命周期、全局快捷键
keel.json           # 应用标识、版本和打包配置
wire.go             # 依赖注入定义
wire_gen.go         # 生成的注入代码
internal/window/    # 启动器、设置、表单和容器界面
internal/service/   # 业务服务
internal/repo/      # 持久化、搜索和 macOS 集成
internal/model/     # 共享数据模型
internal/autostart/ # 登录启动
internal/branding/  # 应用与托盘图标的 SVG 和生成的 PNG
scripts/icons/      # SVG 栅格化工具
scripts/smoke.py    # 离屏界面交互检查
```

修改依赖注入后运行 `go generate .`。应用图标和托盘图标共用偏心圆环轮廓，编辑 [app.svg](internal/branding/app.svg) 或 [tray.svg](internal/branding/tray.svg) 后运行：

```sh
go generate ./internal/branding
```

运行时加载生成的 PNG。Keel 打包时为应用图标添加平台圆角与阴影；菜单栏使用 18pt 的 @2x 单色模板，由 macOS 适配深浅色背景。

## License

[MIT](LICENSE)
