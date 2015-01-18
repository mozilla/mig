#!/usr/bin/env python

# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at http://mozilla.org/MPL/2.0/.
#
# Contributor: Julien Vehent jvehent@mozilla.com [:ulfr]

# This script connects to the management api of rabbitmq to list scheduler
# queues that have no consumer, indicating that the agent has timed out and
# the scheduler stopped consuming that queue. The script then finds a corresponding
# agent queue and, if it has no consumer either, deletes both queues.
#
# rabbitmq admin credentials can be passed via environment variables:
# $ export RABBITMQ_ADMINUSER="admin"
# $ export RABBITMQ_ADMINPASSWD="password12345"
# $ python delete_unused_queues.py

import requests
import json
import sys
import re
import os

def main():
	mqadmin = os.environ['RABBITMQ_ADMINUSER']
	mqpasswd = os.environ['RABBITMQ_ADMINPASSWD']
	r = requests.get('http://localhost:15672/api/queues/mig', auth=(mqadmin, mqpasswd))
	if r.status_code != 200:
		print('request failed')
		sys.exit(1)
	for queue in r.json():
		if queue['consumers'] == 0:
			try:
				agtq = re.search('mig.sched.(.+?)$', queue['name']).group(1)
			except AttributeError:
				continue
			r2 = requests.get('http://localhost:15672/api/queues/mig/mig.agt.'+agtq,
				auth=(mqadmin, mqpasswd))
			if r2.status_code != 200:
				continue
			if r2.json()['consumers'] == 0:
				print('%s has no consumer' % (agtq))
			for loc in ['agt', 'sched']:
				r3 = requests.delete('http://localhost:15672/api/queues/mig/mig.'+loc+'.'+agtq,
					auth=(mqadmin, mqpasswd))
				if r3.status_code == 204:
					print('deleted mig.'+loc+'.'+agtq)
				else:
					print('failed to delete mig.'+loc+'.'+agtq)

if __name__ == "__main__":
	main()
