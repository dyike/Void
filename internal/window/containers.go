package window

import (
	"fmt"
	"strings"

	"github.com/dyike/Void/internal/model"
	"github.com/dyike/keel/ui/el"
	"github.com/dyike/keel/ui/kit"
	"github.com/dyike/keel/ui/theme"
)

func (v *launcherView) containerJob(label string, work func() (model.ContainerCommandResult, error)) {
	v.job(label, func() (string, error) { result, err := work(); return result.Output, err }, func() { v.form = ""; v.refresh() })
}
func (v *launcherView) loadContainers(page, q string) ([]resultRow, error) {
	svc := v.manager.services.Containers
	status := svc.GetStatus()
	if !status.Available || !status.Supported {
		return nil, fmt.Errorf("Apple Containers: %s", status.Error)
	}
	var rows []resultRow
	switch page {
	case "Containers":
		containers, err := svc.ListContainers()
		if err != nil {
			return nil, err
		}
		for _, c := range containers {
			if !matches(q, c.Name, c.Image, c.Status) {
				continue
			}
			logs := func() { v.job("Load logs", func() (string, error) { return svc.GetLogs(c.ID, 200) }, nil) }
			rows = append(rows, resultRow{id: c.ID, title: c.Name, subtitle: c.Status + " · " + c.Image + " · " + c.IP, run: logs, actions: []action{
				{"Start", func() {
					v.containerJob("Start container", func() (model.ContainerCommandResult, error) { return svc.StartContainer(c.ID) })
				}},
				{"Stop", func() {
					v.containerJob("Stop container", func() (model.ContainerCommandResult, error) { return svc.StopContainer(c.ID) })
				}},
				{"Restart", func() {
					v.containerJob("Restart container", func() (model.ContainerCommandResult, error) { return svc.RestartContainer(c.ID) })
				}},
				{"Logs", logs},
				{"Inspect", func() { v.job("Inspect", func() (string, error) { return svc.InspectContainer(c.ID) }, nil) }},
				{"Shell", func() { v.simple("Open shell", func() error { return svc.OpenShell(c.ID) }) }},
				{"Delete", func() {
					v.confirm("Delete "+c.Name+"?", func() {
						v.containerJob("Delete container", func() (model.ContainerCommandResult, error) { return svc.RemoveContainer(c.ID) })
					})
				}},
			}})
		}
	case "Images":
		images, err := svc.ListImages()
		if err != nil {
			return nil, err
		}
		for _, image := range images {
			if !matches(q, image.Name, image.Tag) {
				continue
			}
			ref := image.Name
			if image.Tag != "" && !strings.Contains(ref, ":"+image.Tag) {
				ref += ":" + image.Tag
			}
			rows = append(rows, resultRow{id: ref, title: ref, subtitle: image.Size, run: func() { v.openRunContainer(ref) }, actions: []action{{"Remove image", func() {
				v.confirm("Remove "+ref+"?", func() {
					v.containerJob("Remove image", func() (model.ContainerCommandResult, error) { return svc.RemoveImage(ref) })
				})
			}}}})
		}
	case "Networks":
		networks, err := svc.ListNetworks()
		if err != nil {
			return nil, err
		}
		for _, n := range networks {
			if !matches(q, n.Name, n.IPv4Subnet) {
				continue
			}
			rows = append(rows, resultRow{id: n.Name, title: n.Name, subtitle: n.IPv4Subnet + " · " + n.Mode, run: func() {
				v.output = fmt.Sprintf("%s\nIPv4: %s\nGateway: %s\nIPv6: %s\nPlugin: %s", n.Name, n.IPv4Subnet, n.IPv4Gateway, n.IPv6Subnet, n.Plugin)
			}, actions: []action{{"Remove network", func() {
				v.confirm("Remove network "+n.Name+"?", func() {
					v.containerJob("Remove network", func() (model.ContainerCommandResult, error) { return svc.RemoveNetwork(n.Name) })
				})
			}}}})
		}
	}
	return rows, nil
}
func (v *launcherView) containerToolbar(cx *el.Context) el.Element {
	row := el.Div().Row().Wrap().Gap(5)
	for _, page := range []string{"Containers", "Images", "Networks"} {
		row.Child(kit.Button(page, func() { v.navigate(page) }).Variant(kit.ButtonGhost).Compact(true).Selected(page == v.page).Render(cx))
	}
	row.Child(kit.Button("Refresh", v.refresh).Compact(true).Variant(kit.ButtonSecondary).Render(cx))
	switch v.page {
	case "Containers":
		row.Child(kit.Button("Run container", func() { v.openRunContainer("") }).Compact(true).Render(cx), kit.Button("Start system", func() {
			v.job("Start container system", func() (string, error) { return "", v.manager.services.Containers.StartSystem() }, v.refresh)
		}).Compact(true).Render(cx))
	case "Images":
		row.Child(kit.Button("Pull image", func() { v.form = "pull"; v.field("reference", "Image reference", "", false) }).Compact(true).Render(cx), kit.Button("Prune unused", func() {
			v.confirm("Prune unused images?", func() {
				v.containerJob("Prune images", func() (model.ContainerCommandResult, error) { return v.manager.services.Containers.PruneImages(false) })
			})
		}).Compact(true).Variant(kit.ButtonGhost).Render(cx))
	case "Networks":
		row.Child(kit.Button("Create network", v.openNetwork).Compact(true).Render(cx))
	}
	return row
}

