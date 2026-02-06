package igorserver

const (
	HostStatusUnknown = HostStatus(iota)
	HostStatusOff
	HostStatusOn
	HostStatusPingable
	HostStatusUp
)

// HostStatus is an enum value describing a node's combined current network and power state.
//
//	  1 = off         ; the node has no power according to IPMI
//	  2 = on          ; the node is powered on according to IPMI but doesn't respond to ICMP or TCP requests
//	  3 = pingable    ; the node is powered on and responds to ICMP but not TCP requests.
//	  4 = up          ; the node is powered on and TCP response is alive or RST on at least one port (default 22)
//
//		 Notes:
//
//		 The on(2) status is normal if the node is still in the process of booting but network services are not available yet. Prolonged time in this status would indicate a problem with boot or with internal or external network configuration issues.
//
//		 The pingable(3) status is generally an indicator that the node is visible on the network but something is preventing TCP connections. Since ping is only invoked if TCP connections fail, this status is generally regarded as a warning state that the node has a bad configuration or there may be an external issue like a misconfigured firewall setting.
type HostStatus int
