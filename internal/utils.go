package internal

import (
	"fmt"
	"net"
	"time"

	"github.com/go-ping/ping"
)

func pingIPOnce(ip net.IP) (*ping.Statistics, error) {
	pinger, err := ping.NewPinger(ip.String())
	if err != nil {
		return nil, fmt.Errorf("unable to create pinger for %s: %w", ip.String(), err)
	}
	pinger.Count = 1
	pinger.Timeout = 3 * time.Second
	if err := pinger.Run(); err != nil {
		return nil, fmt.Errorf("unable to ping %s: %w", ip.String(), err)
	}
	return pinger.Statistics(), nil
}
