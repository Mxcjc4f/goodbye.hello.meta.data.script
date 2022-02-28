// +build linux

package specconv

import (
	"os"
	"strings"
	"testing"

	"golang.org/x/sys/unix"

	dbus "github.com/godbus/dbus/v5"
	"github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/configs/validate"
	"github.com/opencontainers/runtime-spec/specs-go"
)

func TestCreateCommandHookTimeout(t *testing.T) {
	timeout := 3600
	hook := specs.Hook{
		Path:    "/some/hook/path",
		Args:    []string{"--some", "thing"},
		Env:     []string{"SOME=value"},
		Timeout: &timeout,
	}
	command := createCommandHook(hook)
	timeoutStr := command.Timeout.String()
	if timeoutStr != "1h0m0s" {
		t.Errorf("Expected the Timeout to be 1h0m0s, got: %s", timeoutStr)
	}
}

func TestCreateHooks(t *testing.T) {
	rspec := &specs.Spec{
		Hooks: &specs.Hooks{
			Prestart: []specs.Hook{
				{
					Path: "/some/hook/path",
				},
				{
					Path: "/some/hook2/path",
					Args: []string{"--some", "thing"},
				},
			},
			CreateRuntime: []specs.Hook{
				{
					Path: "/some/hook/path",
				},
				{
					Path: "/some/hook2/path",
					Args: []string{"--some", "thing"},
				},
			},
			CreateContainer: []specs.Hook{
				{
					Path: "/some/hook/path",
				},
				{
					Path: "/some/hook2/path",
					Args: []string{"--some", "thing"},
				},
			},
			StartContainer: []specs.Hook{
				{
					Path: "/some/hook/path",
				},
				{
					Path: "/some/hook2/path",
					Args: []string{"--some", "thing"},
				},
			},
			Poststart: []specs.Hook{
				{
					Path: "/some/hook/path",
					Args: []string{"--some", "thing"},
					Env:  []string{"SOME=value"},
				},
				{
					Path: "/some/hook2/path",
				},
				{
					Path: "/some/hook3/path",
				},
			},
			Poststop: []specs.Hook{
				{
					Path: "/some/hook/path",
					Args: []string{"--some", "thing"},
					Env:  []string{"SOME=value"},
				},
				{
					Path: "/some/hook2/path",
				},
				{
					Path: "/some/hook3/path",
				},
				{
					Path: "/some/hook4/path",
					Args: []string{"--some", "thing"},
				},
			},
		},
	}
	conf := &configs.Config{}
	createHooks(rspec, conf)

	prestart := conf.Hooks[configs.Prestart]

	if len(prestart) != 2 {
		t.Error("Expected 2 Prestart hooks")
	}

	createRuntime := conf.Hooks[configs.CreateRuntime]

	if len(createRuntime) != 2 {
		t.Error("Expected 2 createRuntime hooks")
	}

	createContainer := conf.Hooks[configs.CreateContainer]

	if len(createContainer) != 2 {
		t.Error("Expected 2 createContainer hooks")
	}

	startContainer := conf.Hooks[configs.StartContainer]

	if len(startContainer) != 2 {
		t.Error("Expected 2 startContainer hooks")
	}

	poststart := conf.Hooks[configs.Poststart]

	if len(poststart) != 3 {
		t.Error("Expected 3 Poststart hooks")
	}

	poststop := conf.Hooks[configs.Poststop]

	if len(poststop) != 4 {
		t.Error("Expected 4 Poststop hooks")
	}

}
func TestSetupSeccomp(t *testing.T) {
	conf := &specs.LinuxSeccomp{
		DefaultAction: "SCMP_ACT_ERRNO",
		Architectures: []specs.Arch{specs.ArchX86_64, specs.ArchARM},
		Syscalls: []specs.LinuxSyscall{
			{
				Names:  []string{"clone"},
				Action: "SCMP_ACT_ALLOW",
				Args: []specs.LinuxSeccompArg{
					{
						Index:    0,
						Value:    unix.CLONE_NEWNS | unix.CLONE_NEWUTS | unix.CLONE_NEWIPC | unix.CLONE_NEWUSER | unix.CLONE_NEWPID | unix.CLONE_NEWNET | unix.CLONE_NEWCGROUP,
						ValueTwo: 0,
						Op:       "SCMP_CMP_MASKED_EQ",
					},
				},
			},
			{
				Names: []string{
					"select",
					"semctl",
					"semget",
					"semop",
					"semtimedop",
					"send",
					"sendfile",
				},
				Action: "SCMP_ACT_ALLOW",
			},
		},
	}
	seccomp, err := SetupSeccomp(conf)

	if err != nil {
		t.Errorf("Couldn't create Seccomp config: %v", err)
	}

	if seccomp.DefaultAction != 2 { // SCMP_ACT_ERRNO
		t.Error("Wrong conversion for DefaultAction")
	}

	if len(seccomp.Architectures) != 2 {
		t.Error("Wrong number of architectures")
	}

	if seccomp.Architectures[0] != "amd64" || seccomp.Architectures[1] != "arm" {
		t.Error("Expected architectures are not found")
	}

	calls := seccomp.Syscalls

	callsLength := len(calls)
	if callsLength != 8 {
		t.Errorf("Expected 8 syscalls, got :%d", callsLength)
	}

	for i, call := range calls {
		if i == 0 {
			expectedCloneSyscallArgs := configs.Arg{
				Index:    0,
				Op:       7, // SCMP_CMP_MASKED_EQ
				Value:    unix.CLONE_NEWNS | unix.CLONE_NEWUTS | unix.CLONE_NEWIPC | unix.CLONE_NEWUSER | unix.CLONE_NEWPID | unix.CLONE_NEWNET | unix.CLONE_NEWCGROUP,
				ValueTwo: 0,
			}
			if expectedCloneSyscallArgs != *call.Args[0] {
				t.Errorf("Wrong arguments conversion for the clone syscall under test")
			}
		}
		if call.Action != 4 {
			t.Error("Wrong conversion for the clone syscall action")
		}

	}

}

