package timer

import (
	"sync"
	"time"
)

type Timer struct {
	Time         time.Duration
	Await        time.Duration
	UserChannels map[string]chan time.Duration
	IsTicking    bool
	Mu           sync.Mutex
	OnTick       func()
	OnClose      func()
}

func (timer *Timer) Tick() {
	for range time.Tick(timer.Await) {
		timer.Mu.Lock()
		timer.Time -= timer.Await
		timer.Mu.Unlock()

		timer.NotifyAll()
		go timer.OnTick()

		if timer.TimesUp() {
			break
		}
	}
	timer.CloseAll()
	timer.OnClose()
}

func (timer *Timer) NotifyAll() {
	timer.Mu.Lock()
	defer timer.Mu.Unlock()

	t := timer.Time
	for _, c := range timer.UserChannels {
		go Notify(c, t)
	}
}

func Notify(c chan time.Duration, t time.Duration) {
	c <- t
}

func (timer *Timer) CloseAll() {
	timer.Mu.Lock()
	defer timer.Mu.Unlock()

	for _, c := range timer.UserChannels {
		select {
		case _ = <-c:
		default:
		}
		close(c)
	}

	timer.UserChannels = make(map[string](chan time.Duration))
}

func (timer *Timer) AddClient(user string) bool {
	timer.Mu.Lock()
	defer timer.Mu.Unlock()

	if timer.UserChannels[user] == nil {
		timer.UserChannels[user] = make(chan time.Duration)
		go Notify(timer.UserChannels[user], timer.Time)
		return true
	}

	return false
}

func (timer *Timer) GetChannel(user string) chan time.Duration {
	timer.Mu.Lock()
	defer timer.Mu.Unlock()

	if !timer.IsTicking {
		timer.IsTicking = true
		go timer.Tick()
	}

	return timer.UserChannels[user]
}

func (timer *Timer) TimesUp() bool {
	timer.Mu.Lock()
	defer timer.Mu.Unlock()

	return timer.Time < 0
}

func (timer *Timer) GetTimeLeft() int64 {
	timer.Mu.Lock()
	defer timer.Mu.Unlock()

	return int64(timer.Time)
}
