#!/usr/bin/env python
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.
#
# Contributors:
# Alexandru Tudorica <tudalex@gmail.com>
# vladimirdiaconescu <vladimirdiaconescu@users.noreply.github.com>
# Teodora Baluta <teobaluta@gmail.com>

import subprocess

# Get the kernel release
p = subprocess.Popen(['uname', '-r'], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
out, err = p.communicate()
kernel_version = out.strip()#.split('-')[0]

# Open file
unistd32_contents = ''
with open('/usr/src/linux-headers-4.2.0-18-generic/arch/x86/include/generated/uapi/asm/unistd_32.h', 'r') as f:
    unistd32_contents = f.read()

split_contents_32 = unistd32_contents.split('\n')

# Construct a python dictionary
mappings = {}
reverse_mappings = {}
syscalls = []

for line in split_contents_32:
    if 'NR_syscalls' in line:
        break
    if '__NR' in line:
        syscalls.append(line)

# Properly parse
# syscalls[1] -> __NR_$NAME
# syscalls[2] -> number
for syscall in syscalls:
    syscall_num = syscall.split()[2]
    syscall_name = syscall.split()[1]
    try:
        mappings[int(syscall_num)] = syscall_name[5:]
        reverse_mappings[syscall_name] = int(syscall_num)
    # Some pesky macro needs expanding
    except ValueError:
        ref_name = syscall_num.split('(')[1].split('+')[0]
        ref_num = reverse_mappings[ref_name]
        syscall_num = syscall_num.replace(ref_name, str(ref_num))
        mappings[int(eval(syscall_num))] = syscall_name[5:]


print "const char * const syscall_mappings[] = {",
for i in range(366):
    try:
        print "\"" + mappings[i] + "\","
    except KeyError:
        print "\"unkown syscall %s\"," % i
print "};\n"
#from pprint import pformat
#print "package main\n"
#print "var idToSyscall = map[int]string" + pformat(mappings).replace("'",'"')
