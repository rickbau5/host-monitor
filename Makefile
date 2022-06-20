BINARY_DIR=bin
TAR=${BINARY_DIR}/${TAR_NAME}

CMD_DIR=./cmd
ARPMON=arpmon
ARPMON_CMD=${CMD_DIR}/${ARPMON}
ARPMON_BINARY=${BINARY_DIR}/${ARPMON}

SNIFFER=sniffer
SNIFFER_CMD=${CMD_DIR}/${SNIFFER}
SNIFFER_BINARY=${BINARY_DIR}/${SNIFFER}

SNIFFER2=sniffer2
SNIFFER2_CMD=${CMD_DIR}/${SNIFFER2}
SNIFFER2_BINARY=${BINARY_DIR}/${SNIFFER2}

BINARIES=${ARPMON_BINARY} ${SNIFFER_BINARY}
BINARY_NAMES=${ARPMON} ${SNIFFER}

TAR_NAME=host-monitor.tar.gz

# this depends on how the Raspberry Pi is set up
TARGET_ARGS=GOOS=linux GOARCH=arm CGO_ENABLED=0

package: build ${TAR}

.PHONY:
build: ${BINARIES}

${ARPMON_BINARY}: ${ARPMON_CMD} vendor
	${TARGET_ARGS} go build -mod=vendor -o $@ ./$<

${SNIFFER_BINARY}: ${SNIFFER_CMD} vendor
	${TARGET_ARGS} go build -mod=vendor -o $@ ./$<

${SNIFFER2_BINARY}: ${SNIFFER2_CMD} vendor
	${TARGET_ARGS} go build -mod=vendor -o $@ ./$<

.PHONY: vendor
vendor: vendor/vendor.txt
vendor/vendor.txt:
	go mod vendor

${TAR}: ${ARPMON_BINARY} ${SNIFFER_BINARY} ${SNIFFER_BINARY}
	pushd ${BINARY_DIR} && tar cvf ${TAR_NAME} ${BINARY_NAMES}; popd

.PHONY: clean
clean:
	go mod tidy
	@rm -rf ${BINARY_DIR}