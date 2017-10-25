// Copyright © 2015 Hilko Bengen <bengen@hilluzination.de>. All rights reserved.
// Use of this source code is governed by the license that can be
// found in the LICENSE file.

package yara

// #cgo !windows,!no_pkg_config  pkg-config: --libs yara
// #cgo !windows,!no_pkg_config  pkg-config: --cflags yara
// #cgo windows no_pkg_config    LDFLAGS: -lyara
import "C"
