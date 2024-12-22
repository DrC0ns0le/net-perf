package management

import (
	"log"
	"net"
	"strconv"

	"google.golang.org/grpc"

	pb "github.com/DrC0ns0le/net-perf/pkg/pb/management"
)

func MustServe(port int) {

	lis, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterManagementServer(s, &managementServer{})

	log.Printf("management gRPC server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
