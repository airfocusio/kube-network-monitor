package internal

import (
	"net"
	"testing"
	"time"
)

func TestPingIPOnce(t *testing.T) {
	stats, err := pingIPOnce(net.ParseIP("127.0.0.1"))
	if err != nil {
		t.Fatal(err)
	}
	if stats.AvgRtt == time.Duration(0) {
		t.Fatal("avg round trip time must be non-zero")
	}
}
