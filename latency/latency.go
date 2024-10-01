package latency

type Result struct {
	Protocol   string
	Status     int
	AvgLatency int64
	Jitter     int64
	Loss       float64
}
