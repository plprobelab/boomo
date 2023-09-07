# `boomo`

`boomo` is a **boo**trapper **mo**nitor. It will probe a list of preconfigured
peers in a constant interval.

In each round it loops through all peers and

1. establishes a connection
2. issues a `FIND_NODE` RPC

The process exposes a single prometheus metric that will indicate the
availability of the configured peers:

```text
probes_total{otel_scope_name="github.com/plprobelab/boomo",otel_scope_version="",peer="QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",success="true",type="CONNECT"} 1
probes_total{otel_scope_name="github.com/plprobelab/boomo",otel_scope_version="",peer="QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",success="true",type="FIND_NODE"} 1
probes_total{otel_scope_name="github.com/plprobelab/boomo",otel_scope_version="",peer="QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",success="true",type="CONNECT"} 1
probes_total{otel_scope_name="github.com/plprobelab/boomo",otel_scope_version="",peer="QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",success="true",type="FIND_NODE"} 1
```

Beware of the metric cardinality when configuring many peers to probe.

Before and after a peer is probed `boomo` will actively disconnect from it and
remove the peer's addresses from the peerstore. The intention here is that this
will trigger another DNS lookup in case the configured peers use `/dnsaddr/`
multiaddresses.

**Note:** the [`msg_sender.go`](./msg_sender.go) file was just copied from [`go-libp2p-kad-dht`](https://github.com/libp2p/go-libp2p-kad-dht/blob/master/internal/net/message_manager.go). It is an `internal` package there, so it cannot be imported straight away. 

## Defaults

By default `boomo` will

1. probe the bootstrap peers of the Amino DHT:

    ```text
    /dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN
    /dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa
    /dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb
    /dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt
    /ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ
    ```
2. Use the `/ipfs/kad/1.0.0` protocol ID
3. Run a round of probes every `5 minutes`
4. Expose the prometheus metrics on port `3232`

## Run

You can run `boomo` locally without much effort. Just run the following command in this repository:

```shell
go run *.go
```

Then you can check the prometheus metrics with:

```shell
curl localhost:3232/metrics | grep boomo
```

## Configuration

```text
NAME:
   boomo - a bootstrapper monitor

USAGE:
   boomo [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --peers value [ --peers value ]  peer multiaddresses (network address + peer ID) (default: IPFS bootstrappers) [$BOOMO_PEERS]
   --protocol value                 the libp2p protocol for the DHT (default: "/ipfs/kad/1.0.0") [$BOOMO_PROTOCOL]
   --metrics-host value             the network musa metrics should bind on (default: "127.0.0.1") [$BOOMO_METRICS_HOST]
   --metrics-port value             the port on which musa metrics should listen on (default: 3232) [$BOOMO_METRICS_PORT]
   --probe-interval value           probe interval (default: 5m0s) [$BOOMO_PROBE_INTERVAL]
   --help, -h                       show help

```