package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	dht_pb "github.com/libp2p/go-libp2p-kad-dht/pb"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	webtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	metricsdk "go.opentelemetry.io/otel/sdk/metric"
	"golang.org/x/exp/slog"
)

func main() {
	app := &cli.App{
		Name:   "boomo",
		Usage:  "a bootstrapper monitor",
		Action: rootAction,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:        "peers",
				DefaultText: "IPFS bootstrappers",
				Usage:       "peer multiaddresses (network address + peer ID)",
				Value:       cfg.BootstrapPeers,
				Destination: cfg.BootstrapPeers,
				EnvVars:     []string{"BOOMO_PEERS"},
			},
			&cli.StringFlag{
				Name:        "protocol",
				Usage:       "the libp2p protocol for the DHT",
				Value:       cfg.ProtocolID,
				Destination: &cfg.ProtocolID,
				EnvVars:     []string{"BOOMO_PROTOCOL"},
			},
			&cli.StringFlag{
				Name:        "metrics-host",
				Usage:       "the network musa metrics should bind on",
				Value:       cfg.MetricsHost,
				Destination: &cfg.MetricsHost,
				EnvVars:     []string{"BOOMO_METRICS_HOST"},
			},
			&cli.IntFlag{
				Name:        "metrics-port",
				Usage:       "the port on which musa metrics should listen on",
				Value:       cfg.MetricsPort,
				Destination: &cfg.MetricsPort,
				EnvVars:     []string{"BOOMO_METRICS_PORT"},
			},
			&cli.DurationFlag{
				Name:        "probe-interval",
				Usage:       "probe interval",
				Value:       cfg.ProbeInterval,
				Destination: &cfg.ProbeInterval,
				EnvVars:     []string{"BOOMO_PROBE_INTERVAL"},
			},
			&cli.StringSliceFlag{
				Name:        "transports",
				Usage:       "the transports to probe",
				Value:       cfg.transports,
				Destination: cfg.transports,
				EnvVars:     []string{"BOOMO_TRANSPORTS"},
			},
		},
	}

	sigs := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	go func() {
		sig := <-sigs
		slog.Info("Received signal - Stopping...", "signal", sig.String())
		signal.Stop(sigs)
		cancel()
	}()

	if err := app.RunContext(ctx, os.Args); err != nil {
		slog.Error("application error", "err", err)
		os.Exit(1)
	}
}

