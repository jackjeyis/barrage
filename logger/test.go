package logger

import (
	"log"
	"net"
	"os"
	"syscall"
)

func daemon() error {
	os.Setenv("daemon", "true")
	procAtrr := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
	}
	pid, err := syscall.ForkExec(os.Args[0], os.Args, procAtrr)
	if err != nil {
		return err
	}
	log.Printf("child process pid is %d,parent pid is %d", pid, os.Getppid())
	return nil
}
func main() {
	if os.Getenv("daemon") != "true" {
		log.Println("daemon")
		daemon()
		os.Exit(0)
	}
	log.Printf("main go ... current pid %d", os.Getpid())
	log.Println("listen on 12345")
	for {
		if ln, err := net.Listen("tcp", ":12345"); err == nil {
			log.Println("listen success")
			if conn, errc := ln.Accept(); errc == nil {
				log.Println("connect succ!")
				var buf []byte
				conn.Read(buf)
			}
		}
	}
}