var runFields = []struct{ key, label, hint string }{
	{"image", "Image", "nginx:latest"}, {"name", "Name", ""}, {"ports", "Ports (one per line)", "8080:80"},
	{"envs", "Environment (one per line)", "KEY=value"}, {"volumes", "Volumes (one per line)", "/host:/container"},
	{"command", "Command", ""}, {"cpus", "CPUs", ""}, {"memory", "Memory", "512m"}, {"network", "Network", ""},
	{"dns", "DNS servers (one per line)", ""}, {"platform", "Platform", "linux/arm64"},
	{"workdir", "Working directory", ""}, {"user", "User", ""}, {"entrypoint", "Entrypoint", ""}, {"labels", "Labels (one per line)", "key=value"},
}

func (v *launcherView) openRunContainer(image string) {
	v.form = "run-container"
	for _, f := range runFields {
		v.field(f.key, f.label, "", f.key == "ports" || f.key == "envs" || f.key == "volumes" || f.key == "dns" || f.key == "labels").Placeholder(f.hint)
	}
	v.fields["image"].SetValue(image)
	for _, label := range []string{"Detach", "Remove on exit", "No DNS", "Init", "Read only", "SSH"} {
		v.checks[label] = kit.Checkbox(label, label == "Detach")
	}
}
func lines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}
func (v *launcherView) runOptions() model.RunContainerOptions {
	return model.RunContainerOptions{
		Image: v.value("image"), Name: v.value("name"), Ports: lines(v.value("ports")), Envs: lines(v.value("envs")), Volumes: lines(v.value("volumes")), Command: v.value("command"),
		CPUs: v.value("cpus"), Memory: v.value("memory"), Network: v.value("network"), DNS: lines(v.value("dns")), Platform: v.value("platform"), Workdir: v.value("workdir"), User: v.value("user"), Entrypoint: v.value("entrypoint"), Labels: lines(v.value("labels")),
		Detach: v.checks["Detach"].Value(), Remove: v.checks["Remove on exit"].Value(), NoDNS: v.checks["No DNS"].Value(), Init: v.checks["Init"].Value(), ReadOnly: v.checks["Read only"].Value(), SSH: v.checks["SSH"].Value(),
	}
}
func (v *launcherView) openNetwork() {
	v.form = "network"
	for _, f := range []struct{ key, label string }{{"net-name", "Name"}, {"plugin", "Plugin"}, {"subnet", "IPv4 subnet"}, {"subnetv6", "IPv6 subnet"}, {"net-labels", "Labels (one per line)"}, {"options", "Options (one per line)"}} {
		v.field(f.key, f.label, "", f.key == "options" || f.key == "net-labels")
	}
	v.checks["Internal"] = kit.Checkbox("Internal network", false)
}
func (v *launcherView) renderContainerForm(cx *el.Context, panel *el.DivEl) el.Element {
	switch v.form {
	case "pull":
		panel.Child(el.Text("Pull image").TextSize(20).Bold(), v.fields["reference"].Render(cx), kit.Button("Pull", func() {
			ref := v.value("reference")
			if ref == "" {
				v.status = "Image reference is required"
				return
			}
			v.containerJob("Pull image", func() (model.ContainerCommandResult, error) { return v.manager.services.Containers.PullImage(ref) })
		}).Loading(v.busy).Render(cx))
	case "run-container":
		panel.Child(el.Text("Run container").TextSize(20).Bold())
		for _, f := range runFields {
			panel.Child(v.fields[f.key].Render(cx))
		}
		for _, label := range []string{"Detach", "Remove on exit", "No DNS", "Init", "Read only", "SSH"} {
			panel.Child(v.checks[label].Render(cx))
		}
		panel.Child(el.Text(v.manager.services.Containers.BuildRunCommand(v.runOptions())).Mono().TextSize(12).TextColor(theme.Muted), kit.Button("Run container", func() {
			opts := v.runOptions()
			if opts.Image == "" {
				v.status = "Image is required"
				return
			}
			v.containerJob("Run container", func() (model.ContainerCommandResult, error) { return v.manager.services.Containers.RunContainer(opts) })
		}).Loading(v.busy).Render(cx))
	case "network":
		panel.Child(el.Text("Create network").TextSize(20).Bold())
		for _, key := range []string{"net-name", "plugin", "subnet", "subnetv6", "net-labels", "options"} {
			panel.Child(v.fields[key].Render(cx))
		}
		panel.Child(v.checks["Internal"].Render(cx))
		panel.Child(kit.Button("Create network", func() {
			opts := model.CreateNetworkOptions{Name: v.value("net-name"), Plugin: v.value("plugin"), Subnet: v.value("subnet"), SubnetV6: v.value("subnetv6"), Labels: lines(v.value("net-labels")), Options: lines(v.value("options")), Internal: v.checks["Internal"].Value()}
			if opts.Name == "" {
				v.status = "Network name is required"
				return
			}
			v.containerJob("Create network", func() (model.ContainerCommandResult, error) { return v.manager.services.Containers.CreateNetwork(opts) })
		}).Loading(v.busy).Render(cx))
	}
	return panel
}