func TestLinuxCgroupWithMemoryResource(t *testing.T) {
	cgroupsPath := "/user/cgroups/path/id"

	spec := &specs.Spec{}
	devices := []specs.LinuxDeviceCgroup{
		{
			Allow:  false,
			Access: "rwm",
		},
	}

	limit := int64(100)
	reservation := int64(50)
	swap := int64(20)
	kernel := int64(40)
	kernelTCP := int64(45)
	swappiness := uint64(1)
	swappinessPtr := &swappiness
	disableOOMKiller := true
	resources := &specs.LinuxResources{
		Devices: devices,
		Memory: &specs.LinuxMemory{
			Limit:            &limit,
			Reservation:      &reservation,
			Swap:             &swap,
			Kernel:           &kernel,
			KernelTCP:        &kernelTCP,
			Swappiness:       swappinessPtr,
			DisableOOMKiller: &disableOOMKiller,
		},
	}
	spec.Linux = &specs.Linux{
		CgroupsPath: cgroupsPath,
		Resources:   resources,
	}

	opts := &CreateOpts{
		CgroupName:       "ContainerID",
		UseSystemdCgroup: false,
		Spec:             spec,
	}

	cgroup, err := CreateCgroupConfig(opts)
	if err != nil {
		t.Errorf("Couldn't create Cgroup config: %v", err)
	}

	if cgroup.Path != cgroupsPath {
		t.Errorf("Wrong cgroupsPath, expected '%s' got '%s'", cgroupsPath, cgroup.Path)
	}
	if cgroup.Resources.Memory != limit {
		t.Errorf("Expected to have %d as memory limit, got %d", limit, cgroup.Resources.Memory)
	}
	if cgroup.Resources.MemoryReservation != reservation {
		t.Errorf("Expected to have %d as memory reservation, got %d", reservation, cgroup.Resources.MemoryReservation)
	}
	if cgroup.Resources.MemorySwap != swap {
		t.Errorf("Expected to have %d as swap, got %d", swap, cgroup.Resources.MemorySwap)
	}
	if cgroup.Resources.KernelMemory != kernel {
		t.Errorf("Expected to have %d as Kernel Memory, got %d", kernel, cgroup.Resources.KernelMemory)
	}
	if cgroup.Resources.KernelMemoryTCP != kernelTCP {
		t.Errorf("Expected to have %d as TCP Kernel Memory, got %d", kernelTCP, cgroup.Resources.KernelMemoryTCP)
	}
	if cgroup.Resources.MemorySwappiness != swappinessPtr {
		t.Errorf("Expected to have %d as memory swappiness, got %d", swappinessPtr, cgroup.Resources.MemorySwappiness)
	}
	if cgroup.Resources.OomKillDisable != disableOOMKiller {
		t.Errorf("The OOMKiller should be enabled")
	}
}

