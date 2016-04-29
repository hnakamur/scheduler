package scheduler

import (
	"testing"
	"time"

	"golang.org/x/net/context"
)

func TestScheduler(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	s := NewScheduler(ctx)

	now := time.Now()
	timeAndValues := []struct {
		t time.Time
		v string
	}{
		{t: now.Add(1 * time.Millisecond), v: "foo"},
		{t: now.Add(3 * time.Millisecond), v: "baz"},
		{t: now.Add(2 * time.Millisecond), v: "bar"},
		{t: now.Add(5 * time.Millisecond), v: "hoge"},
	}
	for _, tv := range timeAndValues {
		s.Schedule(Item{Time: tv.t, Data: tv.v})
	}

	indexes := []int{0, 2, 1, 3}
	i := 0
	for {
		select {
		case schedule := <-s.C:
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
