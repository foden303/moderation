FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Compile proto -> Python stubs
COPY proto/ proto/
RUN python -m grpc_tools.protoc \
    --proto_path=proto \
    --python_out=. \
    --grpc_python_out=. \
    proto/nsfw.proto

# Download model at build time
RUN python -c "from transformers import AutoModelForImageClassification, AutoImageProcessor; \
    AutoImageProcessor.from_pretrained('Falconsai/nsfw_image_detection'); \
    AutoModelForImageClassification.from_pretrained('Falconsai/nsfw_image_detection')"

# Copy application
COPY server.py .

# gRPC port
EXPOSE 50051
# Ray dashboard (optional, for monitoring)
EXPOSE 8265

CMD ["python", "server.py"]
