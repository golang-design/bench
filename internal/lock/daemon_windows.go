//go:build windows
// +build windows

package lock

import (
	"log"
)

// RunDaemon runs lock daemon
func RunDaemon() {
	log.Fatal("running daemon on windows systems are not supported.")
}