func TestLinuxCgroupSystemd(t *testing.T) {
	cgroupsPath := "parent:scopeprefix:name"

	spec := &specs.Spec{}
	spec.Linux = &specs.Linux{
		CgroupsPath: cgroupsPath,
	}

	opts := &CreateOpts{
		UseSystemdCgroup: true,
		Spec:             spec,
	}

	cgroup, err := CreateCgroupConfig(opts)

	if err != nil {
		t.Errorf("Couldn't create Cgroup config: %v", err)
	}

	expectedParent := "parent"
	if cgroup.Parent != expectedParent {
		t.Errorf("Expected to have %s as Parent instead of %s", expectedParent, cgroup.Parent)
	}

	expectedScopePrefix := "scopeprefix"
	if cgroup.ScopePrefix != expectedScopePrefix {
		t.Errorf("Expected to have %s as ScopePrefix instead of %s", expectedScopePrefix, cgroup.ScopePrefix)
	}

	expectedName := "name"
	if cgroup.Name != expectedName {
		t.Errorf("Expected to have %s as Name instead of %s", expectedName, cgroup.Name)
	}
}

func TestLinuxCgroupSystemdWithEmptyPath(t *testing.T) {
	cgroupsPath := ""

	spec := &specs.Spec{}
	spec.Linux = &specs.Linux{
		CgroupsPath: cgroupsPath,
	}

	opts := &CreateOpts{
		CgroupName:       "ContainerID",
		UseSystemdCgroup: true,
		Spec:             spec,
	}

	cgroup, err := CreateCgroupConfig(opts)

	if err != nil {
		t.Errorf("Couldn't create Cgroup config: %v", err)
	}

	expectedParent := "system.slice"
	if cgroup.Parent != expectedParent {
		t.Errorf("Expected to have %s as Parent instead of %s", expectedParent, cgroup.Parent)
	}

	expectedScopePrefix := "runc"
	if cgroup.ScopePrefix != expectedScopePrefix {
		t.Errorf("Expected to have %s as ScopePrefix instead of %s", expectedScopePrefix, cgroup.ScopePrefix)
	}

	if cgroup.Name != opts.CgroupName {
		t.Errorf("Expected to have %s as Name instead of %s", opts.CgroupName, cgroup.Name)
	}
}

func TestLinuxCgroupSystemdWithInvalidPath(t *testing.T) {
	cgroupsPath := "/user/cgroups/path/id"

	spec := &specs.Spec{}
	spec.Linux = &specs.Linux{
		CgroupsPath: cgroupsPath,
	}

	opts := &CreateOpts{
		CgroupName:       "ContainerID",
		UseSystemdCgroup: true,
		Spec:             spec,
	}

	_, err := CreateCgroupConfig(opts)
	if err == nil {
		t.Error("Expected to produce an error if not using the correct format for cgroup paths belonging to systemd")
	}
}
func TestLinuxCgroupsPathSpecified(t *testing.T) {
	cgroupsPath := "/user/cgroups/path/id"

	spec := &specs.Spec{}
	spec.Linux = &specs.Linux{
		CgroupsPath: cgroupsPath,
	}

	opts := &CreateOpts{
		CgroupName:       "ContainerID",
		UseSystemdCgroup: false,
		Spec:             spec,
	}

	cgroup, err := CreateCgroupConfig(opts)
	if err != nil {
		t.Errorf("Couldn't create Cgroup config: %v", err)
	}

	if cgroup.Path != cgroupsPath {
		t.Errorf("Wrong cgroupsPath, expected '%s' got '%s'", cgroupsPath, cgroup.Path)
	}
}

