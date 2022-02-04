package uevent

import (
	"sync"
)

type queue struct {
	cond *sync.Cond

	order []string
	items sync.Map

	backoffItems sync.Map
}

func newQueue() *queue {
	cond := sync.NewCond(&sync.Mutex{})

	return &queue{
		cond:  cond,
		order: []string{},
		items: &sync.Map{},
	}
}

// new item goes to end of queue
func (q *queue) Push(d *deviceEvent) {
	q.cond.L.Lock()
	defer func() {
		q.cond.L.Unlock()
		q.cond.Signal()
	}()

	if item, ok := q.items.Load(d.path); ok {
		// only add if this event is newer than the existing event
		q.items.Store(d.path, d)
		return
	}

	q.order = append(q.order, d.path)
	q.items.Store(d.path, d)
}

// we pop from beginning of queue
func (q *queue) Pop() (d *deviceEvent) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()

	for len(q.order) == 0 {
		q.Cond.Wait()
	}
	
	path := q.order[0]
	q.order = q.order[1:]
	if item, ok := q.items.LoadAndDelete(path); ok {
		return item.(*deviceEvent)
	}
	panic("should never happen")
}
