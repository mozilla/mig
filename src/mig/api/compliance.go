// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"encoding/json"
	"fmt"
	"mig"
	"mig/modules/filechecker"
	"time"
)

type ComplianceItem struct {
	Utctimestamp string           `json:"utctimestamp"`
	Target       string           `json:"target"`
	Policy       CompliancePolicy `json:"policy"`
	Check        ComplianceCheck  `json:"check"`
	Compliance   bool             `json:"compliance"`
	Link         string           `json:"link"`
}

type CompliancePolicy struct {
	Name  string `json:"name"`
	URL   string `json:"url"`
	Level string `json:"level"`
}

type ComplianceCheck struct {
	Ref         string         `json:"ref"`
	Description string         `json:"description"`
	Name        string         `json:"name"`
	Location    string         `json:"location"`
	Test        ComplianceTest `json:"test"`
}

type ComplianceTest struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func commandsToComplianceItems(commands []mig.Command) (items []ComplianceItem, err error) {
	for _, cmd := range commands {
		var bitem ComplianceItem
		bitem.Utctimestamp = cmd.FinishTime.UTC().Format(time.RFC3339Nano)
		bitem.Target = cmd.Agent.Name
		bitem.Policy.Name = cmd.Action.Threat.Type
		bitem.Policy.URL = cmd.Action.Description.URL
		bitem.Policy.Level = cmd.Action.Threat.Level
		bitem.Check.Ref = cmd.Action.Threat.Ref
		bitem.Check.Description = cmd.Action.Name
		bitem.Link = fmt.Sprintf("%s/command?commandid=%.0f", ctx.Server.BaseURL, cmd.ID)
		for i, result := range cmd.Results {
			buf, err := json.Marshal(result)
			if err != nil {
				return items, err
			}
			if i > (len(cmd.Action.Operations) - 1) {
				// skip this entry if the lookup fails
				continue
			}
			switch cmd.Action.Operations[i].Module {
			case "filechecker":
				var r filechecker.Results
				err = json.Unmarshal(buf, &r)
				if err != nil {
					return items, err
				}
				for path, _ := range r.Elements {
					bitem.Check.Location = path
					for method, _ := range r.Elements[path] {
						bitem.Check.Test.Type = method
						for id, _ := range r.Elements[path][method] {
							bitem.Check.Name = id
							for value, _ := range r.Elements[path][method][id] {
								bitem.Check.Test.Value = value
								if r.Elements[path][method][id][value].Matchcount > 0 {
									bitem.Compliance = true
								} else {
									bitem.Compliance = false
								}
								item := bitem
								items = append(items, item)
							}
						}
					}
				}
			}
		}
	}
	return
}