func TestLinuxCgroupsPathNotSpecified(t *testing.T) {
	spec := &specs.Spec{}
	opts := &CreateOpts{
		CgroupName:       "ContainerID",
		UseSystemdCgroup: false,
		Spec:             spec,
	}

	cgroup, err := CreateCgroupConfig(opts)
	if err != nil {
		t.Errorf("Couldn't create Cgroup config: %v", err)
	}

	if cgroup.Path != "" {
		t.Errorf("Wrong cgroupsPath, expected it to be empty string, got '%s'", cgroup.Path)
	}
}

func TestSpecconvExampleValidate(t *testing.T) {
	spec := Example()
	spec.Root.Path = "/"

	opts := &CreateOpts{
		CgroupName:       "ContainerID",
		UseSystemdCgroup: false,
		Spec:             spec,
	}

	config, err := CreateLibcontainerConfig(opts)
	if err != nil {
		t.Errorf("Couldn't create libcontainer config: %v", err)
	}

	if config.NoNewPrivileges != spec.Process.NoNewPrivileges {
		t.Errorf("specconv NoNewPrivileges mismatch. Expected %v got %v",
			spec.Process.NoNewPrivileges, config.NoNewPrivileges)
	}

	validator := validate.New()
	if err := validator.Validate(config); err != nil {
		t.Errorf("Expected specconv to produce valid container config: %v", err)
	}
}

func TestDupNamespaces(t *testing.T) {
	spec := &specs.Spec{
		Root: &specs.Root{
			Path: "rootfs",
		},
		Linux: &specs.Linux{
			Namespaces: []specs.LinuxNamespace{
				{
					Type: "pid",
				},
				{
					Type: "pid",
					Path: "/proc/1/ns/pid",
				},
			},
		},
	}

	_, err := CreateLibcontainerConfig(&CreateOpts{
		Spec: spec,
	})

	if !strings.Contains(err.Error(), "malformed spec file: duplicated ns") {
		t.Errorf("Duplicated namespaces should be forbidden")
	}
}

func TestNonZeroEUIDCompatibleSpecconvValidate(t *testing.T) {
	if _, err := os.Stat("/proc/self/ns/user"); os.IsNotExist(err) {
		t.Skip("userns is unsupported")
	}

	spec := Example()
	spec.Root.Path = "/"
	ToRootless(spec)

	opts := &CreateOpts{
		CgroupName:       "ContainerID",
		UseSystemdCgroup: false,
		Spec:             spec,
		RootlessEUID:     true,
		RootlessCgroups:  true,
	}

	config, err := CreateLibcontainerConfig(opts)
	if err != nil {
		t.Errorf("Couldn't create libcontainer config: %v", err)
	}

	validator := validate.New()
	if err := validator.Validate(config); err != nil {
		t.Errorf("Expected specconv to produce valid rootless container config: %v", err)
	}
}

