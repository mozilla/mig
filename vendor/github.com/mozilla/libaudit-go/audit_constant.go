//audit_constant.go contains constants that are used within libaudit for indicating
//message types, field values etc.
//The list is compiled from linux/audit.h (from the kernel) and libaudit.h from the auditd source

package libaudit

const (
	MAX_AUDIT_MESSAGE_LENGTH = 8970
	AUDIT_MAX_FIELDS         = 64
	AUDIT_BITMASK_SIZE       = 64
	//Rule Flags
	AUDIT_FILTER_USER  = 0x00 /* Apply rule to user-generated messages */
	AUDIT_FILTER_TASK  = 0x01 /* Apply rule at task creation (not syscall) */
	AUDIT_FILTER_ENTRY = 0x02 /* Apply rule at syscall entry */
	AUDIT_FILTER_WATCH = 0x03 /* Apply rule to file system watches */
	AUDIT_FILTER_EXIT  = 0x04 /* Apply rule at syscall exit */
	AUDIT_FILTER_TYPE  = 0x05 /* Apply rule at audit_log_start */
	/* These are used in filter control */
	AUDIT_FILTER_MASK  = 0x07 /* Mask to get actual filter */
	AUDIT_FILTER_UNSET = 0x80 /* This value means filter is unset */

	/* Rule actions */
	AUDIT_NEVER    = 0 /* Do not build context if rule matches */
	AUDIT_POSSIBLE = 1 /* Build context if rule matches  */
	AUDIT_ALWAYS   = 2 /* Generate audit record if rule matches */

	/* Rule fields */
	/* These are useful when checking the
	 * task structure at task creation time
	 * (AUDIT_PER_TASK).  */
	AUDIT_PID                   = 0
	AUDIT_UID                   = 1
	AUDIT_EUID                  = 2
	AUDIT_SUID                  = 3
	AUDIT_FSUID                 = 4
	AUDIT_GID                   = 5
	AUDIT_EGID                  = 6
	AUDIT_SGID                  = 7
	AUDIT_FSGID                 = 8
	AUDIT_LOGINUID              = 9
	AUDIT_OBJ_GID               = 110
	AUDIT_OBJ_UID               = 109
	AUDIT_EXIT                  = 103
	AUDIT_PERS                  = 10
	AUDIT_FILTER_EXCLUDE        = 0x05
	AUDIT_ARCH                  = 11
	PATH_MAX                    = 4096
	AUDIT_MSGTYPE               = 12
	AUDIT_MAX_KEY_LEN           = 256
	AUDIT_PERM                  = 106
	AUDIT_FILTERKEY             = 210
	AUDIT_SUBJ_USER             = 13 /* security label user */
	AUDIT_SUBJ_ROLE             = 14 /* security label role */
	AUDIT_SUBJ_TYPE             = 15 /* security label type */
	AUDIT_SUBJ_SEN              = 16 /* security label sensitivity label */
	AUDIT_SUBJ_CLR              = 17 /* security label clearance label */
	AUDIT_PPID                  = 18
	AUDIT_OBJ_USER              = 19
	AUDIT_OBJ_ROLE              = 20
	AUDIT_OBJ_TYPE              = 21
	AUDIT_WATCH                 = 105
	AUDIT_DIR                   = 107
	AUDIT_OBJ_LEV_LOW           = 22
	AUDIT_OBJ_LEV_HIGH          = 23
	AUDIT_LOGINUID_SET          = 24
	AUDIT_DEVMAJOR              = 100
	AUDIT_DEVMINOR              = 101
	AUDIT_INODE                 = 102
	AUDIT_SUCCESS               = 104
	AUDIT_FIELD_COMPARE         = 111
	AUDIT_EXE                   = 112
	AUDIT_PERM_EXEC             = 1
	AUDIT_PERM_WRITE            = 2
	AUDIT_PERM_READ             = 4
	AUDIT_PERM_ATTR             = 8
	AUDIT_FILETYPE              = 108
	AUDIT_ARG0                  = 200
	AUDIT_ARG1                  = (AUDIT_ARG0 + 1)
	AUDIT_ARG2                  = (AUDIT_ARG0 + 2)
	AUDIT_ARG3                  = (AUDIT_ARG0 + 3)
	AUDIT_BIT_MASK              = 0x08000000
	AUDIT_LESS_THAN             = 0x10000000
	AUDIT_GREATER_THAN          = 0x20000000
	AUDIT_NOT_EQUAL             = 0x30000000
	AUDIT_EQUAL                 = 0x40000000
	AUDIT_BIT_TEST              = (AUDIT_BIT_MASK | AUDIT_EQUAL)
	AUDIT_LESS_THAN_OR_EQUAL    = (AUDIT_LESS_THAN | AUDIT_EQUAL)
	AUDIT_GREATER_THAN_OR_EQUAL = (AUDIT_GREATER_THAN | AUDIT_EQUAL)
	AUDIT_OPERATORS             = (AUDIT_EQUAL | AUDIT_NOT_EQUAL | AUDIT_BIT_MASK)
	/* Status symbols */
	/* Mask values */
	AUDIT_STATUS_ENABLED       = 0x0001
	AUDIT_STATUS_FAILURE       = 0x0002
	AUDIT_STATUS_PID           = 0x0004
	AUDIT_STATUS_RATE_LIMIT    = 0x0008
	AUDIT_STATUS_BACKLOG_LIMIT = 0x0010
	/* Failure-to-log actions */
	AUDIT_FAIL_SILENT = 0
	AUDIT_FAIL_PRINTK = 1
	AUDIT_FAIL_PANIC  = 2

	/* distinguish syscall tables */
	__AUDIT_ARCH_64BIT  = 0x80000000
	__AUDIT_ARCH_LE     = 0x40000000
	AUDIT_ARCH_ALPHA    = (EM_ALPHA | __AUDIT_ARCH_64BIT | __AUDIT_ARCH_LE)
	AUDIT_ARCH_ARM      = (EM_ARM | __AUDIT_ARCH_LE)
	AUDIT_ARCH_ARMEB    = (EM_ARM)
	AUDIT_ARCH_CRIS     = (EM_CRIS | __AUDIT_ARCH_LE)
	AUDIT_ARCH_FRV      = (EM_FRV)
	AUDIT_ARCH_I386     = (EM_386 | __AUDIT_ARCH_LE)
	AUDIT_ARCH_IA64     = (EM_IA_64 | __AUDIT_ARCH_64BIT | __AUDIT_ARCH_LE)
	AUDIT_ARCH_M32R     = (EM_M32R)
	AUDIT_ARCH_M68K     = (EM_68K)
	AUDIT_ARCH_MIPS     = (EM_MIPS)
	AUDIT_ARCH_MIPSEL   = (EM_MIPS | __AUDIT_ARCH_LE)
	AUDIT_ARCH_MIPS64   = (EM_MIPS | __AUDIT_ARCH_64BIT)
	AUDIT_ARCH_MIPSEL64 = (EM_MIPS | __AUDIT_ARCH_64BIT | __AUDIT_ARCH_LE)
	//	AUDIT_ARCH_OPENRISC = (EM_OPENRISC)
	//	AUDIT_ARCH_PARISC   = (EM_PARISC)
	//	AUDIT_ARCH_PARISC64 = (EM_PARISC | __AUDIT_ARCH_64BIT)
	AUDIT_ARCH_PPC     = (EM_PPC)
	AUDIT_ARCH_PPC64   = (EM_PPC64 | __AUDIT_ARCH_64BIT)
	AUDIT_ARCH_S390    = (EM_S390)
	AUDIT_ARCH_S390X   = (EM_S390 | __AUDIT_ARCH_64BIT)
	AUDIT_ARCH_SH      = (EM_SH)
	AUDIT_ARCH_SHEL    = (EM_SH | __AUDIT_ARCH_LE)
	AUDIT_ARCH_SH64    = (EM_SH | __AUDIT_ARCH_64BIT)
	AUDIT_ARCH_SHEL64  = (EM_SH | __AUDIT_ARCH_64BIT | __AUDIT_ARCH_LE)
	AUDIT_ARCH_SPARC   = (EM_SPARC)
	AUDIT_ARCH_SPARC64 = (EM_SPARCV9 | __AUDIT_ARCH_64BIT)
	AUDIT_ARCH_X86_64  = (EM_X86_64 | __AUDIT_ARCH_64BIT | __AUDIT_ARCH_LE)
	///Temporary Solution need to add linux/elf-em.h
	EM_NONE  = 0
	EM_M32   = 1
	EM_SPARC = 2
	EM_386   = 3
	EM_68K   = 4
	EM_88K   = 5
	EM_486   = 6 /* Perhaps disused */
	EM_860   = 7
	EM_MIPS  = 8 /* MIPS R3000 (officially, big-endian only) */
	/* Next two are historical and binaries and
	   modules of these types will be rejected by
	   Linux.  */
	EM_MIPS_RS3_LE = 10 /* MIPS R3000 little-endian */
	EM_MIPS_RS4_BE = 10 /* MIPS R4000 big-endian */

	EM_PARISC      = 15     /* HPPA */
	EM_SPARC32PLUS = 18     /* Sun's "v8plus" */
	EM_PPC         = 20     /* PowerPC */
	EM_PPC64       = 21     /* PowerPC64 */
	EM_SPU         = 23     /* Cell BE SPU */
	EM_ARM         = 40     /* ARM 32 bit */
	EM_SH          = 42     /* SuperH */
	EM_SPARCV9     = 43     /* SPARC v9 64-bit */
	EM_IA_64       = 50     /* HP/Intel IA-64 */
	EM_X86_64      = 62     /* AMD x86-64 */
	EM_S390        = 22     /* IBM S/390 */
	EM_CRIS        = 76     /* Axis Communications 32-bit embedded processor */
	EM_V850        = 87     /* NEC v850 */
	EM_M32R        = 88     /* Renesas M32R */
	EM_MN10300     = 89     /* Panasonic/MEI MN10300, AM33 */
	EM_BLACKFIN    = 106    /* ADI Blackfin Processor */
	EM_TI_C6000    = 140    /* TI C6X DSPs */
	EM_AARCH64     = 183    /* ARM 64 bit */
	EM_FRV         = 0x5441 /* Fujitsu FR-V */
	EM_AVR32       = 0x18ad /* Atmel AVR32 */

	/*
	 * This is an interim value that we will use until the committee comes
	 * up with a final number.
	 */
	EM_ALPHA = 0x9026

	/* Bogus old v850 magic number, used by old tools. */
	EM_CYGNUS_V850 = 0x9080
	/* Bogus old m32r magic number, used by old tools. */
	EM_CYGNUS_M32R = 0x9041
	/* This is the old interim value for S/390 architecture */
	EM_S390_OLD = 0xA390
	/* Also Panasonic/MEI MN10300, AM33 */
	EM_CYGNUS_MN10300 = 0xbeef
	//AUDIT_ARCH determination purpose
	_UTSNAME_LENGTH          = 65
	_UTSNAME_DOMAIN_LENGTH   = _UTSNAME_LENGTH
	_UTSNAME_NODENAME_LENGTH = _UTSNAME_DOMAIN_LENGTH
)

