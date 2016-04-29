// Package scheduler provides a task scheduler which dipatches tasks
// at the specified time for each task.
package scheduler

import (
	"container/heap"
	"time"

	"golang.org/x/net/context"
)

// Task represents a scheduled task. It has the time and user data.
// You may modify the user data after calling Schedule().
type Task struct {
	Data  interface{}
	index int // The index of the task in the heap
	time  time.Time
}

func (t *Task) Time() time.Time {
	return t.time
}

type queue []*Task

func newQueue() queue {
	return make([]*Task, 0)
}

func (q queue) Len() int { return len(q) }
func (q queue) Less(i, j int) bool {
	return q[i].time.Before(q[j].time)
}

func (q queue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *queue) Push(x interface{}) {
	n := len(*q)
	item := x.(*Task)
	item.index = n
	*q = append(*q, item)
}

func (q *queue) Pop() interface{} {
	old := *q
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*q = old[0 : n-1]
	return item
}

// Scheduler sends a scheduled task with the channel C at the specified time.
// The order of tasks at the same time is not specified and may not be the
// same order as calls of Schedule() for those tasks.
type Scheduler struct {
	C     chan *Task
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
		C:     make(chan *Task),
		ctx:   ctx,
		queue: newQueue(),
		timer: timer,
	}
	heap.Init(&s.queue)
	go s.run()
	return s
}

// Schedule a task with the time to dispatch and the user data.
// It returns the task object which you can use for canceling this task.
func (s *Scheduler) Schedule(t time.Time, data interface{}) *Task {
	task := &Task{time: t, Data: data}
	heap.Push(&s.queue, task)
	s.updateTimer()
	return task
}

// Cancel the task.
// It returns true if the task is canceled, or false if the task has been
// already dispatched.
func (s *Scheduler) Cancel(task *Task) bool {
	if task.index == -1 {
		return false
	}
	heap.Remove(&s.queue, task.index)
	s.updateTimer()
	return true
}

func (s *Scheduler) updateTimer() {
	if len(s.queue) == 0 {
		s.timer.Stop()
		return
	}

	var d time.Duration
	now := time.Now()
	t := s.queue[0].time
	if t.After(now) {
		d = t.Sub(now)
	}
	s.timer.Reset(d)
}

func (s *Scheduler) run() {
	for {
		select {
		case <-s.timer.C:
			s.C <- heap.Pop(&s.queue).(*Task)
			s.updateTimer()
		case <-s.ctx.Done():
			s.timer.Stop()
			return
		}
	}
}
