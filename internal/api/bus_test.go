package api

import (
	"sync"
	"testing"
)

func TestBus_SubscribeReceivesNotify(t *testing.T) {
	b := NewBus()
	ch := b.Subscribe("ns", "svc", "key")
	b.Notify("ns", "svc", "key")
	select {
	case <-ch:
	default:
		t.Fatal("expected notification on channel, got none")
	}
}

func TestBus_NotifyNoSubscribers(t *testing.T) {
	b := NewBus()
	b.Notify("ns", "svc", "key") // must not panic
}

func TestBus_NotifyDifferentKeyDoesNotWake(t *testing.T) {
	b := NewBus()
	ch := b.Subscribe("ns", "svc", "key")
	b.Notify("ns", "svc", "other")
	select {
	case <-ch:
		t.Fatal("unexpected notification for different key")
	default:
	}
}

func TestBus_NotifyCoalesces(t *testing.T) {
	b := NewBus()
	ch := b.Subscribe("ns", "svc", "key")
	b.Notify("ns", "svc", "key")
	b.Notify("ns", "svc", "key")
	b.Notify("ns", "svc", "key")
	// Only one signal should be pending (buffered capacity 1).
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count != 1 {
		t.Errorf("expected 1 coalesced signal, got %d", count)
	}
}

func TestBus_MultipleSubscribers(t *testing.T) {
	b := NewBus()
	ch1 := b.Subscribe("ns", "svc", "key")
	ch2 := b.Subscribe("ns", "svc", "key")
	b.Notify("ns", "svc", "key")
	for i, ch := range []<-chan struct{}{ch1, ch2} {
		select {
		case <-ch:
		default:
			t.Errorf("subscriber %d: expected notification, got none", i)
		}
	}
}

func TestBus_Unsubscribe(t *testing.T) {
	b := NewBus()
	ch := b.Subscribe("ns", "svc", "key")
	b.Unsubscribe("ns", "svc", "key", ch)
	b.Notify("ns", "svc", "key")
	select {
	case <-ch:
		t.Fatal("expected no notification after unsubscribe")
	default:
	}
}

func TestBus_UnsubscribeNonExistentSafe(t *testing.T) {
	b := NewBus()
	ch := make(<-chan struct{})
	b.Unsubscribe("ns", "svc", "key", ch) // must not panic
}

func TestBus_UnsubscribeOneOfMany(t *testing.T) {
	b := NewBus()
	ch1 := b.Subscribe("ns", "svc", "key")
	ch2 := b.Subscribe("ns", "svc", "key")
	b.Unsubscribe("ns", "svc", "key", ch1)
	b.Notify("ns", "svc", "key")
	select {
	case <-ch1:
		t.Fatal("ch1 should have been unsubscribed")
	default:
	}
	select {
	case <-ch2:
	default:
		t.Fatal("ch2 should still receive notification")
	}
}

func TestBus_UnsubscribeRemovesKeyWhenEmpty(t *testing.T) {
	b := NewBus()
	ch := b.Subscribe("ns", "svc", "key")
	b.Unsubscribe("ns", "svc", "key", ch)
	b.mu.RLock()
	_, exists := b.subs[busKey("ns", "svc", "key")]
	b.mu.RUnlock()
	if exists {
		t.Error("expected key to be removed from subs map after last unsubscribe")
	}
}

func TestBus_KeySeparatorPreventsCollision(t *testing.T) {
	b := NewBus()
	// "a/b" + "c" vs "a" + "b/c" should produce different bus keys.
	ch1 := b.Subscribe("a/b", "x", "c")
	ch2 := b.Subscribe("a", "x", "b/c")
	b.Notify("a/b", "x", "c")
	select {
	case <-ch1:
	default:
		t.Fatal("ch1 should receive notification")
	}
	select {
	case <-ch2:
		t.Fatal("ch2 should not receive notification for a different key")
	default:
	}
}

func TestBus_Concurrent(t *testing.T) {
	b := NewBus()
	const n = 20
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch := b.Subscribe("ns", "svc", "key")
			b.Notify("ns", "svc", "key")
			<-ch
			b.Unsubscribe("ns", "svc", "key", ch)
		}()
	}
	wg.Wait()
}
