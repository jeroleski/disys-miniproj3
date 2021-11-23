package timer

import (
	"time"
	"sync"
)

type Timer struct {
	Time      time.Duration
	Await     time.Duration
	Read      map[string]bool
	IsTicking bool
	Mu        sync.Mutex
}

func (timer *Timer) Notify() {
	for range time.Tick(timer.Await) {
		timer.Mu.Lock()

		for user := range timer.Read {
			timer.Read[user] = false
		}

		timer.Time -= timer.Await

		timer.Mu.Unlock()

		if timer.TimesUp() {
			break
		}
	}
}

func (timer *Timer) GetTime(user string) (time.Duration, bool) {
	timer.Mu.Lock()
	defer timer.Mu.Unlock()

	if !timer.IsTicking {
		timer.IsTicking = true
		go timer.Notify()
	}

	read := timer.Read[user]
	timer.Read[user] = true

	return timer.Time, read
}

func (timer *Timer) TimesUp() bool {
	timer.Mu.Lock()
	defer timer.Mu.Unlock()

	return timer.Time <= 0
}
