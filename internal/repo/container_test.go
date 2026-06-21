package repo

import (
	"testing"

	"github.com/dyike/Void/internal/model"
)

func TestParseContainerRowsAppleContainerJSON(t *testing.T) {
	rows, err := decodeObjectList([]byte(`[
		{
			"configuration": {
				"creationDate": "2026-06-21T02:36:23Z",
				"id": "95968d88-e929-4ce2-bd20-aeceeffa9cf6",
				"image": {"reference": "docker.io/library/alpine:latest"},
				"platform": {"architecture": "arm64", "os": "linux"},
				"publishedPorts": [],
				"resources": {"cpus": 4, "memoryInBytes": 1073741824}
			},
			"id": "95968d88-e929-4ce2-bd20-aeceeffa9cf6",
			"status": {
				"networks": [],
				"startedDate": "2026-06-21T02:37:05Z",
				"state": "stopped"
			}
		},
		{
			"configuration": {
				"creationDate": "2026-06-21T02:39:21Z",
				"id": "4a129503-e56d-4536-a2c1-0ad7ce20b84b",
				"image": {"reference": "docker.io/library/alpine:latest"},
				"platform": {"architecture": "arm64", "os": "linux"},
				"publishedPorts": [],
				"resources": {"cpus": 4, "memoryInBytes": 1073741824}
			},
			"id": "4a129503-e56d-4536-a2c1-0ad7ce20b84b",
			"status": {
				"networks": [{"ipv4Address": "192.168.64.11/24", "network": "default"}],
				"startedDate": "2026-06-21T02:39:22Z",
				"state": "running"
			}
		}
	]`), "containers", "items", "data")
	if err != nil {
		t.Fatal(err)
	}

	containers := parseContainerRows(rows)
	if len(containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containers))
	}

	stopped := containers[0]
	if stopped.Status != "stopped" {
		t.Fatalf("expected stopped status, got %q", stopped.Status)
	}
	if stopped.Image != "docker.io/library/alpine:latest" {
		t.Fatalf("expected alpine image, got %q", stopped.Image)
	}
	if stopped.IP != "" {
		t.Fatalf("expected stopped container without IP, got %q", stopped.IP)
	}

	running := containers[1]
	if running.Status != "running" {
		t.Fatalf("expected running status, got %q", running.Status)
	}
	if running.IP != "192.168.64.11/24" {
		t.Fatalf("expected running IP, got %q", running.IP)
	}
	if running.CPUs != "4" {
		t.Fatalf("expected 4 CPUs, got %q", running.CPUs)
	}
	if running.Memory != "1.0 GB" {
		t.Fatalf("expected memory 1.0 GB, got %q", running.Memory)
	}
	if running.OS != "linux" || running.Arch != "arm64" {
		t.Fatalf("expected linux/arm64, got %s/%s", running.OS, running.Arch)
	}
}

func TestMacOSMajorVersion(t *testing.T) {
	major, err := macOSMajorVersion("26.5.1")
	if err != nil {
		t.Fatal(err)
	}
	if major != 26 {
		t.Fatalf("expected major version 26, got %d", major)
	}

	major, err = macOSMajorVersion("25")
	if err != nil {
		t.Fatal(err)
	}
	if major != 25 {
		t.Fatalf("expected major version 25, got %d", major)
	}
}

func TestParseImageRowsAppleContainerJSON(t *testing.T) {
	rows, err := decodeObjectList([]byte(`[
		{
			"configuration": {
				"creationDate": "2026-06-10T03:29:33Z",
				"descriptor": {
					"digest": "sha256:ubuntu-index",
					"mediaType": "application/vnd.oci.image.index.v1+json",
					"size": 6694
				},
				"name": "docker.io/library/ubuntu:latest"
			},
			"id": "53958ec7b67c2c9355df922dd08dbf0360611f8c3cdb656875e81873db9ffdba",
			"variants": [
				{"digest": "sha256:amd64", "size": 1048576},
				{"digest": "sha256:arm64", "size": 2097152}
			]
		}
	]`), "images", "items", "data")
	if err != nil {
		t.Fatal(err)
	}

	images := parseImageRows(rows)
	if len(images) != 1 {
		t.Fatalf("expected 1 image, got %d", len(images))
	}

	image := images[0]
	if image.Name != "docker.io/library/ubuntu" {
		t.Fatalf("expected image name docker.io/library/ubuntu, got %q", image.Name)
	}
	if image.Tag != "latest" {
		t.Fatalf("expected tag latest, got %q", image.Tag)
	}
	if image.ID != "53958ec7b67c2c9355df922dd08dbf0360611f8c3cdb656875e81873db9ffdba" {
		t.Fatalf("unexpected image id: %q", image.ID)
	}
	if image.Digest != "sha256:ubuntu-index" {
		t.Fatalf("expected descriptor digest, got %q", image.Digest)
	}
	if image.Size != "3.0 MB" {
		t.Fatalf("expected summed variant size 3.0 MB, got %q", image.Size)
	}
	if image.CreatedAt != "2026-06-10T03:29:33Z" {
		t.Fatalf("expected creation date, got %q", image.CreatedAt)
	}
}

