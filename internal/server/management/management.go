package management

import (
	"context"
	"fmt"

	"github.com/DrC0ns0le/net-perf/internal/route"
	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	pb "github.com/DrC0ns0le/net-perf/pkg/pb/management"
	"github.com/vishvananda/netlink"
)

type Server struct {
	pb.UnimplementedManagementServer

	stateTable       *system.StateTable
	consensusService system.ConsensusInterface
	r                system.RouteInterface
	peers            []int
}

func NewServer(global *system.Node) *Server {
	return &Server{
		stateTable:       global.StateTable,
		consensusService: global.ConsensusService,
		r:                global.RouteService,
		peers:            global.Peers,
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

func (s *Server) GetCentralisedRouteTable(ctx context.Context, req *pb.GetRouteTableRequest) (*pb.GetRouteTableMapResponse, error) {
	routes := make(map[int]int)
	for _, peer := range s.peers {
		routes[peer] = s.r.GetCentralisedRoute(peer)
		if routes[peer] == -1 {
			return nil, fmt.Errorf("no route to peer %d", peer)
		}
	}

	return &pb.GetRouteTableMapResponse{
		Routes: convertToInt32Map(routes),
	}, nil
}

func (s *Server) GetGraphRouteTable(ctx context.Context, req *pb.GetRouteTableRequest) (*pb.GetRouteTableMapResponse, error) {
	routes := make(map[int]int)

	for _, peer := range s.peers {
		routes[peer] = s.r.GetGraphRoute(peer)
		if routes[peer] == -1 {
			return nil, fmt.Errorf("no route to peer %d", peer)
		}
	}

	return &pb.GetRouteTableMapResponse{
		Routes: convertToInt32Map(routes),
	}, nil
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

func (s *Server) GetConsensusState(ctx context.Context, req *pb.GetConsensusStateRequest) (*pb.GetConsensusStateResponse, error) {
	return &pb.GetConsensusStateResponse{
		State: func() string {
			if s.consensusService.Leader() {
				return "leader"
			}
			return "follower"
		}(),
	}, nil
}

func convertToInt32Map(route map[int]int) map[int32]int32 {
	route32 := make(map[int32]int32)
	for k, v := range route {
		route32[int32(k)] = int32(v)
	}
	return route32
}
