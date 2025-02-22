from vllm.engine.async_llm_engine import AsyncLLMEngine
from vllm.engine.arg_utils import AsyncEngineArgs
from vllm.sampling_params import SamplingParams
import asyncio

async def main():
    # 1. Build engine args (similar to CLI args, but in code)
    engine_args = AsyncEngineArgs(
        model="facebook/opt-125m",
        tensor_parallel_size=1,
        # ... any other flags you would have passed ...
    )

    # 2. Create the AsyncLLMEngine
    engine = AsyncLLMEngine.from_engine_args(engine_args)

    # 3. Prepare your sampling params
    sampling_params = SamplingParams(temperature=0.7, max_tokens=30)

    # 4. Run a streaming generation
    request_id = "my-stream-request-123"

    # This returns an async generator
    async for request_output in engine.generate("Hello world", sampling_params, request_id):
        # Each iteration yields partial tokens for all completions in this batch
        for output in request_output.outputs:
            partial_text = output.text
            print(partial_text, end="", flush=True)

        if request_output.finished:
            break

    print("\nDone.")

asyncio.run(main())
