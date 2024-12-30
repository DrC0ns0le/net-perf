package management

import (
	"flag"
	"log"
	"net"
	"strconv"

	"google.golang.org/grpc"

	"github.com/DrC0ns0le/net-perf/internal/system"
	pb "github.com/DrC0ns0le/net-perf/pkg/pb/management"
)

var managementRPCPort = flag.Int("management.rpcport", 5122, "port for management rpc server")

func Serve(global *system.Node) {

	// start gRPC server
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(*managementRPCPort))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	s := grpc.NewServer()
	pb.RegisterManagementServer(s, &managementServer{})

	global.Logger.Infof("management gRPC server listening at %v", listener.Addr())
	if err := s.Serve(listener); err != nil {
		global.Logger.Errorf("failed to serve management gRPC server: %v", err)
	}
}
