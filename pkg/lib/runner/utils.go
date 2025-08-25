package runner

import (
	"os"
	"syscall"
)

type SysProcAttr struct {
	File *os.File
	Raw  *syscall.SysProcAttr
}
