LOCATION = us-west1
PROJECT_ID = zerok-dev
REPOSITORY = stage

SERVER_VERSION = dev
SERVER_IMAGE = zk-wsp-server
SERVER_ART_Repo_URI = $(LOCATION)-docker.pkg.dev/$(PROJECT_ID)/$(REPOSITORY)/$(SERVER_IMAGE)
SERVER_IMG = $(SERVER_ART_Repo_URI):$(SERVER_VERSION)

CLIENT_VERSION = multiarch
CLIENT_IMAGE = zk-wsp-client
CLIENT_ART_Repo_URI = $(LOCATION)-docker.pkg.dev/$(PROJECT_ID)/$(REPOSITORY)/$(CLIENT_IMAGE)
CLIENT_IMG = $(CLIENT_ART_Repo_URI):$(CLIENT_VERSION)

BUILDER_NAME = multi-platform-builder


LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: build-server
build-server:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o wsp_server cmd/wsp_server/main.go
	docker build -t ${SERVER_IMG} . --build-arg APP_FILE=wsp_server
	docker push ${SERVER_IMG}

.PHONY: build-client
build-client:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o zk-wsp-client cmd/wsp_client/main.go
	docker build -f Dockerfile-Client -t ${CLIENT_IMG} .
	docker push ${CLIENT_IMG}

.PHONY: build-client-multiarch
build-client-multiarch:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/zk-wsp-client-amd64 cmd/wsp_client/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/zk-wsp-client-arm64 cmd/wsp_client/main.go
	#Adding remove here again to account for the case when buildx was not removed in previous run.
	docker buildx rm ${BUILDER_NAME} || true
	docker buildx create --use --platform=linux/arm64,linux/amd64 --name ${BUILDER_NAME}
	docker buildx build --platform=linux/arm64,linux/amd64 --push \
	--tag ${CLIENT_IMG} -f Dockerfile-Client .
	docker buildx rm ${BUILDER_NAME}

.PHONY: build-all
build-all: build-client build-server

.PHONY: install-server
install-server: kustomize
	cd k8s/server && $(KUSTOMIZE) edit set image wsp-server=${SERVER_IMG}
	kubectl apply -k k8s/server

.PHONY: uninstall-server
uninstall-server:
	kubectl delete --ignore-not-found=true -k k8s/server

.PHONY: install-client
install-client: kustomize
	cd k8s/client && $(KUSTOMIZE) edit set image wsp-client=${CLIENT_IMG}
	kubectl apply -k k8s/client

.PHONY: uninstall-client
uninstall-client:
	kubectl delete --ignore-not-found=true -k k8s/client

.PHONY: install-all
install-all: install-client install-server

.PHONY: uninstall-all
uninstall-all: uninstall-client uninstall-server

.PHONY: run-test-server
run-test-server:
	go run ./examples/test_api/main.go

# ------- CI-CD ------------
ci-cd-build-client:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o zk-wsp-client cmd/wsp_client/main.go

ci-cd-build-client-multiarch:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/zk-wsp-client-amd64 cmd/wsp_client/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/zk-wsp-client-arm64 cmd/wsp_client/main.go