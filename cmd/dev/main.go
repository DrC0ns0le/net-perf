package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/DrC0ns0le/net-perf/internal/route/finder"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
	pb "github.com/DrC0ns0le/net-perf/pkg/pb/measure"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	measureRPCPort = flag.Int("measure.rpcport", 5122, "port for measure rpc server")

	logger = logging.NewDefaultLogger()
)

func main() {
	ctx := context.Background()
	g, err := finder.NewGraph(ctx)
	if err != nil {
		panic(err)
	}

	args := os.Args
	if len(args) != 3 {
		fmt.Println("invalid number of arguments")
		fmt.Println("usage: dev <start> <end>")
		os.Exit(1)
	}
	start, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Println("invalid start node")
		fmt.Println("usage: dev <start> <end>")
		os.Exit(1)
	}
	end, err := strconv.Atoi(args[2])
	if err != nil {
		fmt.Println("invalid end node")
		fmt.Println("usage: dev <start> <end>")
		os.Exit(1)
	}
	noOfPaths := 50

	// Find 5 unique circular paths that must use the direct edge from node 1 to node 2
	paths, err := g.FindNCircularPathsWithDirectEdge(start, end, noOfPaths)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print the paths
	fmt.Printf("Found %d paths from %d to %d\n", len(paths), start, end)
	for i, path := range paths {
		fmt.Printf("Path %d: %s\n", i+1, path)
		conn, err := grpc.NewClient(fmt.Sprintf("10.201.%d.1:%d", path.Path()[0], *measureRPCPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Errorf("error connecting to site %d daemon: %v\n", path.Path()[0], err)
		}
		defer conn.Close()

		client := pb.NewMeasureClient(conn)

		result, err := client.PathLatency(ctx, &pb.PathLatencyRequest{
			Path: func(p []int) []int32 {
				np := make([]int32, len(p))
				for i, v := range p {
					np[i] = int32(v)
				}
				return np
			}(path.Path()),
			Count:    10,
			Interval: 100,
		})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
		fmt.Printf("Latency: %d ms\n", result.Latency)
	}
}
