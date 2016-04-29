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

	now := time.Now()
	timeAndValues := []struct {
		t time.Time
		v string
	}{
		{t: now.Add(1 * time.Second).Truncate(time.Second), v: "foo"},
		{t: now.Add(3 * time.Second).Truncate(time.Second), v: "baz"},
		{t: now.Add(2 * time.Second).Truncate(time.Second), v: "bar"},
		{t: now.Add(5 * time.Second).Truncate(time.Second), v: "hoge"},
	}
	for _, tv := range timeAndValues {
		s.Schedule(Item{Time: tv.t, Data: tv.v})
	}

	indexes := []int{0, 2, 1, 3}
	i := 0
	for {
		select {
		case schedule := <-s.C:
			if verbose {
				logger.Printf("received schedule=%v", schedule)
			}
			now := time.Now()
			if now.Before(schedule.Time) {
				t.Errorf("schedule received too early. now is %v; want %v", now, schedule.Time)
			}
			if now.After(schedule.Time.Add(maxDelay)) {
				t.Errorf("schedule delayed too much. now is %v; want %v", now, schedule.Time)
			}
			tv := timeAndValues[indexes[i]]
			if !schedule.Time.Equal(tv.t) {
				t.Errorf("schedule time unmatch got %v; want %v", schedule.Time, tv.t)
			}
			if schedule.Data != tv.v {
				t.Errorf("schedule data unmatch got %v; want %v", schedule.Data, tv.v)
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

	now := time.Now()
	timeAndValues := []struct {
		t  time.Time
		v  string
		id int64
	}{
		{t: now.Add(1 * time.Second).Truncate(time.Second), v: "foo"},
		{t: now.Add(3 * time.Second).Truncate(time.Second), v: "baz"},
		{t: now.Add(5 * time.Second).Truncate(time.Second), v: "huga"},
		{t: now.Add(2 * time.Second).Truncate(time.Second), v: "bar"},
		{t: now.Add(5 * time.Second).Truncate(time.Second), v: "hoge"},
	}
	for i, tv := range timeAndValues {
		id := s.Schedule(Item{Time: tv.t, Data: tv.v})
		timeAndValues[i].id = id
	}

	canceled := s.Cancel(timeAndValues[2].id)
	if !canceled {
		t.Errorf("cancel failed")
	}

	indexes := []int{0, 3, 1, 4}
	i := 0
	for {
		select {
		case schedule := <-s.C:
			if verbose {
				logger.Printf("received schedule=%v", schedule)
			}
			now := time.Now()
			if now.Before(schedule.Time) {
				t.Errorf("schedule received too early. now is %v; want %v", now, schedule.Time)
			}
			if now.After(schedule.Time.Add(maxDelay)) {
				t.Errorf("schedule delayed too much. now is %v; want %v", now, schedule.Time)
			}
			tv := timeAndValues[indexes[i]]
			if !schedule.Time.Equal(tv.t) {
				t.Errorf("schedule time unmatch got %v; want %v", schedule.Time, tv.t)
			}
			if schedule.Data != tv.v {
				t.Errorf("schedule data unmatch got %v; want %v", schedule.Data, tv.v)
			}

			i++
			if i == len(timeAndValues)-1 {
				cancel()
				return
			}
		}
	}
}

func TestCancelFirst(t *testing.T) {
	verbose := testing.Verbose()
	var logger *log.Logger
	if verbose {
		logger = log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := NewScheduler(ctx)

	now := time.Now()
	timeAndValues := []struct {
		t  time.Time
		v  string
		id int64
	}{
		{t: now.Add(1 * time.Second).Truncate(time.Second), v: "foo"},
		{t: now.Add(3 * time.Second).Truncate(time.Second), v: "baz"},
		{t: now.Add(5 * time.Second).Truncate(time.Second), v: "huga"},
		{t: now.Add(2 * time.Second).Truncate(time.Second), v: "bar"},
		{t: now.Add(4 * time.Second).Truncate(time.Second), v: "hoge"},
	}
	for i, tv := range timeAndValues {
		id := s.Schedule(Item{Time: tv.t, Data: tv.v})
		timeAndValues[i].id = id
	}

	canceled := s.Cancel(timeAndValues[0].id)
	if !canceled {
		t.Errorf("cancel failed")
	}

	indexes := []int{3, 1, 4, 2}
	i := 0
	for {
		select {
		case schedule := <-s.C:
			if verbose {
				logger.Printf("received schedule=%v", schedule)
			}
			now := time.Now()
			if now.Before(schedule.Time) {
				t.Errorf("schedule received too early. now is %v; want %v", now, schedule.Time)
			}
			if now.After(schedule.Time.Add(maxDelay)) {
				t.Errorf("schedule delayed too much. now is %v; want %v", now, schedule.Time)
			}
			tv := timeAndValues[indexes[i]]
			if !schedule.Time.Equal(tv.t) {
				t.Errorf("schedule time unmatch got %v; want %v", schedule.Time, tv.t)
			}
			if schedule.Data != tv.v {
				t.Errorf("schedule data unmatch got %v; want %v", schedule.Data, tv.v)
			}

			i++
			if i == len(timeAndValues)-1 {
				cancel()
				return
			}
		}
	}
}
