# net-perf

net-perf is a WireGuard mesh network orchestrator and monitoring tool, designed to optimise the performance of the mesh networks by continously monitoring network performance and intelligently manage and select optimal network paths throughout the whole mesh network.

net-perf is currently in active development and is still being tested extensively.

## Architecture Overview

A full-mesh WireGuard mesh network is constructed, where each node has a direct tunnel to another node. Where dual-stack internet endpoints are available, two interfaces are created for each node, one for IPv4 and one for IPv6. Each site operates as a single private ASN, and routing information is shared via BGP over BIRD. Based on the performance of each network path, the routing is then optimised to provide the best network performance possible.

The net-perf service is responsible for monitoring the network performance of all network paths, and then dynamically optimises the routing to provide the best network performance possible.

For generation of the WireGuard configuration files, you can use the following project [wg-mesh-dynamic-v2](https://github.com/DrC0ns0le/wg-mesh-dynamic-v2).

## Features

- Continuous network performance monitoring
  - Per site and per interface monitoring workers orchestration
  - Round-trip time(RTT) measurements of each tunnel endpoint via ICMP and TCP
  - Packet loss measurements with custom UDP-based bandwidth tests
  - Measurement success rate monitoring
  - Custom pathping implementation for true path latency measurements
  - Metrics exposed via Prometheus exposition format via :5120/metrics

- Network path optimisation
  - Integration with BIRD daemon for BGP route management
  - Cost calculation based on network latency, packet loss and availability metrics from prometheus
  - Automatic tunnel endpoint selection(IPv4 or IPv6)
  - Multiple routing algorithms, including:
    - Shortest BGP Path: Traditional shortest-path selection
    - Least Cost BGP Path: Performance-weighted path selection
    - Graph-based Least Cost Path: Topology-aware routing
    - Measured Fastest Path: Real-time performance-based topology-aware routing

- BGP integration
  - BIRD support for BGP route management
  - BGP route table update watchdog
  - Custom routing protocol label(201) for managed routes

- Management interface
  - gRPC-based inter-node communication, for management and measurement orchestration
  - Unix socket for IPC signaling(ie. interface up/down signals)

- Distributed & highly available architecture
  - Automatic node service discovery by parsing /etc/wireguard/*.conf files
  - Failover if unable to query metrics
  - Diverse network path selection algorithms as fallback, balancing decentralisation and complex coordination among nodes, ensuring optimal path selection and resilience

- Network ochestration & management
  - Kernel parameters tuning
  - BIRD route table update watchdog
  - Route table aligment watchdog
 
## To-Do

- add to-do list (theres alot)

## Pre-requisites

- Linux with WireGuard support
- BIRD(for BGP route management)
- Prometheus(for metrics, VictoriaMetrics recommended)
- Grafana(for visualisation)

net-perf makes use of the following assumptions of the network:

### IPv4 Addressing

Site network format: 10.201.X.0/24 where X is the site ID
Gateway addresses: 10.201.X.1 where X is the site ID
Site node addresses:
- IPv4 endpoints: 10.201.X.4
- IPv6 endpoints: 10.201.X.6

### IPv6 Addressing
Site network format: fdac:c9:X::/64 where X is the site ID in hex
Link-local addresses are not used for WireGuard tunnels

### Interface Naming

WireGuard Interfaces
Format: wgX.Y_vZ where:

- X: Local site ID (source)
- Y: Remote site ID (destination)
- Z: IP version (4 or 6)

Examples:
- wg0.1_v4: IPv4 tunnel from site 0 to site 1
- wg1.2_v6: IPv6 tunnel from site 1 to site 2

### BGP Configuration
Private AS range: 64512-65534

Site AS number: 64512 + site ID

Example: Site 0 uses AS64512, Site 1 uses AS64513

## Default
Management RPC: 5122 (default)
HTTP Metrics: 5120
Bandwidth Testing: 5121
PathPing Service: 5124

Socket Paths
- Unix control socket: /opt/wg-mesh/net-perf.sock
- BIRD IPv4 socket: /run/bird/bird.ctl
- BIRD IPv6 socket: /run/bird/bird6.ctl

## Usage

### Build & Run

```bash
make tidy
make build
```

perfc = net-perf client for all nodes route table overview

perfe = net-perf elasticsearch "exporter"


## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- WireGuard® is a registered trademark of Jason A. Donenfeld
- Thanks to the WireGuard and Go communities
- Part of my University Final Year Project
