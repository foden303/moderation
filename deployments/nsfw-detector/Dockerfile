FROM python:3.11-slim

WORKDIR /app

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Download model at build time
RUN python -c "from transformers import AutoModelForImageClassification, AutoImageProcessor; \
    AutoImageProcessor.from_pretrained('Falconsai/nsfw_image_detection'); \
    AutoModelForImageClassification.from_pretrained('Falconsai/nsfw_image_detection')"

# Copy application
COPY app.py .

EXPOSE 8080

CMD ["uvicorn", "app:app", "--host", "0.0.0.0", "--port", "8080"]
