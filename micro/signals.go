package micro

import (
	"os"
	"syscall"
)

// interruptSignals interrupt signals.
// default interrupt signals to catch,you can use InterruptSignal option to append more.
var interruptSignals = []os.Signal{
	syscall.SIGINT, syscall.SIGTERM, os.Interrupt, syscall.SIGHUP,
	syscall.SIGSTOP, syscall.SIGQUIT,
}
