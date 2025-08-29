ifndef ROOTDIR
$(error Please run from package root)
endif

TARGET=nmapparse
VERSION=$(shell pwd)/license/version.go
SRC=$(TARGET).go 
OBJ=$(TARGET).so
BIN=$(TARGET)
INI=base.ini

ALL: $(BINDIR)/$(BIN) $(ETCDIR)/$(INI)

$(BINDIR)/$(BIN): $(BIN) $(INI)
	cp $(BIN) $(BINDIR)

$(ETCDIR)/$(INI): $(INI)
	cp $(INI) $(ETCDIR)

$(VERSION):
	$(GENVER) $(shell dirname $(VERSION))

$(BIN): $(SRC) $(VERSION)
	$(GO) build $(SRC)

clean:
	rm -f $(BIN) .last_commit $(VERSION)
