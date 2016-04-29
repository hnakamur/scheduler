// Package scheduler provides a time scheduler.
package scheduler

import (
	"container/heap"
	"time"

	"golang.org/x/net/context"
)

// Item represents a scheduled item. It has the time and user data.
type Item struct {
	Time time.Time
	Data interface{}
}

type queueItem struct {
	schedule Item
	index    int // The index of the queueItem in the heap
}

type queue []*queueItem

func newQueue() queue {
	return make([]*queueItem, 0)
}

func (q queue) Len() int { return len(q) }
func (q queue) Less(i, j int) bool {
	return q[i].schedule.Time.Before(q[j].schedule.Time)
}

func (q queue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *queue) Push(x interface{}) {
	n := len(*q)
	queueItem := x.(*queueItem)
	queueItem.index = n
	*q = append(*q, queueItem)
}

func (q *queue) Pop() interface{} {
	old := *q
	n := len(old)
	queueItem := old[n-1]
	queueItem.index = -1 // for safety
	*q = old[0 : n-1]
	return queueItem
}

// Scheduler sends a scheduled item with the channel at the specified time.
// The order of items at the same time is not specified and may not the
// same order as calls of Schedule() for those items.
type Scheduler struct {
	C     chan Item
	ctx   context.Context
	queue queue
	timer *time.Timer
}

// NewScheduler creates a scheduler. Pass a context created with context.WithCancel()
// and call cancel() (the cancel function is also returned from context.WithCancel())
// to stop the created scheduler.
func NewScheduler(ctx context.Context) *Scheduler {
	timer := time.NewTimer(time.Second)
	timer.Stop()
	s := &Scheduler{
		C:     make(chan Item),
		ctx:   ctx,
		queue: newQueue(),
		timer: timer,
	}
	heap.Init(&s.queue)
	go s.run()
	return s
}

// Schedule an item.
func (s *Scheduler) Schedule(schedule Item) {
	heap.Push(&s.queue, &queueItem{schedule: schedule})
	s.updateTimer()
}

func (s *Scheduler) updateTimer() {
	if len(s.queue) == 0 {
		s.timer.Stop()
		return
	}

	var d time.Duration
	now := time.Now()
	t := s.queue[0].schedule.Time
	if t.After(now) {
		d = t.Sub(now)
	}
	s.timer.Reset(d)
}

func (s *Scheduler) run() {
	for {
		select {
		case <-s.timer.C:
			queueItem := heap.Pop(&s.queue).(*queueItem)
			s.C <- queueItem.schedule
			s.updateTimer()
		case <-s.ctx.Done():
			s.timer.Stop()
			return
		}
	}
}
