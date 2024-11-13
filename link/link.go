package link

import (
	"log"
	"os/exec"
	"strconv"
	"time"

	"github.com/DrC0ns0le/net-perf/metrics"
	"github.com/DrC0ns0le/net-perf/utils"
)

var (
	localID     string
	justStarted = true
)

func init() {
	enableAsymmetricRoute()
}

func Start() {
	id, err := utils.GetLocalID()
	if err != nil {
		panic(err)
	}

	localID = id
	startWorker()
}

func startWorker() {
	log.Println("starting worker")
	mustUpdateRoutes()
	justStarted = false
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			go mustUpdateRoutes()
		}
	}
}

func mustUpdateRoutes() {

	ifaces, err := utils.GetAllWGInterfaces()
	if err != nil {
		log.Panicf("failed to get interfaces: %v", err)
	}

	var remoteVersionMap = map[string][]string{}

	for _, iface := range ifaces {
		remoteVersionMap[iface.RemoteID] = append(remoteVersionMap[iface.RemoteID], iface.IPVersion)
	}

	lID, err := strconv.Atoi(localID)
	if err != nil {
		log.Printf("failed to convert local ID to int: %v", err)
	}

	for remote, versions := range remoteVersionMap {
		var version string
		var reason string
		if len(versions) > 1 {

			rID, err := strconv.Atoi(remote)
			if err != nil {
				log.Printf("failed to convert remote ID to int: %v", err)
			}

			version, _, reason, err = metrics.GetPreferredVersion(lID, rID)
			if err != nil {
				log.Printf("failed to determine preferred version: %v", err)
			}

			if justStarted {
				version = "4"
			} else if version == "" {
				// log.Printf("neither v4 nor v6 are preferred for %s, skipping", remote)
				continue
			}
		} else {
			version = versions[0]
		}
		// check which interface is being used
		iface := utils.GetOutgoingWGInterface(remote)

		if iface == "" {
			log.Printf("route for %s not found in the routing table, adding route via %s", remote, "wg"+localID+"."+remote+"_v"+version)
			// run "ip route add 10.201.{remote}.0/24 dev wg{local}.{remote}_v{version} scope link src 10.201.{local}.1"
			err := exec.Command("ip", "route", "add", "10.201."+remote+".0/24", "dev", "wg"+localID+"."+remote+"_v"+version, "scope", "link", "src", "10.201."+localID+".1").Run()
			if err != nil {
				log.Printf("Error executing command: %v\n", err)
			}
			// run "ip -6 route add fdac:c9:{remote}::/64 dev wg{local}.{remote}_v{version} scope link"
			err = exec.Command("ip", "-6", "route", "add", "fdac:c9:"+remote+"::/64", "dev", "wg"+localID+"."+remote+"_v"+version, "scope", "link").Run()
			if err != nil {
				log.Printf("Error executing command: %v\n", err)
			}
		} else if iface != "wg"+localID+"."+remote+"_v"+version {
			// run "ip route change 10.201.{remote}.0/24 dev wg{local}.{remote}_v{version} scope link"
			log.Printf("changing route for %s from %s to %s due to %s", remote, iface, "wg"+localID+"."+remote+"_v"+version, reason)
			err := exec.Command("ip", "route", "change", "10.201."+remote+".0/24", "dev", "wg"+localID+"."+remote+"_v"+version, "scope", "link", "src", "10.201."+localID+".1").Run()
			if err != nil {
				log.Printf("Error executing command: %v\n", err)
			}
			// run "ip -6 route change fdac:c9:{remote}::/64 dev wg{local}.{remote}_v{version} scope link"
			err = exec.Command("ip", "-6", "route", "change", "fdac:c9:"+remote+"::/64", "dev", "wg"+localID+"."+remote+"_v"+version, "scope", "link").Run()
			if err != nil {
				log.Printf("Error executing command: %v\n", err)
			}
		} else {
			// log.Printf("route for %s already set to preferred interface %s", remote, iface)
		}
	}
}
