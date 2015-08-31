# This Source Code Form is subject to the terms of the Mozilla Public
# # License, v. 2.0. If a copy of the MPL was not distributed with this
# # file, You can obtain one at http://mozilla.org/MPL/2.0/.

RST2HTML	:= rst2html5

doc:
	for doc in $$(ls *.rst); do \
		$(RST2HTML) --stylesheet=docstyle.css "$$doc" > "$$doc.html"; \
	done
	for name in `ls ../modules/`; do \
		if [ -r "../modules/$$name/doc.rst" ]; then \
			$(RST2HTML) --stylesheet=docstyle.css "..//modules/$$name/doc.rst" > "module_$$name.html"; \
		fi; \
	done
	for name in `ls ../workers/`; do \
		if [ -r "../workers/$$name/doc.rst" ]; then \
			$(RST2HTML) --stylesheet=docstyle.css "../workers/$$name/doc.rst" > "$$name.html"; \
		fi; \
	done
	dot -Tsvg -o .files/action_command_flow.svg .files/action_command_flow.dot
