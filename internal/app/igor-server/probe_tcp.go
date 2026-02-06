// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

//go:build !DEVMODE

package igorserver

import (
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TcpProbe implements IHostProbe.
// It probes a small list of TCP ports concurrently and marks hosts as up
// if connect succeeds or the remote actively refuses (RST).
type TcpProbe struct {
	Ports      []string      // ports to try, e.g. {"22","443"}; stops on first positive
	Timeout    time.Duration // overall timeout for the probeHosts call
	PerDial    time.Duration // per-dial timeout
	MaxWorkers int
}

// NewTcpProbe returns a configured TCP prober. If there is no
// list of probe ports defined in the server config file, default to
// scanning only port 22.
func NewTcpProbe() IHostProbe {

	var probePorts []string

	if len(igor.Server.ProbePorts) > 0 {
		probePorts = make([]string, len(igor.Server.ProbePorts))
		for i, v := range igor.Server.ProbePorts {
			probePorts[i] = strconv.FormatUint(uint64(v), 10) // Convert uint to string
		}
	} else {
		probePorts = []string{"22"} // If no ports configured, fallback to 22
	}

	logger.Info().Msgf("ports used for host status probes = %v", probePorts)

	return &TcpProbe{
		Ports:      probePorts,
		Timeout:    3 * time.Second,
		PerDial:    1 * time.Second,
		MaxWorkers: 64,
	}
}

// dialAlive returns true if a TCP dial indicates the host is reachable.
// Treats connection success OR a "connection refused" as alive.
func dialAlive(ctx context.Context, network, addr string, timeout time.Duration) bool {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, network, addr)
	if err == nil {
		_ = conn.Close()
		return true
	}

	// some systems return wrapped net.OpError with "connection refused"
	// we'll check the string; it's pragmatic and matches common Go errors.
	// Treat "connection refused" as proof the stack is up (RST).
	if strings.Contains(err.Error(), "connection refused") {
		return true
	}
	return false
}

// AliveForIP tries configured ports for the given ip and returns true on first positive.
func (c *TcpProbe) AliveForIP(ctx context.Context, ip string) bool {

	for _, p := range c.Ports {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		addr := net.JoinHostPort(ip, p)
		if dialAlive(ctx, "tcp", addr, c.PerDial) {
			return true
		}
	}
	return false
}

// probeHosts implements IHostProbe.
func (c *TcpProbe) probeHosts(hosts []Host) {
	if len(hosts) == 0 {
		logger.Warn().Msg("TcpProbe.probeHosts called with empty hosts")
		return
	}

	logger.Debug().Msgf("probe tcp: beginning probe for alive tcp ports")

	// Build hostname list
	hostNames := hostNamesOfHosts(hosts)

	// create a per-call context and cancel
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	// Clamp worker count
	workers := c.MaxWorkers
	if workers <= 0 {
		workers = 64
	}
	if workers > len(hostNames) {
		workers = len(hostNames)
	}

	probeJobChan := make(chan string, len(hostNames))
	type result struct {
		name string // node name
		up   bool   // is TCP alive
	}
	resultChan := make(chan result, len(hostNames))

	var tcpWG sync.WaitGroup
	for i := 0; i < workers; i++ {
		tcpWG.Add(1)
		go func() {
			defer tcpWG.Done()
			for name := range probeJobChan {
				ip := hostIpMap[name]
				if ip == "" {
					// unknown IP => nil
					resultChan <- result{name: name, up: false}
					continue
				}

				alive := c.AliveForIP(ctx, ip)
				resultChan <- result{name: name, up: alive}
			}
		}()
	}

	// enqueue
	for _, h := range hostNames {
		probeJobChan <- h
	}
	close(probeJobChan)

	// collect in goroutine to close resultChan when workers done
	go func() {
		tcpWG.Wait()
		close(resultChan)
	}()

	upList := make([]string, 0, len(hostNames))
	noRespList := make([]string, 0, len(hostNames))

	// Build localTcpMap map using only hosts that have active TCP ports
	localTcpMap := make(map[string]HostStatus, len(hostNames))
	for r := range resultChan {
		if r.up {
			localTcpMap[r.name] = HostStatusUp
			upList = append(upList, r.name)
		} else {
			noRespList = append(noRespList, r.name)
		}
	}

	logger.Debug().Msgf("probe tcp: UP [%v]; NO TCP RESP [%v]", strings.Join(upList, ","), strings.Join(noRespList, ","))

	// Swap into global hostStatusMap under lock
	hostStatusMapMU.Lock()

	for k, v := range localTcpMap {
		hostStatusMap[k] = v
		tmpStatusMap[k] = v // update here as well to overwrite any HostStatusOn entries
	}
	hostStatusMapMU.Unlock()
}
