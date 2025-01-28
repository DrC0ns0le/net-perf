package pathping

import (
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
)

func GetAllPeers() (map[int][]int, error) {
	wgIfs, err := netctl.GetAllWGInterfacesFromConfig()
	if err != nil {
		return nil, err
	}

	peers := make(map[int][]int)
	for _, iface := range wgIfs {
		if _, ok := peers[iface.RemoteIDInt]; !ok {
			peers[iface.RemoteIDInt] = []int{iface.IPVersionInt}
		}
		peers[iface.RemoteIDInt] = append(peers[iface.RemoteIDInt], iface.IPVersionInt)
	}

	return peers, nil
}
