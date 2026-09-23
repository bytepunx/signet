package api

import (
	"fmt"
	"sync"
)

// Bus is an in-process pub/sub for WatchSecret and WatchServiceConfig stream
// notifications. When a secret or config is written (via GitOps sync), the
// writer calls Notify/NotifyService and all active Watch subscribers are woken
// to re-fetch the value.
type Bus struct {
	mu   sync.RWMutex
	subs map[string][]chan struct{}
}

// NewBus creates an empty Bus.
func NewBus() *Bus {
	return &Bus{subs: make(map[string][]chan struct{})}
}

// Subscribe returns a buffered channel that receives a signal whenever the named
// secret changes. The caller must call Unsubscribe when done to avoid a leak.
// The channel capacity of 1 ensures Notify never blocks — a second notification
// before the first is consumed simply coalesces.
func (b *Bus) Subscribe(namespace, service, name string) <-chan struct{} {
	ch := make(chan struct{}, 1)
	key := busKey(namespace, service, name)
	b.mu.Lock()
	b.subs[key] = append(b.subs[key], ch)
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes the subscription for ch. Safe to call even if ch has
// already been drained or was never signalled.
func (b *Bus) Unsubscribe(namespace, service, name string, ch <-chan struct{}) {
	key := busKey(namespace, service, name)
	b.mu.Lock()
	defer b.mu.Unlock()
	chans := b.subs[key]
	for i, c := range chans {
		if c == ch {
			last := len(chans) - 1
			chans[i] = chans[last]
			chans[last] = nil
			b.subs[key] = chans[:last]
			if len(b.subs[key]) == 0 {
				delete(b.subs, key)
			}
			return
		}
	}
}

// Notify signals all subscribers watching (namespace, service, name). Each
// subscriber receives at most one pending signal regardless of how many times
// Notify is called before the subscriber drains the channel.
func (b *Bus) Notify(namespace, service, name string) {
	key := busKey(namespace, service, name)
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs[key] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// SubscribeService returns a buffered channel that receives a signal whenever
// any configuration for (namespace, service) changes. The caller must call
// UnsubscribeService when done.
func (b *Bus) SubscribeService(namespace, service string) <-chan struct{} {
	ch := make(chan struct{}, 1)
	key := serviceKey(namespace, service)
	b.mu.Lock()
	b.subs[key] = append(b.subs[key], ch)
	b.mu.Unlock()
	return ch
}

// UnsubscribeService removes a service-level config subscription.
func (b *Bus) UnsubscribeService(namespace, service string, ch <-chan struct{}) {
	key := serviceKey(namespace, service)
	b.mu.Lock()
	defer b.mu.Unlock()
	chans := b.subs[key]
	for i, c := range chans {
		if c == ch {
			last := len(chans) - 1
			chans[i] = chans[last]
			chans[last] = nil
			b.subs[key] = chans[:last]
			if len(b.subs[key]) == 0 {
				delete(b.subs, key)
			}
			return
		}
	}
}

// NotifyService signals all subscribers watching (namespace, service) config.
func (b *Bus) NotifyService(namespace, service string) {
	key := serviceKey(namespace, service)
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs[key] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// busKey uses null bytes as separators to prevent collision between
// ("a/b", "c") and ("a", "b/c") style inputs.
func busKey(namespace, service, name string) string {
	return fmt.Sprintf("%s\x00%s\x00%s", namespace, service, name)
}

// SubscribeBundle returns a buffered channel that receives a signal whenever
// any secret OR config for (namespace, service) changes. Used by WatchServiceBundle.
// The caller must call UnsubscribeBundle when done.
func (b *Bus) SubscribeBundle(namespace, service string) <-chan struct{} {
	ch := make(chan struct{}, 1)
	key := bundleKey(namespace, service)
	b.mu.Lock()
	b.subs[key] = append(b.subs[key], ch)
	b.mu.Unlock()
	return ch
}

// UnsubscribeBundle removes a bundle-level subscription.
func (b *Bus) UnsubscribeBundle(namespace, service string, ch <-chan struct{}) {
	key := bundleKey(namespace, service)
	b.mu.Lock()
	defer b.mu.Unlock()
	chans := b.subs[key]
	for i, c := range chans {
		if c == ch {
			last := len(chans) - 1
			chans[i] = chans[last]
			chans[last] = nil
			b.subs[key] = chans[:last]
			if len(b.subs[key]) == 0 {
				delete(b.subs, key)
			}
			return
		}
	}
}

// NotifyBundle signals all WatchServiceBundle subscribers for (namespace, service).
// Called on any secret write, secret delete, config write, or config delete.
func (b *Bus) NotifyBundle(namespace, service string) {
	key := bundleKey(namespace, service)
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs[key] {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// serviceKey uses a leading \x01 byte to ensure it cannot collide with busKey
// (which always starts with the namespace string, never \x01).
func serviceKey(namespace, service string) string {
	return fmt.Sprintf("\x01%s\x00%s", namespace, service)
}

// bundleKey uses a leading \x02 byte to distinguish bundle subscriptions from
// serviceKey (\x01) and busKey (no prefix).
func bundleKey(namespace, service string) string {
	return fmt.Sprintf("\x02%s\x00%s", namespace, service)
}
