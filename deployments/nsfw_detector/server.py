"""
NSFW Image Detection with Ray Serve + gRPC
HuggingFace Model: https://huggingface.co/Falconsai/nsfw_image_detection

Replaces the FastAPI version with:
  - Ray Serve for model serving (auto-scaling, batching)
  - gRPC for transport (binary protocol, strongly typed)

Run: python server.py
"""

import io
import asyncio
import logging
from concurrent import futures
from typing import List

import httpx
import grpc
import ray
from ray import serve
import torch
from PIL import Image
from transformers import AutoModelForImageClassification, AutoImageProcessor

import nsfw_pb2
import nsfw_pb2_grpc

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("nsfw_detector")

MODEL_NAME = "Falconsai/nsfw_image_detection"
GRPC_PORT = 50051


# ---------------------------------------------------------------------------
# Ray Serve Deployment – wraps the HuggingFace model
# ---------------------------------------------------------------------------
@serve.deployment(
    num_replicas=1,
    ray_actor_options={"num_cpus": 1, "num_gpus": 0.5},
)
class NSFWModel:
    """Ray Serve deployment that loads the NSFW detection model."""

    def __init__(self):
        logger.info(f"Loading model: {MODEL_NAME}")

        # Determine device
        if torch.cuda.is_available():
            self.device = torch.device("cuda")
            logger.info("Using CUDA GPU")
        elif hasattr(torch.backends, "mps") and torch.backends.mps.is_available():
            self.device = torch.device("mps")
            logger.info("Using Apple MPS")
        else:
            self.device = torch.device("cpu")
            logger.info("Using CPU")

        # Load model and processor
        self.processor = AutoImageProcessor.from_pretrained(MODEL_NAME)
        self.model = AutoModelForImageClassification.from_pretrained(MODEL_NAME)
        self.model.to(self.device)
        self.model.eval()
        logger.info("Model loaded successfully!")

    def predict(self, image: Image.Image) -> dict:
        """Run inference on a single PIL image and return scores."""
        inputs = self.processor(images=image, return_tensors="pt")
        inputs = {k: v.to(self.device) for k, v in inputs.items()}

        with torch.no_grad():
            outputs = self.model(**inputs)
            logits = outputs.logits
            probs = torch.softmax(logits, dim=-1)[0]

        labels = self.model.config.id2label
        predicted_idx = probs.argmax().item()
        predicted_label = labels[predicted_idx]
        confidence = probs[predicted_idx].item()

        nsfw_idx = None
        normal_idx = None
        for idx, label in labels.items():
            if "nsfw" in label.lower():
                nsfw_idx = idx
            else:
                normal_idx = idx

        nsfw_score = probs[nsfw_idx].item() if nsfw_idx is not None else 0.0
        normal_score = probs[normal_idx].item() if normal_idx is not None else 1.0

        return {
            "is_nsfw": predicted_label.lower() == "nsfw",
            "nsfw_score": nsfw_score,
            "normal_score": normal_score,
            "label": predicted_label,
            "confidence": confidence,
        }

    @serve.batch(max_batch_size=8, batch_wait_timeout_s=0.1)
    async def predict_batch(self, images: List[Image.Image]) -> List[dict]:
        """Batch inference – Ray Serve auto-batches incoming requests."""
        results = []
        for img in images:
            results.append(self.predict(img))
        return results

    async def __call__(self, image: Image.Image) -> dict:
        """Entry point for Ray Serve handle calls."""
        return await self.predict_batch(image)


