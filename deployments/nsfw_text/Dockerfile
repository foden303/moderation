FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy proto files and generate stubs
COPY proto/ proto/
# Compile proto files relative to project root so they end up in proto/ directory
RUN python -m grpc_tools.protoc \
    --proto_path=. \
    --python_out=. \
    --grpc_python_out=. \
    proto/nsfw_text.proto

# Copy application code
COPY server.py .

# gRPC port
EXPOSE 50052
# Ray dashboard
EXPOSE 8265

CMD ["python", "server.py"]
