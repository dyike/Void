package window

import (
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/dyike/Void/internal/model"
	"github.com/dyike/keel/ui/core"
	"github.com/dyike/keel/ui/el"
	"github.com/dyike/keel/ui/kit"
	"github.com/sahilm/fuzzy"
)

type action struct {
	title string
	run   func()
}
type resultRow struct {
	id, title, subtitle string
	category, iconData  string
	imagePath           string
	icon                kit.IconName
	run                 func()
	actions             []action
}
type launcherView struct {
	clipboardImages                               map[string]*clipboardArtwork
	clipboardImageClock                           uint64
	clipboardImageLoads                           chan struct{}
	clipboardPreview                              *kit.DialogView
	iconImages                                    map[string]*kit.ImageView
	hoveredID                                     string
	actionIndex                                   int
	actionFocus                                   bool
	searchCancel                                  chan struct{}
	manager                                       *Manager
	query                                         string
	list                                          *kit.VirtualListView
	rows                                          []resultRow
	page                                          string
	selected, revision                            int
	focusSearch, hadFocus, actions, busy, loading bool
	reveal                                        bool
	status, output                                string
	form                                          string
	fields                                        map[string]*kit.InputView
	checks                                        map[string]*kit.CheckboxView
	snippetID                                     string
	script                                        *model.ScriptCommand
	dialog                                        *kit.DialogView
}

