package test

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// getAvailableAddress returns a random available port by letting the OS assign one
func getAvailableAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("failed to get available port: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	return fmt.Sprintf("localhost:%d", port)
}

// repoRoot returns the absolute path to the repository root.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	// When this test lives under test/, the repo root is its parent dir.
	dir := filepath.Dir(file)
	if filepath.Base(dir) == "test" {
		return filepath.Dir(dir)
	}
	// Fallback for historical location under cmd/cli/e2e_test.go.
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func setEnv(t *testing.T, addr string, name string, useFakeCert bool, cmd *exec.Cmd) {
	t.Helper()

	root := repoRoot(t)

	var prefix string
	if useFakeCert {
		prefix = "_fake"
	} else {
		prefix = ""
	}

	keyPath := filepath.Join(root, fmt.Sprintf("%s%s_key.pem", name, prefix))
	certPath := filepath.Join(root, fmt.Sprintf("%s%s.pem", name, prefix))
	caCertPath := filepath.Join(root, fmt.Sprintf("ca%s.pem", prefix))

	var env []string
	env = append(env, os.Environ()...)
	env = append(env, fmt.Sprintf("PRN_ADDRESS=%s", addr))
	env = append(env, fmt.Sprintf("PRN_TLS_KEY=%s", readFile(t, keyPath)))
	env = append(env, fmt.Sprintf("PRN_TLS_CERT=%s", readFile(t, certPath)))
	env = append(env, fmt.Sprintf("PRN_CA_TLS_CERT=%s", readFile(t, caCertPath)))

	cmd.Env = env
}

func startServer(t *testing.T, addr string, useFakeCert bool) func() {
	t.Helper()

	cmd := exec.Command("go", "run", "../cmd/server")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	setEnv(t, addr, "server", useFakeCert, cmd)

	cmd.Stdin = nil
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	pid := cmd.Process.Pid

	done := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		done <- err
	}()

	// Wait for readiness by probing TCP port
	deadline := time.Now().Add(8 * time.Second)
	for {
		success := false

		select {
		case <-time.After(100 * time.Millisecond):
			c, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
			if c != nil {
				_ = c.Close()
			}

			if err == nil {
				success = true
				time.Sleep(200 * time.Millisecond)
				break
			}
		case <-time.After(time.Until(deadline)):
			t.Fatalf("server did not become ready in time")
		}

		if success {
			break
		}
	}

	stop := func() {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		<-done
	}

	return stop
}

type startResult struct {
	stdout              io.ReadCloser
	stderr              io.ReadCloser
	cmd                 *exec.Cmd
	done                chan error
	stdOutDrainCallback chan struct{}
	stdErrDrainCallback chan struct{}
}

func startCliCommand(t *testing.T, address string, clientName string, cliCommand string, cliArgs ...string) *startResult {
	t.Helper()

	cmd := exec.Command("go")
	setEnv(t, address, clientName, false, cmd)
	args := append(cmd.Args, "run", "../cmd/cli", cliCommand, "--")
	args = append(args, cliArgs...)
	cmd.Args = args

	cmd.Stdin = nil

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start CLI command: %v", err)
	}

	done := make(chan error, 1)
	stdOutDrainCallback := make(chan struct{})
	stdErrDrainCallback := make(chan struct{})
	go func() {
		<-stdOutDrainCallback
		<-stdErrDrainCallback
		err := cmd.Wait()
		done <- err
	}()

	return &startResult{stdout: stdoutPipe, stderr: stderrPipe, cmd: cmd, done: done, stdOutDrainCallback: stdOutDrainCallback, stdErrDrainCallback: stdErrDrainCallback}
}

type executionResult struct {
	stdout string
	stderr string
}

func executeCliCommand(t *testing.T, address string, clientName string, expectFailure bool, cliCommand string, cliArgs ...string) *executionResult {
	t.Helper()

	startRes := startCliCommand(t, address, clientName, cliCommand, cliArgs...)

	stdoutData, err := io.ReadAll(startRes.stdout)
	startRes.stdOutDrainCallback <- struct{}{}
	// If reading from pipes failed, surface the error to aid debugging.
	if err != nil {
		t.Fatalf("failed to read CLI stdout: %v", err)
	}

	stderrData, err := io.ReadAll(startRes.stderr)
	startRes.stdErrDrainCallback <- struct{}{}
	if err != nil {
		t.Fatalf("failed to read CLI stderr: %v", err)
	}

	select {
	case err := <-startRes.done:
		if err != nil && !expectFailure {
			t.Fatalf("cli command failed: %s. Stdout: %s. Stderr: %s", err, stdoutData, stderrData)
		}

		return &executionResult{
			stdout: string(stdoutData),
			stderr: string(stderrData),
		}
	case <-time.After(8 * time.Second):
		_ = startRes.cmd.Process.Kill()
		t.Fatalf("cli command timed out")
	}

	return nil
}

