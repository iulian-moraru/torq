dbContainer = timescale/timescaledb:latest-pg14
testDbPort = 5433
backendTest = go test ./... -v -count=1
frontendTest = cd web && npm i --legacy-peer-deps && npm test -- --watchAll=false
stopDevDb = ($(MAKE) stop-dev-db && false)
lint = cd web && npm i --legacy-peer-deps && npm run lint

.PHONY: test
test: lint start-dev-db wait-db test-backend-with-db-stop test-frontend-with-db-stop stop-dev-db
	@echo All tests pass!

.PHONY: test-backend-with-db-stop
test-backend-with-db-stop:
	$(backendTest) || $(stopDevDb)

.PHONY: test-frontend-with-db-stop
test-frontend-with-db-stop:
	$(frontendTest) || (cd ../ && $(stopDevDb))

.PHONY: test-backend
test-backend:
	$(backendTest)

.PHONY: test-frontend
test-frontend:
	$(frontendTest)

.PHONY: start-dev-db
start-dev-db:
	docker run -d --rm --name testdb -p $(testDbPort):5432 -e POSTGRES_PASSWORD=password $(dbContainer) \
	|| $(stopDevDb)

.PHONY: stop-dev-db
stop-dev-db:
	docker stop testdb

.PHONY: wait-db
wait-db:
	until docker run \
	--rm \
	--link testdb:pg \
	$(dbContainer) pg_isready \
		-U postgres \
		-h pg; do sleep 1; done

.PHONY: cover
cover:
	go test ./... -coverprofile cover.out && go tool cover -html=cover.out

.PHONY: build-docker
build-docker:
	docker build . -t $(TAG)

.PHONY: test-e2e
test-e2e:
	E2E=true go test -timeout 20m -v -count=1 ./test/e2e/lnd

.PHONY: test-e2e-debug
test-e2e-debug:
	E2E=true DEBUG=true go test -timeout 20m -v -count=1 ./test/e2e/lnd

.PHONY: create-dev-env
create-dev-env:
	go build ./virtual_network/torq_vn && go run ./virtual_network/torq_vn create --db true

.PHONY: start-dev-env
start-dev-env:
	go build ./virtual_network/torq_vn &&  go run ./virtual_network/torq_vn start --db true

.PHONY: stop-dev-env
stop-dev-env:
	go build ./virtual_network/torq_vn &&  go run ./virtual_network/torq_vn stop --db true

.PHONY: purge-dev-env
purge-dev-env:
	go build ./virtual_network/torq_vn &&  go run ./virtual_network/torq_vn purge --db true

.PHONY: lint
lint:
	$(lint)
