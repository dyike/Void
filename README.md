<div align="center">
  <h1>🌌 Void Launcher</h1>
  <p>A fast, extensible, open-source Raycast alternative built with Go and React.</p>

  <img src="https://img.shields.io/badge/macOS-000000?style=for-the-badge&logo=apple&logoColor=white" alt="macOS" />
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/React-20232A?style=for-the-badge&logo=react&logoColor=61DAFB" alt="React" />
  <img src="https://img.shields.io/badge/Wails_3-E00000?style=for-the-badge&logo=wails&logoColor=white" alt="Wails" />
</div>

<br/>

Void is a desktop productivity tool for macOS that gives you quick access to your apps, system commands, AI models, clipboard history, scripts, and snippets. It runs as a menu-bar accessory (no Dock icon) and can be summoned instantly via a customizable global hotkey.

## ✨ Features

- **🚀 App Launcher:** Instantly search and launch any macOS application.
- **⌘  Global Hotkey:** Summon Void from anywhere (defaults to `⌥ + Space`). Fully customizable with auto-detection of modifier keys.
- **📋 Clipboard History:** Auto-records copied text, images, files, and links. Search, preview, and paste past items instantly.
- **🤖 AI Chat:** Built-in AI assistant. Bring your own API key. Supports OpenAI, Anthropic, Ollama (local), and OpenAI-compatible endpoints.
- **💻 Script Commands:** Run custom bash, python, or Node scripts directly from the launcher. Fully compatible with Raycast's `@raycast.xxx` metadata format.
- **✂️ Snippets:** Create, search, and copy your most used text snippets.
- **🧮 Smart Calculator:** Evaluate complex math expressions, do unit conversions (length, weight, data, time, speed), and temperature conversions straight from the search bar.
- **⚙️ System Commands:** Quickly lock screen, sleep, restart, empty trash, or toggle system settings.

## 🛠 Tech Stack

- **Backend:** [Go](https://go.dev/)
- **Desktop Framework:** [Wails 3 (Alpha)](https://wails.io/docs/next/gettingstarted/installation) - Provides native OS integration with a lightweight webview instead of bundled Chromium.
- **Frontend:** [React](https://react.dev/), [TypeScript](https://www.typescriptlang.org/), and [Vite](https://vitejs.dev/).
- **Styling:** Vanilla CSS + Lucide Icons.

## 📦 Installation & Build

### Prerequisites
1. Install [Go (1.21+)](https://go.dev/dl/).
2. Install [Node.js & npm](https://nodejs.org/).
3. Install the **Wails v3 CLI**:
   ```bash
   go install github.com/wailsapp/wails/v3/cmd/wails3@latest
   ```

### Running Locally
1. Clone the repository:
   ```bash
   git clone git@github.com:dyike/Void.git
   cd Void
   ```
2. Start the dev server (compiles Go, runs Vite, and boots the app):
   ```bash
   wails3 dev
   ```
   *Note: On first run, macOS will ask for Accessibility permissions so Void can register the global hotkey.*

### Building for Production
To build a standardized macOS `.app` bundle:
```bash
wails3 build
```
The compiled binary will be placed in the `build/bin/` directory.

## 📂 Project Structure

```text
Void/
├── main.go                # Application entry point & window configuration
├── internal/
│   └── service/           # Go backend business logic
│       ├── ai/            # LLM API integrations
│       ├── calculator/    # Math & Unit conversions
│       ├── clipboard/     # macOS clipboard observer
│       ├── commands/      # App discovery & system actions
│       ├── frecency/      # Smart sorting algorithm 
│       ├── hotkey/        # CGo global EventTap hook
│       ├── scripts/       # Raycast-compatible script runner
│       ├── settings/      # JSON settings persistence
│       └── snippets/      # Text snippet manager
├── frontend/
│   ├── index.html         # Frontend entry
│   ├── src/               # React UI views & components
│   └── bindings/          # Auto-generated Wails IPC bindings
└── build/                 # App icons and build assets
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

MIT
