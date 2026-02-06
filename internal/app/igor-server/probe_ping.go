package igorserver

import (
	"context"
	"errors"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type PingProbe struct {
	Timeout    time.Duration // overall timeout for the probeHosts call
	MaxWorkers int           // max concurrent ping workers (each with its own socket)
}

func NewPingProbe() IHostProbe {
	return &PingProbe{
		Timeout:    2 * time.Second,
		MaxWorkers: 64, // sensible default; tune in config if needed
	}
}

func (p *PingProbe) probeHosts(hosts []Host) {

	// this is called with a list of hosts already known to have power
	if len(hosts) == 0 {
		// if this is true, no nodes in the cluster are currently powered on
		logger.Info().Msgf("no nodes requested for a ping check - all statuses should be current")
		return
	}

	logger.Debug().Msgf("probe ping: beginning ping testing on nodes without tcp response")

	hostNames := hostNamesOfHosts(hosts)

	// Clamp worker count
	workers := p.MaxWorkers
	if workers <= 0 {
		workers = 64
	}
	if workers > len(hostNames) {
		workers = len(hostNames)
	}

	type job struct {
		name string
		ip   string
	}
	type result struct {
		name string
		ok   bool
	}

	// Prepare jobsChan (resolve from hostIpMap)
	jobsChan := make(chan job, len(hostNames))
	resultsChan := make(chan result, len(hostNames))

	for _, name := range hostNames {
		ip := hostIpMap[name]
		if ip == "" {
			logger.Warn().Msgf("ping probe: no IP for %s; skipping", name)
			continue
		}
		jobsChan <- job{name: name, ip: ip}
	}
	close(jobsChan)

	var pingWG sync.WaitGroup
	pingWG.Add(workers)

	// Worker: one ICMP socket per worker, processes many hosts
	for i := 0; i < workers; i++ {
		go func() {
			defer pingWG.Done()

			// Try raw ICMP first
			pc, err := icmp.ListenPacket("ip4:icmp", "")
			if err != nil {
				// If raw ICMP not permitted, try UDP "ping" fallback (Linux allows icmp in udp4)
				pc, err = icmp.ListenPacket("udp4", "")
			}
			if err != nil {
				// If we cannot create ANY socket, drain the jobsChan so the probe doesn’t block
				logger.Warn().Msgf("ping probe: cannot open ICMP socket in worker: %v", err)
				for range jobsChan {
					// report no result; leaving status as-is
				}
				return
			}
			defer pc.Close()

			// Unique ID per worker; seq increases per host attempt
			// Using lower 16 bits as required by Echo
			id := os.Getpid() & 0xffff
			seq := 1

			for j := range jobsChan {
				ctx, cancel := context.WithTimeout(context.Background(), p.Timeout)
				ok := pingOnce(ctx, pc, id, seq, j.ip, j.name)
				cancel()
				seq++

				resultsChan <- result{name: j.name, ok: ok}
			}
		}()
	}

	// Closer for resultsChan
	go func() {
		pingWG.Wait()
		close(resultsChan)
	}()

	pingList := make([]string, 0, len(hosts))
	noPingList := make([]string, 0, len(hosts))

	// Gather successes and promote those hosts to Pingable
	localPingMap := make(map[string]HostStatus, len(hostNames))
	for r := range resultsChan {
		if r.ok {
			localPingMap[r.name] = HostStatusPingable
			pingList = append(pingList, r.name)
		} else {
			noPingList = append(noPingList, r.name)
		}
	}

	logger.Debug().Msgf("probe ping: PING [%v]; NO PING RESP [%v]", strings.Join(pingList, ","), strings.Join(noPingList, ","))

	// Swap into globals under lock (overwriting HostStatusOn)
	hostStatusMapMU.Lock()
	for k, v := range localPingMap {
		hostStatusMap[k] = v
		tmpStatusMap[k] = v
	}
	hostStatusMapMU.Unlock()
}

// pingOnce sends a single ICMP Echo and waits for the matching echo reply
// from the target IP, within ctx deadline, using the provided packetConn.
func pingOnce(ctx context.Context, pc *icmp.PacketConn, id, seq int, ipStr string, hostname string) bool {
	dst := &net.IPAddr{IP: net.ParseIP(ipStr)}
	if dst.IP == nil {
		return false
	}

	// Determine expected echo ID based on socket mode.
	// In udp fallback, replies use the UDP local port as the Echo ID.
	effectiveID := id
	if ua, ok := pc.LocalAddr().(*net.UDPAddr); ok {
		effectiveID = ua.Port & 0xffff
	}

	// Build echo request
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho, Code: 0,
		Body: &icmp.Echo{ID: effectiveID, Seq: seq, Data: []byte("igor-ping|" + hostname)},
	}
	b, err := msg.Marshal(nil)
	if err != nil {
		logger.Debug().Msgf("ping marshal err ip=%s id=%d seq=%d err=%v", ipStr, effectiveID, seq, err)
		return false
	}

	// Set per-attempt deadlines from ctx (used by ReadFrom timeout)
	deadline, has := ctx.Deadline()
	if has {
		_ = pc.SetDeadline(deadline)
	} else {
		_ = pc.SetDeadline(time.Now().Add(2 * time.Second))
	}

	// Choose correct destination address for raw vs udp fallback
	var dstIP net.Addr
	if _, ok := pc.LocalAddr().(*net.UDPAddr); ok || pc.LocalAddr().Network() == "udp" || pc.LocalAddr().Network() == "udp4" {
		dstIP = &net.UDPAddr{IP: dst.IP}
	} else {
		dstIP = &net.IPAddr{IP: dst.IP}
	}

	// Write
	if _, err = pc.WriteTo(b, dstIP); err != nil {
		logger.Debug().Msgf("ping write err ip=%s id=%d seq=%d dst=%T(%s) err=%v",
			ipStr, effectiveID, seq, dstIP, dstIP.String(), err)
		return false
	}

	// Read until deadline, looking for the matching Echo Reply (ID+Seq) from dst.
	reply := make([]byte, 1500)

	peerIP := func(a net.Addr) net.IP {
		switch v := a.(type) {
		case *net.IPAddr:
			return v.IP
		case *net.UDPAddr:
			return v.IP
		default:
			return nil
		}
	}

	for {
		n, peer, err := pc.ReadFrom(reply)
		if err != nil {
			// Deadline exceeded (or other error)
			var nErr net.Error
			if errors.As(err, &nErr) && nErr.Timeout() {
				return false
			}
			logger.Debug().Msgf("ping read err from %s: %v", ipStr, err)
			return false
		}

		// Ignore packets from other hosts (shared socket per worker processes many hosts)
		if pip := peerIP(peer); pip != nil && !pip.Equal(dst.IP) {
			continue
		}

		rm, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), reply[:n])
		if err != nil {
			// Malformed/unexpected packet; keep waiting
			continue
		}

		if rm.Type != ipv4.ICMPTypeEchoReply {
			continue
		}

		body, ok := rm.Body.(*icmp.Echo)
		if !ok {
			continue
		}

		if body.ID == effectiveID && body.Seq == seq {
			return true
		}
		// Echo reply but not our ID/Seq; keep waiting until deadline.
	}
}
