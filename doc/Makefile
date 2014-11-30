doc:
	for doc in $$(ls *.rst); do \
		rst2html5 --stylesheet=docstyle.css "$$doc" > "$$doc.html"; \
	done
	for modname in $(ls ../src/mig/modules/); do \
		if [ -r "../src/mig/modules/$modname/doc.rst" ]; then \
			rst2html5 --stylesheet=docstyle.css "../src/mig/modules/$modname/doc.rst" > "module_$modname.html"; \
		fi; \
	done
	dot -Tsvg -o .files/action_command_flow.svg .files/action_command_flow.dot
