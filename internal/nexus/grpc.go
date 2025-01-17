package nexus

import (
	"flag"
	"fmt"
	"net"
	"strconv"

	"google.golang.org/grpc"

	"github.com/DrC0ns0le/net-perf/internal/nexus/management"
	"github.com/DrC0ns0le/net-perf/internal/nexus/measure"
	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
	managementpb "github.com/DrC0ns0le/net-perf/pkg/pb/management"
	measurepb "github.com/DrC0ns0le/net-perf/pkg/pb/measure"
)

var grpcPort = flag.Int("grpc.port", 5122, "port for grpc server")

type GRPCServer struct {
	port     int
	server   *grpc.Server
	listener net.Listener
	logger   logging.Logger
}

func NewGRPCServer(global *system.Node) *GRPCServer {
	return &GRPCServer{
		port:   *grpcPort,
		logger: global.Logger.With("component", "grpc"),
	}
}

func (s *GRPCServer) Start() error {
	// start gRPC server
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.listener = listener
	s.server = grpc.NewServer()

	// register services
	s.register()

	s.logger.Infof("gRPC server listening at %v", s.listener.Addr())
	if err := s.server.Serve(s.listener); err != nil {
		return fmt.Errorf("failed to serve gRPC server: %w", err)
	}

	return nil
}

func (s *GRPCServer) Stop() error {
	s.server.Stop()
	return s.listener.Close()
}

func (s *GRPCServer) register() {
	managementpb.RegisterManagementServer(s.server, &management.Server{})
	measurepb.RegisterMeasureServer(s.server, &measure.Server{})
}
