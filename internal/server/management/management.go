package management

import (
	"context"

	"github.com/DrC0ns0le/net-perf/internal/route"
	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	pb "github.com/DrC0ns0le/net-perf/pkg/pb/management"
	"github.com/vishvananda/netlink"
)

type Server struct {
	pb.UnimplementedManagementServer

	stateTable *system.StateTable
}

func NewServer(global *system.Node) *Server {
	return &Server{
		stateTable: global.StateTable,
	}
}

func (s *Server) GetRouteTable(ctx context.Context, req *pb.GetRouteTableRequest) (*pb.GetRouteTableResponse, error) {
	routes, err := netctl.ListManagedRoutes(route.CustomRouteProtocol)
	if err != nil {
		return nil, err
	}

	resp := &pb.GetRouteTableResponse{
		Routes: &pb.RouteTable{
			Routes: make([]*pb.Route, 0, len(routes)),
		},
	}

	for _, route := range routes {
		pbRoute := &pb.Route{
			Address:  route.Dst.String(),
			Next:     route.Gw.String(),
			Metric:   int32(route.Priority),
			Protocol: int32(route.Protocol),
		}

		if route.LinkIndex > 0 {
			link, err := netlink.LinkByIndex(route.LinkIndex)
			if err == nil {
				pbRoute.Interface = link.Attrs().Name
			}
		}

		resp.Routes.Routes = append(resp.Routes.Routes, pbRoute)
	}

	return resp, nil
}

func (s *Server) GetState(ctx context.Context, req *pb.GetStateRequest) (*pb.GetStateResponse, error) {
	val, ts, err := s.stateTable.GetFromNamespace(req.Key, req.Namespace)
	if err != nil {
		return nil, err
	}

	return &pb.GetStateResponse{
		Value:     val,
		Timestamp: ts.UnixMicro(),
	}, nil
}
