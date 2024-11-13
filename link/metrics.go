package link

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/DrC0ns0le/net-perf/utils"
)

type InterfaceMetrics struct {
	Availability float64
	Latency      float64
	Jitter       float64
	PacketLoss   float64
}

// getMetrics queries prometheus for the average latency and packet loss for both
// IPv4 and IPv6 paths to the given remote. It returns a map where the keys are the
// IP versions and the values are InterfaceMetrics objects.
//
// The metrics queried are:
//   - network_latency_duration: average latency in microseconds
//   - network_latency_loss and network_bandwidth_packet_loss: average packet loss
//     as a percentage.
//
// The values are averaged over a 5 minute time window.
func getMetrics(remote string) (map[string]*InterfaceMetrics, error) {

	var metrics map[string]*InterfaceMetrics

	// assumes dual-stack
	path := getPathLabel(remote)
	if path == "" {
		return metrics, errors.New("could not generate path label")
	}
	queries := []string{
		fmt.Sprintf(`avg(avg_over_time(network_latency_duration{path=~"%s"}[5m])) by (version)`, path),
		fmt.Sprintf(`avg(avg_over_time(network_latency_loss{path=~"%s"}[5m]),avg_over_time(network_bandwidth_packet_loss{path=~"%s"}[5m])) by (version)`, path, path),
		fmt.Sprintf(`avg(avg_over_time(network_latency_status{path=~"%s"}[5m])) by (version)`, path),
	}

	metrics = map[string]*InterfaceMetrics{
		"4": {},
		"6": {},
	}

	for i, query := range queries {
		response, err := utils.QueryMetrics(query)
		if err != nil {
			return metrics, err
		}

		for _, result := range response.Data.Result {
			switch i {
			case 0:
				latency, err := strconv.ParseFloat(result.Value[1].(string), 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing latency: %v", err)
				}
				metrics[result.Metric.Version].Latency = latency
			case 1:
				loss, err := strconv.ParseFloat(result.Value[1].(string), 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing packet loss: %v", err)
				}
				metrics[result.Metric.Version].PacketLoss = loss
			case 2:
				availability, err := strconv.ParseFloat(result.Value[1].(string), 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing availability: %v", err)
				}
				metrics[result.Metric.Version].Availability = availability
			}
		}
	}

	return metrics, nil
}

// getPathLabel generates a label string for a path between two nodes. The string is
// in the format "X-Y", where X and Y are the node IDs. The IDs are sorted so that
// the smaller one is first, to ensure that the same path is always labelled with
// the same string, regardless of the order in which the nodes are given. If the
// IDs can't be parsed as integers, an empty string is returned.
func getPathLabel(remote string) string {
	intLocalID, err := strconv.Atoi(localID)
	if err != nil {
		return ""
	}

	intRemoteID, err := strconv.Atoi(remote)
	if err != nil {
		return ""
	}

	if intLocalID > intRemoteID {
		intLocalID, intRemoteID = intRemoteID, intLocalID
	}

	return strconv.Itoa(intLocalID) + "-" + strconv.Itoa(intRemoteID)
}
