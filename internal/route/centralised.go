package route

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/measure/pathping"
	"github.com/DrC0ns0le/net-perf/internal/route/finder"
	"github.com/DrC0ns0le/net-perf/internal/system"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DrC0ns0le/net-perf/pkg/logging"
	pb "github.com/DrC0ns0le/net-perf/pkg/pb/distributed"
	measurepb "github.com/DrC0ns0le/net-perf/pkg/pb/measure"
)

const (
	rpcPort             = 5122
	initialStartTimeout = 5 * time.Second
)

var (
	disableCentralisedRoutingFastStart = flag.Bool("router.disable-fast-start", false, "disable fast start for centralised routing")
	fastStartAttempts                  = flag.Int("router.fast-start-attempts", 3, "fast start attempts for centralised routing")
)

type CentralisedRouter struct {
	localID int
	stopCh  chan struct{}
	graph   *finder.Graph

	concensus system.ConsensusInterface

	siteRoutes   map[int]int // site to site routes
	siteRoutesMu sync.RWMutex
	updatedAt    time.Time

	logger logging.Logger
}

type result struct {
	forwardRank int
	returnRank  int
	latency     int
	path        []int
}

func NewCentralisedRouter(logger logging.Logger, graph *finder.Graph, stopCh chan struct{}, consensus system.ConsensusInterface, localID int) *CentralisedRouter {
	return &CentralisedRouter{
		localID:    localID,
		siteRoutes: make(map[int]int),

		concensus: consensus,
		graph:     graph,
		logger:    logger,

		stopCh: stopCh,
	}
}

func (r *CentralisedRouter) Start() error {
	go r.run()
	return nil
}

func (r *CentralisedRouter) run() {
	timer := time.NewTimer(initialStartTimeout)
	ticker := time.NewTicker(*updateInterval / 2)

	routine := func() {
		if r.concensus != nil && r.concensus.Leader() && r.concensus.Healty() {
			ctx, cancel := context.WithTimeout(context.Background(), 10**costContextTimeout)
			defer cancel()
			r.logger.Debug("centralised router updating and distributing routes")
			if err := r.Refresh(ctx); err != nil {
				r.logger.Errorf("error refreshing centralised router: %v", err)
			}
		} else {
			r.logger.Debug("centralised router not leader or not healthy")
		}
	}

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			routine()
		case <-timer.C:
			if !*disableCentralisedRoutingFastStart || *fastStartAttempts == 0 {
				timer.Stop()
				break
			}
			if r.concensus != nil && r.concensus.Healty() {
				routine()
				timer.Stop()
			} else if *fastStartAttempts > 0 {
				*fastStartAttempts--
				timer.Reset(initialStartTimeout)
			}
		}
	}
}

func (r *CentralisedRouter) Refresh(ctx context.Context) error {
	const topN = 5 // number of shortest paths to consider

	// get all active sites
	paths, err := finder.GetAllPaths(ctx)
	if err != nil {
		return err
	}

	// create a map of sites, each site has a map of routes
	sites := make(map[int]map[int]int)
	for _, path := range paths {
		if _, ok := sites[path.Source]; !ok {
			sites[path.Source] = make(map[int]int)
		}

		if _, ok := sites[path.Target]; !ok {
			sites[path.Target] = make(map[int]int)
		}
	}

	// initialise route map
	for _, routes := range sites {
		for site := range sites {
			routes[site] = -1
		}
	}
	if r.graph == nil {
		r.graph, err = finder.NewGraph(ctx)
		if err != nil {
			return fmt.Errorf("error initializing graph: %w", err)
		}
	}
	if err := r.graph.RefreshWeights(ctx); err != nil {
		return fmt.Errorf("error refreshing graph: %w", err)
	}

	for site, routes := range sites {
		for target, via := range routes {
			if via != -1 {
				continue
			}
			if site == target {
				sites[site][target] = site
				continue
			}
			// forward path
			fPath, err := r.graph.GetTopNShortestPaths(site, target, topN)
			if err != nil {
				return fmt.Errorf("error getting forward path: %w", err)
			}

			// return path
			rPath, err := r.graph.GetTopNShortestPaths(target, site, topN)
			if err != nil {
				return fmt.Errorf("error getting return path: %w", err)
			}

			wg := sync.WaitGroup{}
			resultsChan := make(chan result, topN*topN)

			for i, p := range fPath {
				for j, v := range rPath {
					wg.Add(1)
					time.Sleep(10 * time.Millisecond) // Stagger measurements to avoid overloading
					go func(p finder.PathResult, v finder.PathResult, i, j int) {
						defer wg.Done()
						fullPath := append(p.Path, v.Path[1:]...)

						res, err := r.measurePathLatency(ctx, fullPath)
						if err != nil {
							r.logger.Errorf("error measuring path %v: %v", fullPath, err)
							return
						}

						resultsChan <- result{
							forwardRank: i + 1,
							returnRank:  j + 1,
							latency:     res + (10 * len(fullPath)), // add additional penalty of 10ms per hop
							path:        fullPath,
						}
					}(p, v, i, j)
				}
			}
			wg.Wait()
			close(resultsChan)

			results := make([]result, 0, topN*topN)
			for r := range resultsChan {
				results = append(results, r)
			}

			// sort results by latency
			sort.Slice(results, func(i, j int) bool {
				return results[i].latency < results[j].latency
			})

			// update routes
			if len(results) == 0 {
				r.logger.Debugf("forward path: %v", fPath)
				r.logger.Debugf("return path: %v", rPath)
				return fmt.Errorf("no routes found for site %d to %d", site, target)
			}

			// temporarily workaround to prevent route loops
			success := false
		selectionLoop:
			for rank, res := range results {
				bestPath := res.path

				// find the middle site
				mid := -1
				for i, v := range bestPath {
					if v == target {
						mid = i
						break
					}
				}

				// update routes
				for i := 0; i < len(bestPath)-2; i++ {
					if i < mid {
						if sites[bestPath[i]][target] != -1 && sites[bestPath[i]][target] != bestPath[i+1] {
							r.logger.Infof("skipping route %v ranked %d to prevent loop", bestPath, rank)
							continue selectionLoop
						}
						sites[bestPath[i]][target] = bestPath[i+1]
					} else {
						if sites[bestPath[i]][site] != -1 && sites[bestPath[i]][site] != bestPath[i+1] {
							r.logger.Infof("skipping route %v ranked %d to prevent loop", bestPath, rank)
							continue selectionLoop
						}
						sites[bestPath[i]][site] = bestPath[i+1]
					}
				}

				success = true
				break
			}
			if !success {
				return fmt.Errorf("no routes found for site %d to %d", site, target)
			}

		}
	}

	if err := r.checkTooManyHops(sites); err != nil {
		return err
	}

	// distribute routes
	if err := r.Distribute(ctx, sites); err != nil {
		r.logger.Errorf("error distributing centralised routes: %v", err)
	}

	return nil
}

