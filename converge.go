package converge

import (
	"context"
	"sync"
)

type Converge[T comparable, V any] struct {
	p          sync.RWMutex
	l          sync.Mutex
	pending    []pendingReq[T, V]
	f          func([]T) (map[T]V, error)
	requesting chan struct{}
	cancel     context.CancelFunc
	ctx        context.Context
}

type Config[T comparable, V any] struct {
	batchFunc  func([]T) (map[T]V, error)
	concurrent int
}

func NewConfig[T comparable, V any](batchFunc func([]T) (map[T]V, error)) *Config[T, V] {
	return &Config[T, V]{batchFunc: batchFunc, concurrent: 1}
}

func (c *Config[T, V]) WithConcurrent(concurrent int) *Config[T, V] {
	if concurrent <= 0 {
		panic("invalid concurrent")
	}
	c.concurrent = concurrent
	return c
}

func New[T comparable, V any](cfg *Config[T, V]) *Converge[T, V] {
	ctx, cancel := context.WithCancel(context.TODO())
	c := &Converge[T, V]{
		pending:    make([]pendingReq[T, V], 0, 10),
		f:          cfg.batchFunc,
		requesting: make(chan struct{}, cfg.concurrent),
		cancel:     cancel,
		ctx:        ctx,
	}
	for i := 0; i < cfg.concurrent; i++ {
		go c.run()
	}
	return c
}

type Result[V any] struct {
	val    V
	exist  bool
	shared bool
}

type ResWrap[T comparable, V any] struct {
	data map[T]Result[V]
	err  error
}

type pendingReq[T comparable, V any] struct {
	elms    []T
	resChan chan ResWrap[T, V]
}

func (c *Converge[T, V]) Do(elms []T) (map[T]Result[V], error) {
	select {
	case <-c.ctx.Done():
		return nil, c.ctx.Err()
	default:
		resWrap := <-c.req(elms)
		return resWrap.data, resWrap.err
	}
}

func (c *Converge[T, V]) Stop() {
	c.cancel()
}

func (c *Converge[T, V]) addPending(elms []T, resChan chan ResWrap[T, V]) {
	c.p.RLock()
	defer c.p.RUnlock()
	c.l.Lock()
	defer c.l.Unlock()
	c.pending = append(c.pending, pendingReq[T, V]{
		elms:    elms,
		resChan: resChan,
	})
}

func (c *Converge[T, V]) clearPending() []pendingReq[T, V] {
	c.p.Lock()
	defer c.p.Unlock()

	c.l.Lock()
	defer c.l.Unlock()

	tmp := c.pending
	c.pending = make([]pendingReq[T, V], 0, 10)
	return tmp
}

func (c *Converge[T, V]) req(elms []T) chan ResWrap[T, V] {
	resChan := make(chan ResWrap[T, V], 1)
	c.addPending(elms, resChan)
	select {
	case c.requesting <- struct{}{}:
	default:
	}
	return resChan
}

func (c *Converge[T, V]) run() {
	for {
		select {
		case <-c.requesting:
			c.doCall()
		case <-c.ctx.Done():
			select {
			case <-c.requesting:
				c.doCall()
			default:
			}
			return
		}
	}
}

func (c *Converge[T, V]) doCall() {
	pendingReqs := c.clearPending()
	if len(pendingReqs) == 0 {
		return
	}
	dupMap := make(map[T]bool)
	items := []T{}
	for _, v := range pendingReqs {
		for _, elm := range v.elms {
			_, ok := dupMap[elm]
			if ok {
				dupMap[elm] = true
			} else {
				dupMap[elm] = false
				items = append(items, elm)
			}
		}
	}

	resMap, err := c.f(items)
	for _, v := range pendingReqs {
		result := make(map[T]Result[V])
		for _, k := range v.elms {
			val, ok := resMap[k]
			result[k] = Result[V]{
				val:    val,
				exist:  ok,
				shared: dupMap[k],
			}
		}
		v.resChan <- ResWrap[T, V]{
			data: result,
			err:  err,
		}
	}
}
