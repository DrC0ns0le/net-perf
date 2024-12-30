package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/route/finder"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/DrC0ns0le/net-perf/pkg/pb/management"
)

var (
	managementRPCPort = flag.Int("management.rpcport", 5122, "port for management rpc server")
	elasticPort       = flag.Int("elastic.port", 9200, "port for elastic server")
	elasticHost       = flag.String("elastic.host", "localhost", "host for elastic server")
	elasticUser       = flag.String("elastic.user", "elastic", "user for elastic server")
	elasticPass       = flag.String("elastic.pass", "password", "pass for elastic server")

	updateInterval = flag.Duration("update.interval", 1*time.Minute, "interval for updating routes")

	printRoutes = flag.Bool("print.routes", false, "print routes")

	sites = []int{
		0, 1, 2, 3,
	}

	sitesSet = map[int]bool{}

	logger = logging.NewDefaultLogger()
)

func main() {

	flag.Parse()

	// setup gRPC client
	nodes := map[int]*grpc.ClientConn{}

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

	// Configure the Elasticsearch client
	cfg := elasticsearch.Config{
		Addresses: []string{
			fmt.Sprintf("https://%s:%d", *elasticHost, *elasticPort),
		},
		Username: *elasticUser, // Your Elasticsearch username
		Password: *elasticPass, // Your Elasticsearch password

		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec
			},
		},
	}

	// Create the client
	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating Elasticsearch client: %s", err)
	}

	res, err := esClient.Info()
	if err != nil {
		log.Fatalf("Error getting response: %s", err)
	}
	defer res.Body.Close()

	prevRoutes, err := initializeCacheFromES(esClient)
	if err != nil {
		log.Fatalf("Failed to initialize cache from Elasticsearch: %v", err)
	}

	ticker := time.NewTicker(*updateInterval)
	defer ticker.Stop()

	logger.Infof("running initial route check")
	if err := runRouteCheck(nodes, esClient, prevRoutes); err != nil {
		log.Printf("Error in route check: %v", err)
	}

	for range ticker.C {
		logger.Infof("running route check")
		if err := runRouteCheck(nodes, esClient, prevRoutes); err != nil {
			log.Printf("Error in route check: %v", err)
		}
	}

}

type Route struct {
	Dest int
	Via  []int
}

type RouteRecord struct {
	Timestamp       time.Time `json:"@timestamp"`
	Source          int       `json:"source"`
	Destination     int       `json:"destination"`
	BGPPath         []int     `json:"bgp_path"`
	DijkstraPath    []int     `json:"dijkstra_path"`
	HopCount        int       `json:"hop_count"`
	IsOptimal       bool      `json:"is_optimal"`
	RouteChanged    bool      `json:"route_changed"`
	LastBGPPath     []int     `json:"last_bgp_path,omitempty"`
	BGPPathStr      string    `json:"bgp_path_str"`
	DijkstraPathStr string    `json:"dijkstra_path_str"`
	LastBGPPathStr  string    `json:"last_bgp_path_str,omitempty"`
}

func initializeCacheFromES(esClient *elasticsearch.Client) (map[string][]int, error) {
	prevRoutes := make(map[string][]int)

	// Query Elasticsearch for the most recent route records
	res, err := esClient.Search(
		esClient.Search.WithIndex("routing-history"),
		esClient.Search.WithSort("@timestamp:desc"),
		esClient.Search.WithSize(250),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query Elasticsearch: %v", err)
	}
	defer res.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse Elasticsearch response: %v", err)
	}

	hits, _ := result["hits"].(map[string]interface{})["hits"].([]interface{})
	for _, hit := range hits {
		source := hit.(map[string]interface{})["_source"].(map[string]interface{})
		routeKey := fmt.Sprintf("%d-%d", int(source["source"].(float64)), int(source["destination"].(float64)))
		bgpPath, _ := source["bgp_path"].([]interface{})
		intPath := make([]int, len(bgpPath))
		for i, v := range bgpPath {
			intPath[i] = int(v.(float64))
		}
		prevRoutes[routeKey] = intPath
	}

	return prevRoutes, nil
}

