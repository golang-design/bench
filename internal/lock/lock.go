package lock

import "sync"

type perflock struct {
	l sync.Mutex
	q []*locker
}

type locker struct {
	C      <-chan bool
	c      chan<- bool
	shared bool
	woken  bool

	msg string
}

func (l *perflock) Enqueue(shared, nonblocking bool, msg string) *locker {
	ch := make(chan bool, 1)
	locker := &locker{ch, ch, shared, false, msg}

	// Enqueue.
	l.l.Lock()
	defer l.l.Unlock()
	l.setQ(append(l.q, locker))

	if nonblocking && !locker.woken {
		// Acquire failed. Dequeue.
		l.setQ(l.q[:len(l.q)-1])
		return nil
	}

	return locker
}

func (l *perflock) Dequeue(locker *locker) {
	l.l.Lock()
	defer l.l.Unlock()
	for i, o := range l.q {
		if locker == o {
			copy(l.q[i:], l.q[i+1:])
			l.setQ(l.q[:len(l.q)-1])
			return
		}
	}
	panic("Dequeue of non-enqueued locker")
}

func (l *perflock) Queue() []string {
	var q []string

	l.l.Lock()
	defer l.l.Unlock()
	for _, locker := range l.q {
		q = append(q, locker.msg)
	}
	return q
}

func (l *perflock) setQ(q []*locker) {
	l.q = q
	if len(q) == 0 {
		return
	}

	wake := func(locker *locker) {
		if locker.woken == false {
			locker.woken = true
			locker.c <- true
		}
	}
	if q[0].shared {
		// Wake all shared acquires at the head of the queue.
		for _, locker := range q {
			if !locker.shared {
				break
			}
			wake(locker)
		}
	} else {
		wake(q[0])
	}
}
