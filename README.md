# 💥 `boomo`

`boomo` is a **boo**trapper **mo**nitor. It will probe a list of preconfigured
peers at a constant frequency with different transports.

In each round, it loops through the configured transports and peers and does two things. It

1. establishes a connection (using a single transport)
2. issues a `FIND_NODE` RPC (if the connection was successful).

By default, TCP, QUIC, Websocket, and WebTransport are configured.

The process exposes two prometheus metrics that will indicate the
availability of probed peers:

1. A counter that tracks the performed probes

   ```text
   boomo_probes_total{peer="QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",success="true",type="CONNECT",transport="tcp"} 1
   boomo_probes_total{peer="QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",success="true",type="FIND_NODE",transport="tcp"} 1
   boomo_probes_total{peer="QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",success="true",type="CONNECT",transport="tcp"} 1
   boomo_probes_total{peer="QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",success="true",type="FIND_NODE",transport="tcp"} 1
   ```
   
   `CONNECT` counts the established connections to a peer by transport/success. `FIND_NODE` counts the number of `FIND_NODE` RPCs to a peer by transport/success.

2. A gauge indicating if a probed peer is considered online by transport:

   ```text
   boomo_up{peer="QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",transport="quic"} 1
   boomo_up{peer="QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",transport="tcp"} 0
   boomo_up{peer="QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",transport="ws"} 0
   boomo_up{peer="QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",transport="wt"} 0
   ```

Beware of the metric cardinality when configuring many peers/more transports to probe.

Before and after a peer is probed `boomo` will actively disconnect from a peer and
remove the peer's addresses from the peerstore.

**Note:** the [`msg_sender.go`](./msg_sender.go) file was just copied from [`go-libp2p-kad-dht`](https://github.com/libp2p/go-libp2p-kad-dht/blob/master/internal/net/message_manager.go). It is an `internal` package there, so it cannot be imported straight away. 

## Defaults

By default `boomo` will

1. Probe the bootstrap peers of the Amino DHT:

    ```text
    /dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN
    /dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa
    /dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb
    /dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt
    /ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ
    ```
   
2. Use the `/ipfs/kad/1.0.0` protocol ID
3. Probe with the TCP, QUIC, Websocket and WebTransport transports
3. Run a round of probes every `5 minutes`
4. Expose prometheus metrics on port `3232`

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
   --peers value [ --peers value ]            peer multiaddresses (network address + peer ID) (default: IPFS bootstrappers) [$BOOMO_PEERS]
   --protocol value                           the libp2p protocol for the DHT (default: "/ipfs/kad/1.0.0") [$BOOMO_PROTOCOL]
   --metrics-host value                       the network musa metrics should bind on (default: "127.0.0.1") [$BOOMO_METRICS_HOST]
   --metrics-port value                       the port on which musa metrics should listen on (default: 3232) [$BOOMO_METRICS_PORT]
   --probe-interval value                     probe interval (default: 5m0s) [$BOOMO_PROBE_INTERVAL]
   --transports value [ --transports value ]  the transports to probe [$BOOMO_TRANSPORTS]
   --help, -h                                 show help
```