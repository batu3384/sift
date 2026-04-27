package tui

func (m appModel) routeView() string {
	switch m.route {
	case RouteHome:
		m.home.width, m.home.height = m.width, m.height
		return m.home.View()
	case RouteClean:
		m.clean.width, m.clean.height = m.width, m.height
		m.cleanFlow.width, m.cleanFlow.height = m.width, m.height
		return m.cleanFlow.View()
	case RouteTools:
		m.tools.width, m.tools.height = m.width, m.height
		return m.tools.View()
	case RouteProtect:
		m.protect.width, m.protect.height = m.width, m.height
		return m.protect.View()
	case RouteUninstall:
		m.uninstall.width, m.uninstall.height = m.width, m.height
		m.uninstallFlow.width, m.uninstallFlow.height = m.width, m.height
		return m.uninstallFlow.View(m.uninstall)
	case RouteStatus:
		m.status.width, m.status.height = m.width, m.height
		return m.status.View()
	case RouteDoctor:
		m.doctor.width, m.doctor.height = m.width, m.height
		return m.doctor.View()
	case RouteAnalyze:
		m.analyze.width, m.analyze.height = m.width, m.height
		m.analyzeFlow.width, m.analyzeFlow.height = m.width, m.height
		return m.analyzeFlow.View(m.analyze)
	case RouteReview:
		m.review.width, m.review.height = m.width, m.height
		return m.review.View()
	case RoutePreflight:
		m.preflight.width, m.preflight.height = m.width, m.height
		return m.preflight.View()
	case RouteProgress:
		m.progress.width, m.progress.height = m.width, m.height
		if m.reducedMotion {
			m.progress.spinnerFrame = 0
		} else {
			m.progress.spinnerFrame = m.spinnerFrame
		}
		return m.progress.View()
	case RouteResult:
		m.result.width, m.result.height = m.width, m.height
		if m.reducedMotion {
			m.result.spinnerFrame = 0
		} else {
			m.result.spinnerFrame = m.spinnerFrame
		}
		return m.result.View()
	default:
		return "Unknown route"
	}
}
