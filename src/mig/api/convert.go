/* Mozilla InvestiGator API

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
	"fmt"
	"github.com/jvehent/cljs"
	"mig"
)

// extendedActionToItem receives an ExtendedAction and return an Item
// in the Collection+JSON format
func extendedActionToItem(ea mig.ExtendedAction) (item cljs.Item, err error) {
	item.Href = "/api/action?actionid=" + fmt.Sprintf("%d", ea.Action.ID)
	links := make([]cljs.Link, 0)
	for _, cmdid := range ea.CommandIDs {
		link := cljs.Link{
			Rel:  "command",
			Href: "/api/command?actionid=" + fmt.Sprintf("%d", ea.Action.ID) + "&commandid=" + fmt.Sprintf("%d", cmdid),
		}
		links = append(links, link)
	}
	item.Links = links
	item.Data = []cljs.Data{
		{Name: "action", Value: ea},
	}
	return
}

// commandToItem receives a command and returns an Item in Collection+JSON
func commandToItem(cmd mig.Command) (item cljs.Item, err error) {
	item.Href = "/api/command?actionid=" + fmt.Sprintf("%d", cmd.Action.ID) + "&commandid=" + fmt.Sprintf("%d", cmd.ID)
	links := make([]cljs.Link, 0)
	link := cljs.Link{
		Rel:  "action",
		Href: "/api/action?actionid=" + fmt.Sprintf("%d", cmd.Action.ID),
	}
	links = append(links, link)
	item.Links = links
	item.Data = []cljs.Data{
		{Name: "command", Value: cmd},
	}
	return
}
