from vllm.engine.async_llm_engine import AsyncLLMEngine
from vllm.engine.arg_utils import AsyncEngineArgs
from vllm.sampling_params import SamplingParams
import asyncio

async def main():
    engine_args = AsyncEngineArgs(
        model="merged_model",
        tensor_parallel_size=1,
    )
    engine = AsyncLLMEngine.from_engine_args(engine_args)
    sampling_params = SamplingParams(temperature=0.7, max_tokens=15)
    request_id = "my-stream-request-123"

    currently_seen = 0
    async for request_output in engine.generate("Hello how are you", sampling_params, request_id):
        for output in request_output.outputs:
            partial_text = output.text[currently_seen:]
            currently_seen = len(output.text)
            print(partial_text, end="", flush=True)

        if request_output.finished:
            break

    print("\nDone.")

asyncio.run(main())
