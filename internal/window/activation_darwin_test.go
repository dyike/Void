//go:build darwin && keelnativeqa

package window

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/dyike/Void/internal/repo"
	"github.com/dyike/Void/internal/service"
	"github.com/dyike/keel/ui/core"
	keelwindow "github.com/dyike/keel/ui/window"
)

func TestMain(m *testing.M) {
	if os.Getenv("VOID_NATIVE_FOCUS_HELPER") == "background" {
		keelwindow.Open(keelwindow.Options{Title: "Focus test background", Width: 200, Height: 100})
		go func() {
			for {
				s := keelwindow.NativeQAWindowFocus("Focus test background")
				if s.FramePresented && s.Active && s.Key && s.InputResponder {
					fmt.Println("READY")
					return
				}
				time.Sleep(20 * time.Millisecond)
			}
		}()
		keelwindow.Main()
		return
	}
	if os.Getenv("VOID_NATIVE_FOCUS_HELPER") != "" {
		nativeLauncherActivationHelper()
		return
	}
	os.Exit(m.Run())
}

// Headless input cannot detect an inactive AppKit application or a settings
// window ordered below a floating launcher. Exercise the actual event path.
func TestNativeLauncherActivation(t *testing.T) {
	if os.Getenv("KEEL_DESKTOP") != "1" {
		t.Skip("set KEEL_DESKTOP=1 and build with -tags keelnativeqa")
	}
	binary, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, material := range []string{"frosted", "clear", "regular"} {
		t.Run(material, func(t *testing.T) {
			dir := t.TempDir()
			bundle := filepath.Join(dir, "Void Focus.app")
			macOS := filepath.Join(bundle, "Contents", "MacOS")
			if err := os.MkdirAll(macOS, 0o755); err != nil {
				t.Fatal(err)
			}
			exe := filepath.Join(macOS, "focus")
			if err := os.WriteFile(exe, binary, 0o755); err != nil {
				t.Fatal(err)
			}
			plist := `<?xml version="1.0"?><plist version="1.0"><dict><key>CFBundleExecutable</key><string>focus</string><key>CFBundleIdentifier</key><string>com.dyike.void.focus-regression</string><key>CFBundlePackageType</key><string>APPL</string><key>LSUIElement</key><true/></dict></plist>`
			if err := os.WriteFile(filepath.Join(bundle, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
				t.Fatal(err)
			}
			if out, err := exec.Command("codesign", "--force", "--sign", "-", bundle).CombinedOutput(); err != nil {
				t.Fatalf("sign test bundle: %v\n%s", err, out)
			}
			// A second test process provides a real inactive-app transition.
			// NSApplication.deactivate can be ignored on newer macOS releases.
			backgroundBundle := filepath.Join(dir, "Background.app")
			backgroundMacOS := filepath.Join(backgroundBundle, "Contents", "MacOS")
			if err := os.MkdirAll(backgroundMacOS, 0o755); err != nil {
				t.Fatal(err)
			}
			backgroundExe := filepath.Join(backgroundMacOS, "focus")
			if err := os.WriteFile(backgroundExe, binary, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(backgroundBundle, "Contents", "Info.plist"), []byte(strings.ReplaceAll(plist, "focus-regression", "focus-background")), 0o644); err != nil {
				t.Fatal(err)
			}
			if out, err := exec.Command("codesign", "--force", "--sign", "-", backgroundBundle).CombinedOutput(); err != nil {
				t.Fatalf("sign background fixture: %v\n%s", err, out)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, exe)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
			defer func() {
				if cmd.Process != nil {
					_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				}
			}()
			cmd.Env = append(os.Environ(), "VOID_NATIVE_FOCUS_HELPER="+material, "VOID_NATIVE_FOCUS_DATA="+filepath.Join(dir, "data"), "VOID_NATIVE_FOCUS_BACKGROUND="+backgroundExe)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("native activation: %v\n%s", err, out)
			}
			t.Log(string(out))
		})
	}
}

