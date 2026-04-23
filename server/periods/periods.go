package periods

import (
	"errors"
	"sort"
	"time"
)

type nextType int

const (
	start nextType = iota
	stop
	any
)

type Periods struct {
	p [][2]time.Time
}

func (s *Periods) Add(start, stop string) error {
	t1, e := time.Parse("15:04", start)
	if e != nil {
		return e
	}
	var t2 time.Time
	if stop == "00:00" {
		t2, _ = time.Parse(time.TimeOnly, "23:59:59")
	} else {
		t2, e = time.Parse("15:04", stop)
		if e != nil {
			return e
		}
	}
	if t1.Before(t2) {
		s.p = append(s.p, [2]time.Time{t1, t2})
	}
	s.normalize()
	return nil
}

func (s *Periods) normalize() {
	if len(s.p) < 2 {
		return
	}
	sort.SliceStable(s.p, func(i, j int) bool {
		return s.p[i][0].Before(s.p[j][0]) || (s.p[i][0].Compare(s.p[j][0])) == 0 && s.p[i][1].Before(s.p[j][1])
	})
	var p [][2]time.Time
	for i, v := range s.p {
		if i == 0 {
			p = append(p, v)
		} else if v[0].Compare(p[len(p)-1][1]) <= 0 {
			if v[1].After(p[len(p)-1][1]) {
				p[len(p)-1][1] = v[1]
			}
		} else {
			p = append(p, v)
		}
	}
	s.p = p
}

func (s *Periods) In(t time.Time) bool {
	tm, _ := time.Parse(time.TimeOnly, t.Format(time.TimeOnly))
	tm = tm.Add(100 * time.Millisecond)
	for _, v := range s.p {
		if v[0].Before(tm) && v[1].After(tm) {
			return true
		}
	}
	return false
}

func (s *Periods) NowIn() bool {
	return s.In(time.Now())
}

func (s *Periods) NextStart() (time.Time, error) {
	return s.next(start)
}

func (s *Periods) NextStop() (time.Time, error) {
	return s.next(stop)
}

func (s *Periods) NextAny() (time.Time, error) {
	return s.next(any)
}

func (s *Periods) next(t nextType) (time.Time, error) {
	now := time.Now()
	if now.Format(time.TimeOnly) == "23:59:59" {
		now = now.Add(time.Second)
	}

	tm, _ := time.Parse(time.TimeOnly, now.Format(time.TimeOnly))
	var t0, t1 time.Time
	for i := len(s.p) - 1; i >= 0; i-- {
		if s.p[i][0].After(tm) {
			t0 = s.p[i][0]
		}
		if s.p[i][1].After(tm) {
			t1 = s.p[i][1]
		}
	}

	e := errors.New("not found")
	if t0.Equal(time.Time{}) && t1.Equal(time.Time{}) {
		return time.Time{}, e
	}

	switch t {
	case start:
		if t0.Equal(time.Time{}) {
			return time.Time{}, e
		}
	case stop:
		if t1.Equal(time.Time{}) {
			return time.Time{}, e
		}
		t0 = t1
	default:
		if t0.Equal(time.Time{}) || (!t1.Equal(time.Time{}) && t1.Before(t0)) {
			t0 = t1
		}
	}
	return time.Date(now.Year(), now.Month(), now.Day(), t0.Hour(), t0.Minute(), t0.Second(), int(100*time.Millisecond), time.Local), nil
}
