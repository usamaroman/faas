BIN=bin

export GOBIN=$(PWD)/$(BIN)

bin_dir:
	mkdir -p $(BIN)

goose: bin_dir
	rm -f $(BIN)/goose && go install github.com/pressly/goose/v3/cmd/goose@latest

install_all_deps: goose

build_all:
	go work sync
	go build -o bin/control_plane ./control_plane/cmd
	go build -o bin/invoicer ./invoicer/cmd
	go build -o bin/notifier ./notifier/cmd
	go build -o bin/meter ./meter/cmd
	go build -o bin/meter_agent ./meter_agent/cmd
	go build -o bin/price_service ./price_service/cmd

setup:
	bash setup.sh