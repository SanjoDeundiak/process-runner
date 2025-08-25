package main

import (
	"fmt"
	"strings"

	apiv1 "github.com/SanjoDeundiak/process-runner/api/v1"
)

func printStatusTable(id string, st *apiv1.ProcessStatus, p *apiv1.Process) {
	state := ""
	if st != nil {
		switch st.GetState() {
		case apiv1.ProcessState_PROCESS_STATE_RUNNING:
			state = "Running"
		case apiv1.ProcessState_PROCESS_STATE_STOPPED:
			state = "Stopped"
		default:
			state = "Unknown"
		}
	}
	cmd := ""
	if p != nil {
		all := append([]string{p.GetCommand()}, p.GetArgs()...)
		cmd = strings.TrimSpace(strings.Join(all, " "))
	}

	// Determine column widths
	idW := maxInt(36, len(id))
	stateW := maxInt(7, len(state))
	cmdW := maxInt(7, len(cmd))

	// There is more data in the process state that might be useful
	sep := fmt.Sprintf("+-%s-+-%s-+-%s-+\n", strings.Repeat("-", idW), strings.Repeat("-", stateW), strings.Repeat("-", cmdW))
	fmt.Print(sep)
	fmt.Printf("| %s | %s | %s |\n", pad("ID", idW), pad("STATE", stateW), pad("COMMAND", cmdW))
	fmt.Print(sep)
	fmt.Printf("| %s | %s | %s |\n", pad(id, idW), pad(state, stateW), pad(cmd, cmdW))
	fmt.Print(sep)
}

func pad(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
