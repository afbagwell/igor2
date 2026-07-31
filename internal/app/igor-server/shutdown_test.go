// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// A request that never completes must not be able to block shutdown indefinitely. This is
// what made every wedge recoverable only by SIGKILL, which destroyed the goroutine state
// that would have identified the cause.
func TestShutdownIsBoundedByAStuckRequest(t *testing.T) {
	stall := make(chan struct{})
	defer close(stall)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-stall // simulates a handler parked on dbAccess or an unbounded external call
		}),
	}
	go func() { _ = srv.Serve(ln) }()

	// get one request in flight and confirm the handler has been entered
	reqStarted := make(chan struct{})
	go func() {
		close(reqStarted)
		c := &http.Client{Timeout: 30 * time.Second}
		resp, rErr := c.Get("http://" + ln.Addr().String())
		if resp != nil {
			_ = resp.Body.Close()
		}
		_ = rErr
	}()
	<-reqStarted
	time.Sleep(200 * time.Millisecond)

	// Shutdown with an unbounded context would never return here. Bounded, it must give up.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- srv.Shutdown(ctx) }()

	select {
	case sErr := <-done:
		assert.True(t, errors.Is(sErr, context.DeadlineExceeded),
			"a stuck request should make Shutdown hit its deadline, got: %v", sErr)
	case <-time.After(10 * time.Second):
		t.Fatal("Shutdown never returned; it is not bounded")
	}

	assert.NoError(t, srv.Close(), "force close must succeed after the deadline")
}

// waitForWorkers must return even when a background manager is parked and never reaches
// its shutdownChan select -- otherwise the database session is never closed.
func TestWaitForWorkersGivesUpOnAStuckWorker(t *testing.T) {
	prev := shutdownGrace
	shutdownGrace = 500 * time.Millisecond
	defer func() { shutdownGrace = prev }()

	release := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-release // never signalled until after the assertion below
	}()
	defer close(release)

	start := time.Now()
	done := make(chan struct{})
	go func() {
		waitForWorkers()
		close(done)
	}()

	select {
	case <-done:
		assert.GreaterOrEqual(t, time.Since(start), shutdownGrace,
			"waitForWorkers returned early; it should have waited out its grace period")
	case <-time.After(shutdownGrace + 10*time.Second):
		t.Fatal("waitForWorkers never returned; shutdown is still unbounded")
	}
}
