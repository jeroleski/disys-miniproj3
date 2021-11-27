package timer

import (
	"fmt"
	"sync"
	"time"
)

type Timer struct {
	Time      time.Duration
	Await     time.Duration
	Read      map[string](chan time.Duration)
	IsTicking bool
	Mu        sync.Mutex
}

func (timer *Timer) Tick() {
	for range time.Tick(timer.Await) {
		timer.Mu.Lock()

		timer.NotifyAll()

		timer.Time -= timer.Await

		timer.Mu.Unlock()

		if timer.TimesUp() {
			break
		}
	}
}

func (timer *Timer) NotifyAll() {
	t := timer.Time
	for _, c := range timer.Read {
		go func() {
			c <- t
		}()
	}
}

func Send(t time.Duration, c chan time.Duration) {
	fmt.Println("sending time")
	c <- t
	fmt.Println("time has been send")
}

func (timer *Timer) GetChannel(user string) chan time.Duration {
	timer.Mu.Lock()
	defer timer.Mu.Unlock()

	if !timer.IsTicking {
		timer.IsTicking = true
		go timer.Tick()
	}

	if timer.Read[user] == nil {
		timer.Read[user] = make(chan time.Duration)
	}

	return timer.Read[user]
}

func (timer *Timer) TimesUp() bool {
	timer.Mu.Lock()
	defer timer.Mu.Unlock()

	return timer.Time <= 0
}
