package config

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestConfigLockExcludesAndReleasesAfterCallbackError(t *testing.T) {
	dir := t.TempDir()
	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseFirst) }) }
	t.Cleanup(release)
	firstDone := make(chan error, 1)
	wantErr := errors.New("fixture callback failure")
	go func() {
		firstDone <- withConfigLock(dir, func() error {
			close(firstEntered)
			<-releaseFirst
			return wantErr
		})
	}()
	select {
	case <-firstEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("first callback never acquired the lock")
	}
	secondStarted := make(chan struct{})
	secondEntered := make(chan struct{})
	secondDone := make(chan error, 1)
	go func() {
		close(secondStarted)
		secondDone <- withConfigLock(dir, func() error {
			close(secondEntered)
			return nil
		})
	}()
	<-secondStarted
	select {
	case <-secondEntered:
		t.Error("second callback entered while the first held the lock")
	case <-time.After(100 * time.Millisecond):
	}
	release()
	select {
	case err := <-firstDone:
		if !errors.Is(err, wantErr) {
			t.Fatalf("callback error = %v, want %v", err, wantErr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("first callback did not finish")
	}
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("lock was not reacquired after callback failure: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("second callback could not acquire the released lock")
	}
}

func TestConfigLockFailsBeforeCallbackOnInvalidDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "file-instead-of-directory")
	if err := os.WriteFile(dir, []byte("preserve fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	called := false
	err := withConfigLock(dir, func() error { called = true; return nil })
	if err == nil || called {
		t.Fatalf("invalid lock directory: err=%v, callback ran=%v", err, called)
	}
	if data, err := os.ReadFile(dir); err != nil || string(data) != "preserve fixture" {
		t.Fatalf("lock failure changed fixture: %q, %v", data, err)
	}
}
