PROTOC=protoc
PROTO_DIR=proto
OUT_DIR=.

proto:
	python -m grpc_tools.protoc \
  -I. \
  --python_out=$(OUT_DIR) \
  --grpc_python_out=$(OUT_DIR) \
  $(PROTO_DIR)/*.proto
.PHONY: proto