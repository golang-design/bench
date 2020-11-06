package lock

import (
	"encoding/gob"
	"fmt"
	"log"
	"net"
)

// Client is a lock client
type Client struct {
	c net.Conn

	gr *gob.Encoder
	gw *gob.Decoder
}

// NewClient returns a lock client
func NewClient() *Client {
	c, err := net.Dial("unix", Socketpath)
	if err != nil {
		log.Printf("failed to connect bench daemon: %v", err)
		return nil
	}

	// Send credentials.
	err = writeCredentials(c.(*net.UnixConn))
	if err != nil {
		log.Fatal("failed to send credentials: ", err)
	}

	gr, gw := gob.NewEncoder(c), gob.NewDecoder(c)

	return &Client{c, gr, gw}
}

func (c *Client) do(action perflockAction, response interface{}) {
	err := c.gr.Encode(action)
	if err != nil {
		log.Fatal(err)
	}

	err = c.gw.Decode(response)
	if err != nil {
		log.Fatal(err)
	}
}

// Acquire acuiqres the lock
func (c *Client) Acquire(shared, nonblocking bool, msg string) bool {
	var ok bool
	c.do(perflockAction{actionAcquire{Shared: shared, NonBlocking: nonblocking, Msg: msg}}, &ok)
	return ok
}

// List lists all perflock actions
func (c *Client) List() []string {
	var list []string
	c.do(perflockAction{actionList{}}, &list)
	return list
}

// SetCPUFreq sets the given cpu frequency
func (c *Client) SetCPUFreq(percent int) error {
	var err string
	c.do(perflockAction{actionSetCPUFreq{Percent: percent}}, &err)
	if err == "" {
		return nil
	}
	return fmt.Errorf("%s", err)
}
