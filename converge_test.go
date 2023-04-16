package converge

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewConverge(t *testing.T) {
	c, err := New[int, int](func(elms []int) (map[int]int, error) {
		time.Sleep(time.Second)
		res := make(map[int]int)
		for _, v := range elms {
			res[v] = 2 * v
		}
		return res, nil
	}, NewConfig(10, 5*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Stop()
	wg := sync.WaitGroup{}
	wg.Add(5000)

	var totalMillSec int64
	var lessone, lesstwo, lessthree int32

	for i := 0; i < 5000; i++ {
		go func(n int) {
			//time.Sleep(time.Duration(n) * time.Millisecond)
			defer wg.Done()
			begin := time.Now()
			_, err := c.Do([]int{n, n + 1, n + 2, n + 3})
			if err != nil {
				panic(err)
			}
			cost := time.Now().Sub(begin).Milliseconds()
			atomic.AddInt64(&totalMillSec, cost)
			if cost < 1000 {
				atomic.AddInt32(&lessone, 1)
			} else if cost < 2000 {
				atomic.AddInt32(&lesstwo, 1)
			} else {
				atomic.AddInt32(&lessthree, 1)
			}
			//t.Log(cost)
		}(i)
	}

	wg.Wait()
	tt := atomic.LoadInt64(&totalMillSec)
	t.Log("total time", atomic.LoadInt64(&totalMillSec))
	t.Log("< 1s", atomic.LoadInt32(&lessone))
	t.Log("1s< and <2s", atomic.LoadInt32(&lesstwo))
	t.Log(">2s", atomic.LoadInt32(&lessthree))
	t.Log("average time per req", time.Duration(tt)*time.Millisecond/5000)
}

var num int64

func BenchmarkConverge_Do(b *testing.B) {
	c, err := New[int, int](func(elms []int) (map[int]int, error) {
		time.Sleep(time.Second)
		res := make(map[int]int)
		for _, v := range elms {
			res[v] = 2 * v
		}
		return res, nil
	}, NewConfig(10, 5*time.Millisecond))
	if err != nil {
		b.Fatal(err)
	}
	defer c.Stop()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			doval := int(atomic.AddInt64(&num, 1))
			c.Do([]int{doval, doval + 1, doval + 2, doval + 3})
		}
	})
}
