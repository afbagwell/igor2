// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

//go:build DEVMODE

package igorserver

var _doInitOn = true

type TcpProbe struct{}

func NewTcpProbe() IHostProbe {
	return &TcpProbe{}
}

func (nr *TcpProbe) probeHosts(hosts []Host) {

	if len(hosts) == 0 {
		logger.Warn().Msg("no hosts provided on call to probeHosts")
		return
	}

	if _doInitOn {
		initToOn(hosts)
		return
	}
}

func initToOn(hosts []Host) {

	_doInitOn = false

	hostNames := hostNamesOfHosts(hosts)
	rHosts, _ := getReservedHosts()

	hostStatusMapMU.Lock()
	for _, h := range hostNames {
		powerVal := HostStatusOff
		for _, rh := range rHosts {
			if h == rh.HostName && rh.State > HostAvailable {
				powerVal = HostStatusUp
				break
			}
		}
		hostStatusMap[h] = powerVal
	}
	hostStatusMapMU.Unlock()
}
