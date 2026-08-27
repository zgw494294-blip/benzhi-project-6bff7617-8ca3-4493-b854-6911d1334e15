package persistence

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEventLogAssignsAndVerifiesDigestChain(t *testing.T) {
	store := New(t.TempDir())
	if err := store.AppendEvent(Event{Type: "project.created", Payload: map[string]any{"projectID": "p"}}); err != nil {
		t.Fatal(err)
	}
	if err := store.AppendEvent(Event{Type: "segment.added", Payload: map[string]any{"segmentID": "s"}}); err != nil {
		t.Fatal(err)
	}
	events, err := store.Events()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Sequence != 1 || events[1].Sequence != 2 {
		t.Fatalf("事件序号错误: %+v", events)
	}
	if events[0].Digest == "" || events[1].PreviousDigest != events[0].Digest {
		t.Fatalf("事件摘要链错误: %+v", events)
	}
}

func TestEventLogRejectsTampering(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	if err := store.AppendEvent(Event{Type: "project.created", Payload: "p"}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "events.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := range data {
		if data[i] == 'p' {
			data[i] = 'x'
			break
		}
	}
	if err = os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Events(); err == nil {
		t.Fatal("摘要不匹配的事件日志必须被拒绝")
	}
}
