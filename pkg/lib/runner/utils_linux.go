//go:build linux
// +build linux

package runner

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	cgroupRoot = "/sys/fs/cgroup"
	cgroupNs   = "prn"
)

// TODO: Test

func GetSysProcAttr(id string) (*SysProcAttr, error) {
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
	cgDir := filepath.Join(cgroupRoot, cgroupNs, id)
	err := writeString(filepath.Join(cgDir, "cgroup.kill"), "1")

	return err == nil, err
}

func CleanupCgroup(id string) error {
	cgDir := filepath.Join(cgroupRoot, cgroupNs, id)
	return os.Remove(cgDir)
}

func setupCgroupFor(processId string) (*string, error) {
	if !isV2() {
		return nil, fmt.Errorf("cgroup v2 not enabled")
	}

	rel, err := readSelfCgroupV2()
	if err != nil {
		return nil, err
	}
	// Absolute path to our current cgroup directory
	base := filepath.Join("/sys/fs/cgroup", strings.TrimPrefix(rel, "/"))

	// We will create children under our own cgroup. Usually writable if you have delegation/root.
	err = enableControllers(base, []string{"memory", "cpu", "io"})
	if err != nil {
		return nil, err
	}

	cgDir := filepath.Join(base, "prn", processId)
	err = os.MkdirAll(cgDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("mkdir %s: %v", cgDir, err)
	}

	// Write limits
	cpuW := 100
	_ = writeString(filepath.Join(cgDir, "cpu.weight"), strconv.Itoa(cpuW))
	ioW := 100
	_ = writeString(filepath.Join(cgDir, "io.weight"), fmt.Sprintf("default %d", ioW))
	memoryHigh := int64(512) * 1024 * 1024
	_ = writeString(filepath.Join(cgDir, "memory.high"), strconv.FormatInt(memoryHigh, 10))

	// FIXME
	//// Ensure namespace directory exists
	//nsDir := filepath.Join(cgroupRoot, cgroupNs)
	//if err := os.MkdirAll(nsDir, 0o755); err != nil {
	//	return nil, err
	//}
	//// Enable controllers on nsDir so children can configure limits
	//_ = writeString(filepath.Join(nsDir, "cgroup.subtree_control"), "+cpu +io +memory")

	// FIXME
	// Optional cleanup: kill everything and remove
	// _ = mustWrite(filepath.Join(childCg, "cgroup.kill"), "1")
	// _ = os.Remove(childCg)

	return &cgDir, nil
}

func writeString(path, val string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(val)

	return err
}

// isV2: /sys/fs/cgroup/cgroup.controllers exists on v2
func isV2() bool {
	_, err := os.Stat("/sys/fs/cgroup/cgroup.controllers")
	return err == nil
}

// readSelfCgroupV2 returns the cgroup-relative path ("" for root) of this process.
func readSelfCgroupV2() (string, error) {
	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	defer f.Close()
	// v2 line looks like: "0::/user.slice/user-1000.slice/user@1000.service/app.slice"
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		parts := strings.SplitN(line, ":", 3)
		if len(parts) == 3 && parts[0] == "0" {
			return parts[2], nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", errors.New("no v2 line in /proc/self/cgroup")
}

// enableControllers enables a subset (+memory +cpu +pids) in the parent cgroup’s subtree_control.
func enableControllers(parent string, want []string) error {
	ctrlsBytes, err := os.ReadFile(filepath.Join(parent, "cgroup.controllers"))
	if err != nil {
		return fmt.Errorf("read cgroup.controllers: %w", err)
	}
	available := map[string]bool{}
	for _, f := range strings.Fields(string(ctrlsBytes)) {
		available[f] = true
	}
	var toEnable []string
	for _, w := range want {
		if available[w] {
			toEnable = append(toEnable, "+"+w)
		}
	}
	if len(toEnable) == 0 {
		return nil // nothing to do, or not available
	}
	line := strings.Join(toEnable, " ")
	return os.WriteFile(filepath.Join(parent, "cgroup.subtree_control"), []byte(line), 0644)
}
