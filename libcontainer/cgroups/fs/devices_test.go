// +build linux

package fs

import (
	"testing"

	"github.com/opencontainers/runc/libcontainer/cgroups/fscommon"
	"github.com/opencontainers/runc/libcontainer/devices"
)

func TestDevicesSetAllow(t *testing.T) {
	helper := NewCgroupTestUtil("devices", t)
	defer helper.cleanup()

	helper.writeFileContents(map[string]string{
		"devices.allow": "",
		"devices.deny":  "",
		"devices.list":  "a *:* rwm",
	})

	helper.CgroupData.config.Resources.Devices = []*devices.Rule{
		{
			Type:        devices.CharDevice,
			Major:       1,
			Minor:       5,
			Permissions: devices.Permissions("rwm"),
			Allow:       true,
		},
	}

	d := &DevicesGroup{testingSkipFinalCheck: true}
	if err := d.Set(helper.CgroupPath, helper.CgroupData.config.Resources); err != nil {
		t.Fatal(err)
	}

	// The default deny rule must be written.
	value, err := fscommon.GetCgroupParamString(helper.CgroupPath, "devices.deny")
	if err != nil {
		t.Fatalf("Failed to parse devices.deny: %s", err)
	}
	if value[0] != 'a' {
		t.Errorf("Got the wrong value (%q), set devices.deny failed.", value)
	}

	// Permitted rule must be written.
	if value, err := fscommon.GetCgroupParamString(helper.CgroupPath, "devices.allow"); err != nil {
		t.Fatalf("Failed to parse devices.allow: %s", err)
	} else if value != "c 1:5 rwm" {
		t.Errorf("Got the wrong value (%q), set devices.allow failed.", value)
	}
}
