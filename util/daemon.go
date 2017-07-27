package util

import (
	"os"
	"syscall"
)

func restart() (int, error) {
	os.Setenv("daemon", "true")
	procAttr := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
	}
	pid, err := syscall.ForkExec(os.Args[0], os.Args, procAttr)
	if err != nil {
		return 0, err
	}
	return pid, nil
}

func Daemon() {
	if os.Getenv("daemon") != "true" {
		restart()
		os.Exit(0)
	}
}