func (r *CentralisedRouter) getSiteRPCConn(site int) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(fmt.Sprintf("10.201.%d.1:%d", site, rpcPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("error connecting to site %d daemon: %w", site, err)
	}
	return conn, nil
}

func (r *CentralisedRouter) GetSiteRouteMap() map[int]int { return r.siteRoutes }

func (r *CentralisedRouter) Distribute(ctx context.Context, siteRoutes map[int]map[int]int) error {
	for site, routes := range siteRoutes {
		conn, err := r.getSiteRPCConn(site)
		if err != nil {
			return err
		}
		defer conn.Close()

		client := pb.NewRouteServiceClient(conn)

		r.logger.Debugf("distributing routes for site %d", site)

		_, err = client.UpdateRoute(ctx, &pb.SiteRoute{
			Route: convertToInt32Map(routes),
		})
		if err != nil {
			return fmt.Errorf("error updating route for site %d: %w", site, err)
		}
	}

	return nil
}

// UpdateSiteRoutes updates the routes for a specific site.
// NOTE, this purges all other routes
func (r *CentralisedRouter) UpdateSiteRoutes(routes map[int]int) bool {
	r.siteRoutesMu.Lock()
	defer r.siteRoutesMu.Unlock()

	// Assume no changes until we find one
	changed := false

	// Check each route
	for k, v := range routes {
		if r.siteRoutes[k] != v {
			changed = true
			break
		}
	}

	// Check if all keys in existing are in routes
	if !changed {
		for k := range r.siteRoutes {
			if _, ok := routes[k]; !ok {
				changed = true
				break
			}
		}
	}
	if changed {
		r.logger.Infof("centralised route table updated %v", routes)
		r.siteRoutes = make(map[int]int)
		for k, v := range routes {
			r.siteRoutes[k] = v
		}
	}
	r.updatedAt = time.Now()
	return changed
}

func (r *CentralisedRouter) measurePathLatency(ctx context.Context, path []int) (int, error) {
	measurementPath := func(p []int) []int32 {
		np := make([]int32, len(p))
		for i, v := range p {
			np[i] = int32(v)
		}
		return np
	}(path)

	if path[0] == r.localID {
		c := pathping.NewClient(10, 100*time.Millisecond)
		res, err := c.Measure(path)
		return int(res.Duration.Microseconds()), err
	} else {
		conn, err := r.getSiteRPCConn(path[0])
		if err != nil {
			return 0, err
		}
		defer conn.Close()

		client := measurepb.NewMeasureClient(conn)

		res, err := client.PathLatency(ctx, &measurepb.PathLatencyRequest{
			Path:     measurementPath,
			Count:    10,
			Interval: 100,
		})
		if err != nil {
			return 0, err
		}
		return int(res.Latency), nil
	}
}

func (r *CentralisedRouter) checkTooManyHops(routeMap map[int]map[int]int) error {
	for source, paths := range routeMap {
		for dst, via := range paths {
			if dst != via {
				count := 0
				currentVia := via

				for {
					next, exists := routeMap[currentVia][dst]
					if !exists {
						break
					}

					count += 1
					if count > len(routeMap) { // hop count should be less than the total number of sites
						return fmt.Errorf("too many hops for route %d -> %d", source, dst)
					}

					if next == dst {
						break
					}

					currentVia = next
				}
			}
		}
	}

	return nil
}

func convertToInt32Map(input map[int]int) map[int32]int32 {
	output := make(map[int32]int32, len(input))
	for k, v := range input {
		output[int32(k)] = int32(v)
	}
	return output
}
