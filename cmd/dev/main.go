package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/route/cost"
	"github.com/DrC0ns0le/net-perf/internal/route/finder"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
	pb "github.com/DrC0ns0le/net-perf/pkg/pb/measure"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	measureRPCPort = flag.Int("measure.rpcport", 5122, "port for measure rpc server")
	noOfPaths      = flag.Int("noofpaths", 5, "number of paths to find")

	logger = logging.NewDefaultLogger()
)

func main() {
	flag.Parse()
	args := os.Args

	var (
		start, end int
		err        error
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	g, err := finder.NewGraph(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if len(args) == 3 {
		start, err = strconv.Atoi(args[1])
		if err != nil {
			fmt.Println("invalid start node")
			fmt.Println("usage: dev <start> <end>")
			os.Exit(1)
		}
		end, err = strconv.Atoi(args[2])
		if err != nil {
			fmt.Println("invalid end node")
			fmt.Println("usage: dev <start> <end>")
			os.Exit(1)
		}

		findPaths(g, start, end, *noOfPaths)
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		paths, err := finder.GetAllPaths(ctx)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		for _, path := range paths {
			findPaths(g, path.Source, path.Target, *noOfPaths)
		}
	}

}

func findPaths(graph *finder.Graph, start, end, noOfPaths int) {

	derivedLatencies := make([]float64, 0)

	// Find 5 unique circular paths that must use the direct edge from node 1 to node 2
	paths, err := graph.FindNCircularPathsWithDirectEdge(start, end, noOfPaths)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print the paths
	fmt.Printf("Found %d paths from %d to %d\n", len(paths), start, end)
	for i, path := range paths {
		conn, err := grpc.NewClient(fmt.Sprintf("10.201.%d.1:%d", path.Path()[0], *measureRPCPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Errorf("error connecting to site %d daemon: %v\n", path.Path()[0], err)
		}
		defer conn.Close()

		client := pb.NewMeasureClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		result, err := client.PathLatency(ctx, &pb.PathLatencyRequest{
			Path: func(p []int) []int32 {
				np := make([]int32, len(p))
				for i, v := range p {
					np[i] = int32(v)
				}
				return np
			}(path.Path()),
			Count:    10,
			Interval: 150,
		})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		expectedLatency := time.Duration(0)
		for i, p := range path.Path() {
			if i == len(path.Path())-1 {
				break
			}
			rtt, err := cost.GetPathLatency(ctx, p, path.Path()[i+1])
			if err != nil {
				fmt.Printf("GetPathLatency error: %v\n", err)
			}
			expectedLatency += time.Duration(rtt/2) * time.Microsecond
		}
		fmt.Printf("Path %d: %s\n", i+1, path)
		fmt.Printf("Latency: %d ms\n", result.Latency)
		fmt.Printf("Expected Latency: %d ms\n", expectedLatency.Milliseconds())

		derivedLatency, err := calculateOneWayLatency(path.Path(), start, end, float64(result.Latency))
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		derivedLatencies = append(derivedLatencies, derivedLatency)
	}

	avgDerivedLatency := 0.0
	for _, l := range derivedLatencies {
		avgDerivedLatency += l
	}
	avgDerivedLatency /= float64(len(derivedLatencies))
	fmt.Printf("Average Derived Latency: %.2f ms\n", avgDerivedLatency)

	// Calculate the half RTT
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	l, err := cost.GetPathLatency(ctx, start, end)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Half RTT: %.2f ms\n\n", l/2000)
}

func calculateOneWayLatency(path []int, src, dst int, totalLatency float64) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i := 0; i < len(path)-1; i++ {
		if path[i] == src && path[i+1] == dst {
			continue
		}
		l, err := cost.GetPathLatency(ctx, path[i], path[i+1])
		if err != nil {
			return 0, err
		}
		totalLatency -= l / 2000 // half the rtt and convert to ms
	}

	return totalLatency, nil
}
