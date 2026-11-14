package cache

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func decodeTestJSON(data []byte) (map[string]string, error) {
	var value map[string]string
	err := json.Unmarshal(data, &value)
	return value, err
}

func TestLoadRecoversCorruptCacheAndRejectsBadResponses(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())
	if err := Write("k", []byte("partial JSON")); err != nil {
		t.Fatal(err)
	}
	calls := 0
	loader := func() ([]byte, error) {
		calls++
		if calls == 1 {
			return []byte("invalid upstream JSON"), nil
		}
		return []byte(`{"value":"recovered"}`), nil
	}
	if _, err := Load("k", time.Hour, loader, decodeTestJSON); err == nil {
		t.Fatal("invalid response accepted")
	}
	if _, ok := Read("k", 0); ok {
		t.Fatal("invalid response was cached")
	}
	for i := 0; i < 2; i++ {
		value, err := Load("k", time.Hour, loader, decodeTestJSON)
		if err != nil || value["value"] != "recovered" {
			t.Fatalf("recovery: %v, %v", value, err)
		}
	}
	if calls != 2 {
		t.Fatalf("requests=%d, want 2", calls)
	}
}

func TestLoadKeepsResponseWhenCacheCannotBeWritten(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("alfred_workflow_cache", filepath.Join(file, "cache"))
	value, err := Load("k", time.Hour, func() ([]byte, error) { return []byte(`{"value":"success"}`), nil }, decodeTestJSON)
	if err != nil || value["value"] != "success" {
		t.Fatalf("successful response lost: %v, %v", value, err)
	}
	want := errors.New("offline")
	_, err = Load("k", time.Hour, func() ([]byte, error) { return nil, want }, decodeTestJSON)
	if !errors.Is(err, want) {
		t.Fatalf("loader error lost: %v", err)
	}
}

func TestAtomicWriteNeverExposesPartialData(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())
	a, b := bytes.Repeat([]byte("a"), 32768), bytes.Repeat([]byte("b"), 16384)
	if err := Write("k", a); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	failures := make(chan error, 1)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				data, ok := Read("k", time.Hour)
				if !ok || (!bytes.Equal(data, a) && !bytes.Equal(data, b)) {
					select {
					case failures <- fmt.Errorf("partial read: hit=%v size=%d", ok, len(data)):
					default:
					}
					return
				}
			}
		}()
	}
	for i := 0; i < 50; i++ {
		data := a
		if i%2 == 0 {
			data = b
		}
		if err := Write("k", data); err != nil {
			t.Error(err)
			break
		}
	}
	close(stop)
	wg.Wait()
	select {
	case err := <-failures:
		t.Fatal(err)
	default:
	}
	entries, err := os.ReadDir(dir())
	if err != nil || len(entries) != 1 {
		t.Fatalf("temporary files leaked: %v, %v", entries, err)
	}
}

func TestFailedRenameCleansTemporaryFile(t *testing.T) {
	t.Setenv("alfred_workflow_cache", t.TempDir())
	if err := os.Mkdir(path("k"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := Write("k", []byte("data")); err == nil {
		t.Fatal("expected rename failure")
	}
	files, err := os.ReadDir(dir())
	if err != nil || len(files) != 1 {
		t.Fatalf("temporary file leaked: %v, %v", files, err)
	}
}
