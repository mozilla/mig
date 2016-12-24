package headers

// Location: include/uapi/asm-generic/mman.h  >0x100
//           include/uapi/asm-generic/mman-common.h < 0x100
var MmapLookUp = map[int]string{
	0x00001: "MAP_SHARED",
	0x00002: "MAP_PRIVATE",
	0x00010: "MAP_FIXED",
	0x00020: "MAP_ANONYMOUS",
	0x00040: "MAP_32BIT",
	0x00100: "MAP_GROWSDOWN",
	0x00800: "MAP_DENYWRITE",
	0x01000: "MAP_EXECUTABLE",
	0x02000: "MAP_LOCKED",
	0x04000: "MAP_NORESERVE",
	0x08000: "MAP_POPULATE",
	0x10000: "MAP_NONBLOCK",
	0x20000: "MAP_STACK",
	0x40000: "MAP_HUGETLB",
}