func runRouteCheck(nodes map[int]*grpc.ClientConn, esClient *elasticsearch.Client, prevRoutes map[string][]int) error {
	routeMap := make(map[int]map[int]int)

	// Fetch current routes from nodes
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

	if err := tracePath(routeMap, esClient, prevRoutes); err != nil {
		return fmt.Errorf("failed to trace path: %v", err)
	}

	return nil
}

func tracePath(routeMap map[int]map[int]int, esClient *elasticsearch.Client, prevRoutes map[string][]int) error {
	routingTable := make(map[int][]Route, len(routeMap))
	routeHistory := make([]RouteRecord, 0)

	// Process routes
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

	// Get sorted sources for consistent output
	sources := make([]int, 0, len(routingTable))
	for source := range routingTable {
		sources = append(sources, source)
	}
	sort.Ints(sources)

	// Initialize graph for Dijkstra calculations
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	graph, err := finder.NewGraph(ctx)
	if err != nil {
		return fmt.Errorf("failed to create graph: %v", err)
	}

	// Process and store route information
	for _, source := range sources {
		routes := routingTable[source]
		sort.Slice(routes, func(i, j int) bool {
			return routes[i].Dest < routes[j].Dest
		})

		for _, route := range routes {
			dijkstraPath, _, err := graph.GetShortestPath(source, route.Dest)
			if err != nil {
				logger.Errorf("Failed to get shortest path for %d -> %d: %v", source, route.Dest, err)
				continue
			}

			// Create unique key for route
			routeKey := fmt.Sprintf("%d-%d", source, route.Dest)

			// Check if route changed from previous state
			lastPath, exists := prevRoutes[routeKey]
			routeChanged := exists && !reflect.DeepEqual(lastPath, route.Via)

			// Create historical record
			record := RouteRecord{
				Timestamp:       time.Now(),
				Source:          source,
				Destination:     route.Dest,
				BGPPath:         route.Via,
				DijkstraPath:    dijkstraPath[1:],
				HopCount:        len(route.Via),
				IsOptimal:       reflect.DeepEqual(route.Via, dijkstraPath[1:]),
				RouteChanged:    routeChanged,
				LastBGPPath:     lastPath,
				BGPPathStr:      pathToString(route.Via),
				DijkstraPathStr: pathToString(dijkstraPath[1:]),
				LastBGPPathStr:  pathToString(lastPath),
			}

			routeHistory = append(routeHistory, record)

			// Update previous routes
			prevRoutes[routeKey] = route.Via

			if *printRoutes {
				// Print current route
				fmt.Printf("\nSource: %d\n", source)
				fmt.Printf(" To: %d\n", route.Dest)
				fmt.Printf(" Bird BGP: %v\n", route.Via)
				fmt.Printf(" Dijkstra: %v\n", dijkstraPath[1:])
			}
		}
	}

	// Send data to Elasticsearch using bulk indexer
	bulkIndexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:         "routing-history",
		Client:        esClient,
		FlushBytes:    5242880, // 5MB
		FlushInterval: 30 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("failed to create bulk indexer: %v", err)
	}

	for _, record := range routeHistory {
		data, err := json.Marshal(record)
		if err != nil {
			logger.Errorf("Failed to marshal route record: %v", err)
			continue
		}

		err = bulkIndexer.Add(
			context.Background(),
			esutil.BulkIndexerItem{
				Action: "index",
				Body:   bytes.NewReader(data),
				OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
					if err != nil {
						logger.Errorf("Failed to index route record: %v", err)
					}
				},
			},
		)
		if err != nil {
			logger.Errorf("Failed to add record to bulk indexer: %v", err)
		}
	}

	if err := bulkIndexer.Close(context.Background()); err != nil {
		return fmt.Errorf("failed to close bulk indexer: %v", err)
	}

	return nil
}

func pathToString(path []int) string {
	strPath := make([]string, len(path))
	for i, as := range path {
		strPath[i] = fmt.Sprintf("%d", as)
	}
	return strings.Join(strPath, " -> ")
}
