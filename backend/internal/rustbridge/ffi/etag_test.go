package ffi

import "testing"

func TestBuildETagFromBytes_IsDeterministic(t *testing.T) {
	first := BuildETagFromBytes([]byte("sub2api"))
	second := BuildETagFromBytes([]byte("sub2api"))
	if first == "" {
		t.Fatal("expected non-empty etag")
	}
	if first != second {
		t.Fatalf("expected deterministic etag, got %q and %q", first, second)
	}
}

func TestBuildETagFromAny_HandlesMarshalFailure(t *testing.T) {
	if got := BuildETagFromAny(make(chan int)); got != "" {
		t.Fatalf("expected empty etag for unmarshalable payload, got %q", got)
	}
}
