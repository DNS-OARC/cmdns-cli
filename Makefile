SOURCES := $(wildcard *.go)

all: cmdns-cli

fmt: format

clean:
	rm -f cmdns-cli

format:
	gofmt -w *.go
	sed -i -e 's%	%    %g' *.go

cmdns-cli: $(SOURCES)
	go build -v -x
