// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.
//
// Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]
package main

import (
	"encoding/json"
	"fmt"
	"mig.ninja/mig"
	"mig.ninja/mig/modules"
	"mig.ninja/mig/modules/file"
	"time"
)

type ComplianceItem struct {
	Utctimestamp string           `json:"utctimestamp"`
	Target       string           `json:"target"`
	Policy       CompliancePolicy `json:"policy"`
	Check        ComplianceCheck  `json:"check"`
	Compliance   bool             `json:"compliance"`
	Link         string           `json:"link"`
	Tags         interface{}      `json:"tags"`
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

type ComplianceTags struct {
	Operator string `json:"operator"`
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
		if _, ok := cmd.Agent.Tags.(map[string]interface{})["operator"]; ok {
			var t ComplianceTags
			t.Operator = cmd.Agent.Tags.(map[string]interface{})["operator"].(string)
			bitem.Tags = t
		}
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
			case "file":
				var el file.SearchResults
				var r modules.Result
				err = json.Unmarshal(buf, &r)
				if err != nil {
					return items, err
				}
				err = r.GetElements(&el)
				if err != nil {
					return items, err
				}
				for label, sr := range el {
					for _, mf := range sr {
						bitem.Check.Location = mf.File
						bitem.Check.Name = label
						bitem.Check.Test.Type = "file"
						bitem.Check.Test.Value = ""
						for _, v := range mf.Search.Names {
							if len(bitem.Check.Test.Value) > 0 {
								bitem.Check.Test.Value += " and "
							}
							bitem.Check.Test.Value += fmt.Sprintf("name='%s'", v)
						}
						for _, v := range mf.Search.Sizes {
							if len(bitem.Check.Test.Value) > 0 {
								bitem.Check.Test.Value += " and "
							}
							bitem.Check.Test.Value += fmt.Sprintf("size='%s'", v)
						}
						for _, v := range mf.Search.Modes {
							if len(bitem.Check.Test.Value) > 0 {
								bitem.Check.Test.Value += " and "
							}
							bitem.Check.Test.Value += fmt.Sprintf("mode='%s'", v)
						}
						for _, v := range mf.Search.Mtimes {
							if len(bitem.Check.Test.Value) > 0 {
								bitem.Check.Test.Value += " and "
							}
							bitem.Check.Test.Value += fmt.Sprintf("mtime='%s'", v)
						}
						for _, v := range mf.Search.Contents {
							if len(bitem.Check.Test.Value) > 0 {
								bitem.Check.Test.Value += " and "
							}
							bitem.Check.Test.Value += fmt.Sprintf("content='%s'", v)
						}
						for _, v := range mf.Search.MD5 {
							if len(bitem.Check.Test.Value) > 0 {
								bitem.Check.Test.Value += " and "
							}
							bitem.Check.Test.Value += fmt.Sprintf("md5='%s'", v)
						}
						for _, v := range mf.Search.SHA1 {
							if len(bitem.Check.Test.Value) > 0 {
								bitem.Check.Test.Value += " and "
							}
							bitem.Check.Test.Value += fmt.Sprintf("sha1='%s'", v)
						}
						for _, v := range mf.Search.SHA2 {
							if len(bitem.Check.Test.Value) > 0 {
								bitem.Check.Test.Value += " and "
							}
							bitem.Check.Test.Value += fmt.Sprintf("sha2='%s'", v)
						}
						for _, v := range mf.Search.SHA3 {
							if len(bitem.Check.Test.Value) > 0 {
								bitem.Check.Test.Value += " and "
							}
							bitem.Check.Test.Value += fmt.Sprintf("sha3='%s'", v)
						}
						if mf.File == "" {
							for i, p := range mf.Search.Paths {
								if i > 0 {
									bitem.Check.Location += ", "
								}
								bitem.Check.Location += p
							}
							bitem.Compliance = false
						} else {
							bitem.Compliance = true
						}
						items = append(items, bitem)
					}
				}
			}
		}
	}
	return
}
