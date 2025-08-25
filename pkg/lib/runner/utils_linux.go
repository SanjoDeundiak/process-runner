//go:build linux
// +build linux

package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

	cGroupFile, err = os.Open(cgPath)
	if err != nil {
		return nil, err
	}

	return &syscall.SysProcAttr{
		// New process group to manage children as a unit
		Setpgid:     true,
		UseCgroupFD: 1,
		CgroupFd:    cGroupFile.Fd(),
	}
}

func KillCgroup(id string) error {
	cgDir := filepath.Join(cgroupRoot, cgroupNs, id)
	return writeString(filepath.Join(cgDir, "cgroup.kill"), "1")
}

func CleanupCgroup(id string) error {
	cgDir := filepath.Join(cgroupRoot, cgroupNs, id)
	return os.Remove(cgDir)
}

func setupCgroupFor(id string) (*string, error) {
	// Ensure namespace directory exists
	nsDir := filepath.Join(cgroupRoot, cgroupNs)
	if err := os.MkdirAll(nsDir, 0o755); err != nil {
		return nil, err
	}
	// Enable controllers on nsDir so children can configure limits
	_ = writeString(filepath.Join(nsDir, "cgroup.subtree_control"), "+cpu +io +memory")
	// Create process-specific cgroup
	cgDir := filepath.Join(nsDir, id)
	if err := os.MkdirAll(cgDir, 0o755); err != nil {
		return err
	}
	// Write limits
	cpuW := 100
	_ = writeString(filepath.Join(cgDir, "cpu.weight"), strconv.Itoa(cpuW))
	ioW := 100
	_ = writeString(filepath.Join(cgDir, "io.weight"), fmt.Sprintf("default %d", ioW))
	memoryHigh := 512 * 1024 * 1024
	_ = writeString(filepath.Join(cgDir, "memory.high"), strconv.FormatInt(opts.MemoryHigh, 10))

	return nil
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
