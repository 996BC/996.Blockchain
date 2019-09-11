package utils

import "sync"

/*
LoopMode is a working mode that indicates the owner seperates its working logic in one or many long-term running goroutines.
The owner should call StartWorking() in its setup function and Stop() in its cleanup function,
and each of its long-term running goroutine should work like:
```go
func loop() {
	lm.Add()
	defer lm.Done()
	for {
		select {
		case <-lm.D:
			return
		// case ...:
		// do the jobs
		}
	}
}
```
*/
type LoopMode struct {
	working     bool
	routinesNum int
	waitGroup   sync.WaitGroup
	D           chan bool
}

// NewLoop return a LoopMode.Param routines is the number of long-term running go routines(must >0)
func NewLoop(routines int) *LoopMode {
	if routines <= 0 {
		return nil
	}
	return &LoopMode{
		working:     false,
		routinesNum: routines,
		D:           make(chan bool, routines),
	}
}

func (l *LoopMode) StartWorking() {
	l.working = true
}

// Stop stops the long-term running go routines.If it's not working, return false; otherwise return true.
func (l *LoopMode) Stop() bool {
	if !l.working {
		return false
	}

	l.working = false
	for i := 0; i < l.routinesNum; i++ {
		l.D <- true
	}
	l.waitGroup.Wait()
	return true
}

func (l *LoopMode) Add() {
	l.waitGroup.Add(1)
}

func (l *LoopMode) Done() {
	l.waitGroup.Done()
}

func (l *LoopMode) IsWorking() bool {
	return l.working
}
