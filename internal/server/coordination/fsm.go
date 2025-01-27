package concensus

import (
	"github.com/hashicorp/raft"
)

type FSM struct{}

func (f *FSM) Apply(*raft.Log) interface{}         { return nil }
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) { return nil, nil }
func (f *FSM) Restore(raft.FSMSnapshot) error      { return nil }
