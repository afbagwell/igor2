// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"sync"
	"time"
)

var (
	// hostStatusMap stores the authoritative network status of each cluster node (off,on,pingable,up)
	hostStatusMap   map[string]HostStatus
	tmpStatusMap    map[string]HostStatus
	hostIpMap       map[string]string
	hostStatusMapMU sync.Mutex
)

// IHostProbe is an interface that provides methods to fetch network status information about
// cluster nodes.
type IHostProbe interface {
	// probeHosts gathers network status information about the slice of hosts provided and
	// updates hostStatusMap with the results.
	probeHosts(hosts []Host)
}

// hostProbeManager is called as a go routine and calls its suite of IHostProbe routines
// more frequently when clients are active and less frequently otherwise.
func hostProbeManager(hosts []Host) {
	defer wg.Done()

	logger.Info().Msg("starting host probe manager")

	var hostList = hosts

	createIPProbeMap(hostList)

	startup := 10 * time.Millisecond
	timeoutFast := 3 * time.Second
	timeoutSlow := 10 * time.Second // during no user activity, reduce call frequency
	timeout := timeoutFast
	fastRefreshes := 20
	countdown := time.NewTimer(startup)
	hostNames := hostNamesOfHosts(hostList)
	hostStatusMap = make(map[string]HostStatus, len(hostNames))
	for _, h := range hostNames {
		hostStatusMap[h] = HostStatusUnknown
	}

	for {
		select {
		case <-shutdownChan:
			logger.Info().Msg("stopping host probe manager")
			if !countdown.Stop() {
				<-countdown.C
			}
			return
		case <-clusterUpdateChan:
			hostList, _ = dbReadHostsTx(map[string]interface{}{})
			createIPProbeMap(hostList)
			hostNames = hostNamesOfHosts(hostList)
			newHostStatusMap := make(map[string]HostStatus, len(hostNames))
			for _, h := range hostNames {
				newHostStatusMap[h] = HostStatusUnknown
				if prevStatus, exists := hostStatusMap[h]; exists {
					newHostStatusMap[h] = prevStatus
				}
			}
			hostStatusMap = newHostStatusMap
		case <-refreshStatusChan:
			if fastRefreshes == 0 {
				if !countdown.Stop() {
					<-countdown.C
				}
				// when user activity starts after slow period, refresh immediately
				countdown.Reset(startup)
			}
			fastRefreshes = 20
		case <-countdown.C:
			if fastRefreshes == 0 {
				logger.Debug().Msg("slow host probe update")
				timeout = timeoutSlow
			} else {
				logger.Debug().Msgf("fast host probe update - countdown %d", fastRefreshes)
				timeout = timeoutFast
				fastRefreshes--
			}

			// at the top of each countdown check, restart with a fresh tmpStatusMap
			tmpStatusMap = make(map[string]HostStatus, len(hostStatusMap))

			if DEVMODE {
				// cheating a little here until we can figure out how to test in a pure dev environment without
				// a cluster.
				igor.PortProbe.probeHosts(hostList)
			} else {
				// check and set hosts that have no power first
				igor.PowerProbe.probeHosts(hostList)
				// for hosts with power, check for TCP alive
				igor.PortProbe.probeHosts(getPoweredHosts(hostList))
				// for hosts with power but not TCP alive, check with ping
				igor.PingProbe.probeHosts(getPoweredHosts(hostList))

				// finally, update the authoritative map with everything that is left with HostStatusOn
				hostStatusMapMU.Lock()
				for k, v := range tmpStatusMap {
					if v == HostStatusOn {
						hostStatusMap[k] = v
					}
				}
				hostStatusMapMU.Unlock()
			}

			countdown.Reset(timeout)
		}
	}
}

func createIPProbeMap(hosts []Host) {
	hostIpMap = make(map[string]string, len(hosts))
	for _, h := range hosts {
		ip := h.IP
		if ip == "" {
			logger.Warn().Msgf("no IP found for host %s; skipping", h.HostName)
			continue
		}
		hostIpMap[h.HostName] = ip
	}
	logger.Debug().Msgf("%v", hostIpMap)
}

// getPoweredHosts reads the tmpStatusMap to find only those
// hosts reporting a status of HostStatusOn.
func getPoweredHosts(hosts []Host) []Host {
	poweredHosts := make([]Host, 0, len(hosts))
	for _, h := range hosts {
		if tmpStatusMap[h.HostName] == HostStatusOn {
			poweredHosts = append(poweredHosts, h)
		}
	}
	return poweredHosts
}
