package repository

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestJSONStoreAtomicWritePreservesOriginalOnMarshalFailure(t *testing.T) {
	store := NewJSONStore(t.TempDir())
	if err := store.Write(context.Background(), func(tx *JSONTx) error {
		return tx.Save("config.json", map[string]string{"value": "original"})
	}); err != nil {
		t.Fatal(err)
	}

	err := store.Write(context.Background(), func(tx *JSONTx) error {
		return tx.Save("config.json", make(chan int))
	})
	if err == nil {
		t.Fatal("expected marshal failure")
	}

	var result map[string]string
	if err := store.Read(context.Background(), func(tx *JSONTx) error {
		return tx.Load("config.json", &result)
	}); err != nil {
		t.Fatal(err)
	}
	if result["value"] != "original" {
		t.Fatalf("original data was not preserved: %#v", result)
	}
	matches, err := filepath.Glob(filepath.Join(store.dir, ".config.json.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files were not cleaned up: %v", matches)
	}
}

func TestJSONStoreLockTimeout(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, ".tracker-data.lock")
	release, err := tryFileLock(lockPath, true)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	store := NewJSONStore(dir)
	store.lockTimeout = 50 * time.Millisecond
	started := time.Now()
	err = store.Read(context.Background(), func(tx *JSONTx) error { return nil })
	if !errors.Is(err, ErrDataBusy) {
		t.Fatalf("expected ErrDataBusy, got %v", err)
	}
	if elapsed := time.Since(started); elapsed < 40*time.Millisecond || elapsed > time.Second {
		t.Fatalf("unexpected lock wait duration: %v", elapsed)
	}
}

func TestJSONStoreLockWaitHonorsCancellation(t *testing.T) {
	dir := t.TempDir()
	release, err := tryFileLock(filepath.Join(dir, ".tracker-data.lock"), true)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	store := NewJSONStore(dir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	err = store.Write(ctx, func(tx *JSONTx) error { return nil })
	if !errors.Is(err, ErrDataBusy) {
		t.Fatalf("expected ErrDataBusy, got %v", err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("canceled lock wait did not stop promptly")
	}
}

func TestAtomicWritesAreAlwaysReadable(t *testing.T) {
	store := NewJSONStore(t.TempDir())
	type payload struct {
		Version int      `json:"version"`
		Values  []string `json:"values"`
	}
	initial := payload{Version: 0, Values: []string{"initial"}}
	if err := store.Write(context.Background(), func(tx *JSONTx) error {
		return tx.Save("errors.json", initial)
	}); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 1; i <= 100; i++ {
			value := payload{Version: i, Values: make([]string, 256)}
			for j := range value.Values {
				value.Values[j] = "complete-value"
			}
			if err := store.Write(context.Background(), func(tx *JSONTx) error {
				return tx.Save("errors.json", value)
			}); err != nil {
				errs <- err
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			var value payload
			if err := store.Read(context.Background(), func(tx *JSONTx) error {
				return tx.Load("errors.json", &value)
			}); err != nil {
				errs <- err
				return
			}
		}
	}()
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}
