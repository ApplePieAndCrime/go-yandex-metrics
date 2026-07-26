package pool

import "testing"

type reusable struct {
	value  int
	resets int
}

func (r *reusable) Reset() {
	r.value = 0
	r.resets++
}

func TestPool(t *testing.T) {
	created := 0
	p := New(func() *reusable {
		created++
		return &reusable{}
	})

	first := p.Get()
	if created != 1 {
		t.Fatalf("factory called %d times, want 1", created)
	}
	first.value = 42
	p.Put(first)

	if first.value != 0 {
		t.Fatalf("Put did not reset value: got %d", first.value)
	}
	if first.resets != 1 {
		t.Fatalf("Put called Reset %d times, want 1", first.resets)
	}

	if got := p.Get(); got.value != 0 {
		t.Fatalf("Get returned an object with value %d, want 0", got.value)
	}
}
