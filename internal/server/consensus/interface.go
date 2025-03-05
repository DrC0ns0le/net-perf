package consensus

func (s *Server) Healty() bool {
	healthy, _ := s.IsClusterHealthy()
	return healthy
}

func (s *Server) Leader() bool {
	return s.IsLeader()
}
