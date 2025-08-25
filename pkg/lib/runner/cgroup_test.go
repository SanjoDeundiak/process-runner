package runner

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
)

// Runs only as root on linux
func TestCgroup(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping: not running on Linux")
	}

	if os.Geteuid() != 0 {
		t.Skip("Skipping: not running as root")
	}

	runner, err := NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}

	res, err := runner.Start("sh", "-c", "sleep 60")
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	processId := res.ID
	pid := fmt.Sprintf("%v", res.pid)
	procsData, err := os.ReadFile(fmt.Sprintf("/sys/fs/cgroup/prn/%s/cgroup.procs", processId))
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Check that pid was attached to cgroup
	procsStr := strings.TrimSpace(string(procsData))
	if procsStr != pid {
		t.Fatalf("cgroup fail: %s. Expected: %s", procsStr, pid)
	}

	cpuWeightData, err := os.ReadFile(fmt.Sprintf("/sys/fs/cgroup/prn/%s/cpu.weight", processId))
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	cpuWeightStr := strings.TrimSpace(string(cpuWeightData))
	if cpuWeightStr != "100" {
		t.Fatalf("cgroup fail cpu weight: %s", cpuWeightStr)
	}

	memoryHighData, err := os.ReadFile(fmt.Sprintf("/sys/fs/cgroup/prn/%s/memory.high", processId))
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	memoryHighStr := strings.TrimSpace(string(memoryHighData))
	if memoryHighStr != fmt.Sprint(int64(512)*1024*1024) {
		t.Fatalf("cgroup fail memory high: %s", memoryHighStr)
	}

	ioWeightData, err := os.ReadFile(fmt.Sprintf("/sys/fs/cgroup/prn/%s/io.weight", processId))
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	ioWeightStr := strings.TrimSpace(string(ioWeightData))
	if ioWeightStr != "default 100" {
		t.Fatalf("cgroup fail io weight: %s.", ioWeightStr)
	}
}
