//go:build linux

package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

const (
	cgroupRoot = "/sys/fs/cgroup/prn"
)

var (
	cgroupInitOnce sync.Once
	cgroupInitErr  error
)

// InitCgroups initializes the cgroup root for process-runner.
// It is safe to call multiple times; real work happens only once.
// As non-root, this is a no-op.
func initCgroups() error {
	cgroupInitOnce.Do(func() {
		cgroupInitErr = initCgroupsImpl()
	})
	return cgroupInitErr
}

func initCgroupsImpl() error {
	if os.Geteuid() != 0 {
		// Not running as root; don't attempt to create/modify cgroups
		return nil
	}

	if err := os.MkdirAll(cgroupRoot, 0755); err != nil {
		return err
	}

	// Determine which controllers are available and already enabled on this cgroup
	available, err := readControllerSet(filepath.Join(cgroupRoot, "cgroup.controllers"))
	if err != nil {
		return err
	}
	enabled, err := readControllerSet(filepath.Join(cgroupRoot, "cgroup.subtree_control"))
	if err != nil {
		return err
	}

	desired := []string{"cpu", "io", "memory"}
	var toAdd []string
	for _, ctrl := range desired {
		if available[ctrl] && !enabled[ctrl] {
			toAdd = append(toAdd, "+"+ctrl)
		}
	}
	if len(toAdd) > 0 {
		if err := writeString(filepath.Join(cgroupRoot, "cgroup.subtree_control"), strings.Join(toAdd, " ")); err != nil {
			return err
		}
	}
	return nil
}

func readControllerSet(path string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool)
	fields := strings.Fields(string(data))
	for _, f := range fields {
		// Entries in these files are names like "cpu", "io", "memory"
		// subtree_control may present names without "+" prefix when read
		f = strings.TrimPrefix(f, "+")
		set[f] = true
	}
	return set, nil
}

func GetSysProcAttr(id string) (*SysProcAttr, error) {
	// available only as root. Better api is required here
	if os.Geteuid() != 0 {
		return &SysProcAttr{
			File: nil,
			Raw: &syscall.SysProcAttr{
				Setpgid: true,
			},
		}, nil
	}

	// Ensure cgroup root is initialized
	_ = initCgroups()

	cgPath, err := setupCgroupFor(id)
	if err != nil {
		return nil, err
	}

	cGroupFile, err := os.Open(*cgPath)
	if err != nil {
		return nil, err
	}

	cGroupFd := cGroupFile.Fd()

	sysProcAttr := &syscall.SysProcAttr{
		// New process group to manage children as a unit
		Setpgid:     true,
		UseCgroupFD: true,
		CgroupFD:    int(cGroupFd),
	}
	return &SysProcAttr{
		File: cGroupFile,
		Raw:  sysProcAttr,
	}, nil
}

func KillCgroup(id string) (bool, error) {
	cgDir := filepath.Join(cgroupRoot, id)
	err := writeString(filepath.Join(cgDir, "cgroup.kill"), "1")

	return err == nil, err
}

func CleanupCgroup(id string) error {
	cgDir := filepath.Join(cgroupRoot, id)
	return os.Remove(cgDir)
}

func setupCgroupFor(processId string) (*string, error) {
	processRoot := filepath.Join(cgroupRoot, processId)
	if err := os.MkdirAll(processRoot, 0755); err != nil {
		return nil, err
	}

	// Only write controller-specific files if controllers are enabled
	if controllerEnabled(cgroupRoot, "cpu") {
		if err := writeString(filepath.Join(processRoot, "cpu.weight"), "100"); err != nil {
			return nil, err
		}
	}
	if controllerEnabled(cgroupRoot, "io") {
		if err := writeString(filepath.Join(processRoot, "io.weight"), "100"); err != nil {
			return nil, err
		}
	}
	if controllerEnabled(cgroupRoot, "memory") {
		if err := writeString(filepath.Join(processRoot, "memory.high"), fmt.Sprint(int64(512)*1024*1024)); err != nil {
			return nil, err
		}
	}

	return &processRoot, nil
}

func controllerEnabled(cgPath, controller string) bool {
	enabled, err := readControllerSet(filepath.Join(cgPath, "cgroup.subtree_control"))
	if err != nil {
		return false
	}
	return enabled[controller]
}

func writeString(path, val string) error {
	return os.WriteFile(path, []byte(val), 0644)
}
