//go:build darwin
// +build darwin

package lock

import (
	"errors"
	"net"
)

func writeCredentials(c *net.UnixConn) error {
	return errors.New("unimplemented")
}
