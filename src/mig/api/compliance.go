/* Mozilla InvestiGator API: Compliance parsing functions

Version: MPL 1.1/GPL 2.0/LGPL 2.1

The contents of this file are subject to the Mozilla Public License Version
1.1 (the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at
http://www.mozilla.org/MPL/

Software distributed under the License is distributed on an "AS IS" basis,
WITHOUT WARRANTY OF ANY KIND, either express or implied. See the License
for the specific language governing rights and limitations under the
License.

The Initial Developer of the Original Code is
Mozilla Corporation
Portions created by the Initial Developer are Copyright (C) 2013
the Initial Developer. All Rights Reserved.

Contributor(s):
Julien Vehent jvehent@mozilla.com [:ulfr]

Alternatively, the contents of this file may be used under the terms of
either the GNU General Public License Version 2 or later (the "GPL"), or
the GNU Lesser General Public License Version 2.1 or later (the "LGPL"),
in which case the provisions of the GPL or the LGPL are applicable instead
of those above. If you wish to allow use of your version of this file only
under the terms of either the GPL or the LGPL, and not to allow others to
use your version of this file under the terms of the MPL, indicate your
decision by deleting the provisions above and replace them with the notice
and other provisions required by the GPL or the LGPL. If you do not delete
the provisions above, a recipient may use your version of this file under
the terms of any one of the MPL, the GPL or the LGPL.
*/

package main

import (
	"encoding/json"
	"fmt"
	"mig"
	"mig/modules/filechecker"
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

const RFC3339Nano = "2006-01-02T15:04:05.999999999+07:00"

func commandsToComplianceItems(commands []mig.Command) (items []ComplianceItem, err error) {
	for _, cmd := range commands {
		var bitem ComplianceItem
		bitem.Utctimestamp = cmd.FinishTime.Format(RFC3339Nano)
		bitem.Target = cmd.Agent.Name
		bitem.Policy.Name = cmd.Action.Threat.Type
		bitem.Policy.URL = cmd.Action.Description.URL
		bitem.Policy.Level = cmd.Action.Threat.Level
		bitem.Check.Ref = cmd.Action.Threat.Ref
		bitem.Check.Description = cmd.Action.Name
		bitem.Link = fmt.Sprintf("%s/command?commandid=%d", ctx.Server.BaseURL, cmd.ID)
		for i, result := range cmd.Results {
			buf, err := json.Marshal(result)
			if err != nil {
				return items, err
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
