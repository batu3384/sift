package tui

import (
	"github.com/batu3384/sift/internal/config"
	"github.com/batu3384/sift/internal/domain"
)

func newAppModel(opts AppOptions, callbacks AppCallbacks) appModel {
	model := appModel{
		route:                      opts.InitialRoute,
		cfg:                        config.Normalize(opts.Config),
		executable:                 opts.Executable,
		hasHome:                    opts.InitialRoute == RouteHome,
		reducedMotion:              opts.ReducedMotion,
		keys:                       defaultKeyMap(),
		help:                       newHelpModel(),
		callbacks:                  callbacks,
		permissionWarmup:           defaultPermissionWarmupCmd,
		permissionKeepalive:        defaultPermissionKeepalive,
		acceptedPermissionProfiles: map[string]struct{}{},
		home: homeModel{
			actions:    buildHomeActions(opts.Config),
			executable: opts.Executable,
			cfg:        config.Normalize(opts.Config),
		},
		clean: menuModel{
			title:    "Clean",
			subtitle: "choose scope",
			hint:     "Quick for routine cleanup, workstation for cache-heavy days, deep for maximum reclaim.",
			actions:  buildCleanActions(),
		},
		cleanFlow: newCleanFlowModel(),
		tools: menuModel{
			title:    "More Tools",
			subtitle: "more tools",
			hint:     "Check, fixes, installer cleanup, protect, purge, and diagnostics live here.",
			actions:  buildToolsActions(config.Normalize(opts.Config)),
		},
		protect:       newProtectModel(config.Normalize(opts.Config).ProtectedPaths),
		uninstall:     newUninstallModel(),
		uninstallFlow: newUninstallFlowModel(),
		analyzeFlow:   newAnalyzeFlowModel(),
	}
	model.syncMotionSettings()
	model.protect.syncFamilies(model.cfg.ProtectedFamilies)
	model.protect.syncScopes(model.cfg.CommandExcludes)
	switch opts.InitialRoute {
	case RouteHome:
		model.setHomeLoading("dashboard")
	case RouteStatus:
		model.setStatusLoading("dashboard")
	case RouteDoctor:
		model.setDoctorLoading("doctor")
	case RouteUninstall:
		model.setUninstallLoading("installed apps")
	}
	seedInitialPlanAndResult(&model, opts.InitialPlan, opts.InitialResult)
	return model
}

func seedInitialPlanAndResult(model *appModel, plan *domain.ExecutionPlan, result *domain.ExecutionResult) {
	if plan != nil {
		switch model.route {
		case RouteAnalyze:
			model.setAnalyzePlan(*plan)
		case RouteReview:
			model.setReviewPlan(*plan, shouldExecutePlan(*plan))
		}
	}
	if result == nil {
		return
	}
	model.result = resultModel{result: *result}
	if plan != nil {
		model.result.plan = *plan
	}
}
