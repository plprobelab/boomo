package main

import (
	"encoding/json"
	"sort"
	"time"

	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slog"
)

var cfg = &Config{
	BootstrapPeers: cli.NewStringSlice(),
	ProbeInterval:  5 * time.Minute,
	ProtocolID:     string(kaddht.ProtocolDHT),
	MetricsHost:    "127.0.0.1",
	MetricsPort:    3232,
	transports:     cli.NewStringSlice(),
}

type Config struct {
	ProbeInterval  time.Duration
	ProtocolID     string
	BootstrapPeers *cli.StringSlice
	MetricsHost    string
	MetricsPort    int
	transports     *cli.StringSlice
}

func (c *Config) String() string {
	data, _ := json.MarshalIndent(c, "", "  ")
	return string(data)
}

func (c *Config) BootstrapAddrInfos() ([]peer.AddrInfo, error) {
	if len(c.BootstrapPeers.Value()) == 0 && c.ProtocolID == string(kaddht.ProtocolDHT) {
		return kaddht.GetDefaultBootstrapPeerAddrInfos(), nil
	}

	bootstrappers := make([]peer.AddrInfo, len(c.BootstrapPeers.Value()))
	for i, bp := range cfg.BootstrapPeers.Value() {
		addrInfo, err := peer.AddrInfoFromString(bp)
		if err != nil {
			slog.Error("failed parsing addr info from string", "addrInfo", bp)
			return nil, err
		}
		bootstrappers[i] = *addrInfo
	}
	return bootstrappers, nil
}

func (c *Config) Transports() []string {
	var transports []string
	if c.transports == nil || len(c.transports.Value()) == 0 {
		transports = []string{"tcp", "quic", "ws", "wt"}
	} else {
		transports = c.transports.Value()
	}

	sort.Strings(transports)

	return transports
}
