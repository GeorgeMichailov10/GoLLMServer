from peft import PeftModel, PeftConfig
from transformers import AutoModelForCausalLM, AutoTokenizer

lora_model_path = "custom-model"
base_model_name = "Qwen/Qwen2.5-1.5B-Instruct"
peft_config = PeftConfig.from_pretrained(lora_model_path)
base_model = AutoModelForCausalLM.from_pretrained(
    base_model_name,
    torch_dtype="auto",
    device_map="auto"
)
lora_model = PeftModel.from_pretrained(base_model, lora_model_path)
merged_model = lora_model.merge_and_unload()
tokenizer = AutoTokenizer.from_pretrained(base_model_name)
output_path = "merged_model"
merged_model.save_pretrained(output_path)
tokenizer.save_pretrained(output_path)
