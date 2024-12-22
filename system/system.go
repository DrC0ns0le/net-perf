package system

import (
	"log"

	"github.com/DrC0ns0le/net-perf/system/tunables"
)

func init() {
	err := tunables.Init()
	if err != nil {
		log.Panicf("failed to configure init sysctls: %v", err)
	}
}
