HOST     ?= ti.sspaeti.duckdns.org
SSH      := ssh $(HOST)
SRC_DIR  := /home/sspaeti/survey-src
BIN      := /usr/local/bin/survey

# duckdb_use_lib dynamically links against the system libduckdb (1.5.3) on the
# FreeBSD host, since duckdb-go's v2 bindings ship pre-built libs for darwin/
# linux/windows only.
BUILD_FLAGS := -tags=duckdb_use_lib

.PHONY: test fmt vet sync build deploy logs status query help

help:
	@echo "Targets:"
	@echo "  test    - run unit tests locally"
	@echo "  fmt     - gofmt -w ."
	@echo "  vet     - go vet ./..."
	@echo "  sync    - rsync source to $(HOST)"
	@echo "  build   - sync, then build on $(HOST)"
	@echo "  deploy  - build, then atomically swap binary and restart service"
	@echo "  logs    - tail service log on $(HOST)"
	@echo "  status  - show service status on $(HOST)"
	@echo "  query   - print the Quack ATTACH snippet to paste into duckdb"

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

sync:
	rsync -az --delete \
	  --exclude build/ --exclude .git/ --exclude docs/ --exclude .DS_Store \
	  ./ $(HOST):$(SRC_DIR)/

build: sync
	$(SSH) "cd $(SRC_DIR) && \
	        CGO_CFLAGS='-I/usr/local/include' \
	        CGO_LDFLAGS='-L/usr/local/lib -lduckdb' \
	        go build $(BUILD_FLAGS) -o build/survey ./cmd/survey"

deploy: build
	$(SSH) "doas cp $(SRC_DIR)/build/survey $(BIN).new && \
	        doas mv $(BIN).new $(BIN) && \
	        doas service survey restart"

logs:
	$(SSH) "doas tail -f /var/log/survey/survey.log"

status:
	$(SSH) "doas service survey status"

query:
	@echo "Paste into duckdb on your laptop:"
	@echo ""
	@echo "  CREATE SECRET (TYPE quack, TOKEN '<token from /usr/local/etc/survey/survey.env>');"
	@echo "  ATTACH 'quack:quack.sspaeti.duckdns.org' AS s;"
	@echo "  FROM s.votes ORDER BY ts DESC LIMIT 20;"
