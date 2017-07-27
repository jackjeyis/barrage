package logger

import (
	"fmt"
	"io"
	"os"
)

const (
	red = iota + 91
	green
	yellow
	blue
	magenta

	info = "[INFO]"
	trac = "[TRACE]"
	err  = "[ERROR]"
	warn = "[WARN]"

	exper = "\x1b[1;%dm%s\x1b[0m"
)

type Logger struct {
	f io.Writer
}

func newLogger(fname string) *Logger {
	logger := &Logger{}
	if fname == "" {
		logger.f = os.Stdout
	} else {
		if f, err := os.Create(fname); err == nil {
			logger.f = f
		} else {
			return nil
		}
	}
	return logger
}

func Color(c int, s string) string {
	return fmt.Sprintf(exper, c, s)
}

func (l *Logger) Info(s string, a ...interface{}) {
	fmt.Fprintln(l.f, Color(green, fmt.Sprintf(info+"  "+s, a)))
}
func (l *Logger) Trace(s string, a ...interface{}) {
	fmt.Fprintln(l.f, Color(blue, fmt.Sprintf(trac+"  "+s, a)))
}

func main1() {
	logger := newLogger("")
	logger.Info("helelo %s", "23423")
}
