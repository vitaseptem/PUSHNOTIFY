package queue

import (
	"testing"
	"time"
)

func TestCalculateBackoffGrows(t *testing.T) {
	b1 := CalculateBackoff(1, 30)
	b2 := CalculateBackoff(2, 30)
	b3 := CalculateBackoff(3, 30)
	if !(b1 < b2 && b2 < b3) {
		t.Fatalf("backoff should grow: %v %v %v", b1, b2, b3)
	}
	if b1 < 30*time.Second {
		t.Errorf("attempt 1 backoff too small: %v", b1)
	}
}

func TestCalculateBackoffCap(t *testing.T) {
	if got := CalculateBackoff(20, 30); got > time.Hour {
		t.Errorf("backoff exceeded cap: %v", got)
	}
}

func TestShouldRetry(t *testing.T) {
	if !ShouldRetry(QueueJob{Attempt: 1, MaxAttempts: 3}) {
		t.Error("expected retry allowed")
	}
	if ShouldRetry(QueueJob{Attempt: 3, MaxAttempts: 3}) {
		t.Error("expected retry denied at max")
	}
}

func TestQueueForPriority(t *testing.T) {
	cases := map[int]string{1: KeyQueueHigh, 3: KeyQueueHigh, 5: KeyQueueDefault, 9: KeyQueueLow}
	for prio, want := range cases {
		if got := queueForPriority(prio); got != want {
			t.Errorf("priority %d -> %s, want %s", prio, got, want)
		}
	}
}
