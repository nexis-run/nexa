//go:build !windows

package entgen

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func configureWorkerCancellation(command *exec.Cmd) {
	// worker 和 Ent 启动的 Go 子进程使用同一独立进程组
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}

		return err
	}
}
