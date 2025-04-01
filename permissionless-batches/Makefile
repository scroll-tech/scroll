.PHONY: batch-production-submission launch_local_prover launch_cloud_prover

IMAGE_VERSION=latest
REPO_ROOT_DIR=./..

batch_production_submission:
	docker compose --profile batch-production-submission up

launch_local_prover:
	docker compose --profile cloud-prover up

launch_cloud_prover:
    docker compose --profile local-prover up