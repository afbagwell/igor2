// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/kballard/go-shellquote"
)

// cmdTargetRE matches a valid DNS hostname per RFC 1123:
// - Consists of labels separated by dots
// - Each label 1–63 chars: [A–Z a–z 0–9 -]
// - Labels cannot start or end with a hyphen
// - Total length ≤ 253 chars
var cmdTargetRE = regexp.MustCompile(
	`^(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)(?:\.(?i:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?))*$`,
)

// processWrapper executes the given arg list and returns a combined
// stdout/stderr and any errors. processWrapper blocks until the process exits.
func processWrapper(ctx context.Context, timeout time.Duration, argv ...string) (string, error) {
	if len(argv) == 0 {
		return "", fmt.Errorf("processWrapper: empty argument list")
	}

	logger.Trace().Msgf("running %v", argv)

	// create a child context with timeout if requested
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		// still derive a cancellable ctx to keep symmetry (caller cancels if they want)
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	out, err := cmd.CombinedOutput()
	stop := time.Now()

	logger.Trace().Msgf("cmd %v completed in %v", argv[0], stop.Sub(start))

	// If the context timed out/canceled, exec.CommandContext returns a context error.
	// Log the output and the error (if any).
	if err != nil {
		// Note: this will include an error if the process exited with non-zero status,
		// or if the context was canceled/timed out.
		logger.Warn().Msgf("problem running '%v': %v - %v", argv, err, strings.ReplaceAll(string(out), "\n", "; "))
	}

	return string(out), err
}

// parseTemplate builds argv from a single free-form command string.
//  1. Split command using shell-like parsing (respects quotes/escapes).
//  2. Replace placeholders inside each token (%s, %v, or {target}) with the target.
//     Since we replace *inside the token*, it remains one argv element (e.g. "-H", "kn67.ipmi").
//  3. Validate target to avoid argument smuggling/injection.
func parseTemplate(cmdTemplate string, target string) ([]string, error) {
	if !cmdTargetRE.MatchString(target) {
		return nil, fmt.Errorf("invalid target string %q", target)
	}
	tokens, err := shellquote.Split(cmdTemplate)
	if err != nil {
		return nil, fmt.Errorf("invalid command template: %w", err)
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty command template")
	}
	for i, tok := range tokens {
		// Replace any of these placeholder styles
		// (keep %s/%v for convenience; {target} is explicit and safer)
		if strings.Contains(tok, "%s") || strings.Contains(tok, "%v") || strings.Contains(tok, "{target}") {
			tok = strings.ReplaceAll(tok, "%s", target)
			tok = strings.ReplaceAll(tok, "%v", target)
			tok = strings.ReplaceAll(tok, "{target}", target)
			tokens[i] = tok
		}
	}
	return tokens, nil
}

// runAll runs the command using an argument template against every target in targets.
//
//   - cmdTemplate; each template may include a single %s, %v or {target} which will be replaced with the individual
//     target strings. For example:
//     "ipmitool -I lanplus -H {target}.ipmi -U admin -P **** chassis power on"
//     results in arguments where {target} is replaced by a hostname.
//   - targets is the list of strings to pass into templates, for example a list of hostnames in the example.
//   - timeout is the per-command timeout. Pass 0 for no timeout.
//
// This function uses DefaultRunner to run the commands in parallel with the
// configured concurrency and retry policy.
func runAll(cmdTemplate string, targets []string, timeout time.Duration) error {
	r := DefaultRunner(func(target string) error {
		argv, err := parseTemplate(cmdTemplate, target)
		if err != nil {
			return err
		}
		// Background ctx here; wire a ctx through if you want whole-batch cancel.
		ctx := context.Background()
		_, err = processWrapper(ctx, timeout, argv...)
		return err
	})
	return r.RunAll(targets)
}

func runAllCapture(cmdTemplate string, targets []string, timeout time.Duration) (map[string]string, error) {
	var (
		mu          sync.Mutex
		outByTarget = make(map[string]string, len(targets))
	)
	r := DefaultRunner(func(target string) error {
		argv, err := parseTemplate(cmdTemplate, target)
		if err != nil {
			return err
		}
		ctx := context.Background()
		out, err := processWrapper(ctx, timeout, argv...)

		mu.Lock()
		outByTarget[target] = out
		mu.Unlock()

		return err
	})

	return outByTarget, r.RunAll(targets)
}
