//go:build !linux

package runner

import (
	"syscall"
)

func GetSysProcAttr(id string) (*SysProcAttr, error) {
	return &SysProcAttr{
		File: nil,
		Raw: &syscall.SysProcAttr{
			// New process group to manage children as a unit
			Setpgid: true,
		}}, nil
}

func KillCgroup(id string) (bool, error) {
	return false, nil
}

func CleanupCgroup(id string) error {
	return nil
}
