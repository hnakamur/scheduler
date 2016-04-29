package scheduler

import (
	"log"
	"os"
	"testing"
	"time"

	"golang.org/x/net/context"
)

const maxDelay = 10 * time.Millisecond

func TestSchedule(t *testing.T) {
	verbose := testing.Verbose()
	var logger *log.Logger
	if verbose {
		// NOTE: Usually we should use testing.T.Logf for logging, but
		// we create a logger to see times of receiving schedules.
		logger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := NewScheduler(ctx)

	now := time.Now().Truncate(100 * time.Millisecond)
	timeAndValues := []struct {
		t time.Time
		v string
	}{
		{t: now.Add(100 * time.Millisecond), v: "foo"},
		{t: now.Add(300 * time.Millisecond), v: "baz"},
		{t: now.Add(200 * time.Millisecond), v: "bar"},
		{t: now.Add(500 * time.Millisecond), v: "hoge"},
	}
	for _, tv := range timeAndValues {
		s.Schedule(tv.t, tv.v)
	}

	indexes := []int{0, 2, 1, 3}
	i := 0
	for {
		select {
		case task := <-s.C:
			if verbose {
				logger.Printf("received task.time=%v, data=%s", task.Time(), task.Data)
			}
			now := time.Now()
			if now.Before(task.Time()) {
				t.Errorf("task received too early. now is %v; want %v", now, task.Time())
			}
			if now.After(task.Time().Add(maxDelay)) {
				t.Errorf("task delayed too much. now is %v; want %v", now, task.Time())
			}
			tv := timeAndValues[indexes[i]]
			if !task.Time().Equal(tv.t) {
				t.Errorf("task time unmatch got %v; want %v", task.Time(), tv.t)
			}
			if task.Data != tv.v {
				t.Errorf("task data unmatch got %v; want %v", task.Data, tv.v)
			}

			i++
			if i == len(timeAndValues) {
				cancel()
				return
			}
		}
	}
}

func TestCancel(t *testing.T) {
	verbose := testing.Verbose()
	var logger *log.Logger
	if verbose {
		logger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := NewScheduler(ctx)

	now := time.Now().Truncate(100 * time.Millisecond)
	timeAndValues := []struct {
		t    time.Time
		v    string
		task *Task
	}{
		{t: now.Add(100 * time.Millisecond), v: "foo"},
		{t: now.Add(300 * time.Millisecond), v: "baz"},
		{t: now.Add(500 * time.Millisecond), v: "huga"},
		{t: now.Add(200 * time.Millisecond), v: "bar"},
		{t: now.Add(500 * time.Millisecond), v: "hoge"},
	}
	for i, tv := range timeAndValues {
		timeAndValues[i].task = s.Schedule(tv.t, tv.v)
	}

	canceled := s.Cancel(timeAndValues[2].task)
	if !canceled {
		t.Errorf("cancel failed")
	}

	indexes := []int{0, 3, 1, 4}
	i := 0
	for {
		select {
		case task := <-s.C:
			if verbose {
				logger.Printf("received task.time=%v, data=%s", task.Time(), task.Data)
			}
			now := time.Now()
			if now.Before(task.Time()) {
				t.Errorf("task received too early. now is %v; want %v", now, task.Time())
			}
			if now.After(task.Time().Add(maxDelay)) {
				t.Errorf("task delayed too much. now is %v; want %v", now, task.Time())
			}
			tv := timeAndValues[indexes[i]]
			if !task.Time().Equal(tv.t) {
				t.Errorf("task time unmatch got %v; want %v", task.Time(), tv.t)
			}
			if task.Data != tv.v {
				t.Errorf("task data unmatch got %v; want %v", task.Data, tv.v)
			}

			i++
			if i == len(timeAndValues)-1 {
				cancel()
				return
			}
		}
	}
}

func TestCancelEarliest(t *testing.T) {
	verbose := testing.Verbose()
	var logger *log.Logger
	if verbose {
		logger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := NewScheduler(ctx)

	now := time.Now().Truncate(100 * time.Millisecond)
	timeAndValues := []struct {
		t    time.Time
		v    string
		task *Task
	}{
		{t: now.Add(100 * time.Millisecond), v: "foo"},
		{t: now.Add(300 * time.Millisecond), v: "baz"},
		{t: now.Add(500 * time.Millisecond), v: "huga"},
		{t: now.Add(200 * time.Millisecond), v: "bar"},
		{t: now.Add(400 * time.Millisecond), v: "hoge"},
	}
	for i, tv := range timeAndValues {
		timeAndValues[i].task = s.Schedule(tv.t, tv.v)
	}

	task := s.CancelEarliest()
	if task == nil {
		t.Errorf("cancel earliest failed")
	}

	indexes := []int{3, 1, 4, 2}
	i := 0
	for {
		select {
		case task := <-s.C:
			if verbose {
				logger.Printf("received task.time=%v, data=%s", task.Time(), task.Data)
			}
			now := time.Now()
			if now.Before(task.Time()) {
				t.Errorf("task received too early. now is %v; want %v", now, task.Time())
			}
			if now.After(task.Time().Add(maxDelay)) {
				t.Errorf("task delayed too much. now is %v; want %v", now, task.Time())
			}
			tv := timeAndValues[indexes[i]]
			if !task.Time().Equal(tv.t) {
				t.Errorf("task time unmatch got %v; want %v", task.Time(), tv.t)
			}
			if task.Data != tv.v {
				t.Errorf("task data unmatch got %v; want %v", task.Data, tv.v)
			}

			i++
			if i == len(timeAndValues)-1 {
				cancel()
				return
			}
		}
	}
}
