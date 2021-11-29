//go:build darwin
// +build darwin

package lock

import (
	"log"
)

// RunDaemon runs lock daemon
func RunDaemon() {
	log.Fatal("running daemon on darwin systems are not supported.")
}
