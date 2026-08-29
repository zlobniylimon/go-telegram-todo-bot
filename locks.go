package main

import "sync"

var (
	chatLocksMu sync.Mutex
	chatLocks   = make(map[string]*sync.Mutex)
)

func lockChat(key string) func() {
	chatLocksMu.Lock()
	m, ok := chatLocks[key]
	if !ok {
		m = &sync.Mutex{}
		chatLocks[key] = m
	}
	chatLocksMu.Unlock()
	m.Lock()
	return m.Unlock
}
