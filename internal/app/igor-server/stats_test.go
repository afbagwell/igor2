// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"net/http"
	"strings"
	"testing"
)

// TestRunStatsRejectsBadDuration covers BUG-002. The rejection message for a bad
// stats duration is built with Sprintf and then used to construct the returned
// error. When that pre-built message was passed to fmt.Errorf as a format string,
// any '%' in the client-supplied value was re-interpreted as a verb and both the
// log line and the API response came back mangled.
//
// Only the rejection paths are exercised here; a valid duration continues on to a
// database transaction, which this test deliberately avoids.
func TestRunStatsRejectsBadDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration string
	}{
		{name: "trailing percent", duration: "50%"},
		{name: "embedded verb", duration: "10%d"},
		{name: "bare percent", duration: "%"},
		{name: "non-numeric", duration: "seven"},
		{name: "negative", duration: "-5"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, status, err := runStats(map[string][]string{"duration": {tc.duration}})

			if err == nil {
				t.Fatalf("runStats(duration=%q) returned no error, want a rejection", tc.duration)
			}
			if status != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", status, http.StatusBadRequest)
			}

			msg := err.Error()
			if strings.Contains(msg, "%!") {
				t.Errorf("error message was format-mangled: %q", msg)
			}
			if !strings.Contains(msg, tc.duration) {
				t.Errorf("error message %q does not report the offending value %q", msg, tc.duration)
			}
		})
	}
}
