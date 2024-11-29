package system

import "log"

func init() {
	err := configureInitSysctls()
	if err != nil {
		log.Panicf("failed to configure init sysctls: %v", err)
	}
}
