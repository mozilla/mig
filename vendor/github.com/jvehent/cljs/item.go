/* Go module for Collection+JSON

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
Portions created by the Initial Developer are Copyright (C) 2014
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

package cljs

import (
	"fmt"
)

type Item struct {
	Href  string `json:"href"`            //required
	Data  []Data `json:"data,omitempty"`  //optional
	Links []Link `json:"links,omitempty"` //optional
}

// AddItem inserts a new Item inside a Resource. It takes
// care of the allocation if needed
func (r *Resource) AddItem(item Item) (err error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	var tmpitems []Item
	tmpitems = r.Collection.Items
	tmpitems = append(tmpitems, item)
	r.Collection.Items = tmpitems
	return
}

func (item Item) Validate() (err error) {
	if item.Href == "" {
		return fmt.Errorf("'href' attribute is empty")
	}
	for i, data := range item.Data {
		err = data.Validate()
		if err != nil {
			return fmt.Errorf("failed to validate data entry %d: %v", i, err)
		}
	}
	for i, link := range item.Links {
		err = link.Validate()
		if err != nil {
			return fmt.Errorf("failed to validate link entry %d: %v", i, err)
		}
	}
	return
}
