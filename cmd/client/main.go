package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"sort"
	"strconv"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
	pb "github.com/DrC0ns0le/net-perf/pkg/pb/management"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	managementRPCPort = flag.Int("management.rpcport", 5122, "port for management rpc server")

	sites = []int{
		0, 1, 2, 3,
	}

	sitesSet = map[int]bool{}

	routeMap = map[int]map[int]int{}

	logger = logging.NewDefaultLogger()

	nodes = map[int]*grpc.ClientConn{}
)

func main() {

	flag.Parse()

	for _, site := range sites {
		sitesSet[site] = true
		conn, err := grpc.NewClient(fmt.Sprintf("10.201.%d.1:%d", site, *managementRPCPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Errorf("error connecting to site %d daemon: %v\n", site, err)
			continue
		}
		defer conn.Close()
		nodes[site] = conn
	}

	for site, conn := range nodes {
		s := pb.NewManagementClient(conn)

		r, err := s.GetRouteTable(context.Background(), &pb.GetRouteTableRequest{})
		if err != nil {
			logger.Errorf("error getting routes from site %d: %v\n", site, err)
			continue
		}

		routeMap[site] = map[int]int{}

		for _, route := range r.Routes.Routes {
			if route.Protocol == 201 {
				ipAddr, _, _ := net.ParseCIDR(route.Address)

				ipv4 := ipAddr.To4()
				if ipv4 == nil {
					continue
				}

				if sitesSet[int(ipv4[1])] {
					wgIf, err := netctl.ParseWGInterface(route.Interface)
					if err != nil {
						logger.Errorf("error parsing interface %s: %v", route.Interface, err)
						continue
					}

					localID, err := strconv.Atoi(wgIf.LocalID)
					if err != nil {
						logger.Errorf("error parsing local ID %s: %v", wgIf.LocalID, err)
						continue
					}

					remoteID, err := strconv.Atoi(wgIf.RemoteID)
					if err != nil {
						logger.Errorf("error parsing remote ID %s: %v", wgIf.RemoteID, err)
						continue
					}

					routeMap[localID][int(ipv4[1])] = remoteID
				}
			}
		}
	}

	tracePath(routeMap)

}

type Route struct {
	Dest int
	Via  []int
}

func tracePath(routeMap map[int]map[int]int) {

	routingTable := make(map[int][]Route, len(routeMap))

	for source, paths := range routeMap {
		for dst, via := range paths {
			newRoute := Route{Dest: dst, Via: []int{via}}
			if dst != via {
				count := 0
				for {
					next := routeMap[via][dst]
					newRoute.Via = append(newRoute.Via, next)

					if next == dst {
						break
					}

					count += 1
					if count > 10 {
						logger.Errorf("too many hops for route %d -> %d", source, dst)
						break
					}

					via = next
				}
			}

			routingTable[source] = append(routingTable[source], newRoute)
		}

	}

	// Get sorted list of sources for consistent output
	sources := make([]int, 0, len(routingTable))
	for source := range routingTable {
		sources = append(sources, source)
	}
	sort.Ints(sources)

	// Print routing table
	fmt.Println("\nRouting Table:")
	fmt.Println("-------------")
	for _, source := range sources {
		fmt.Printf("\nSource: %d\n", source)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		conn := pb.NewManagementClient(nodes[source])
		s, err := conn.GetState(ctx, &pb.GetStateRequest{Key: "update_method", Namespace: "route"})
		if err != nil {
			logger.Errorf("error getting update method for source %d: %v", source, err)
			continue
		}
		fmt.Printf("Update Method: %s\n", s.Value)
		fmt.Printf("Updated at: %v\n", time.Unix(0, s.Timestamp*1000))

		c, err := conn.GetConsensusState(ctx, &pb.GetConsensusStateRequest{})
		if err != nil {
			logger.Errorf("error getting consensus state for source %d: %v", source, err)
			continue
		}

		fmt.Printf("Consensus State: %s\n", c.State)

		routes := routingTable[source]

		// Sort routes by destination for consistent output
		sort.Slice(routes, func(i, j int) bool {
			return routes[i].Dest < routes[j].Dest
		})

		for _, route := range routes {
			fmt.Printf("  To: %d\n", route.Dest)
			fmt.Printf("    Via: %v\n", route.Via)
		}
	}

}
