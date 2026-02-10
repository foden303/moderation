"""
NSFW Image Detection API using Falconsai/nsfw_image_detection
HuggingFace Model: https://huggingface.co/Falconsai/nsfw_image_detection

Run: uvicorn app:app --host 0.0.0.0 --port 8080
"""

import io
import logging
from typing import Optional

import httpx
from fastapi import FastAPI, File, UploadFile, HTTPException
from fastapi.responses import JSONResponse
from pydantic import BaseModel
from PIL import Image
import torch
from transformers import AutoModelForImageClassification, AutoImageProcessor

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Initialize FastAPI
app = FastAPI(
    title="NSFW Image Detection API",
    description="API for detecting NSFW content in images using Falconsai/nsfw_image_detection",
    version="1.0.0"
)

# Global model and processor
model = None
processor = None
device = None

MODEL_NAME = "Falconsai/nsfw_image_detection"


class PredictionResponse(BaseModel):
    is_nsfw: bool
    nsfw_score: float
    normal_score: float
    label: str
    confidence: float


class URLRequest(BaseModel):
    url: str


class HealthResponse(BaseModel):
    status: str
    model: str
    device: str


@app.on_event("startup")
async def load_model():
    """Load model on startup."""
    global model, processor, device
    
    logger.info(f"Loading model: {MODEL_NAME}")
    
    # Determine device
    if torch.cuda.is_available():
        device = torch.device("cuda")
        logger.info("Using CUDA GPU")
    elif hasattr(torch.backends, "mps") and torch.backends.mps.is_available():
        device = torch.device("mps")
        logger.info("Using Apple MPS")
    else:
        device = torch.device("cpu")
        logger.info("Using CPU")
    
    # Load model and processor
    processor = AutoImageProcessor.from_pretrained(MODEL_NAME)
    model = AutoModelForImageClassification.from_pretrained(MODEL_NAME)
    model.to(device)
    model.eval()
    
    logger.info("Model loaded successfully!")


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint."""
    if model is None:
        raise HTTPException(status_code=503, detail="Model not loaded")
    
    return HealthResponse(
        status="healthy",
        model=MODEL_NAME,
        device=str(device)
    )


def predict_image(image: Image.Image) -> PredictionResponse:
    """Core prediction logic for a PIL Image."""
    # Process image
    inputs = processor(images=image, return_tensors="pt")
    inputs = {k: v.to(device) for k, v in inputs.items()}
    
    # Inference
    with torch.no_grad():
        outputs = model(**inputs)
        logits = outputs.logits
        probs = torch.softmax(logits, dim=-1)[0]
    
    # Get predictions
    labels = model.config.id2label
    
    predicted_idx = probs.argmax().item()
    predicted_label = labels[predicted_idx]
    confidence = probs[predicted_idx].item()
    
    # Get individual scores
    nsfw_idx = None
    normal_idx = None
    for idx, label in labels.items():
        if "nsfw" in label.lower():
            nsfw_idx = idx
        else:
            normal_idx = idx
    
    nsfw_score = probs[nsfw_idx].item() if nsfw_idx is not None else 0.0
    normal_score = probs[normal_idx].item() if normal_idx is not None else 1.0
    
    return PredictionResponse(
        is_nsfw=predicted_label.lower() == "nsfw",
        nsfw_score=nsfw_score,
        normal_score=normal_score,
        label=predicted_label,
        confidence=confidence
    )


@app.post("/predict", response_model=PredictionResponse)
async def predict(file: UploadFile = File(...)):
    """
    Predict NSFW content in an uploaded image.
    
    Returns:
        - is_nsfw: True if image is NSFW
        - nsfw_score: Probability of NSFW (0.0-1.0)
        - normal_score: Probability of normal (0.0-1.0)
        - label: Predicted label ("nsfw" or "normal")
        - confidence: Confidence score of prediction
    """
    if model is None:
        raise HTTPException(status_code=503, detail="Model not loaded")
    
    # Validate content type
    if not file.content_type.startswith("image/"):
        raise HTTPException(status_code=400, detail="File must be an image")
    
    try:
        contents = await file.read()
        image = Image.open(io.BytesIO(contents)).convert("RGB")
        return predict_image(image)
    except Exception as e:
        logger.error(f"Prediction error: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/predict/url", response_model=PredictionResponse)
async def predict_from_url(request: URLRequest):
    """
    Predict NSFW content from a public image URL.
    
    Args:
        url: Public URL of the image
    
    Returns:
        - is_nsfw: True if image is NSFW
        - nsfw_score: Probability of NSFW (0.0-1.0)
        - normal_score: Probability of normal (0.0-1.0)
        - label: Predicted label ("nsfw" or "normal")
        - confidence: Confidence score of prediction
    """
    if model is None:
        raise HTTPException(status_code=503, detail="Model not loaded")
    
    try:
        # Download image from URL
        async with httpx.AsyncClient(timeout=30.0) as client:
            response = await client.get(request.url)
            response.raise_for_status()
            
            # Check content type
            content_type = response.headers.get("content-type", "")
            if not content_type.startswith("image/"):
                raise HTTPException(status_code=400, detail=f"URL does not point to an image: {content_type}")
            
            image = Image.open(io.BytesIO(response.content)).convert("RGB")
        
        return predict_image(image)
        
    except httpx.HTTPError as e:
        logger.error(f"Failed to download image: {e}")
        raise HTTPException(status_code=400, detail=f"Failed to download image: {str(e)}")
    except Exception as e:
        logger.error(f"Prediction error: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/predict/batch")
async def predict_batch(files: list[UploadFile] = File(...)):
    """Predict NSFW content for multiple images."""
    results = []
    for file in files:
        try:
            result = await predict(file)
            results.append({"filename": file.filename, **result.dict()})
        except HTTPException as e:
            results.append({"filename": file.filename, "error": e.detail})
    
    return {"predictions": results}


class BatchURLRequest(BaseModel):
    urls: list[str]


@app.post("/predict/batch/url")
async def predict_batch_from_urls(request: BatchURLRequest):
    """
    Predict NSFW content for multiple image URLs.
    
    Args:
        urls: List of public image URLs
    """
    if model is None:
        raise HTTPException(status_code=503, detail="Model not loaded")
    
    results = []
    async with httpx.AsyncClient(timeout=30.0) as client:
        for url in request.urls:
            try:
                response = await client.get(url)
                response.raise_for_status()
                
                content_type = response.headers.get("content-type", "")
                if not content_type.startswith("image/"):
                    results.append({"url": url, "error": f"Not an image: {content_type}"})
                    continue
                
                image = Image.open(io.BytesIO(response.content)).convert("RGB")
                pred = predict_image(image)
                results.append({"url": url, **pred.dict()})
                
            except httpx.HTTPError as e:
                results.append({"url": url, "error": f"Download failed: {str(e)}"})
            except Exception as e:
                results.append({"url": url, "error": str(e)})
    
    return {"predictions": results}


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8080)

