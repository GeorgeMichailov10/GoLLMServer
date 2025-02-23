import asyncio
import grpc
from vllm_service_pb2 import QueryResponse
import vllm_service_pb2_grpc
from vllm.engine.async_llm_engine import AsyncLLMEngine
from vllm.engine.arg_utils import AsyncEngineArgs
from vllm.sampling_params import SamplingParams

# Define parameters and initialize engine.
SAMPLING_PARAMS = SamplingParams(temperature=0.7, top_p=0.95, max_tokens=30)
MODEL_NAME = "facebook/opt-125m"
engine_args = AsyncEngineArgs(
    model=MODEL_NAME,
    tensor_parallel_size=1,
)
engine = AsyncLLMEngine.from_engine_args(engine_args)

class VLLMServiceServicer(vllm_service_pb2_grpc.VLLMServiceServicer):
    async def Query(self, request, context):
        print(f"[gRPC Server] Received query: {request.query}")
        async for token in self.stream_inference_async(request.query):
            print(f"[gRPC Server] Sending token: '{token}' for query: {request.query}")
            yield QueryResponse(token=token)
        print(f"[gRPC Server] Completed query: {request.query}")

    async def stream_inference_async(self, query: str):
        print(f"[gRPC Server] Starting inference for query: {query}")
        request_id = "my-stream-request-123"
        async for request_output in engine.generate(query, SAMPLING_PARAMS, request_id):
            for output in request_output.outputs:
                yield output.text
            if request_output.finished:
                break
        yield "[END]"

async def serve():
    server = grpc.aio.server()
    vllm_service_pb2_grpc.add_VLLMServiceServicer_to_server(VLLMServiceServicer(), server)
    listen_addr = "[::]:50051"
    server.add_insecure_port(listen_addr)
    print(f"[gRPC Server] Starting gRPC server on {listen_addr}")
    await server.start()
    await server.wait_for_termination()

if __name__ == '__main__':
    asyncio.run(serve())
