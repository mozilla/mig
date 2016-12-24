package headers

// Location:       include/uapi/linux/kd.h
//	include/uapi/linux/cdrom.h
//	include/uapi/asm-generic/ioctls.h
//	include/uapi/drm/drm.h

var IoctlLookup = map[int]string{
	0x4B3A:     "KDSETMODE",
	0x4B3B:     "KDGETMODE",
	0x5309:     "CDROMEJECT",
	0x530F:     "CDROMEJECT_SW",
	0x5311:     "CDROM_GET_UPC",
	0x5316:     "CDROMSEEK",
	0x5401:     "TCGETS",
	0x5402:     "TCSETS",
	0x5403:     "TCSETSW",
	0x5404:     "TCSETSF",
	0x5409:     "TCSBRK",
	0x540B:     "TCFLSH",
	0x540E:     "TIOCSCTTY",
	0x540F:     "TIOCGPGRP",
	0x5410:     "TIOCSPGRP",
	0x5413:     "TIOCGWINSZ",
	0x5414:     "TIOCSWINSZ",
	0x541B:     "TIOCINQ",
	0x5421:     "FIONBIO",
	0x8901:     "FIOSETOWN",
	0x8903:     "FIOGETOWN",
	0x8910:     "SIOCGIFNAME",
	0x8927:     "SIOCGIFHWADDR",
	0x8933:     "SIOCGIFINDEX",
	0x89a2:     "SIOCBRADDIF",
	0x40045431: "TIOCSPTLCK",
	0x80045430: "TIOCGPTN",
	0x80045431: "TIOCSPTLCK",
	0xC01C64A3: "DRM_IOCTL_MODE_CURSOR",
	0xC01864B0: "DRM_IOCTL_MODE_PAGE_FLIP",
	0xC01864B1: "DRM_IOCTL_MODE_DIRTYFB"}
