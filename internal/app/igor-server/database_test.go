// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// assertDbAccessFree fails the test if the global write mutex is still held, and leaves
// it unlocked either way so later tests are unaffected.
func assertDbAccessFree(t *testing.T, msg string) {
	t.Helper()
	if !dbAccess.TryLock() {
		t.Fatal(msg)
	}
	dbAccess.Unlock()
}

func TestLockedDbWriteReleasesOnNormalReturn(t *testing.T) {
	ran := false
	lockedDbWrite(func() { ran = true })

	assert.True(t, ran, "the function must actually run")
	assertDbAccessFree(t, "dbAccess still held after a normal return")
}

// The reason this helper exists. handleCreateReservations used to unlock with a bare
// statement after the call, which a panic unwound straight past -- leaving the global
// write mutex held by a goroutine that no longer existed. httprouter recovers the panic,
// so the server stayed up serving reads while every write blocked forever.
func TestLockedDbWriteReleasesOnPanic(t *testing.T) {
	func() {
		defer func() {
			assert.NotNil(t, recover(), "the panic must propagate to the caller")
		}()
		lockedDbWrite(func() {
			panic("simulated panic inside the locked region")
		})
	}()

	assertDbAccessFree(t, "dbAccess leaked: a panic inside the locked region did not release it")
}

// A leaked mutex is only observable through its effect on the next writer, so check that
// a second call can still acquire it after the first one panicked.
func TestLockedDbWriteUsableAfterPanic(t *testing.T) {
	func() {
		defer func() { _ = recover() }()
		lockedDbWrite(func() { panic("first call panics") })
	}()

	done := make(chan struct{})
	go func() {
		lockedDbWrite(func() { close(done) })
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a later write could not acquire dbAccess; the mutex was leaked")
	}
}
