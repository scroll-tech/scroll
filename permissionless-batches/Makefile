.PHONY: batch-production-submission launch_local_prover launch_cloud_prover psql check_proving_status

export SCROLL_ZKVM_VERSION=0.3.0
PG_URL=postgres://postgres@localhost:5432/scroll

batch_production_submission:
	docker compose --profile batch-production-submission up

local_prover:
	docker compose --profile local-prover up

cloud_prover:
	docker compose --profile cloud-prover up

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