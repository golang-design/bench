package lock

// +build darwin

import (
	"errors"
	"net"
)

func writeCredentials(c *net.UnixConn) error {
	return errors.New("unimplemented")
}
