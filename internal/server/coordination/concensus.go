package concensus

import (
	"flag"
)

var (
	raftDirectory = flag.String("raft.directory", "/opt/wg-mesh/raft", "path to raft directory")
)

type Raft struct {
	directory string
}