func TestE2E_StartStatusStop(t *testing.T) {
	addr := getAvailableAddress(t)
	stop := startServer(t, addr, false)
	defer stop()

	// 1) Start a process
	res := executeCliCommand(t, addr, "client1", false, "start", "sleep", "1")
	processId := strings.TrimSpace(res.stdout)
	if processId == "" {
		t.Fatalf("expected process id, got empty string")
	}

	// 2) Status should print a table containing the id and command
	res = executeCliCommand(t, addr, "client1", false, "status", processId)
	if res.stderr != "" {
		t.Fatalf("unexpected stderr: %q", res.stderr)
	}
	outStatus := res.stdout
	if !strings.Contains(outStatus, processId) || !strings.Contains(outStatus, "sleep 1") {
		t.Fatalf("status output does not contain expected fields: %q", outStatus)
	}

	// 3) Stop should stop the process and print Stopped
	res = executeCliCommand(t, addr, "client1", false, "stop", processId)
	if res.stderr != "" {
		t.Fatalf("unexpected stderr: %q", res.stderr)
	}
	outStatus = res.stdout
	if !strings.Contains(outStatus, "Stopped") {
		t.Fatalf("expected Stopped state, got: %q", outStatus)
	}
}

func TestE2E_OutputLiveProcess(t *testing.T) {
	addr := getAvailableAddress(t)
	stop := startServer(t, addr, false)
	defer stop()

	res := executeCliCommand(t, addr, "client1", false, "start", "sh", "-c", "echo hello; sleep 1; echo goodbye")
	processId := strings.TrimSpace(res.stdout)

	startRes := startCliCommand(t, addr, "client1", "logs", processId)
	time.Sleep(200 * time.Millisecond)
	buffer := make([]byte, 128)
	n, err := startRes.stdout.Read(buffer)
	if err != nil {
		t.Fatalf("failed to read from stdout: %v", err)
	}
	str := string(buffer[0:n])
	if str != "hello\n" {
		t.Fatalf("expected 'hello\\n', got %q", str)
	}

	n, err = startRes.stdout.Read(buffer)
	if err != nil {
		t.Fatalf("failed to read from stdout: %v", err)
	}
	str = string(buffer[0:n])
	if str != "goodbye\n" {
		t.Fatalf("expected 'hello\\n', got %q", str)
	}
}

func TestE2E_OutputStoppedProcess(t *testing.T) {
	addr := getAvailableAddress(t)
	stop := startServer(t, addr, false)
	defer stop()

	res := executeCliCommand(t, addr, "client1", false, "start", "sh", "-c", "echo hello")
	processId := strings.TrimSpace(res.stdout)
	time.Sleep(200 * time.Millisecond)

	res = executeCliCommand(t, addr, "client1", false, "logs", processId)
	str := res.stdout
	if res.stdout != "hello\n" {
		t.Fatalf("expected 'hello\\n', got %q", str)
	}
}

func TestE2E_Authz(t *testing.T) {
	addr := getAvailableAddress(t)
	stop := startServer(t, addr, false)
	defer stop()

	res := executeCliCommand(t, addr, "client1", false, "start", "sh", "-c", "echo hello")
	processId := strings.TrimSpace(res.stdout)
	time.Sleep(200 * time.Millisecond)

	res = executeCliCommand(t, addr, "client2", false, "logs", processId)
	str := res.stdout
	if res.stdout != "hello\n" {
		t.Fatalf("expected 'hello\\n', got %q", str)
	}

	res = executeCliCommand(t, addr, "client2", true, "status", processId)
	if !strings.Contains(res.stderr, "Forbidden") {
		t.Fatalf("status output does not contain expected value: %q", res.stderr)
	}

	res = executeCliCommand(t, addr, "client2", true, "stop", processId)
	if !strings.Contains(res.stderr, "Forbidden") {
		t.Fatalf("status output does not contain expected value: %q", res.stderr)
	}
}
