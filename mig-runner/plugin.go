// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Aaron Meihm ameihm@mozilla.com [:alm]
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mig.ninja/mig"
	"os/exec"
	"path"
)

var pluginList []plugin

type plugin struct {
	name string
	path string
}

func runPlugin(r mig.RunnerResult) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("runPlugin() -> %v", e)
		}
	}()

	var pent *plugin
	for i := range pluginList {
		if pluginList[i].name == r.UsePlugin {
			pent = &pluginList[i]
			break
		}
	}
	if pent == nil {
		panic("unable to locate plugin")
	}

	go func() {
		buf, err := json.Marshal(r)
		if err != nil {
			mlog("%v: %v", pent.name, err)
			return
		}
		c := exec.Command(pent.path)
		stdin, err := c.StdinPipe()
		if err != nil {
			mlog("%v: %v", pent.name, err)
			return
		}
		err = c.Start()
		if err != nil {
			mlog("%v: %v", pent.name, err)
			return
		}
		wb, err := stdin.Write(buf)
		if err != nil {
			mlog("%v: %v", pent.name, err)
		}
		stdin.Close()
		err = c.Wait()
		if err != nil {
			mlog("%v: %v", pent.name, err)
		}
		mlog("%v: wrote %v bytes to plugin", pent.name, wb)
	}()

	return nil
}

func loadPlugins() (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("loadPlugins() -> %v", e)
		}
	}()

	// Identify any available output plugins
	dirents, err := ioutil.ReadDir(ctx.Runner.PluginDirectory)
	if err != nil {
		panic(err)
	}
	for _, pluginEnt := range dirents {
		pluginName := pluginEnt.Name()
		pluginMode := pluginEnt.Mode()
		if (pluginMode & 0111) == 0 {
			mlog("plugins: skipping %v (not executable)", pluginName)
			continue
		}
		ppath := path.Join(ctx.Runner.PluginDirectory, pluginName)
		mlog("plugins: registering %v", pluginName)
		np := plugin{
			name: pluginName,
			path: ppath,
		}
		pluginList = append(pluginList, np)
	}

	return nil
}
