package consensus

import (
	"io"

	"github.com/hashicorp/raft"
)

// FSM is a minimal FSM implementation that does nothing
type FSM struct{}

// Apply is called when a log entry is committed (no-op)
func (f *FSM) Apply(log *raft.Log) interface{} {
	return nil
}

// Snapshot returns a snapshot of the FSM state (no-op)
func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	return &FSMSnapshot{}, nil
}

// Restore restores the FSM state from a snapshot (no-op)
func (f *FSM) Restore(snapshot io.ReadCloser) error {
	return nil
}

// FSMSnapshot is a minimal implementation of FSMSnapshot
// Since we don't have state to persist, this is minimal but still required by the Raft interface
type FSMSnapshot struct{}

// Persist writes the snapshot (we have nothing to write, but must implement the interface)
func (s *FSMSnapshot) Persist(sink raft.SnapshotSink) error {
	// Nothing to persist, just close the sink successfully
	return sink.Close()
}

// Release is called when we're done with the snapshot
func (s *FSMSnapshot) Release() {
	// No resources to release
}