func rootAction(c *cli.Context) error {
	slog.Info("Starting to monitor bootstrappers with configuration:")
	fmt.Println(cfg.String())

	bootstrappers, err := cfg.BootstrapAddrInfos()
	if err != nil {
		return fmt.Errorf("parse peer strings: %w", err)
	}

	hosts, err := initHosts()
	if err != nil {
		return fmt.Errorf("init hosts: %w", err)
	}

	// Initializing OpenTelemetry - omg is that clunky.
	exporter, err := prometheus.New()
	if err != nil {
		return fmt.Errorf("new prometheus exporter: :%w", err)
	}

	meterProvider := metricsdk.NewMeterProvider(metricsdk.WithReader(exporter))
	meter := meterProvider.Meter("github.com/plprobelab/boomo")

	probeIns, err := meter.Int64Counter("boomo_probes")
	if err != nil {
		return fmt.Errorf("boomo_probes meter: %w", err)
	}

	// up state keeps state about whether a peer is up for a certain protocol.
	var upStateMu sync.RWMutex
	upState := map[string]map[peer.ID]int64{}
	for _, t := range cfg.Transports() {
		for _, p := range bootstrappers {
			if _, found := upState[t]; !found {
				upState[t] = map[peer.ID]int64{}
			}
			upState[t][p.ID] = 0
		}
	}

	_, err = meter.Int64ObservableGauge("boomo_up", metric.WithInt64Callback(func(ctx context.Context, o metric.Int64Observer) error {
		upStateMu.RLock()
		defer upStateMu.RUnlock()
		for trpt, peers := range upState {
			for p, up := range peers {
				o.Observe(up, metric.WithAttributes(
					attribute.String("peer", p.String()),
					attribute.String("transport", trpt),
				))
			}
		}
		return nil
	}))
	if err != nil {
		return fmt.Errorf("boomo_up meter: %w", err)
	}

	go serveMetrics()

	for {
		slog.Info("Sleeping until next probe...", "duration", cfg.ProbeInterval)
		select {
		case <-c.Context.Done():
			return c.Context.Err()
		case <-time.After(cfg.ProbeInterval):
		}

		for _, ref := range hosts {
			for _, addrInfo := range bootstrappers {
				select {
				case <-c.Context.Done():
					return c.Context.Err()
				default:
				}

				slogEntry := slog.With("peer", addrInfo.ID.String(), "transport", ref.trpt)

				if err := forgetPeer(ref.host, addrInfo.ID); err != nil {
					slogEntry.Warn("failed forgetting peer", "err", err.Error())
					continue
				}

				slogEntry.Info("Connecting")
				err := ref.host.Connect(c.Context, addrInfo)
				mattrs := metric.WithAttributes(
					attribute.Bool("success", err == nil),
					attribute.String("peer", addrInfo.ID.String()),
					attribute.String("type", "CONNECT"),
					attribute.String("transport", ref.trpt),
				)
				probeIns.Add(c.Context, 1, mattrs)

				upStateMu.Lock()
				if err == nil {
					upState[ref.trpt][addrInfo.ID] = 1
				} else {
					upState[ref.trpt][addrInfo.ID] = 0
				}
				upStateMu.Unlock()

				if err != nil {
					slogEntry.Warn("failed connecting to peer", "err", err.Error())
					continue
				}

				slogEntry.Info("Getting closer peers")
				closer, err := ref.pm.GetClosestPeers(c.Context, addrInfo.ID, ref.host.ID())
				mattrs = metric.WithAttributes(
					attribute.Bool("success", err == nil && len(closer) != 0),
					attribute.String("peer", addrInfo.ID.String()),
					attribute.String("type", "FIND_NODE"),
					attribute.String("transport", ref.trpt),
				)
				probeIns.Add(c.Context, 1, mattrs)
				if err != nil {
					slogEntry.Warn("failed getting closer peers", "err", err.Error())
				}

				slogEntry.Info("Disconnecting from peer")
				if err := forgetPeer(ref.host, addrInfo.ID); err != nil {
					slogEntry.Warn("failed forgetting peer", "err", err.Error())
					continue
				}
			}
		}
	}
}

// hostRef is a struct that captures information about a host that
// was initialized with a specific transport protocol.
type hostRef struct {
	host host.Host
	trpt string
	pm   *dht_pb.ProtocolMessenger
}

func initHosts() (map[string]*hostRef, error) {
	refs := map[string]*hostRef{}
	for _, trpt := range cfg.Transports() {
		var transportOpt libp2p.Option
		switch trpt {
		case "tcp":
			transportOpt = libp2p.Transport(tcp.NewTCPTransport)
		case "quic":
			transportOpt = libp2p.Transport(quic.NewTransport)
		case "ws":
			transportOpt = libp2p.Transport(ws.New)
		case "wt":
			transportOpt = libp2p.Transport(webtransport.New)
		default:
			return nil, fmt.Errorf("unknown transport: %s", trpt)
		}

		h, err := libp2p.New(
			transportOpt,
			libp2p.NoListenAddrs,
		)
		if err != nil {
			return nil, fmt.Errorf("new libp2p host: %w", err)
		}

		pm, err := dht_pb.NewProtocolMessenger(NewMessageSenderImpl(h, []protocol.ID{protocol.ID(cfg.ProtocolID)}))
		if err != nil {
			return nil, err
		}

		slog.Info("Initialized new libp2p host", "peerID", h.ID().String(), "transport", trpt)
		refs[trpt] = &hostRef{
			host: h,
			trpt: trpt,
			pm:   pm,
		}
	}
	return refs, nil
}

func forgetPeer(h host.Host, pid peer.ID) error {
	h.Peerstore().RemovePeer(pid)
	return h.Network().ClosePeer(pid)
}

func serveMetrics() {
	addr := fmt.Sprintf("%s:%d", cfg.MetricsHost, cfg.MetricsPort)
	slog.Info("serving metrics", "endpoint", addr+"/metrics")
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		slog.Warn("error serving metrics", "err", err.Error())
		return
	}
}
