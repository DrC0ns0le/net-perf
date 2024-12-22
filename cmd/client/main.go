package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"sort"
	"strconv"

	"github.com/DrC0ns0le/net-perf/logging"
	pb "github.com/DrC0ns0le/net-perf/pkg/pb/management"
	"github.com/DrC0ns0le/net-perf/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	managementRPCPort = flag.Int("management.rpcport", 5122, "port for management rpc server")

	// showInfo       = flag.Bool("i", false, "Show all routes info")
	// showRouteTable = flag.Bool("t", false, "Show managed routes")

	sites = []int{
		0, 1, 2, 3,
	}

	sitesSet = map[int]bool{}

	routeMap = map[int]map[int]int{}
)

func main() {

	flag.Parse()

	// setup gRPC client
	nodes := map[int]*grpc.ClientConn{}

	for _, site := range sites {
		sitesSet[site] = true
		conn, err := grpc.NewClient(fmt.Sprintf("10.201.%d.1:%d", site, *managementRPCPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logging.Errorf("Error connecting to site %d daemon: %v\n", site, err)
			continue
		}
		defer conn.Close()
		nodes[site] = conn
	}

	// adjacency matrix
	matrix := make([][]int, len(sites))
	for i := range matrix {
		matrix[i] = make([]int, len(sites))
	}

	for site, conn := range nodes {
		s := pb.NewManagementClient(conn)

		r, err := s.GetRouteTable(context.Background(), &pb.GetRouteTableRequest{})
		if err != nil {
			logging.Errorf("Error getting routes from site %d: %v\n", site, err)
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
					wgIf, err := utils.ParseWGInterface(route.Interface)
					if err != nil {
						logging.Errorf("Error parsing interface %s: %v", route.Interface, err)
						continue
					}

					localID, err := strconv.Atoi(wgIf.LocalID)
					if err != nil {
						logging.Errorf("Error parsing local ID %s: %v", wgIf.LocalID, err)
						continue
					}

					remoteID, err := strconv.Atoi(wgIf.RemoteID)
					if err != nil {
						logging.Errorf("Error parsing remote ID %s: %v", wgIf.RemoteID, err)
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
						logging.Errorf("Too many hops for route %d -> %d", source, dst)
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
		routes := routingTable[source]

		// Sort routes by destination for consistent output
		sort.Slice(routes, func(i, j int) bool {
			return routes[i].Dest < routes[j].Dest
		})

		for _, route := range routes {
			fmt.Printf("  To %d via: %v\n", route.Dest, route.Via)
		}
	}

}
