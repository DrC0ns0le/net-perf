package route

func (rm *RouteManager) GetCentralisedRoute(destination int) int {
	if _, ok := rm.CentralisedRouter.GetSiteRouteMap()[destination]; !ok {
		rm.logger.Errorf("route not found for destination %d", destination)
		return -1
	}
	return rm.CentralisedRouter.GetSiteRouteMap()[destination]
}

func (rm *RouteManager) UpdateLocalRoutes(routes map[int]int) {
	updated := rm.CentralisedRouter.UpdateSiteRoutes(routes)

	if updated {
		rm.rtUpdateCh <- struct{}{}
	}
}

func (rm *RouteManager) GetGraphRoute(destination int) int {
	if rm.Graph == nil {
		rm.logger.Errorf("graph is not initialized")
		return -1
	}
	path, _, err := rm.Graph.GetShortestPath(asToSiteID(rm.Config.ASNumber), destination)
	if err != nil {
		rm.logger.Errorf("error getting shortest path: %v", err)
	}

	return path[1]
}
