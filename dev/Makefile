SHELL = bash

files/id_ed25519:
	mkdir -p files
	ssh-keygen -t ed25519 -a 64 -N "" -f "$@"

files/config.toml: config.toml
ifndef RUNNER_TOKEN
	$(error RUNNER_TOKEN is not defined)
endif
ifndef SCW_ACCESS_KEY
	$(error SCW_ACCESS_KEY is not defined)
endif
ifndef SCW_SECRET_KEY
	$(error SCW_SECRET_KEY is not defined)
endif
ifndef SCW_ORGANIZATION_ID
	$(error SCW_ORGANIZATION_ID is not defined)
endif
ifndef SCW_PROJECT_ID
	$(error SCW_PROJECT_ID is not defined)
endif
ifndef SCW_DEFAULT_ZONE
	$(error SCW_DEFAULT_ZONE is not defined)
endif
	mkdir -p files
	envsubst '$$RUNNER_TOKEN,$$SCW_ACCESS_KEY,$$SCW_SECRET_KEY,$$SCW_ORGANIZATION_ID,$$SCW_PROJECT_ID,$$SCW_DEFAULT_ZONE' < config.toml > "$@"

up: files/id_ed25519 files/config.toml
	BUILDKIT_PROGRESS=plain docker compose up --build --detach

down:
	docker compose down
	bash destroy.sh

clean: down
	rm -Rf files/
