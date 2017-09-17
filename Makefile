GO=go

COMPILE=$(GO) build
LDFLAGS="-s -w"
BUILD_FLAGS=-x -v -ldflags $(LDFLAGS)

GLIDE=glide
VENDOR_DIR=vendor

LINT=golint
LINT_FLAGS=-set_exit_status

EXE=mc
MAIN=$(wildcard ./marlowc/main.go)

COVERAGE=goverage
COVERAGE_REPORT=coverage.out

LIB_DIR=./marlow
SRC_DIR=./marlowc
EXAMPLE_DIR=./examples

LIB_SRC=$(wildcard $(LIB_DIR)/**/*.go $(LIB_DIR)/*.go)
GO_SRC=$(wildcard $(MAIN) $(SRC_DIR)/**/*.go $(SRC_DIR)/*.go)
EXAMPLE_OBJS=$(wildcard $(EXAMPLE_DIR)/library/**/*.marlow.go)

VET=$(GO) vet
VET_FLAGS=

MAX_TEST_CONCURRENCY=10
TEST_FLAGS=-covermode=atomic -coverprofile=.coverprofile 

all: $(EXE)

$(EXE): $(VENDOR_DIR) $(GO_SRC) $(LIB_SRC)
	$(COMPILE) $(BUILD_FLAGS) -o $(EXE) $(MAIN)

lint: $(GO_SRC)
	$(LINT) $(LINT_FLAGS) $(LIB_DIR)
	$(LINT) $(LINT_FLAGS) $(MAIN)

test: $(GO_SRC) $(VENDOR_DIR) $(INTERCHANGE_OBJ) lint
	$(VET) $(VET_FLAGS) $(SRC_DIR)
	$(VET) $(VET_FLAGS) $(LIB_DIR)
	$(VET) $(VET_FLAGS) $(MAIN)
	$(GO) test $(TEST_FLAGS) -p=$(MAX_TEST_CONCURRENCY) $(LIB_DIR)

$(VENDOR_DIR):
	go get -u github.com/Masterminds/glide
	go get -u github.com/golang/lint/golint
	$(GLIDE) install

clean-example:
	rm -rf $(EXAMPLE_OBJS)

clean: clean-example
	rm -rf $(COVERAGE_REPORT)
	rm -rf $(LINT_RESULT)
	rm -rf $(VENDOR_DIR)
	rm -rf $(EXE)
