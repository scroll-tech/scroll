.PHONY: batch_production_submission launch_prover psql check_proving_status

export SCROLL_ZKVM_VERSION=0.4.2
PG_URL=postgres://postgres@localhost:5432/scroll

batch_production_submission:
	docker compose --profile batch-production-submission up

launch_prover:
	docker compose up -d

psql:
	psql 'postgres://postgres@localhost:5432/scroll'

check_proving_status:
	@echo "Checking proving status..."
	@result=$$(psql "${PG_URL}" -t -c "SELECT proving_status = 4 AS is_status_success FROM batch ORDER BY index LIMIT 1;" | tr -d '[:space:]'); \
	if [ "$$result" = "t" ]; then \
		echo "✅ Prove succeeded! You're ready to submit permissionless batch and proof!"; \
	else \
		echo "Proof is not ready..."; \
	fi