package measure

import (
	"context"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/measure/pathping"
	pb "github.com/DrC0ns0le/net-perf/pkg/pb/measure"
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
