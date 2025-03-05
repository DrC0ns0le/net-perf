package server

import (
	"flag"
	"fmt"

	"github.com/DrC0ns0le/net-perf/internal/server/consensus"
	"github.com/DrC0ns0le/net-perf/internal/system"
)

var consensusPort = flag.Int("consensus.port", 5123, "port for concensus server")

func NewConcensusServer(global *system.Node) *consensus.Server {
	cs := consensus.NewServer(global, fmt.Sprintf("10.201.%d.1:%d", global.SiteID, *consensusPort))

	global.ConsensusService = cs

	return cs
}
