package internal

import (
	"fmt"
	"net"
	"time"

	"github.com/go-ping/ping"
)

func pingIP(ip net.IP) (*ping.Statistics, error) {
	pinger, err := ping.NewPinger(ip.String())
	if err != nil {
		return nil, fmt.Errorf("unable to create pinger for %s: %w", ip.String(), err)
	}
	pinger.SetPrivileged(true)
	pinger.Count = 3
	pinger.Interval = 500 * time.Millisecond
	pinger.Timeout = 10 * time.Second
	err = pinger.Run()
	if err != nil {
		return nil, fmt.Errorf("unable to ping %s", ip.String())
	}
	return pinger.Statistics(), nil
}
