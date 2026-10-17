#!/usr/bin/env python3
"""Exercise the native UI with isolated data and no host clipboard/hotkeys."""
import json
import base64
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import time
import struct
import zlib

ROOT = Path(__file__).resolve().parents[1]


def main():
    with tempfile.TemporaryDirectory(prefix="void-smoke-") as temporary:
        temp = Path(temporary)
        binary = temp / "void"
        subprocess.run(["go", "build", "-race", "-o", str(binary), "."], cwd=ROOT, check=True)
        data_dir = temp / "data"
        data_dir.mkdir()
        image_path = data_dir / "clipboard sample #50%.png"
        def png_chunk(kind, data):
            return struct.pack(">I", len(data)) + kind + data + struct.pack(">I", zlib.crc32(kind + data))
        pixels = b"".join(b"\0" + bytes((46, 154, 224, 255)) * 400 + bytes((237, 125, 79, 255)) * 400 for _ in range(400))
        image_path.write_bytes(b"\x89PNG\r\n\x1a\n" + png_chunk(b"IHDR", struct.pack(">IIBBBBB", 800, 400, 8, 6, 0, 0, 0)) + png_chunk(b"IDAT", zlib.compress(pixels)) + png_chunk(b"IEND", b""))
        (data_dir / "clipboard-history.json").write_text(json.dumps([
            {"id": "smoke-image", "type": "image", "content": str(image_path), "timestamp": int(time.time())},
            {"id": "smoke-text", "type": "text", "content": "Clipboard smoke text", "timestamp": int(time.time())},
        ]))
        (data_dir / "settings.json").write_text(json.dumps({
            "globalShortcut": "option+space",
            "windowMaterial": "clear",
            "ai": {"enabled": True, "openaiApiKey": "test-secret"},
            "pinnedCommands": ["system-ai-chat"],
            "commandHotkeys": {"system-ai-chat": "cmd+j"},
        }))
        address = str(temp / "ui.sock")
        env = dict(os.environ, KEEL_AUTOMATION=address, KEEL_HEADLESS="1")
        with (temp / "app.log").open("w+") as logs:
            app = subprocess.Popen([str(binary), "-data-dir", str(temp / "data")], cwd=ROOT, env=env, stdout=logs, stderr=logs)
            try:
                deadline = time.monotonic() + 20
                while not Path(address).exists():
                    if app.poll() is not None or time.monotonic() > deadline:
                        raise RuntimeError("App did not start")
                    time.sleep(0.05)
                with socket.socket(socket.AF_UNIX) as connection:
                    connection.settimeout(15)
                    connection.connect(address)
                    stream = connection.makefile("rwb")
                    launcher = "w1"

                    def call(method, **params):
                        stream.write((json.dumps({"method": method, "params": params}) + "\n").encode())
                        stream.flush()
                        response = json.loads(stream.readline())
                        if response.get("error"):
                            if method != "snapshot":
                                print(json.dumps(call("snapshot", window=params.get("window", launcher)), indent=2))
                            raise RuntimeError(response["error"])
                        return response.get("result")

                    def find(snapshot, role, name):
                        return next(e["ref"] for e in snapshot["elements"] if e["role"] == role and e.get("name") == name)

                    def type_field(name, value, window=launcher):
                        snap = call("snapshot", window=window)
                        return call("type", ref=find(snap, "textbox", name), text=value, clear=True, window=window)

                    call("wait_for", text="Clipboard History", timeout_ms=10000)
                    call("press",key="down")
                    snap=call("snapshot")
                    assert any(e.get("selected") and e.get("name")=="Snippets" for e in snap["elements"])
                    call("press",key="up")
                    call("press",key="enter")
                    call("wait_for",text="Clear history")
                    call("wait_for",text="Hover thumbnail to preview")
                    call("press",key="mod+k")
                    call("press",key="down")
                    call("press",key="enter")
                    call("wait_for",text="800 × 400")
                    snap = call("snapshot")
                    assert any(e.get("role") == "image" and e.get("name") == "Clipboard image preview" and e.get("value") == "loaded" for e in snap["elements"])
                    if os.environ.get("VOID_SMOKE_ARTIFACT_DIR"):
                        artifacts = Path(os.environ["VOID_SMOKE_ARTIFACT_DIR"])
                        artifacts.mkdir(parents=True, exist_ok=True)
                        (artifacts / "clipboard-preview.png").write_bytes(base64.b64decode(call("screenshot")))
                    call("press",key="esc")
                    call("wait_for",text="Clear history")
                    call("press",key="mod+0")
                    call("wait_for",text="Clipboard History")
                    # Calculator uses the same input/submit path as the launcher.
                    snap = call("snapshot")
                    assert all("AI Chat" not in e.get("name", "") for e in snap["elements"])
                    migrated = json.loads((data_dir / "settings.json").read_text())
                    assert "ai" not in migrated and "system-ai-chat" not in json.dumps(migrated)
                    search = next(e["ref"] for e in snap["elements"] if e["role"] == "textbox")
                    call("type", ref=search, text="2+3", clear=True)
                    call("wait_for", text="Enter to copy")
                    # Navigate with results loading; stale search results must not reappear.
                    call("press", key="mod+2")
                    call("click", text="New snippet")
                    type_field("Name", "Smoke snippet")
                    type_field("Expansion prefix", ";smoke")
                    type_field("Content", "Native Keel content")
                    call("scroll", x=400, y=350, dy=450, window=launcher)
                    call("click", text="Save snippet")
                    call("wait_for", text="Smoke snippet")
                    saved = json.loads((temp / "data" / "snippets.json").read_text())
                    assert "Native Keel content" in json.dumps(saved)
                    call("press", key="mod+k")
                    call("press", key="down")
                    call("press", key="enter")
                    call("wait_for", text="Content")
                    type_field("Content", "Updated content")
                    call("scroll", x=400, y=350, dy=450, window=launcher)
                    call("click", text="Save snippet")
                    call("wait_for", text="Updated content")
                    # Open an independent settings window; save and close it.
                    call("press", key="mod+,")
                    windows = call("windows")
                    settings = next(w["id"] for w in windows if w["title"] == "Settings")

                    def save_settings():
                        nonlocal settings
                        call("press", key="mod+s", window=settings)
                        deadline = time.monotonic() + 10
                        while time.monotonic() < deadline:
                            candidates = [w for w in call("windows") if w["title"] == "Settings"]
                            if candidates:
                                settings = max(candidates, key=lambda w: int(w["id"][1:]))["id"]
                                snap = call("snapshot", window=settings)
                                if "Settings saved" in json.dumps(snap):
                                    return
                            time.sleep(0.05)
                        raise RuntimeError("Settings did not finish saving: " + json.dumps(call("snapshot",window=settings),ensure_ascii=False))

                    snap = call("snapshot", window=settings)
                    assert all("AI" not in e.get("name", "") and "API key" not in e.get("name", "") for e in snap["elements"])
                    assert any(e.get("role") == "link" and e.get("name") == "General" for e in snap["elements"])
                    assert any(e.get("role") == "switch" and e.get("name") == "Launch at login" for e in snap["elements"])
                    call("click", ref=find(snap, "switch", "Show in menu bar"), window=settings)
                    assert any(e.get("name") == "Search" for e in snap["elements"])
                    call("click", ref=find(call("snapshot", window=settings), "button", "Global shortcut"), window=settings)
                    call("press", key="mod+s", window=settings)
                    assert json.loads((temp / "data" / "settings.json").read_text())["globalShortcut"] == "option+space", "Recording Cmd+S must not save settings"
                    call("click", ref=find(call("snapshot", window=settings), "button", "Global shortcut"), window=settings)
                    call("press", key="option+shift+space", window=settings)
                    assert next(e for e in call("snapshot",window=settings)["elements"] if e.get("name")=="Global shortcut" and e.get("role")=="button")["value"]=="⌥⇧Space"
                    # Drafts survive navigation; appearance controls live on their own page.
                    call("click", ref=find(call("snapshot", window=settings), "link", "Appearance"), window=settings)
                    snap=call("snapshot",window=settings)
                    call("click",ref=find(snap,"select","Launcher material"),window=settings)
                    call("click",text="Frosted",window=settings)
                    save_settings()
                    config = json.loads((temp / "data" / "settings.json").read_text())
                    assert config["globalShortcut"] == "option+shift+space"
                    assert config["windowMaterial"] == "frosted"
                    assert config["showInMenuBar"] is False
                    launcher=next(w["id"] for w in call("windows") if w["title"]=="Void")
                    assert launcher!="w1", "Changing material must recreate the native launcher"
                    # Repeated replacement must not let an old OnClose clear the new window.
                    snap=call("snapshot",window=settings)
                    assert any(e.get("role") == "select" and e.get("name") == "Launcher material" for e in snap["elements"])
                    call("click",ref=find(snap,"select","Launcher material"),window=settings)
                    call("click",text="Clear glass",window=settings)
                    save_settings()
                    launcher=next(w["id"] for w in call("windows") if w["title"]=="Void")
                    assert len(call("windows"))==2
                    call("click", ref=find(call("snapshot", window=settings), "link", "Advanced"), window=settings)
                    type_field("Terminal application", "Draft terminal", window=settings)
                    call("click", text="Revert", window=settings)
                    snap = call("snapshot", window=settings)
                    assert any(e.get("name") == "Terminal application" and e.get("value") == "Terminal" for e in snap["elements"])
                    assert not any(e.get("role") == "select" and e.get("name") == "Launcher material" for e in snap["elements"])
                    # Escape cancels recording without closing Settings or changing its value.
                    call("click", ref=find(snap, "link", "General"), window=settings)
                    call("click", ref=find(call("snapshot", window=settings), "button", "Global shortcut"), window=settings)
                    call("press", key="a", window=settings)
                    call("wait_for", text="Hold Command, Option or Control", window=settings)
                    call("press", key="esc", window=settings)
                    snap = call("snapshot", window=settings)
                    assert any(e.get("name") == "Global shortcut" and e.get("role") == "button" for e in snap["elements"])
                    assert json.loads((temp / "data" / "settings.json").read_text())["globalShortcut"] == "option+shift+space"
                    call("click", ref=find(snap, "link", "Advanced"), window=settings)
                    snap = call("snapshot", window=settings)
                    # Settings search spans categories and has an explicit empty state.
                    search = next(e["ref"] for e in snap["elements"] if e.get("role") == "textbox" and e.get("name") != "Terminal application")
                    call("type", ref=search, text="glass", clear=True, window=settings)
                    snap = call("snapshot", window=settings)
                    assert any(e.get("role") == "select" and e.get("name") == "Launcher material" for e in snap["elements"])
                    search = next(e["ref"] for e in snap["elements"] if e.get("role") == "textbox")
                    call("type", ref=search, text="unmatched-setting-query", clear=True, window=settings)
                    snap = call("snapshot", window=settings)
                    assert not any(e.get("role") == "select" for e in snap["elements"])
                    call("close", window=settings)
                    assert len(call("windows")) == 1
                    # Visit the remaining views, without executing OS or container actions.
                    for shortcut in ("mod+4", "mod+1", "mod+3", "mod+5"):
                        call("press", key=shortcut, window=launcher)
                        call("snapshot", window=launcher)
                    call("click", text="Run container", window=launcher)
                    call("scroll", x=400, y=350, dy=4000, window=launcher)
                    call("click", text="Run container", window=launcher)
                    call("wait_for", text="Image is required", window=launcher)
                    call("press", key="esc", window=launcher)
                    call("press", key="mod+0", window=launcher)
                    call("wait_for", text="Clipboard History", window=launcher)
                    call("close", window=launcher)
                    assert call("windows") == []
                    assert app.poll() is None, "Closing the final window must keep the resident app alive"
                    logs.flush()
                    logs.seek(0)
                    assert "WARNING: DATA RACE" not in logs.read(), "UI data race detected"
                    print("PASS: clipboard image preview, calculator, snippets, settings navigation/search/revert/persistence, material recreation, container validation, resident lifecycle")
            except Exception:
                logs.flush()
                logs.seek(0)
                print(logs.read())
                raise
            finally:
                app.terminate()
                try:
                    app.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    app.kill()
                    app.wait()


if __name__ == "__main__":
    main()
