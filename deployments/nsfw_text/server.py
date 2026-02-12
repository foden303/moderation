"""
NSFW Text Detection with Ray Serve + gRPC
Model: https://huggingface.co/Qwen/Qwen3Guard-Gen-0.6B
Run: python server.py
"""

import re
import asyncio
import logging
from concurrent import futures
from typing import List

import grpc
import ray
from ray import serve

import proto.nsfw_text_pb2 as nsfw_text_pb2
import proto.nsfw_text_pb2_grpc as nsfw_text_pb2_grpc
from dotenv import load_dotenv

# CONFIG
MODEL_NAME = "Qwen/Qwen3Guard-Gen-0.6B"
GRPC_PORT = 50052
MAX_NEW_TOKENS = 128

# Load env
load_dotenv()

# Logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("nsfw_text")

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

SAFE_PATTERN = re.compile(r"Safety:\s*(Safe|Unsafe|Controversial)", re.IGNORECASE)
CATEGORY_PATTERN = re.compile(
    r"(Violent|Non-violent Illegal Acts|Sexual Content or Sexual Acts|"
    r"PII|Suicide & Self-Harm|Unethical Acts|Politically Sensitive Topics|"
    r"Copyright Violation|Jailbreak|None)"
)


def parse_guard_output(content: str) -> dict:
    """Parse Qwen3Guard model output into structured result."""
    safe_match = SAFE_PATTERN.search(content)
    safety_label = safe_match.group(1) if safe_match else "Unknown"
    categories = CATEGORY_PATTERN.findall(content)
    # Filter out "None"
    categories = [c for c in categories if c != "None"]

    return {
        "is_nsfw": safety_label.lower() == "unsafe",
        "safety_label": safety_label,
        "categories": categories,
    }


# ---------------------------------------------------------------------------
# Ray Serve Deployment
# ---------------------------------------------------------------------------


@serve.deployment(
    num_replicas=1,
    ray_actor_options={"num_cpus": 1, "num_gpus": 1},
)
class NSFWTextModel:
    def __init__(self):
        logger.info(f"🚀 Loading model: {MODEL_NAME}")
        import torch
        from transformers import AutoModelForCausalLM, AutoTokenizer
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

        # Load tokenizer + model
        self.tokenizer = AutoTokenizer.from_pretrained(MODEL_NAME)
        self.model = AutoModelForCausalLM.from_pretrained(
            MODEL_NAME,
            torch_dtype="auto",
            device_map="auto",
        )
        self.model.eval()
        self.torch.set_grad_enabled(False)

        # Warmup
        self._predict_single("Hello, how are you?")
        logger.info("✅ Model loaded and warmed up")

    def _predict_single(self, text: str) -> dict:
        """Run moderation on a single text input."""
        messages = [{"role": "user", "content": text}]
        prompt = self.tokenizer.apply_chat_template(
            messages, tokenize=False
        )
        model_inputs = self.tokenizer(
            [prompt], return_tensors="pt"
        ).to(self.model.device)

        generated_ids = self.model.generate(
            **model_inputs, max_new_tokens=MAX_NEW_TOKENS
        )
        output_ids = generated_ids[0][len(model_inputs.input_ids[0]):].tolist()
        content = self.tokenizer.decode(output_ids, skip_special_tokens=True)
        logger.debug(f"Model raw output: {content}")

        return parse_guard_output(content)

    @serve.batch(max_batch_size=8, batch_wait_timeout_s=0.1)
    async def predict_batch(self, texts: List[str]) -> List[dict]:
        """Batch inference — processes each text sequentially within the batch
        since generative models have variable-length outputs."""
        results = []
        for text in texts:
            result = self._predict_single(text)
            results.append(result)
        return results

    async def __call__(self, text: str) -> dict:
        return await self.predict_batch(text)


# ---------------------------------------------------------------------------
# gRPC Servicer
# ---------------------------------------------------------------------------


class NSFWTextDetectorServicer(nsfw_text_pb2_grpc.NSFWTextDetectorServicer):

    def __init__(self, model_handle):
        self.model_handle = model_handle

    async def Predict(self, request, context):
        try:
            result = await self.model_handle.remote(request.text)
            return self._to_response(result)

        except Exception as e:
            logger.error(f"Predict error: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return nsfw_text_pb2.PredictResponse()

    async def PredictBatch(self, request, context):
        predictions = []

        for text in request.texts:
            batch_result = nsfw_text_pb2.BatchPredictionResult(text=text)

            try:
                result = await self.model_handle.remote(text)
                batch_result.result.CopyFrom(self._to_response(result))

            except Exception as e:
                batch_result.error = str(e)

            predictions.append(batch_result)

        return nsfw_text_pb2.PredictBatchResponse(predictions=predictions)

    async def HealthCheck(self, request, context):
        try:
            await self.model_handle.remote("health check")

            return nsfw_text_pb2.HealthCheckResponse(
                status="healthy",
                model=MODEL_NAME,
                device="ray-serve",
            )

        except Exception as e:
            context.set_code(grpc.StatusCode.UNAVAILABLE)
            context.set_details(str(e))
            return nsfw_text_pb2.HealthCheckResponse(status="unhealthy")

    # ---------------- Helpers ----------------
    @staticmethod
    def _to_response(result: dict):
        return nsfw_text_pb2.PredictResponse(
            is_nsfw=result["is_nsfw"],
            safety_label=result["safety_label"],
            categories=result["categories"],
        )


# ---------------------------------------------------------------------------
# gRPC Server
# ---------------------------------------------------------------------------


async def run_grpc_server(model_handle):
    server = grpc.aio.server(futures.ThreadPoolExecutor(max_workers=10))

    nsfw_text_pb2_grpc.add_NSFWTextDetectorServicer_to_server(
        NSFWTextDetectorServicer(model_handle),
        server,
    )

    listen_addr = f"0.0.0.0:{GRPC_PORT}"
    server.add_insecure_port(listen_addr)

    logger.info(f"gRPC server listening on {listen_addr}")

    await server.start()
    await server.wait_for_termination()


def main():
    ray.init(ignore_reinit_error=True)
    app = NSFWTextModel.bind()
    handle = serve.run(app, name="nsfw_text_model", route_prefix="/nsfw-text")
    logger.info("Ray Serve deployment started")
    asyncio.run(run_grpc_server(handle))


if __name__ == "__main__":
    main()