func nativeLauncherActivationHelper() {
	time.AfterFunc(10*time.Second, func() {
		fmt.Fprintln(os.Stderr, "FAIL: native focus fixture stalled")
		pprof.Lookup("goroutine").WriteTo(os.Stderr, 2)
		_ = syscall.Kill(-os.Getpid(), syscall.SIGKILL)
	})
	var background *exec.Cmd
	stopBackground := func() {
		if background != nil && background.Process != nil {
			_ = background.Process.Kill()
			_ = background.Wait()
			background = nil
		}
	}
	fail := func(message string) {
		fmt.Fprintf(os.Stderr, "FAIL: %s\nlauncher=%+v\nsettings=%+v\n", message, keelwindow.NativeQAWindowFocus("Void"), keelwindow.NativeQAWindowFocus("Settings"))
		pprof.Lookup("goroutine").WriteTo(os.Stderr, 2)
		stopBackground()
		os.Exit(1)
	}
	data := os.Getenv("VOID_NATIVE_FOCUS_DATA")
	settings := service.NewSettingsService(repo.NewSettingsRepo(data))
	settings.SetLaunchAtLoginFunc(nil)
	m := NewManager()
	m.Init(Services{Settings: settings, Commands: service.NewCommandService(data, settings, repo.NewCommandRepo(data)), Calculator: service.NewCalculatorService(repo.NewCalculatorRepo())})
	if err := m.SetGlassStyle(os.Getenv("VOID_NATIVE_FOCUS_HELPER")); err != nil {
		fail(err.Error())
	}
	m.ShowLauncher()
	go func() {
		onUI := func(fn func()) {
			done := make(chan struct{})
			core.Update(func() { fn(); close(done) })
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				fail("UI queue stalled")
			}
		}
		wait := func(label string, fn func() bool) {
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				if fn() {
					return
				}
				time.Sleep(20 * time.Millisecond)
			}
			fail(label)
		}
		active := func(title string) bool {
			s := keelwindow.NativeQAWindowFocus(title)
			return s.Visible && s.Active && s.Key && s.InputResponder && s.FramePresented && s.MaterialInstalled
		}
		openAndCloseSettings := func() {
			if err := keelwindow.NativeQAKey("Void", ",", true, 43); err != nil {
				fail(err.Error())
			}
			wait("Cmd+, did not show and focus settings", func() bool { return active("Settings") })
			if state := keelwindow.NativeQAWindowFocus("Settings"); state.Level != 3 || state.ActivationPolicy != 1 {
				fail("settings changed accessory policy or appeared below the launcher")
			}
			onUI(func() { m.settingsWindow.Close() })
			wait("settings did not close", func() bool {
				var closed bool
				onUI(func() { closed = m.settingsWindow == nil })
				return closed
			})
		}
		wait("launcher did not acquire native focus", func() bool { return active("Void") })
		if err := keelwindow.NativeQAKey("Void", "x", false, 7); err != nil {
			fail(err.Error())
		}
		wait("visible caret did not accept keyboard input", func() bool {
			var query string
			onUI(func() { query = m.launcher.query })
			return query == "x"
		})
		openAndCloseSettings()
		for cycle := range 3 {
			onUI(m.HideLauncher)
			wait("launcher did not close", func() bool {
				var closed bool
				onUI(func() { closed = m.mainWindow == nil })
				return closed
			})
			background = exec.Command(os.Getenv("VOID_NATIVE_FOCUS_BACKGROUND"))
			background.Env = append(os.Environ(), "VOID_NATIVE_FOCUS_HELPER=background")
			output, err := background.StdoutPipe()
			if err != nil {
				fail(err.Error())
			}
			background.Stderr = os.Stderr
			if err := background.Start(); err != nil {
				fail(err.Error())
			}
			if line, err := bufio.NewReader(output).ReadString('\n'); err != nil || strings.TrimSpace(line) != "READY" {
				fail(fmt.Sprintf("background app did not finish opening: %q, %v", line, err))
			}
			wait("test application did not deactivate", func() bool { return !keelwindow.NativeQAWindowFocus("Void").Active })
			onUI(m.ShowLauncher)
			wait(fmt.Sprintf("reopen %d did not activate the application", cycle), func() bool { return active("Void") })
			if err := keelwindow.NativeQAKey("Void", "y", false, 16); err != nil {
				fail(err.Error())
			}
			wait("reopened launcher did not accept typing", func() bool {
				var query string
				onUI(func() { query = m.launcher.query })
				return strings.Count(query, "x") == 1 && strings.Count(query, "y") == cycle+1 && len(query) == cycle+2
			})
			openAndCloseSettings()
			stopBackground()
		}
		fmt.Println("PASS: native typing, Cmd+, settings focus and 3 inactive-app reopen cycles")
		os.Exit(0)
	}()
	keelwindow.MainWithOptions(keelwindow.MainOptions{KeepAlive: true})
}
