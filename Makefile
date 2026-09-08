PREFIX ?= /usr/local
BINDIR = $(PREFIX)/bin

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

LDFLAGS = -ldflags "-X gotcode.org/tally/internal/version.Version=$(VERSION) -X gotcode.org/tally/internal/version.Commit=$(COMMIT) -X gotcode.org/tally/internal/version.Branch=$(BRANCH)"

.PHONY: all build install clean

all: build

build:
	go build $(LDFLAGS) -o tally ./cmd/tally

install: build
	install -d $(DESTDIR)$(BINDIR)
	install -m 755 tally $(DESTDIR)$(BINDIR)/tally

clean:
	rm -f tally
