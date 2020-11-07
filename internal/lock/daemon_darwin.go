package lock

import (
	"log"
)

// +build darwin

// RunDaemon runs lock daemon
func RunDaemon() {
	log.Fatal("running daemon on darwin systems are not supported.")
}
