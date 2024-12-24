package management

import (
	"context"

	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	pb "github.com/DrC0ns0le/net-perf/pkg/pb/management"
	"github.com/vishvananda/netlink"
)

type managementServer struct {
	pb.UnimplementedManagementServer
}

func (s *managementServer) GetRouteTable(ctx context.Context, req *pb.GetRouteTableRequest) (*pb.GetRouteTableResponse, error) {
	routes, err := netctl.ListManagedRoutes()
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