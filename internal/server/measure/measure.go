package measure

import (
	"context"
	"log"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/analyser"
	"github.com/DrC0ns0le/net-perf/internal/measure/pathping"
	pb "github.com/DrC0ns0le/net-perf/pkg/pb/measure"
	analyserpb "github.com/DrC0ns0le/net-perf/pkg/pb/networkanalysis"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Server struct {
	pb.UnimplementedMeasureServer
}

func (s *Server) PathLatency(ctx context.Context, req *pb.PathLatencyRequest) (*pb.PathLatencyResponse, error) {
	c := pathping.NewClient(int(req.Count), time.Duration(req.Interval)*time.Millisecond)

	result, err := c.Measure(func(path []int32) []int {
		p := make([]int, len(path))
		for i, v := range path {
			p[i] = int(v)
		}
		return p
	}(req.Path))
	return &pb.PathLatencyResponse{
		Status:  int32(result.Status),
		Latency: int32(result.Duration.Microseconds()),
	}, err
}

func (s *Server) ProjectPathLatency(ctx context.Context, req *pb.ProjectedPathLatencyRequest) (*pb.ProjectedPathLatencyResponse, error) {
	start := int(req.Start)
	end := int(req.End)

	a := analyser.NewNetworkAnalyzer()
	if err := a.CreateGraph(ctx); err != nil {
		return nil, err
	}

	conn, err := grpc.NewClient("10.2.1.15:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to solver: %v", err)
	}
	defer conn.Close()

	solverClient := analyserpb.NewNetworkAnalyzerClient(conn)

	cost, err := a.AnalyzeTargetPath(ctx, start, end, solverClient)
	return &pb.ProjectedPathLatencyResponse{
		Status:  int32(1),
		Latency: cost,
	}, err
}
