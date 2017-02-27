package main

import (
	"os"
	"path"
	"runtime"

	"mig.ninja/mig"
)

func setPlatformDefaults(runOpt *runtimeOptions) {
	// platform fallback defaults
	runOpt.config = "/etc/mig/mig-agent.cfg"

	if runtime.GOOS == `windows` {
		setWindowsDefaults(runOpt)
	}
}

func setWindowsDefaults(runOpt *runtimeOptions) {
	root := os.Getenv(mig.Env_Win_Root)
	if root != "" && root != mig.Env_Win_Root_Defaut {
		// setup non default paths
		runOpt.config = path.Join(root, "mig", "mig-agent.cfg")
	}
	return
}
