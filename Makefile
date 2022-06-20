BINARY_DIR=bin
TAR=${BINARY_DIR}/${TAR_NAME}

CMD_DIR=./cmd
ARPMON=arpmon
ARPMON_CMD=${CMD_DIR}/${ARPMON}
ARPMON_BINARY=${BINARY_DIR}/${ARPMON}

SNIFFER=sniffer
SNIFFER_CMD=${CMD_DIR}/${SNIFFER}
SNIFFER_BINARY=${BINARY_DIR}/${SNIFFER}

TAR_NAME=host-monitor.tar.gz

# this depends on how the Raspberry Pi is set up
TARGET_ARGS=GOOS=linux GOARCH=arm CGO_ENABLED=0

package: build ${TAR}

.PHONY:
build: ${ARPMON_BINARY}

${ARPMON_BINARY}: vendor
	${TARGET_ARGS} go build -mod=vendor -o ${ARPMON_BINARY} ${ARPMON_CMD}

${SNIFFER_BINARY}: vendor
	${TARGET_ARGS} go build -mod=vendor -o ${SNIFFER_BINARY} ${SNIFFER_CMD}

.PHONY: vendor
# vendor: vendor/vendor.txt
# vendor/vendor.txt:
# 	go mod vendor

${TAR}: ${ARPMON_BINARY} ${SNIFFER_BINARY}
	pushd ${BINARY_DIR} && tar cvf ${TAR_NAME} ${ARPMON} ${SNIFFER}; popd

.PHONY: clean
clean:
	go mod tidy
	@rm -rf ${BINARY_DIR}