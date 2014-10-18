doc:
	for doc in $$(ls *.rst); do \
		rst2html5 --stylesheet=docstyle.css "$$doc" > "$$doc.html"; \
	done
	dot -Tsvg -o .files/action_command_flow.svg .files/action_command_flow.dot
