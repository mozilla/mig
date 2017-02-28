// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package libaudit

import (
	"strconv"
)

var bufferMap map[uint64][]*AuditEvent

// bufferEvent buffers an incoming audit event which contains partial record informatioa.
func bufferEvent(a *AuditEvent) {
	if bufferMap == nil {
		bufferMap = make(map[uint64][]*AuditEvent)
	}

	serial, err := strconv.ParseUint(a.Serial, 10, 64)
	if err != nil {
		return
	}
	if _, ok := bufferMap[serial]; !ok {
		bufferMap[serial] = make([]*AuditEvent, 5)
	}
	bufferMap[serial] = append(bufferMap[serial], a)
}

// bufferGet returns the complete audit event from the buffer, given the AUDIT_EOE event a.
func bufferGet(a *AuditEvent) *AuditEvent {
	serial, err := strconv.ParseUint(a.Serial, 10, 64)
	if err != nil {
		return nil
	}

	var (
		bm []*AuditEvent
		ok bool
	)
	if bm, ok = bufferMap[serial]; !ok {
		return nil
	}
	rlen := len(a.Raw)
	for i := range bm {
		if bm[i] == nil {
			continue
		}
		for k, v := range bm[i].Data {
			a.Data[k] = v
		}
		if len(bm[i].Raw) > rlen {
			a.Raw += " " + bm[i].Raw[rlen:]
		}
	}

	delete(bufferMap, serial)
	return a
}
