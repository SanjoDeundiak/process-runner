package runner

import (
	"os"
	"syscall"
)

type SysProcAttr struct {
	Fd  *os.File
	Raw *syscall.SysProcAttr
}
