package converge

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type PriorityMutex struct {
	l      sync.Mutex
	pcount atomic.Int64
}

func (p *PriorityMutex) PLock() {
	p.pcount.Add(1)
	p.l.Lock()
}

func (p *PriorityMutex) PUnlock() {
	p.l.Unlock()
	p.pcount.Add(-1)
}

func (p *PriorityMutex) Lock() {
	if p.pcount.Load() > 0 {
		//log.Println("GoSched")
		runtime.Gosched()
	}
	p.l.Lock()
}

func (p *PriorityMutex) Unlock() {
	p.l.Unlock()
}
