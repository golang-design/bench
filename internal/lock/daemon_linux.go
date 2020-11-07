package lock

// +build !darwin

import (
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/user"
	"time"

	"golang.design/x/bench/internal/cpupower"
)

var theLock perflock

// RunDaemon runs lock daemon
func RunDaemon() {
	// check if daemon is running
	c, _ := net.Dial("unix", Socketpath)
	if c != nil {
		c.Close()
		log.Fatalf("The bench daemon is already running at %s !", Socketpath)
		return
	}

	os.Remove(Socketpath)
	l, err := net.Listen("unix", Socketpath)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	// Make the socket world-writable/connectable.
	err = os.Chmod(Socketpath, 0777)
	if err != nil {
		log.Fatal(err)
	}

	// Receive connections.
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go func(c net.Conn) {
			defer c.Close()
			NewServer(c).Serve()
		}(conn)
	}
}

// Server is the bench lock server
type Server struct {
	c        net.Conn
	userName string

	locker    *locker
	acquiring bool

	oldCPUFreqs []*cpuFreqSettings
}

// NewServer returns a bench lock server
func NewServer(c net.Conn) *Server {
	return &Server{c: c}
}

// Serve serves the bench lock server
func (s *Server) Serve() {
	// Drop any held locks if we exit for any reason.
	defer s.drop()

	// Get connection credentials.
	ucred, err := readCredentials(s.c.(*net.UnixConn))
	if err != nil {
		log.Print("reading credentials: ", err)
		return
	}

	u, err := user.LookupId(fmt.Sprintf("%d", ucred.Uid))
	s.userName = "???"
	if err == nil {
		s.userName = u.Username
	}

	// Receive incoming actions. We do this in a goroutine so the
	// main handler can select on EOF or lock acquisition.
	actions := make(chan perflockAction)
	go func() {
		gr := gob.NewDecoder(s.c)
		for {
			var msg perflockAction
			err := gr.Decode(&msg)
			if err != nil {
				if err != io.EOF {
					log.Print(err)
				}
				close(actions)
				return
			}
			actions <- msg
		}
	}()

	// Process incoming actions.
	var acquireC <-chan bool
	gw := gob.NewEncoder(s.c)
	for {
		select {
		case action, ok := <-actions:
			if !ok {
				// Connection closed.
				return
			}
			if s.acquiring {
				log.Printf("protocol error: message while acquiring")
				return
			}
			switch action := action.Action.(type) {
			case actionAcquire:
				if s.locker != nil {
					log.Printf("protocol error: acquiring lock twice")
					return
				}
				msg := fmt.Sprintf("%s\t%s\t%s", s.userName, time.Now().Format(time.Stamp), action.Msg)
				if action.Shared {
					msg += " [shared]"
				}
				s.locker = theLock.Enqueue(action.Shared, action.NonBlocking, msg)
				if s.locker != nil {
					// Enqueued. Wait for acquire.
					s.acquiring = true
					acquireC = s.locker.C
				} else {
					// Non-blocking acquire failed.
					if err := gw.Encode(false); err != nil {
						log.Print(err)
						return
					}
				}

			case actionList:
				list := theLock.Queue()
				if err := gw.Encode(list); err != nil {
					log.Print(err)
					return
				}

			case actionSetCPUFreq:
				if s.locker == nil {
					log.Printf("protocol error: setting cpuFreq without lock")
					return
				}
				err := s.setCPUFreq(action.Percent)
				errString := ""
				if err != nil {
					errString = err.Error()
				}
				if err := gw.Encode(errString); err != nil {
					log.Print(err)
					return
				}

			default:
				log.Printf("unknown message")
				return
			}

		case <-acquireC:
			// Lock acquired.
			s.acquiring, acquireC = false, nil
			if err := gw.Encode(true); err != nil {
				log.Print(err)
				return
			}
		}
	}
}

func (s *Server) drop() {
	// Restore the CPU cpuFreq before releasing the lock.
	if s.oldCPUFreqs != nil {
		s.restoreCPUFreq()
		s.oldCPUFreqs = nil
	}
	// Release the lock.
	if s.locker != nil {
		theLock.Dequeue(s.locker)
		s.locker = nil
	}
}

type cpuFreqSettings struct {
	domain   *cpupower.Domain
	min, max int
}

func (s *Server) setCPUFreq(percent int) error {
	domains, err := cpupower.Domains()
	if err != nil {
		return err
	}
	if len(domains) == 0 {
		return fmt.Errorf("no power domains")
	}

	// Save current frequency settings.
	old := []*cpuFreqSettings{}
	for _, d := range domains {
		min, max, err := d.CurrentRange()
		if err != nil {
			return err
		}
		old = append(old, &cpuFreqSettings{d, min, max})
	}
	s.oldCPUFreqs = old

	// Set new settings.
	abs := func(x int) int {
		if x < 0 {
			return -x
		}
		return x
	}
	for _, d := range domains {
		min, max, avail := d.AvailableRange()
		target := (max-min)*percent/100 + min

		// Find the nearest available frequency.
		if len(avail) != 0 {
			closest := avail[0]
			for _, a := range avail {
				if abs(target-a) < abs(target-closest) {
					closest = a
				}
			}
			target = closest
		}

		err := d.SetRange(target, target)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) restoreCPUFreq() error {
	var err error
	for _, g := range s.oldCPUFreqs {
		// Try to set all of the domains, even if one fails.
		err1 := g.domain.SetRange(g.min, g.max)
		if err1 != nil && err == nil {
			err = err1
		}
	}
	return err
}
