package entgen

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"time"
)

func configureWorkerCancellation(command *exec.Cmd) {
	command.Cancel = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// taskkill 在终止 worker 前收集并终止其子进程树
		terminate := exec.CommandContext(ctx, "taskkill", "/T", "/F", "/PID", strconv.Itoa(command.Process.Pid))
		terminate.WaitDelay = time.Second

		err := terminate.Run()
		if err != nil {
			return errors.Join(err, command.Process.Kill())
		}

		return nil
	}
}