# ---------------------------------------------------------------------------
# gRPC Servicer – translates gRPC calls → Ray Serve handle
# ---------------------------------------------------------------------------
class NSFWDetectorServicer(nsfw_pb2_grpc.NSFWDetectorServicer):
    """gRPC servicer that forwards requests to the Ray Serve deployment."""

    def __init__(self, model_handle: serve.DeploymentHandle):
        self.model_handle = model_handle

    async def Predict(
        self,
        request: nsfw_pb2.PredictRequest,
        context: grpc.aio.ServicerContext,
    ) -> nsfw_pb2.PredictResponse:
        """Predict NSFW from raw image bytes."""
        try:
            image = Image.open(io.BytesIO(request.image_data)).convert("RGB")
            result = await self.model_handle.remote(image)
            return self._to_response(result)
        except Exception as e:
            logger.error(f"Predict error: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return nsfw_pb2.PredictResponse()

    async def PredictFromURL(
        self,
        request: nsfw_pb2.PredictURLRequest,
        context: grpc.aio.ServicerContext,
    ) -> nsfw_pb2.PredictResponse:
        """Predict NSFW from a public image URL."""
        try:
            image = await self._download_image(request.url)
            result = await self.model_handle.remote(image)
            return self._to_response(result)
        except Exception as e:
            logger.error(f"PredictFromURL error: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return nsfw_pb2.PredictResponse()

    async def PredictBatchFromURLs(
        self,
        request: nsfw_pb2.PredictBatchURLRequest,
        context: grpc.aio.ServicerContext,
    ) -> nsfw_pb2.PredictBatchResponse:
        """Batch predict NSFW from multiple URLs."""
        predictions = []
        for url in request.urls:
            batch_result = nsfw_pb2.BatchPredictionResult(url=url)
            try:
                image = await self._download_image(url)
                result = await self.model_handle.remote(image)
                batch_result.result.CopyFrom(self._to_response(result))
            except Exception as e:
                batch_result.error = str(e)
            predictions.append(batch_result)

        return nsfw_pb2.PredictBatchResponse(predictions=predictions)

    async def HealthCheck(
        self,
        request: nsfw_pb2.HealthCheckRequest,
        context: grpc.aio.ServicerContext,
    ) -> nsfw_pb2.HealthCheckResponse:
        """Health check – verify Ray Serve deployment is alive."""
        try:
            # Ping the deployment with a tiny 1x1 image
            tiny = Image.new("RGB", (1, 1), color=(0, 0, 0))
            await self.model_handle.remote(tiny)
            return nsfw_pb2.HealthCheckResponse(
                status="healthy",
                model=MODEL_NAME,
                device="ray-serve",
            )
        except Exception as e:
            context.set_code(grpc.StatusCode.UNAVAILABLE)
            context.set_details(str(e))
            return nsfw_pb2.HealthCheckResponse(status="unhealthy")

    # -- Helpers ---------------------------------------------------------------

    @staticmethod
    def _to_response(result: dict) -> nsfw_pb2.PredictResponse:
        return nsfw_pb2.PredictResponse(
            is_nsfw=result["is_nsfw"],
            nsfw_score=result["nsfw_score"],
            normal_score=result["normal_score"],
            label=result["label"],
            confidence=result["confidence"],
        )

    @staticmethod
    async def _download_image(url: str) -> Image.Image:
        async with httpx.AsyncClient(timeout=30.0) as client:
            response = await client.get(url)
            response.raise_for_status()
            content_type = response.headers.get("content-type", "")
            if not content_type.startswith("image/"):
                raise ValueError(f"URL does not point to an image: {content_type}")
            return Image.open(io.BytesIO(response.content)).convert("RGB")


# ---------------------------------------------------------------------------
# Main – start Ray, deploy model, run gRPC server
# ---------------------------------------------------------------------------
async def run_grpc_server(model_handle: serve.DeploymentHandle):
    """Start the async gRPC server."""
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=10))
    nsfw_pb2_grpc.add_NSFWDetectorServicer_to_server(
        NSFWDetectorServicer(model_handle), server
    )
    listen_addr = f"0.0.0.0:{GRPC_PORT}"
    server.add_insecure_port(listen_addr)
    logger.info(f"gRPC server listening on {listen_addr}")
    await server.start()
    await server.wait_for_termination()


def main():
    # Initialize Ray (connects to existing cluster or starts local)
    ray.init(ignore_reinit_error=True)

    # Deploy the model via Ray Serve
    app = NSFWModel.bind()
    handle = serve.run(app, name="nsfw_model", route_prefix="/nsfw")
    logger.info("Ray Serve deployment started")

    # Start gRPC server
    asyncio.run(run_grpc_server(handle))


if __name__ == "__main__":
    main()
