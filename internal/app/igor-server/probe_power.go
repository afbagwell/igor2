package igorserver

import (
	"strings"
	"time"
)

type PowerProbe struct {
	Timeout    time.Duration // overall timeout for the probeHosts call
	MaxWorkers int
}

func NewPowerProbe() IHostProbe {
	return &PowerProbe{
		Timeout: 2 * time.Second,
	}
}

func (c *PowerProbe) probeHosts(hosts []Host) {

	// Build hostname list
	hostNames := hostNamesOfHosts(hosts)

	// assume this ipmitool command:
	// ipmitool -I lanplus -H %v.ipmi -U admin -P admin power status

	logger.Debug().Msgf("running IPMI status on hosts")
	outputMap, err := runAllCapture(igor.ExternalCmds.PowerStatus, hostNames, c.Timeout)
	if err != nil {
		logger.Error().Msgf("error running host power status commands: %v", err)
	}

	powerList := make([]string, 0, len(hostNames))
	noPowerList := make([]string, 0, len(hostNames))

	localPowerMap := make(map[string]HostStatus, len(hostNames))
	for k, v := range outputMap {
		w := strings.ReplaceAll(v, "\n", "; ")
		w = strings.ToLower(w)
		code := HostStatusOff
		if strings.Contains(w, "fail") || strings.Contains(w, "error") {
			logger.Debug().Msgf("probe power: node %v power status returned fail or error - %v", k, w)
			code = HostStatusUnknown
		} else if strings.Contains(w, PowerOn) {
			code = HostStatusOn
			powerList = append(powerList, k)
		} else if strings.Contains(w, PowerOff) {
			code = HostStatusOff
			noPowerList = append(noPowerList, k)
		} else {
			logger.Debug().Msgf("probe power: node %v no power status returned - %v", k, w)
			code = HostStatusUnknown
		}
		localPowerMap[k] = code
	}

	logger.Debug().Msgf("probe power: ON [%v]; OFF [%v]", strings.Join(powerList, ","), strings.Join(noPowerList, ","))

	// immediately update the power map if node has no power
	hostStatusMapMU.Lock()

	for k, v := range localPowerMap {
		if v == HostStatusOff {
			hostStatusMap[k] = v
		}
		tmpStatusMap[k] = v // update here to completely overwrite the tmpStatusMap with off/on values
	}
	hostStatusMapMU.Unlock()
}
