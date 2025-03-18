package route

func (rm *RouteManager) GetFullSiteRoutes() map[int]map[int]int {
	return rm.CentralisedRouter.GetFullRouteMap()
}

func (rm *RouteManager) GetSiteRoutes(site int) map[int]int {
	return rm.CentralisedRouter.GetSiteRouteMap(site)
}

func (rm *RouteManager) UpdateLocalRoutes(routes map[int]int) {
	updated := rm.CentralisedRouter.UpdateSiteRoutes(rm.siteID, routes)

	if updated {
		rm.rtUpdateCh <- struct{}{}
	}
}