/* Audit message types as of 2.6.29 kernel:
 * 1000 - 1099 are for commanding the audit system
 * 1100 - 1199 user space trusted application messages
 * 1200 - 1299 messages internal to the audit daemon
 * 1300 - 1399 audit event messages
 * 1400 - 1499 kernel SE Linux use
 * 1500 - 1599 AppArmor events
 * 1600 - 1699 kernel crypto events
 * 1700 - 1799 kernel anomaly records
 * 1800 - 1899 kernel integrity labels and related events
 * 1800 - 1999 future kernel use
 * 2001 - 2099 unused (kernel)
 * 2100 - 2199 user space anomaly records
 * 2200 - 2299 user space actions taken in response to anomalies
 * 2300 - 2399 user space generated LSPP events
 * 2400 - 2499 user space crypto events
 * 2500 - 2599 user space virtualization management events
 * 2600 - 2999 future user space (maybe integrity labels and related events)
 */

//go:generate stringer -type=auditConstant
type auditConstant uint16

const (
	AUDIT_GET         auditConstant = 1000 /* Get status */
	AUDIT_SET         auditConstant = 1001 /* Set status (enable/disable/auditd) */
	AUDIT_LIST        auditConstant = 1002 /* List syscall rules -- deprecated */
	AUDIT_ADD         auditConstant = 1003 /* Add syscall rule -- deprecated */
	AUDIT_DEL         auditConstant = 1004 /* Delete syscall rule -- deprecated */
	AUDIT_USER        auditConstant = 1005 /* Message from userspace -- deprecated */
	AUDIT_LOGIN       auditConstant = 1006 /* Define the login id and information */
	AUDIT_WATCH_INS   auditConstant = 1007 /* Insert file/dir watch entry */
	AUDIT_WATCH_REM   auditConstant = 1008 /* Remove file/dir watch entry */
	AUDIT_WATCH_LIST  auditConstant = 1009 /* List all file/dir watches */
	AUDIT_SIGNAL_INFO auditConstant = 1010 /* Get info about sender of signal to auditd */
	AUDIT_ADD_RULE    auditConstant = 1011 /* Add syscall filtering rule */
	AUDIT_DEL_RULE    auditConstant = 1012 /* Delete syscall filtering rule */
	AUDIT_LIST_RULES  auditConstant = 1013 /* List syscall filtering rules */
	AUDIT_TRIM        auditConstant = 1014 /* Trim junk from watched tree */
	AUDIT_MAKE_EQUIV  auditConstant = 1015 /* Append to watched tree */
	AUDIT_TTY_GET     auditConstant = 1016 /* Get TTY auditing status */
	AUDIT_TTY_SET     auditConstant = 1017 /* Set TTY auditing status */
	AUDIT_SET_FEATURE auditConstant = 1018 /* Turn an audit feature on or off */
	AUDIT_GET_FEATURE auditConstant = 1019 /* Get which features are enabled */

	AUDIT_FIRST_USER_MSG   auditConstant = 1100 /* First user space message */
	AUDIT_LAST_USER_MSG    auditConstant = 1199 /* Last user space message */
	AUDIT_USER_AUTH        auditConstant = 1100 /* User space authentication */
	AUDIT_USER_ACCT        auditConstant = 1101 /* User space acct change */
	AUDIT_USER_MGMT        auditConstant = 1102 /* User space acct management */
	AUDIT_CRED_ACQ         auditConstant = 1103 /* User space credential acquired */
	AUDIT_CRED_DISP        auditConstant = 1104 /* User space credential disposed */
	AUDIT_USER_START       auditConstant = 1105 /* User space session start */
	AUDIT_USER_END         auditConstant = 1106 /* User space session end */
	AUDIT_USER_AVC         auditConstant = 1107 /* User space avc message */
	AUDIT_USER_CHAUTHTOK   auditConstant = 1108 /* User space acct attr changed */
	AUDIT_USER_ERR         auditConstant = 1109 /* User space acct state err */
	AUDIT_CRED_REFR        auditConstant = 1110 /* User space credential refreshed */
	AUDIT_USYS_CONFIG      auditConstant = 1111 /* User space system config change */
	AUDIT_USER_LOGIN       auditConstant = 1112 /* User space user has logged in */
	AUDIT_USER_LOGOUT      auditConstant = 1113 /* User space user has logged out */
	AUDIT_ADD_USER         auditConstant = 1114 /* User space user account added */
	AUDIT_DEL_USER         auditConstant = 1115 /* User space user account deleted */
	AUDIT_ADD_GROUP        auditConstant = 1116 /* User space group added */
	AUDIT_DEL_GROUP        auditConstant = 1117 /* User space group deleted */
	AUDIT_DAC_CHECK        auditConstant = 1118 /* User space DAC check results */
	AUDIT_CHGRP_ID         auditConstant = 1119 /* User space group ID changed */
	AUDIT_TEST             auditConstant = 1120 /* Used for test success messages */
	AUDIT_TRUSTED_APP      auditConstant = 1121 /* Trusted app msg - freestyle text */
	AUDIT_USER_SELINUX_ERR auditConstant = 1122 /* SE Linux user space error */
	AUDIT_USER_CMD         auditConstant = 1123 /* User shell command and args */
	AUDIT_USER_TTY         auditConstant = 1124 /* Non-ICANON TTY input meaning */
	AUDIT_CHUSER_ID        auditConstant = 1125 /* Changed user ID supplemental data */
	AUDIT_GRP_AUTH         auditConstant = 1126 /* Authentication for group password */
	AUDIT_SYSTEM_BOOT      auditConstant = 1127 /* System boot */
	AUDIT_SYSTEM_SHUTDOWN  auditConstant = 1128 /* System shutdown */
	AUDIT_SYSTEM_RUNLEVEL  auditConstant = 1129 /* System runlevel change */
	AUDIT_SERVICE_START    auditConstant = 1130 /* Service (daemon) start */
	AUDIT_SERVICE_STOP     auditConstant = 1131 /* Service (daemon) stop */
	AUDIT_GRP_MGMT         auditConstant = 1132 /* Group account attr was modified */
	AUDIT_GRP_CHAUTHTOK    auditConstant = 1133 /* Group acct password or pin changed */
	AUDIT_MAC_CHECK        auditConstant = 1134 /* User space MAC decision results */
	AUDIT_ACCT_LOCK        auditConstant = 1135 /* User's account locked by admin */
	AUDIT_ACCT_UNLOCK      auditConstant = 1136 /* User's account unlocked by admin */

	AUDIT_FIRST_DAEMON    auditConstant = 1200
	AUDIT_LAST_DAEMON     auditConstant = 1299
	AUDIT_DAEMON_CONFIG   auditConstant = 1203 /* Daemon config change */
	AUDIT_DAEMON_RECONFIG auditConstant = 1204 /* Auditd should reconfigure */
	AUDIT_DAEMON_ROTATE   auditConstant = 1205 /* Auditd should rotate logs */
	AUDIT_DAEMON_RESUME   auditConstant = 1206 /* Auditd should resume logging */
	AUDIT_DAEMON_ACCEPT   auditConstant = 1207 /* Auditd accepted remote connection */
	AUDIT_DAEMON_CLOSE    auditConstant = 1208 /* Auditd closed remote connection */

	AUDIT_SYSCALL auditConstant = 1300 /* Syscall event */
	/* AUDIT_FS_WATCH     auditConstant       =   1301     * Deprecated */
	AUDIT_PATH           auditConstant = 1302 /* Filename path information */
	AUDIT_IPC            auditConstant = 1303 /* IPC record */
	AUDIT_SOCKETCALL     auditConstant = 1304 /* sys_socketcall arguments */
	AUDIT_CONFIG_CHANGE  auditConstant = 1305 /* Audit system configuration change */
	AUDIT_SOCKADDR       auditConstant = 1306 /* sockaddr copied as syscall arg */
	AUDIT_CWD            auditConstant = 1307 /* Current working directory */
	AUDIT_EXECVE         auditConstant = 1309 /* execve arguments */
	AUDIT_IPC_SET_PERM   auditConstant = 1311 /* IPC new permissions record type */
	AUDIT_MQ_OPEN        auditConstant = 1312 /* POSIX MQ open record type */
	AUDIT_MQ_SENDRECV    auditConstant = 1313 /* POSIX MQ send/receive record type */
	AUDIT_MQ_NOTIFY      auditConstant = 1314 /* POSIX MQ notify record type */
	AUDIT_MQ_GETSETATTR  auditConstant = 1315 /* POSIX MQ get/set attribute record type */
	AUDIT_KERNEL_OTHER   auditConstant = 1316 /* For use by 3rd party modules */
	AUDIT_FD_PAIR        auditConstant = 1317 /* audit record for pipe/socketpair */
	AUDIT_OBJ_PID        auditConstant = 1318 /* ptrace target */
	AUDIT_TTY            auditConstant = 1319 /* Input on an administrative TTY */
	AUDIT_EOE            auditConstant = 1320 /* End of multi-record event */
	AUDIT_BPRM_FCAPS     auditConstant = 1321 /* Information about fcaps increasing perms */
	AUDIT_CAPSET         auditConstant = 1322 /* Record showing argument to sys_capset */
	AUDIT_MMAP           auditConstant = 1323 /* Record showing descriptor and flags in mmap */
	AUDIT_NETFILTER_PKT  auditConstant = 1324 /* Packets traversing netfilter chains */
	AUDIT_NETFILTER_CFG  auditConstant = 1325 /* Netfilter chain modifications */
	AUDIT_SECCOMP        auditConstant = 1326 /* Secure Computing event */
	AUDIT_PROCTITLE      auditConstant = 1327 /* Proctitle emit event */
	AUDIT_FEATURE_CHANGE auditConstant = 1328 /* audit log listing feature changes */

	/* AUDIT_FIRST_EVENT       1300 */ //TODO: libaudit define this as AUDIT_FIRST_EVENT but audit.h differently.
	AUDIT_LAST_EVENT                   auditConstant = 1399

	/* AUDIT_FIRST_SELINUX     1400 */ // TODO: libaudit define this as AUDIT_FIRST_SELINUX but audit.h as AUDIT_AVC
	AUDIT_AVC                          auditConstant = 1400 /* SE Linux avc denial or grant */
	AUDIT_SELINUX_ERR                  auditConstant = 1401 /* internal SE Linux Errors */
	AUDIT_AVC_PATH                     auditConstant = 1402 /* dentry, vfsmount pair from avc */
	AUDIT_MAC_POLICY_LOAD              auditConstant = 1403 /* Policy file load */
	AUDIT_MAC_STATUS                   auditConstant = 1404 /* Changed enforcing,permissive,off */
	AUDIT_MAC_CONFIG_CHANGE            auditConstant = 1405 /* Changes to booleans */
	AUDIT_MAC_UNLBL_ALLOW              auditConstant = 1406 /* NetLabel: allow unlabeled traffic */
	AUDIT_MAC_CIPSOV4_ADD              auditConstant = 1407 /* NetLabel: add CIPSOv4 DOI entry */
	AUDIT_MAC_CIPSOV4_DEL              auditConstant = 1408 /* NetLabel: del CIPSOv4 DOI entry */
	AUDIT_MAC_MAP_ADD                  auditConstant = 1409 /* NetLabel: add LSM domain mapping */
	AUDIT_MAC_MAP_DEL                  auditConstant = 1410 /* NetLabel: del LSM domain mapping */
	AUDIT_MAC_IPSEC_ADDSA              auditConstant = 1411 /* Not used */
	AUDIT_MAC_IPSEC_DELSA              auditConstant = 1412 /* Not used  */
	AUDIT_MAC_IPSEC_ADDSPD             auditConstant = 1413 /* Not used */
	AUDIT_MAC_IPSEC_DELSPD             auditConstant = 1414 /* Not used */
	AUDIT_MAC_IPSEC_EVENT              auditConstant = 1415 /* Audit an IPSec event */
	AUDIT_MAC_UNLBL_STCADD             auditConstant = 1416 /* NetLabel: add a static label */
	AUDIT_MAC_UNLBL_STCDEL             auditConstant = 1417 /* NetLabel: del a static label */
	AUDIT_LAST_SELINUX                 auditConstant = 1499

	AUDIT_FIRST_APPARMOR auditConstant = 1500
	AUDIT_LAST_APPARMOR  auditConstant = 1599

	AUDIT_AA               auditConstant = 1500 /* Not upstream yet*/
	AUDIT_APPARMOR_AUDIT   auditConstant = 1501
	AUDIT_APPARMOR_ALLOWED auditConstant = 1502
	AUDIT_APPARMOR_DENIED  auditConstant = 1503
	AUDIT_APPARMOR_HT      auditConstant = 1504
	AUDIT_APPARMOR_STATUS  auditConstant = 1505
	AUDIT_APPARMOR_ERROR   auditConstant = 1506

	AUDIT_FIRST_KERN_CRYPTO_MSG auditConstant = 1600
	AUDIT_LAST_KERN_CRYPTO_MSG  auditConstant = 1699

	// AUDIT_FIRST_KERN_ANOM_MSG auditConstant = 1700
	AUDIT_LAST_KERN_ANOM_MSG auditConstant = 1799
	AUDIT_ANOM_PROMISCUOUS   auditConstant = 1700 /* Device changed promiscuous mode */
	AUDIT_ANOM_ABEND         auditConstant = 1701 /* Process ended abnormally */
	AUDIT_ANOM_LINK          auditConstant = 1702 /* Suspicious use of file links */

	AUDIT_INTEGRITY_FIRST_MSG auditConstant = 1800
	AUDIT_TINTEGRITY_LAST_MSG auditConstant = 1899

	AUDIT_INTEGRITY_DATA     auditConstant = 1800 /* Data integrity verification */
	AUDIT_INTEGRITY_METADATA auditConstant = 1801 // Metadata integrity verification
	AUDIT_INTEGRITY_STATUS   auditConstant = 1802 /* integrity enable status */
	AUDIT_INTEGRITY_HASH     auditConstant = 1803 /* integrity HASH type */
	AUDIT_INTEGRITY_PCR      auditConstant = 1804 /* PCR invalidation msgs */
	AUDIT_INTEGRITY_RULE     auditConstant = 1805 /* Policy rule */
	AUDIT_KERNEL             auditConstant = 2000 /* Asynchronous audit record. NOT A REQUEST. */

	AUDIT_FIRST_ANOM_MSG           auditConstant = 2100
	AUDIT_LAST_ANOM_MSG            auditConstant = 2199
	AUDIT_ANOM_LOGIN_FAILURES      auditConstant = 2100 // Failed login limit reached
	AUDIT_ANOM_LOGIN_TIME          auditConstant = 2101 // Login attempted at bad time
	AUDIT_ANOM_LOGIN_SESSIONS      auditConstant = 2102 // Max concurrent sessions reached
	AUDIT_ANOM_LOGIN_ACCT          auditConstant = 2103 // Login attempted to watched acct
	AUDIT_ANOM_LOGIN_LOCATION      auditConstant = 2104 // Login from forbidden location
	AUDIT_ANOM_MAX_DAC             auditConstant = 2105 // Max DAC failures reached
	AUDIT_ANOM_MAX_MAC             auditConstant = 2106 // Max MAC failures reached
	AUDIT_ANOM_AMTU_FAIL           auditConstant = 2107 // AMTU failure
	AUDIT_ANOM_RBAC_FAIL           auditConstant = 2108 // RBAC self test failure
	AUDIT_ANOM_RBAC_INTEGRITY_FAIL auditConstant = 2109 // RBAC file Tegrity failure
	AUDIT_ANOM_CRYPTO_FAIL         auditConstant = 2110 // Crypto system test failure
	AUDIT_ANOM_ACCESS_FS           auditConstant = 2111 // Access of file or dir
	AUDIT_ANOM_EXEC                auditConstant = 2112 // Execution of file
	AUDIT_ANOM_MK_EXEC             auditConstant = 2113 // Make an executable
	AUDIT_ANOM_ADD_ACCT            auditConstant = 2114 // Adding an acct
	AUDIT_ANOM_DEL_ACCT            auditConstant = 2115 // Deleting an acct
	AUDIT_ANOM_MOD_ACCT            auditConstant = 2116 // Changing an acct
	AUDIT_ANOM_ROOT_TRANS          auditConstant = 2117 // User became root

	AUDIT_FIRST_ANOM_RESP        auditConstant = 2200
	AUDIT_LAST_ANOM_RESP         auditConstant = 2299
	AUDIT_RESP_ANOMALY           auditConstant = 2200 /* Anomaly not reacted to */
	AUDIT_RESP_ALERT             auditConstant = 2201 /* Alert email was sent */
	AUDIT_RESP_KILL_PROC         auditConstant = 2202 /* Kill program */
	AUDIT_RESP_TERM_ACCESS       auditConstant = 2203 /* Terminate session */
	AUDIT_RESP_ACCT_REMOTE       auditConstant = 2204 /* Acct locked from remote access*/
	AUDIT_RESP_ACCT_LOCK_TIMED   auditConstant = 2205 /* User acct locked for time */
	AUDIT_RESP_ACCT_UNLOCK_TIMED auditConstant = 2206 /* User acct unlocked from time */
	AUDIT_RESP_ACCT_LOCK         auditConstant = 2207 /* User acct was locked */
	AUDIT_RESP_TERM_LOCK         auditConstant = 2208 /* Terminal was locked */
	AUDIT_RESP_SEBOOL            auditConstant = 2209 /* Set an SE Linux boolean */
	AUDIT_RESP_EXEC              auditConstant = 2210 /* Execute a script */
	AUDIT_RESP_SINGLE            auditConstant = 2211 /* Go to single user mode */
	AUDIT_RESP_HALT              auditConstant = 2212 /* take the system down */

	AUDIT_FIRST_USER_LSPP_MSG    auditConstant = 2300
	AUDIT_LAST_USER_LSPP_MSG     auditConstant = 2399
	AUDIT_USER_ROLE_CHANGE       auditConstant = 2300 /* User changed to a new role */
	AUDIT_ROLE_ASSIGN            auditConstant = 2301 /* Admin assigned user to role */
	AUDIT_ROLE_REMOVE            auditConstant = 2302 /* Admin removed user from role */
	AUDIT_LABEL_OVERRIDE         auditConstant = 2303 /* Admin is overriding a label */
	AUDIT_LABEL_LEVEL_CHANGE     auditConstant = 2304 /* Object's level was changed */
	AUDIT_USER_LABELED_EXPORT    auditConstant = 2305 /* Object exported with label */
	AUDIT_USER_UNLABELED_EXPORT  auditConstant = 2306 /* Object exported without label */
	AUDIT_DEV_ALLOC              auditConstant = 2307 /* Device was allocated */
	AUDIT_DEV_DEALLOC            auditConstant = 2308 /* Device was deallocated */
	AUDIT_FS_RELABEL             auditConstant = 2309 /* Filesystem relabeled */
	AUDIT_USER_MAC_POLICY_LOAD   auditConstant = 2310 /* Userspc daemon loaded policy */
	AUDIT_ROLE_MODIFY            auditConstant = 2311 /* Admin modified a role */
	AUDIT_USER_MAC_CONFIG_CHANGE auditConstant = 2312 /* Change made to MAC policy */

	AUDIT_FIRST_CRYPTO_MSG         auditConstant = 2400
	AUDIT_CRYPTO_TEST_USER         auditConstant = 2400 /* Crypto test results */
	AUDIT_CRYPTO_PARAM_CHANGE_USER auditConstant = 2401 /* Crypto attribute change */
	AUDIT_CRYPTO_LOGIN             auditConstant = 2402 /* Logged in as crypto officer */
	AUDIT_CRYPTO_LOGOUT            auditConstant = 2403 /* Logged out from crypto */
	AUDIT_CRYPTO_KEY_USER          auditConstant = 2404 /* Create,delete,negotiate */
	AUDIT_CRYPTO_FAILURE_USER      auditConstant = 2405 /* Fail decrypt,encrypt,randomiz */
	AUDIT_CRYPTO_REPLAY_USER       auditConstant = 2406 /* Crypto replay detected */
	AUDIT_CRYPTO_SESSION           auditConstant = 2407 /* Record parameters set during TLS session establishment */
	AUDIT_CRYPTO_IKE_SA            auditConstant = 2408 /* Record parameters related to IKE SA */
	AUDIT_CRYPTO_IPSEC_SA          auditConstant = 2409 /* Record parameters related to IPSEC SA */

	AUDIT_LAST_CRYPTO_MSG auditConstant = 2499

	AUDIT_FIRST_VIRT_MSG  auditConstant = 2500
	AUDIT_VIRT_CONTROL    auditConstant = 2500 /* Start, Pause, Stop VM */
	AUDIT_VIRT_RESOURCE   auditConstant = 2501 /* Resource assignment */
	AUDIT_VIRT_MACHINE_ID auditConstant = 2502 /* Binding of label to VM */
	AUDIT_LAST_VIRT_MSG   auditConstant = 2599
	AUDIT_LAST_USER_MSG2  auditConstant = 2999
	// Field Comparing Constants
	AUDIT_COMPARE_UID_TO_OBJ_UID   auditConstant = 1
	AUDIT_COMPARE_GID_TO_OBJ_GID   auditConstant = 2
	AUDIT_COMPARE_EUID_TO_OBJ_UID  auditConstant = 3
	AUDIT_COMPARE_EGID_TO_OBJ_GID  auditConstant = 4
	AUDIT_COMPARE_AUID_TO_OBJ_UID  auditConstant = 5
	AUDIT_COMPARE_SUID_TO_OBJ_UID  auditConstant = 6
	AUDIT_COMPARE_SGID_TO_OBJ_GID  auditConstant = 7
	AUDIT_COMPARE_FSUID_TO_OBJ_UID auditConstant = 8
	AUDIT_COMPARE_FSGID_TO_OBJ_GID auditConstant = 9
	AUDIT_COMPARE_UID_TO_AUID      auditConstant = 10
	AUDIT_COMPARE_UID_TO_EUID      auditConstant = 11
	AUDIT_COMPARE_UID_TO_FSUID     auditConstant = 12
	AUDIT_COMPARE_UID_TO_SUID      auditConstant = 13
	AUDIT_COMPARE_AUID_TO_FSUID    auditConstant = 14
	AUDIT_COMPARE_AUID_TO_SUID     auditConstant = 15
	AUDIT_COMPARE_AUID_TO_EUID     auditConstant = 16
	AUDIT_COMPARE_EUID_TO_SUID     auditConstant = 17
	AUDIT_COMPARE_EUID_TO_FSUID    auditConstant = 18
	AUDIT_COMPARE_SUID_TO_FSUID    auditConstant = 19
	AUDIT_COMPARE_GID_TO_EGID      auditConstant = 20
	AUDIT_COMPARE_GID_TO_FSGID     auditConstant = 21
	AUDIT_COMPARE_GID_TO_SGID      auditConstant = 22
	AUDIT_COMPARE_EGID_TO_FSGID    auditConstant = 23
	AUDIT_COMPARE_EGID_TO_SGID     auditConstant = 24
	AUDIT_COMPARE_SGID_TO_FSGID    auditConstant = 25
)