func newLauncher(m *Manager) *launcherView {
	v := &launcherView{manager: m, page: "Commands", fields: map[string]*kit.InputView{}, checks: map[string]*kit.CheckboxView{}, dialog: kit.Dialog("")}
	v.iconImages = make(map[string]*kit.ImageView)
	v.clipboardImages = make(map[string]*clipboardArtwork)
	v.clipboardImageLoads = make(chan struct{}, 2)
	v.clipboardPreview = kit.Dialog("Image preview").Width(600).CloseButton(true)
	v.list = kit.VirtualList(0, 48, v.renderRow).Fill().ItemKey(func(i int) string { return v.rows[i].id })
	return v
}
func (v *launcherView) navigate(page string) {
	v.clipboardPreview.SetValue(false)
	v.pruneClipboardImages(nil)
	v.page = page
	v.form = ""
	v.output = ""
	v.status = ""
	v.actions = false
	v.query = ""
	v.focusSearch = true
	v.refresh()
}
func (v *launcherView) escape() {
	if v.clipboardPreview.Value() {
		v.clipboardPreview.SetValue(false)
		return
	}
	if v.dialog.Value() {
		v.dialog.SetValue(false)
		return
	}
	if v.form != "" {
		v.form = ""
		return
	}
	if v.actions {
		v.actions = false
		return
	}
	if v.output != "" {
		v.output = ""
		return
	}
	if v.query != "" {
		v.query = ""
		v.refresh()
		return
	}
	if v.page != "Commands" {
		v.navigate("Commands")
		return
	}
	v.manager.HideLauncher()
}
func (v *launcherView) move(delta int) {
	if len(v.rows) > 0 && v.form == "" {
		v.selected = (v.selected + delta + len(v.rows)) % len(v.rows)
		v.reveal = true
	}
}
func (v *launcherView) activate() {
	if v.form == "" && v.selected < len(v.rows) && v.selected >= 0 && !v.loading {
		v.rows[v.selected].run()
	}
}
func (v *launcherView) job(label string, work func() (string, error), after func()) {
	if v.busy {
		return
	}
	v.busy = true
	v.status = label
	go func() {
		out, err := work()
		core.Update(func() {
			v.busy = false
			if err != nil {
				v.status = err.Error()
				if out != "" {
					v.output = out
				}
				return
			}
			v.status = "Done"
			v.output = out
			if after != nil {
				after()
			}
		})
	}()
}
func (v *launcherView) simple(label string, fn func() error) {
	v.job(label, func() (string, error) { return "", fn() }, nil)
}
func (v *launcherView) confirm(title string, fn func()) {
	v.dialog.ConfirmDanger(title, "This action cannot be undone.", "Confirm", fn)
}
func (v *launcherView) refresh() {
	if v.searchCancel != nil {
		close(v.searchCancel)
	}
	v.searchCancel = make(chan struct{})
	cancelled := v.searchCancel
	v.revision++
	rev := v.revision
	page, q := v.page, strings.TrimSpace(v.query)
	v.selected = 0
	v.loading = true
	v.rows = nil
	v.list.SetCount(0)
	go func() {
		if q != "" {
			select {
			case <-cancelled:
				return
			case <-time.After(140 * time.Millisecond):
			}
		}
		rows, err := v.loadRows(page, q)
		core.Update(func() {
			if rev != v.revision {
				return
			}
			v.loading = false
			if err != nil {
				v.status = err.Error()
			}
			v.rows = rows
			v.pruneClipboardImages(rows)
			v.list.SetCount(len(rows))
		})
	}()
}
func matches(q string, parts ...string) bool {
	return q == "" || strings.Contains(strings.ToLower(strings.Join(parts, " ")), strings.ToLower(q)) || (len(parts) > 0 && len(fuzzy.Find(q, parts[:1])) > 0)
}
func (v *launcherView) loadRows(page, q string) ([]resultRow, error) {
	s := v.manager.services
	var rows []resultRow
	switch page {
	case "Commands", "Files", "Contents":
		var commands []model.CommandInfo
		var err error
		if page == "Files" || page == "Contents" {
			if len([]rune(q)) < 2 {
				return nil, nil
			}
			if page == "Files" {
				commands, err = s.Commands.SearchFiles(q)
			} else {
				commands, err = s.Commands.SearchFileContents(q)
			}
		} else {
			commands, err = s.Commands.GetCommands()
			cfg, _ := s.Settings.GetSettings()
			commands = slices.DeleteFunc(commands, func(c model.CommandInfo) bool { return cfg != nil && slices.Contains(cfg.DisabledCommands, c.ID) })
			if cfg != nil {
				slices.SortStableFunc(commands, func(a, b model.CommandInfo) int {
					ap, bp := slices.Contains(cfg.PinnedCommands, a.ID), slices.Contains(cfg.PinnedCommands, b.ID)
					if ap && !bp {
						return -1
					}
					if bp && !ap {
						return 1
					}
					return 0
				})
			}
			if s.Calculator.IsLikelyExpression(q) {
				if result, e := s.Calculator.Evaluate(q); e == nil && result.IsValid {
					value := result.Result
					rows = append(rows, resultRow{id: "calculator", title: value, subtitle: q + " · Enter to copy", category: "Calculator", icon: kit.IconReceipt, run: func() { v.simple("Copy result", func() error { return s.Commands.CopyToClipboard(value) }) }})
				}
			}
			if strings.HasPrefix(q, "@") {
				term := strings.TrimSpace(strings.TrimPrefix(q, "@"))
				for _, engine := range []struct{ name, base string }{{"Google", "https://www.google.com/search?q="}, {"Baidu", "https://www.baidu.com/s?wd="}, {"Bing", "https://www.bing.com/search?q="}, {"GitHub", "https://github.com/search?q="}, {"YouTube", "https://www.youtube.com/results?search_query="}} {
					target := engine.base + url.QueryEscape(term)
					rows = append(rows, resultRow{id: engine.name, title: "Search " + engine.name, subtitle: term, category: "Web Search", icon: kit.IconSearch, run: func() { v.simple("Open search", func() error { return s.Commands.OpenURL(target) }) }})
				}
			}
		}
		if err != nil {
			return rows, err
		}
		if page == "Commands" {
			commands = searchCommands(commands, q)
		}
		for _, c := range commands {
			row := resultRow{id: c.ID, title: c.Title, subtitle: c.Subtitle, category: c.Category, icon: commandIcon(c), iconData: c.IconDataURL, run: func() { v.execute(c) }}
			if c.Category == "file" {
				row.actions = []action{
					{"Reveal in Finder", func() { v.simple("Reveal file", func() error { return s.Commands.RevealInFinder(c.Path) }) }},
					{"Quick Look", func() { v.simple("Preview", func() error { return s.Commands.QuickLook(c.Path) }) }},
					{"Copy path", func() { v.simple("Copy path", func() error { return s.Commands.CopyToClipboard(c.Path) }) }},
				}
			} else {
				row.actions = []action{{"Pin / unpin", func() { v.pin(c.ID) }}}
			}
			rows = append(rows, row)
		}
	case "Clipboard":
		for _, item := range s.Clipboard.GetHistory() {
			if !matches(q, item.Content, item.Preview) {
				continue
			}
			title := item.Preview
			if title == "" {
				title = item.Content
			}
			if item.Type == "image" {
				title = "Image · " + title
			}
			imagePath := clipboardImagePath(item)
			subtitle := item.Type
			if imagePath != "" {
				subtitle = "Image · Hover thumbnail to preview"
			}
			rows = append(rows, resultRow{id: item.ID, title: title, subtitle: subtitle, category: "Clipboard", icon: kit.IconCopy, imagePath: imagePath, run: func() { v.simple("Copy item", func() error { return s.Clipboard.CopyItem(item.ID) }) }, actions: []action{
				{"Preview", func() {
					if imagePath != "" {
						v.previewClipboardImage(imagePath)
					} else {
						v.output = item.Content
					}
				}},
				{"Delete", func() {
					v.confirm("Delete clipboard item?", func() {
						v.job("Delete", func() (string, error) { return "", s.Clipboard.DeleteItem(item.ID) }, v.refresh)
					})
				}},
			}})
		}
	case "Snippets":
		for _, sn := range s.Snippets.GetSnippets(q) {
			rows = append(rows, resultRow{id: sn.ID, title: sn.Name, subtitle: sn.Prefix + "  " + sn.Content, category: "Snippet", icon: kit.IconFile, run: func() { v.simple("Copy snippet", func() error { return s.Snippets.CopyToClipboard(sn.ID) }) }, actions: []action{
				{"Edit", func() { v.editSnippet(sn) }},
				{"Delete", func() {
					v.confirm("Delete snippet?", func() {
						v.job("Delete", func() (string, error) { return "", s.Snippets.DeleteSnippet(sn.ID) }, v.refresh)
					})
				}},
			}})
		}
	case "Scripts":
		for _, script := range s.Scripts.GetScriptCommands() {
			if !matches(q, script.Title, script.Description) {
				continue
			}
			rows = append(rows, resultRow{id: script.ID, title: script.Title, subtitle: script.Description, category: "Script", icon: kit.IconReceipt, run: func() { v.openScript(script) }})
		}
	case "Containers", "Images", "Networks":
		return v.loadContainers(page, q)
	}
	return rows, nil
}
func (v *launcherView) execute(c model.CommandInfo) {
	if v.busy {
		return
	}
	switch c.ID {
	case "system-file-search":
		v.navigate("Files")
		return
	case "system-file-contents":
		v.navigate("Contents")
		return
	case "system-void-settings":
		v.manager.OpenSettings()
		return
	case "system-clipboard-manager":
		v.navigate("Clipboard")
		return
	case "system-snippets":
		v.navigate("Snippets")
		return
	case "system-scripts":
		v.navigate("Scripts")
		return
	case "system-apple-containers":
		v.navigate("Containers")
		return
	}
	s := v.manager.services
	run := func() {
		// Dismiss before executing window-management actions, so the target app owns focus.
		v.manager.HideLauncher()
		v.job("Running "+c.Title, func() (string, error) {
			var err error
			switch c.Category {
			case "app", "file":
				err = s.Commands.LaunchApp(c.Path)
			case "settings":
				err = s.Commands.OpenSystemSettings(c.Path)
			default:
				err = s.Commands.ExecuteSystemCommand(c.ID)
			}
			if err == nil {
				s.Commands.RecordUsage(c.ID)
			}
			return "", err
		}, nil)
	}
	if slices.Contains([]string{"system-restart", "system-shutdown", "system-empty-trash"}, c.ID) {
		v.confirm(c.Title+"?", run)
	} else {
		run()
	}
}
func (v *launcherView) pin(id string) {
	s := v.manager.services.Settings
	v.job("Save pin", func() (string, error) {
		cfg, e := s.GetSettings()
		if e != nil {
			return "", e
		}
		if slices.Contains(cfg.PinnedCommands, id) {
			cfg.PinnedCommands = slices.DeleteFunc(cfg.PinnedCommands, func(x string) bool { return x == id })
		} else {
			cfg.PinnedCommands = append(cfg.PinnedCommands, id)
		}
		_, e = s.SaveSettings(*cfg)
		return "", e
	}, v.refresh)
}
func (v *launcherView) renderToolbar(cx *el.Context) el.Element {
	row := el.Div().Row().Wrap().Gap(6)
	switch v.page {
	case "Snippets":
		row.Child(kit.Button("New snippet", func() { v.editSnippet(model.Snippet{}) }).Compact(true).Render(cx))
	case "Scripts":
		row.Child(kit.Button("Open scripts folder", func() { v.simple("Open folder", v.manager.services.Scripts.OpenScriptsDir) }).Compact(true).Render(cx), kit.Button("Refresh", func() {
			v.job("Refresh scripts", func() (string, error) { v.manager.services.Scripts.RefreshScriptCommands(); return "", nil }, v.refresh)
		}).Compact(true).Render(cx))
	case "Clipboard":
		row.Child(kit.Button("Clear history", func() {
			v.confirm("Clear clipboard history?", func() {
				v.job("Clear history", func() (string, error) { return "", v.manager.services.Clipboard.ClearHistory() }, v.refresh)
			})
		}).Variant(kit.ButtonGhost).Compact(true).Render(cx))
	case "Containers", "Images", "Networks":
		return v.containerToolbar(cx)
	}
	return row
}
