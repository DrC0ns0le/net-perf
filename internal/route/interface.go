package route

func (rm *RouteManager) GetSiteRoutes(site int) map[int]int {
	return rm.CentralisedRouter.GetSiteRouteMap(site)
}

func (rm *RouteManager) UpdateLocalRoutes(routes map[int]int) {
	updated := rm.CentralisedRouter.UpdateSiteRoutes(routes)

	if updated {
		rm.rtUpdateCh <- struct{}{}
	}
}