func TestParseNetworkRowsAppleContainerJSON(t *testing.T) {
	rows, err := decodeObjectList([]byte(`[
		{
			"configuration": {
				"creationDate": "2026-06-20T12:59:55Z",
				"labels": {
					"com.apple.container.resource.role": "builtin"
				},
				"mode": "nat",
				"name": "default",
				"options": {},
				"plugin": "container-network-vmnet"
			},
			"id": "default",
			"status": {
				"ipv4Gateway": "192.168.64.1",
				"ipv4Subnet": "192.168.64.0/24",
				"ipv6Subnet": "fd43:bdef:b661:2fc9::/64"
			}
		}
	]`), "networks", "items", "data")
	if err != nil {
		t.Fatal(err)
	}

	networks := parseNetworkRows(rows)
	if len(networks) != 1 {
		t.Fatalf("expected 1 network, got %d", len(networks))
	}

	network := networks[0]
	if network.ID != "default" || network.Name != "default" {
		t.Fatalf("expected default network, got id=%q name=%q", network.ID, network.Name)
	}
	if network.Mode != "nat" {
		t.Fatalf("expected nat mode, got %q", network.Mode)
	}
	if network.Plugin != "container-network-vmnet" {
		t.Fatalf("unexpected plugin: %q", network.Plugin)
	}
	if network.IPv4Subnet != "192.168.64.0/24" || network.IPv4Gateway != "192.168.64.1" {
		t.Fatalf("unexpected ipv4 fields: subnet=%q gateway=%q", network.IPv4Subnet, network.IPv4Gateway)
	}
	if network.IPv6Subnet != "fd43:bdef:b661:2fc9::/64" {
		t.Fatalf("unexpected ipv6 subnet: %q", network.IPv6Subnet)
	}
	if len(network.Labels) != 1 || network.Labels[0] != "com.apple.container.resource.role=builtin" {
		t.Fatalf("unexpected labels: %#v", network.Labels)
	}
}

func TestCreateNetworkArgs(t *testing.T) {
	args, err := createNetworkArgs(model.CreateNetworkOptions{
		Name:     "devnet",
		Internal: true,
		Labels:   []string{"env=dev"},
		Options:  []string{"mtu=1500"},
		Plugin:   "container-network-vmnet",
		Subnet:   "192.168.130.0/24",
		SubnetV6: "fd00:130::/64",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"network", "create",
		"--internal",
		"--label", "env=dev",
		"--option", "mtu=1500",
		"--plugin", "container-network-vmnet",
		"--subnet", "192.168.130.0/24",
		"--subnet-v6", "fd00:130::/64",
		"devnet",
	}
	if len(args) != len(expected) {
		t.Fatalf("expected %#v, got %#v", expected, args)
	}
	for i := range args {
		if args[i] != expected[i] {
			t.Fatalf("expected %#v, got %#v", expected, args)
		}
	}
}

func TestDeleteNetworkArgs(t *testing.T) {
	args, err := deleteNetworkArgs("devnet")
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"network", "delete", "devnet"}
	if len(args) != len(expected) {
		t.Fatalf("expected %#v, got %#v", expected, args)
	}
	for i := range args {
		if args[i] != expected[i] {
			t.Fatalf("expected %#v, got %#v", expected, args)
		}
	}
}

func TestSplitImageTagKeepsRegistryPort(t *testing.T) {
	name, tag := splitImageTag("localhost:5000/team/image:dev")
	if name != "localhost:5000/team/image" || tag != "dev" {
		t.Fatalf("unexpected split with tag: name=%q tag=%q", name, tag)
	}

	name, tag = splitImageTag("localhost:5000/team/image")
	if name != "localhost:5000/team/image" || tag != "" {
		t.Fatalf("unexpected split without tag: name=%q tag=%q", name, tag)
	}
}
