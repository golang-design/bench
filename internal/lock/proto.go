package lock

import "encoding/gob"

type perflockAction struct {
	Action interface{}
}

// actionAcquire acquires the lock. The response is a boolean
// indicating whether or not the lock was acquired (which may be false
// for a non-blocking acquire).
type actionAcquire struct {
	Shared      bool
	NonBlocking bool
	Msg         string
}

// actionList returns the list of current and pending lock
// acquisitions as a []string.
type actionList struct {
}

// actionSetCPUFreq sets the CPU frequency of all CPUs. The caller
// must hold the lock.
type actionSetCPUFreq struct {
	// Percent indicates the percent to set the CPU cpuFreq to
	// between the lower and highest available frequencies.
	Percent int
}

func init() {
	gob.Register(actionAcquire{})
	gob.Register(actionList{})
	gob.Register(actionSetCPUFreq{})
}
