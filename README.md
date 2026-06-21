<div align="center">
  <h1>Void Launcher</h1>
  <p>A fast, keyboard-first macOS launcher and desktop assistant built with Go, Wails, React, and TypeScript.</p>

  <img src="https://img.shields.io/badge/macOS-000000?style=for-the-badge&logo=apple&logoColor=white" alt="macOS" />
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/React-20232A?style=for-the-badge&logo=react&logoColor=61DAFB" alt="React" />
  <img src="https://img.shields.io/badge/Wails_3-E00000?style=for-the-badge&logo=wails&logoColor=white" alt="Wails" />
  <img src="https://img.shields.io/badge/license-MIT-green?style=for-the-badge" alt="MIT License" />
</div>

Void is a macOS productivity launcher inspired by Raycast and Spotlight. It runs as a menu-bar accessory, stays out of the Dock, and can be opened from anywhere with a global shortcut. From one command palette you can launch apps, search files, run scripts, manage snippets, inspect clipboard history, ask local or remote AI models, and work with Apple Container workloads.

## Status

Void is macOS-first and still evolving. The repository contains Wails build scaffolding for other platforms, but the implemented native integrations currently depend on macOS APIs and command-line tools such as `open`, `pbcopy`, `pbpaste`, `osascript`, `mdfind`, and AppKit event taps.

## Features

- **Application launcher**: discovers macOS applications with Spotlight, falls back to scanning common app directories, caches app icons, and ranks frequently used commands with frecency.
- **System commands**: lock screen, sleep, empty trash, restart, shut down, open Finder or Terminal, show desktop, and move the frontmost window to halves, quarters, center, or fullscreen.
- **File search**: fuzzy-search files under the home directory and open, reveal in Finder, Quick Look, or copy paths through the action panel.
- **Web search**: type `@` to choose Google, Baidu, Bing, GitHub, or YouTube, then search directly from the launcher.
- **Clipboard history**: records recent text, URLs, file references, and images, with preview, copy, delete, and clear actions.
- **Snippets**: create reusable text snippets, search and copy them from the launcher, and expand configured prefixes globally.
- **Script commands**: run local `.sh`, `.py`, `.rb`, `.js`, `.ts`, `.zsh`, `.fish`, or `.swift` scripts. Void understands Raycast-style `@raycast.*` metadata, including title, mode, icon, package name, description, and up to five arguments.
- **Smart calculator**: evaluate math expressions and convert units for length, mass, data, time, speed, area, and temperature.
- **AI chat**: bring your own provider configuration for OpenAI, DeepSeek, Ollama, or OpenAI-compatible endpoints. AI requests stream progress events back to the UI.
- **Apple Containers**: view containers, images, and networks; pull and remove images; create and delete networks; run containers with ports, env vars, volumes, resources, DNS, platform, labels, SSH, and command preview; inspect logs and open shells.
- **Settings**: configure global hotkey, launch at login, menu-bar visibility, terminal app, text size, theme mode, AI provider, and optional skills directory.

## Screens and Workflows

The launcher opens with a single search box. Press `Enter` to run the selected item, `Cmd/Ctrl + K` to open the action panel, and `Cmd/Ctrl + ,` to open Settings.

Common entry points:

- `Clipboard History` opens the clipboard manager.
- `AI Chat` opens the assistant view.
- `Snippets` opens snippet management.
- `Script Commands` opens the local script runner.
- `Apple Containers` opens the container workspace.
- `Run Container` and `Pull Image` jump directly into Apple Container workflows.

## Requirements

- macOS. Native launcher, clipboard, hotkey, snippet expansion, and window-management features are macOS-specific.
- Go `1.26` or newer, matching `go.mod`.
- Node.js `20` or newer and npm.
- Wails v3 CLI:

  ```bash
  go install github.com/wailsapp/wails/v3/cmd/wails3@latest
  ```

Optional dependencies:

- [Task](https://taskfile.dev/) if you want to use `task dev`, `task build`, and `task package`.
- Python 3 for the calculator's fallback evaluator.
- Ollama for local AI models.
- Apple `container` CLI for the Apple Containers workspace. That feature requires macOS 26 or later and a running `container system`.
- Homebrew if you want Void's container install action to open `brew install container`.

## Development

Clone the repository and install frontend dependencies:

```bash
git clone https://github.com/dyike/Void.git
cd Void
cd frontend && npm install && cd ..
```

Run in development mode:

```bash
wails3 dev -config ./build/config.yml -port 9245
```

Or, with Task installed:

```bash
task dev
```

On first launch, macOS may ask for Accessibility permissions. Grant them in System Settings > Privacy & Security > Accessibility if you want global hotkeys, snippet expansion, and window-management commands to work reliably.

## Build

Create a production build:

```bash
wails3 build -config ./build/config.yml
```

Package the app using the Wails task configuration:

```bash
wails3 package
```

Or use the Taskfile wrappers:

```bash
task build
task package
```

Build artifacts are written under `bin/` and Wails build directories.

## Test

Run Go tests:

```bash
go test ./...
```

Run the frontend type-check and production build:

```bash
cd frontend
npm run build
```

If you change exported Go services or models, regenerate Wails bindings before committing generated frontend bindings:

```bash
wails3 generate bindings
```

## Data and Privacy

Void stores local state in:

```text
~/Library/Application Support/Void
```

Important files and directories:

- `settings.json`: app settings and AI provider configuration.
- `clipboard-history.json`: clipboard history metadata.
- `clipboard-assets/`: persisted image clipboard items.
- `snippets.json`: snippet definitions.
- `script-commands/`: user script commands and generated examples.
- `frecency.json`: command usage scores.
- `icon-cache/`: extracted application icons.

API keys are stored locally in `settings.json`. Do not commit files from the application support directory.

## Script Command Metadata

Example script:

```bash
#!/bin/bash
# @raycast.schemaVersion 1
# @raycast.title Current Time
# @raycast.mode compact
# @raycast.icon Clock
# @raycast.packageName Utilities
# @raycast.description Show current date and time

date "+%Y-%m-%d %H:%M:%S %Z"
```

Scripts live in:

```text
~/Library/Application Support/Void/script-commands
```

Void creates a few example scripts on first run. You can open the folder from the Script Commands view.

## Project Structure

```text
Void/
├── main.go                  # Wails app bootstrap, service registration, window setup
├── wire.go                  # Dependency injection providers
├── wire_gen.go              # Generated Wire injector
├── internal/
│   ├── autostart/           # Launch-at-login integration
│   ├── model/               # Shared Go models exposed to Wails bindings
│   ├── repo/                # OS integration, persistence, command execution
│   │   └── filesearch/      # Pure-Go file name and content search
│   ├── service/             # Wails-facing application services
│   └── window/              # Main window, settings window, tray, dock service
├── frontend/
│   ├── src/                 # React views and UI logic
│   ├── components/          # Shared UI components
│   ├── bindings/            # Generated Wails TypeScript bindings
│   └── public/              # Static frontend assets
├── build/                   # Wails platform build configuration and assets
├── Taskfile.yml             # Development and packaging tasks
└── release.sh               # Release helper script
```

## Contributing

Issues and pull requests are welcome. For code changes, please include focused tests when behavior changes and run:

```bash
go test ./...
cd frontend && npm run build
```

Keep changes scoped and avoid committing local app data, build outputs, or generated files unrelated to the change.

## License

Void is released under the [MIT License](LICENSE).