func TestInitSystemdProps(t *testing.T) {
	type inT struct {
		name, value string
	}
	type expT struct {
		isErr bool
		name  string
		value interface{}
	}

	testCases := []struct {
		desc string
		in   inT
		exp  expT
	}{
		{
			in:  inT{"org.systemd.property.TimeoutStopUSec", "uint64 123456789"},
			exp: expT{false, "TimeoutStopUSec", uint64(123456789)},
		},
		{
			desc: "convert USec to Sec (default numeric type)",
			in:   inT{"org.systemd.property.TimeoutStopSec", "456"},
			exp:  expT{false, "TimeoutStopUSec", uint64(456000000)},
		},
		{
			desc: "convert USec to Sec (byte)",
			in:   inT{"org.systemd.property.TimeoutStopSec", "byte 234"},
			exp:  expT{false, "TimeoutStopUSec", uint64(234000000)},
		},
		{
			desc: "convert USec to Sec (int16)",
			in:   inT{"org.systemd.property.TimeoutStopSec", "int16 234"},
			exp:  expT{false, "TimeoutStopUSec", uint64(234000000)},
		},
		{
			desc: "convert USec to Sec (uint16)",
			in:   inT{"org.systemd.property.TimeoutStopSec", "uint16 234"},
			exp:  expT{false, "TimeoutStopUSec", uint64(234000000)},
		},
		{
			desc: "convert USec to Sec (int32)",
			in:   inT{"org.systemd.property.TimeoutStopSec", "int32 234"},
			exp:  expT{false, "TimeoutStopUSec", uint64(234000000)},
		},
		{
			desc: "convert USec to Sec (uint32)",
			in:   inT{"org.systemd.property.TimeoutStopSec", "uint32 234"},
			exp:  expT{false, "TimeoutStopUSec", uint64(234000000)},
		},
		{
			desc: "convert USec to Sec (int64)",
			in:   inT{"org.systemd.property.TimeoutStopSec", "int64 234"},
			exp:  expT{false, "TimeoutStopUSec", uint64(234000000)},
		},
		{
			desc: "convert USec to Sec (uint64)",
			in:   inT{"org.systemd.property.TimeoutStopSec", "uint64 234"},
			exp:  expT{false, "TimeoutStopUSec", uint64(234000000)},
		},
		{
			desc: "convert USec to Sec (float)",
			in:   inT{"org.systemd.property.TimeoutStopSec", "234.789"},
			exp:  expT{false, "TimeoutStopUSec", uint64(234789000)},
		},
		{
			desc: "convert USec to Sec (bool -- invalid value)",
			in:   inT{"org.systemd.property.TimeoutStopSec", "false"},
			exp:  expT{true, "", ""},
		},
		{
			desc: "convert USec to Sec (string -- invalid value)",
			in:   inT{"org.systemd.property.TimeoutStopSec", "'covfefe'"},
			exp:  expT{true, "", ""},
		},
		{
			in:  inT{"org.systemd.property.CollectMode", "'inactive-or-failed'"},
			exp: expT{false, "CollectMode", "inactive-or-failed"},
		},
		{
			desc: "unrelated property",
			in:   inT{"some.other.annotation", "0"},
			exp:  expT{false, "", ""},
		},
		{
			desc: "too short property name",
			in:   inT{"org.systemd.property.Xo", "1"},
			exp:  expT{true, "", ""},
		},
		{
			desc: "invalid character in property name",
			in:   inT{"org.systemd.property.Number1", "1"},
			exp:  expT{true, "", ""},
		},
		{
			desc: "invalid property value",
			in:   inT{"org.systemd.property.ValidName", "invalid-value"},
			exp:  expT{true, "", ""},
		},
	}

	spec := &specs.Spec{}

	for _, tc := range testCases {
		tc := tc
		spec.Annotations = map[string]string{tc.in.name: tc.in.value}

		outMap, err := initSystemdProps(spec)
		//t.Logf("input %+v, expected %+v, got err:%v out:%+v", tc.in, tc.exp, err, outMap)

		if tc.exp.isErr != (err != nil) {
			t.Errorf("input %+v, expecting error: %v, got %v", tc.in, tc.exp.isErr, err)
		}
		expLen := 1 // expect a single item
		if tc.exp.name == "" {
			expLen = 0 // expect nothing
		}
		if len(outMap) != expLen {
			t.Fatalf("input %+v, expected %d, got %d entries: %v", tc.in, expLen, len(outMap), outMap)
		}
		if expLen == 0 {
			continue
		}

		out := outMap[0]
		if tc.exp.name != out.Name {
			t.Errorf("input %+v, expecting name: %q, got %q", tc.in, tc.exp.name, out.Name)
		}
		expValue := dbus.MakeVariant(tc.exp.value).String()
		if expValue != out.Value.String() {
			t.Errorf("input %+v, expecting value: %s, got %s", tc.in, expValue, out.Value)
		}
	}
}

func TestNullProcess(t *testing.T) {
	spec := Example()
	spec.Process = nil

	_, err := CreateLibcontainerConfig(&CreateOpts{
		Spec: spec,
	})

	if err != nil {
		t.Errorf("Null process should be forbidden")
	}
}
