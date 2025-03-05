package distributed

import (
	"context"

	"github.com/DrC0ns0le/net-perf/internal/system"
	pb "github.com/DrC0ns0le/net-perf/pkg/pb/distributed"
	"github.com/golang/protobuf/ptypes/empty"
)

type Server struct {
	r system.RouteInterface
	pb.UnimplementedRouteServiceServer
}

func NewServer(global *system.Node) *Server {
	return &Server{
		r: global.RouteService,
	}
}

func (s *Server) GetRoute(ctx context.Context, req *pb.GetRouteRequest) (*pb.SiteRoute, error) {
	route := s.r.GetSiteRoutes(int(req.Id))

	return &pb.SiteRoute{
		Route: convertToInt32Map(route),
	}, nil
}

func (s *Server) GetFullRoute(ctx context.Context, _ *empty.Empty) (*pb.FullRoute, error) {
	route := s.r.GetFullSiteRoutes()

	fullRoute := make(map[int32]*pb.SiteRoute)

	for site, route := range route {
		fullRoute[int32(site)] = &pb.SiteRoute{
			Route: convertToInt32Map(route),
		}
	}

	return &pb.FullRoute{
		Route: fullRoute,
	}, nil
}

func (s *Server) UpdateRoute(ctx context.Context, req *pb.SiteRoute) (*pb.UpdateRouteResponse, error) {
	route := convertToIntMap(req.Route)
	s.r.UpdateLocalRoutes(route)

	return &pb.UpdateRouteResponse{}, nil
}

func convertToInt32Map(route map[int]int) map[int32]int32 {
	route32 := make(map[int32]int32)
	for k, v := range route {
		route32[int32(k)] = int32(v)
	}
	return route32
}

func convertToIntMap(route map[int32]int32) map[int]int {
	routeInt := make(map[int]int)
	for k, v := range route {
		routeInt[int(k)] = int(v)
	}
	return routeInt
}
