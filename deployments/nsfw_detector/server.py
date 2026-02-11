"""
NSFW Image Detection with Ray Serve + gRPC
HuggingFace Model: https://huggingface.co/Falconsai/nsfw_image_detection
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
from PIL import Image

import proto.nsfw_pb2 as nsfw_pb2
import proto.nsfw_pb2_grpc as nsfw_pb2_grpc
from dotenv import load_dotenv

# CONFIG
MODEL_NAME = "Falconsai/nsfw_image_detection"
GRPC_PORT = 50051
# Load env
load_dotenv()
# Logging

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("nsfw_detector")

# ---------------------------------------------------------------------------
# Ray Serve Deployment
# ---------------------------------------------------------------------------


@serve.deployment(
    num_replicas=1,
    ray_actor_options={"num_cpus": 1, "num_gpus": 1},
)
class NSFWModel:
    def __init__(self):
        logger.info(f"🚀 Loading model: {MODEL_NAME}")
        import torch
        from transformers import AutoModelForImageClassification, AutoImageProcessor

        self.torch = torch
        # Device detect
        if self.torch.cuda.is_available():
            self.device = self.torch.device("cuda")
            logger.info("Using CUDA GPU")
        elif hasattr(self.torch.backends, "mps") and self.torch.backends.mps.is_available():
            self.device = self.torch.device("mps")
            logger.info("Using Apple MPS")
        else:
            self.device = self.torch.device("cpu")
            logger.info("Using CPU")
        # Load processor + model
        self.processor = AutoImageProcessor.from_pretrained(MODEL_NAME)

        self.model = AutoModelForImageClassification.from_pretrained(
            MODEL_NAME,
            torch_dtype=self.torch.float16 if self.device.type == "cuda" else self.torch.float32
        )
        self.model.to(self.device)
        self.model.eval()
        self.torch.set_grad_enabled(False)
        # Warmup
        dummy = Image.new("RGB", (224, 224))
        self._predict_single(dummy)

        logger.info("✅ Model loaded and warmed up")

    # Internal single predict
    def _predict_single(self, image: Image.Image) -> dict:
        inputs = self.processor(images=image, return_tensors="pt")
        inputs = {k: v.to(self.device) for k, v in inputs.items()}

        outputs = self.model(**inputs)
        probs = self.torch.softmax(outputs.logits, dim=-1)[0]

        labels = self.model.config.id2label
        predicted_idx = probs.argmax().item()
        predicted_label = labels[predicted_idx]

        nsfw_idx = next((i for i, l in labels.items()
                        if "nsfw" in l.lower()), None)
        normal_idx = next((i for i, l in labels.items()
                          if "nsfw" not in l.lower()), None)

        return {
            "is_nsfw": predicted_label.lower() == "nsfw",
            "nsfw_score": probs[nsfw_idx].item() if nsfw_idx is not None else 0.0,
            "normal_score": probs[normal_idx].item() if normal_idx is not None else 1.0,
            "label": predicted_label,
            "confidence": probs[predicted_idx].item(),
        }

    # Batch inference (REAL batching)
    @serve.batch(max_batch_size=8, batch_wait_timeout_s=0.1)
    async def predict_batch(self, images: List[Image.Image]) -> List[dict]:

        inputs = self.processor(images=images, return_tensors="pt")
        inputs = {k: v.to(self.device) for k, v in inputs.items()}

        outputs = self.model(**inputs)
        probs_batch = self.torch.softmax(outputs.logits, dim=-1)

        labels = self.model.config.id2label

        results = []
        for probs in probs_batch:
            predicted_idx = probs.argmax().item()
            predicted_label = labels[predicted_idx]

            nsfw_idx = next((i for i, l in labels.items()
                            if "nsfw" in l.lower()), None)
            normal_idx = next((i for i, l in labels.items()
                              if "nsfw" not in l.lower()), None)

            results.append({
                "is_nsfw": predicted_label.lower() == "nsfw",
                "nsfw_score": probs[nsfw_idx].item() if nsfw_idx is not None else 0.0,
                "normal_score": probs[normal_idx].item() if normal_idx is not None else 1.0,
                "label": predicted_label,
                "confidence": probs[predicted_idx].item(),
            })

        return results

    # Serve entry
    async def __call__(self, image: Image.Image) -> dict:
        results = await self.predict_batch([image])
        if isinstance(results, list):
            return results[0]
        return results


# gRPC Servicer
class NSFWDetectorServicer(nsfw_pb2_grpc.NSFWDetectorServicer):

    def __init__(self, model_handle):
        self.model_handle = model_handle

    async def Predict(self, request, context):
        try:
            image = Image.open(io.BytesIO(request.image_data)).convert("RGB")
            result = await self.model_handle.remote(image)
            return self._to_response(result)

        except Exception as e:
            logger.error(f"Predict error: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return nsfw_pb2.PredictResponse()

    async def PredictFromURL(self, request, context):
        try:
            image = await self._download_image(request.url)
            result = await self.model_handle.remote(image)
            return self._to_response(result)

        except Exception as e:
            logger.error(f"PredictFromURL error: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return nsfw_pb2.PredictResponse()

    async def PredictBatchFromURLs(self, request, context):

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

    async def HealthCheck(self, request, context):

        try:
            tiny = Image.new("RGB", (1, 1))
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

    # ---------------- Helpers ----------------
    @staticmethod
    def _to_response(result: dict):

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
                raise ValueError("URL is not an image")

            return Image.open(io.BytesIO(response.content)).convert("RGB")


# gRPC Server
async def run_grpc_server(model_handle):

    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=10))

    nsfw_pb2_grpc.add_NSFWDetectorServicer_to_server(
        NSFWDetectorServicer(model_handle),
        server
    )

    listen_addr = f"0.0.0.0:{GRPC_PORT}"
    server.add_insecure_port(listen_addr)

    logger.info(f"gRPC server listening on {listen_addr}")

    await server.start()
    await server.wait_for_termination()


def main():
    ray.init(ignore_reinit_error=True)
    app = NSFWModel.bind()
    handle = serve.run(app, name="nsfw_model", route_prefix="/nsfw")
    logger.info("Ray Serve deployment started")
    asyncio.run(run_grpc_server(handle))


if __name__ == "__main__":
    main()
