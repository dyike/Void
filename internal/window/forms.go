package window

import (
	"fmt"
	"strings"

	"github.com/dyike/Void/internal/model"
	"github.com/dyike/keel/ui/el"
	"github.com/dyike/keel/ui/kit"
)

func (v *launcherView) field(key, label, value string, multiline bool) *kit.InputView {
	var input *kit.InputView
	if multiline {
		input = kit.TextArea(label).Rows(4)
	} else {
		input = kit.Input(label)
	}
	input.SetValue(value)
	v.fields[key] = input
	return input
}
func (v *launcherView) value(key string) string {
	if f := v.fields[key]; f != nil {
		return strings.TrimSpace(f.Value())
	}
	return ""
}
func (v *launcherView) editSnippet(sn model.Snippet) {
	v.form = "snippet"
	v.snippetID = sn.ID
	v.field("name", "Name", sn.Name, false)
	v.field("prefix", "Expansion prefix", sn.Prefix, false)
	v.field("content", "Content", sn.Content, true)
	v.field("tags", "Tags (comma separated)", strings.Join(sn.Tags, ", "), false)
}
func (v *launcherView) saveSnippet() {
	name, content, prefix, id := v.value("name"), v.fields["content"].Value(), v.value("prefix"), v.snippetID
	if name == "" || content == "" {
		v.status = "Name and content are required"
		return
	}
	var tags []string
	for _, tag := range strings.Split(v.value("tags"), ",") {
		if tag = strings.TrimSpace(tag); tag != "" {
			tags = append(tags, tag)
		}
	}
	v.job("Save snippet", func() (string, error) {
		var err error
		if id == "" {
			_, err = v.manager.services.Snippets.CreateSnippet(name, content, prefix, tags)
		} else {
			_, err = v.manager.services.Snippets.UpdateSnippet(id, name, content, prefix, tags)
		}
		return "", err
	}, func() { v.form = ""; v.refresh() })
}
func (v *launcherView) openScript(script model.ScriptCommand) {
	v.script = &script
	v.form = "script"
	for i, arg := range script.Arguments {
		field := v.field(fmt.Sprint(i), arg.Name, "", false).Placeholder(arg.Placeholder)
		if arg.Type == "password" {
			field.Password()
		}
	}
	if len(script.Arguments) == 0 {
		v.executeScript()
	}
}
func (v *launcherView) executeScript() {
	if v.script == nil {
		return
	}
	script := *v.script
	args := map[string]string{}
	for i, arg := range script.Arguments {
		value := v.value(fmt.Sprint(i))
		if arg.Required && value == "" {
			v.status = arg.Name + " is required"
			return
		}
		args[fmt.Sprintf("arg%d", i)] = value
		args[arg.Name] = value
	}
	v.job("Running "+script.Title, func() (string, error) {
		result, err := v.manager.services.Scripts.Execute(script.ID, args)
		if err != nil {
			return "", err
		}
		if result.ExitCode != 0 {
			return result.Output, fmt.Errorf("script exited with code %d", result.ExitCode)
		}
		return result.Output, nil
	}, func() { v.form = "" })
}
func (v *launcherView) renderForm(cx *el.Context) el.Element {
	panel := el.Div().ID("form").Grow().ScrollY().Gap(10)
	panel.Child(kit.Button("Back", func() { v.form = "" }).Variant(kit.ButtonGhost).Render(cx))
	switch v.form {
	case "snippet":
		panel.Child(el.Text("Snippet").TextSize(20).Bold())
		for _, key := range []string{"name", "prefix", "content", "tags"} {
			panel.Child(v.fields[key].Render(cx))
		}
		panel.Child(kit.Button("Save snippet", v.saveSnippet).Loading(v.busy).Render(cx))
	case "script":
		panel.Child(el.Text(v.script.Title).TextSize(20).Bold())
		for i := range v.script.Arguments {
			panel.Child(v.fields[fmt.Sprint(i)].Render(cx))
		}
		panel.Child(kit.Button("Run script", v.executeScript).Loading(v.busy).Render(cx))
	default:
		return v.renderContainerForm(cx, panel)
	}
	return panel
}
