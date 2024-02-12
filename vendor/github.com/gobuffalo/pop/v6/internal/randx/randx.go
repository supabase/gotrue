package randx

import (
	"math/rand"
	"sync"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type safeSrc struct {
	src  rand.Source
	moot sync.Mutex
}

func (s *safeSrc) Int63() int64 {
	s.moot.Lock()
	n := s.src.Int63()
	s.moot.Unlock()
	return n
}

func newSafeSrc(s rand.Source) *safeSrc {
	return &safeSrc{
		src:  s,
		moot: sync.Mutex{},
	}
}
