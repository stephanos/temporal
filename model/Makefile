.PHONY: check check-regression

REGRESSION_ID := nexus-caller-closure-upgrade
REGRESSION_INSPECTOR := temporal-experiment-inspect

check:
	mise exec -- lake build

check-regression:
	mise exec -- lake build ExperimentTests $(REGRESSION_INSPECTOR)
	@tmp_dir=$$(mktemp -d); \
	trap 'rm -rf "$$tmp_dir"' EXIT; \
	mise exec -- lake exe $(REGRESSION_INSPECTOR) $(REGRESSION_ID) > "$$tmp_dir/first.json"; \
	mise exec -- lake exe $(REGRESSION_INSPECTOR) $(REGRESSION_ID) > "$$tmp_dir/second.json"; \
	cmp -s "$$tmp_dir/first.json" "$$tmp_dir/second.json"; \
	if mise exec -- lake exe $(REGRESSION_INSPECTOR) missing-pilot \
		> "$$tmp_dir/negative.stdout" 2> "$$tmp_dir/negative.stderr"; then \
		echo "expected the inspector to reject an unknown pilot" >&2; \
		exit 1; \
	fi; \
	test ! -s "$$tmp_dir/negative.stdout"; \
	grep -Fx '{"kind":"unknownPilot","subject":"missing-pilot","context":"pilot registry"}' \
		"$$tmp_dir/negative.stderr" > /dev/null
