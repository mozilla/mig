package headers

import "syscall"

// Location: include/uapi/linux/fs.h
// when updating look at printMount
var MountLookUp = map[int]string{
	syscall.MS_RDONLY:      "MS_RDONLY",
	syscall.MS_NOSUID:      "MS_NOSUID",
	syscall.MS_NODEV:       "MS_NODEV",
	syscall.MS_NOEXEC:      "MS_NOEXEC",
	syscall.MS_SYNCHRONOUS: "MS_SYNCHRONOUS",
	syscall.MS_REMOUNT:     "MS_REMOUNT",
	syscall.MS_MANDLOCK:    "MS_MANDLOCK",
	syscall.MS_DIRSYNC:     "MS_DIRSYNC",
	syscall.MS_NOATIME:     "MS_NOATIME",
	syscall.MS_NODIRATIME:  "MS_NODIRATIME",
	syscall.MS_BIND:        "MS_BIND",
	syscall.MS_MOVE:        "MS_MOVE",
	syscall.MS_REC:         "MS_REC",
	syscall.MS_SILENT:      "MS_SILENT",
	syscall.MS_POSIXACL:    "MS_POSIXACL",
	syscall.MS_UNBINDABLE:  "MS_UNBINDABLE",
	syscall.MS_PRIVATE:     "MS_PRIVATE",
	syscall.MS_SLAVE:       "MS_SLAVE",
	syscall.MS_SHARED:      "MS_SHARED",
	syscall.MS_RELATIME:    "MS_RELATIME",
	syscall.MS_KERNMOUNT:   "MS_KERNMOUNT",
	syscall.MS_I_VERSION:   "MS_I_VERSION",
	1 << 24:                "MS_STRICTATIME",
	1 << 27:                "MS_SNAP_STABLE",
	1 << 28:                "MS_NOSEC",
	1 << 29:                "MS_BORN",
	syscall.MS_ACTIVE:      "MS_ACTIVE",
	syscall.MS_NOUSER:      "MS_NOUSER",
}
