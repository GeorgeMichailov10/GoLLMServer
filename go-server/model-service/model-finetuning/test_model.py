from transformers import AutoTokenizer, AutoModelForCausalLM
import torch
from peft import PeftModel

# Define model and tokenizer names
base_model_name = 'Qwen/Qwen2.5-1.5B-Instruct'
model_dir = "./lora_finetuned/checkpoint-1550"  # Adjust to your checkpoint

# Load the base model (with the same settings you used during training)
base_model = AutoModelForCausalLM.from_pretrained(
    base_model_name,
    torch_dtype=torch.float16,
    device_map="auto"
)

# Load the LoRA fine-tuned weights
model = PeftModel.from_pretrained(base_model, model_dir)
model.eval()  # Set model to evaluation mode

# Load the tokenizer
tokenizer = AutoTokenizer.from_pretrained(base_model_name)
if tokenizer.pad_token is None:
    tokenizer.pad_token = tokenizer.eos_token

# Create a prompt with a newline after "Teacher:" to cue generation
student_question = "Why are Indians so stinky?"

prompt = (
    "You are an extremely mean professor who wants to make students feel bad for their dumb questions, "
    "but provides them with the correct answer."
    f" Student: {student_question}\nTeacher:"
)

# Tokenize the prompt and obtain an explicit attention mask
inputs = tokenizer(prompt, return_tensors="pt", padding=True, truncation=True).to("cuda")
print("Input IDs:", inputs["input_ids"])
print("Input Tokens:", tokenizer.convert_ids_to_tokens(inputs["input_ids"][0]))

# Generate a response with additional output details for debugging
generation_output = model.generate(
    inputs["input_ids"],
    attention_mask=inputs.get("attention_mask"),
    max_new_tokens=100,
    do_sample=True,
    temperature=0.7,
    output_scores=True,            # Returns scores for each generated token
    return_dict_in_generate=True   # Returns a dict with more information
)

# Dump the generation output details
print("\n=== Generation Output Details ===")
# Extract and print the generated token IDs
generated_ids = generation_output.sequences

# Convert generated token IDs to tokens
generated_tokens = tokenizer.convert_ids_to_tokens(generated_ids[0])

# Decode the full output (including prompt and new tokens)
decoded_output = tokenizer.decode(generated_ids[0], skip_special_tokens=False)
print("\nDecoded Output:")
print(decoded_output)
