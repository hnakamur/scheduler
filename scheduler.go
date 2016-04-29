// Package scheduler provides a task scheduler which dipatches tasks
// at the specified time for each task.
package scheduler

import (
	"container/heap"
	"sync"
	"time"

	"golang.org/x/net/context"
)

// Task represents a scheduled task. It has the time and user data.
// You may modify the user data after calling Scheduler.Schedule().
type Task struct {
	time  time.Time
	Data  interface{}
	index int // The index of the task in the heap
}

// NewTask creates a task.
func NewTask(t time.Time, data interface{}) *Task {
	return &Task{
		time:  t,
		Data:  data,
		index: -1, // for safety
	}
}

// Time returns the time when the task will be dispatched.
func (t *Task) Time() time.Time {
	return t.time
}

type taskHeap []*Task

func newTaskHeap() taskHeap {
	return make([]*Task, 0)
}

func (h taskHeap) Len() int { return len(h) }
func (h taskHeap) Less(i, j int) bool {
	return h[i].time.Before(h[j].time)
}

func (h taskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *taskHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*Task)
	item.index = n
	*h = append(*h, item)
}

func (h *taskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*h = old[0 : n-1]
	return item
}

// TaskQueue represents a queue for tasks. Tasks are sorted by time.
type TaskQueue struct {
	taskHeap taskHeap
}

// NewTaskQueue creates a TaskQueue.
func NewTaskQueue() *TaskQueue {
	q := &TaskQueue{
		taskHeap: newTaskHeap(),
	}
	heap.Init(&q.taskHeap)
	return q
}

// Len returns the length of the TaskQueue.
func (q *TaskQueue) Len() int {
	return q.taskHeap.Len()
}

// Push a task to the TaskQueue.
func (q *TaskQueue) Push(task *Task) {
	heap.Push(&q.taskHeap, task)
}

// Pop a task from the TaskQueue. Returns nil if the TaskQueue is empty.
func (q *TaskQueue) Pop() *Task {
	if len(q.taskHeap) == 0 {
		return nil
	}
	return heap.Pop(&q.taskHeap).(*Task)
}

// Peek returns the first task without removing it from the TaskQueue.
// Returns nil if the TaskQueue is empty.
func (q *TaskQueue) Peek() *Task {
	if len(q.taskHeap) == 0 {
		return nil
	}
	return q.taskHeap[0]
}

// Remove a task from the TaskQueue. Returns true if the task was removed,
// or false if the task was already removed.
func (q *TaskQueue) Remove(task *Task) bool {
	if task.index == -1 {
		return false
	}
	heap.Remove(&q.taskHeap, task.index)
	return true
}

// Scheduler sends a scheduled task with the channel C at the specified time.
// The order of tasks at the same time is not specified and may not be the
// same order as calls of Schedule() for those tasks.
type Scheduler struct {
	C     chan *Task
	ctx   context.Context
	queue *TaskQueue
	timer *time.Timer
	mu    sync.Mutex
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
		queue: NewTaskQueue(),
		timer: timer,
	}
	go s.run()
	return s
}

// Schedule a task with the time to dispatch and the user data.
// It returns the task object which you can use for canceling this task.
func (s *Scheduler) Schedule(t time.Time, data interface{}) *Task {
	s.mu.Lock()
	task := NewTask(t, data)
	s.queue.Push(task)
	s.updateTimer()
	s.mu.Unlock()
	return task
}

// Cancel the task.
// It returns true if the task is canceled, or false if the task has been
// already dispatched.
func (s *Scheduler) Cancel(task *Task) bool {
	s.mu.Lock()
	removed := s.queue.Remove(task)
	s.updateTimer()
	s.mu.Unlock()
	return removed
}

func (s *Scheduler) updateTimer() {
	if s.queue.Len() == 0 {
		s.timer.Stop()
		return
	}

	var d time.Duration
	now := time.Now()
	t := s.queue.Peek().Time()
	if t.After(now) {
		d = t.Sub(now)
	}
	s.timer.Reset(d)
}

func (s *Scheduler) run() {
	for {
		select {
		case <-s.timer.C:
			s.mu.Lock()
			task := s.queue.Pop()
			if task != nil {
				s.C <- task
			}
			s.updateTimer()
			s.mu.Unlock()
		case <-s.ctx.Done():
			s.timer.Stop()
			return
		}
	}
}
