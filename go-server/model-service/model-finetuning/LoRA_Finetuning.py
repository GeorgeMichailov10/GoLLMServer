from datasets import load_dataset
from transformers import AutoModelForCausalLM, AutoTokenizer
import torch
import bitsandbytes as bnb
from peft import LoraConfig, get_peft_model
from dataclasses import dataclass
from transformers import TrainingArguments, Trainer

# Configuration Class --------------------------------------------------
@dataclass
class Config():
    # Config Args -------
    EIGHT_BIT_PRECISION = True
    MODEL_NAME = 'Qwen/Qwen2.5-1.5B-Instruct'
    # LoRA Args----------
    rank = 16                                       # Smaller rank scale for smaller updates to the model itself.
    lora_alpha = 32                                 # Medium value to keep model stability during finetuning.
    lora_dropout = 0.05
    target_modules = ['q_proj', 'v_proj']
    bias = 'none'
    task_type = "CAUSAL_LM"
    # Training Args -----
    output_dir = './lora_finetuned'
    per_device_train_batch_size = 2
    num_train_epochs = 10
    save_strategy = "epoch"
    logging_steps = 10
    evaluation_strategy = "no"
    learning_rate = 2e-5
    weight_decay = 0.01                             # Want fairly aggressive weight decay because LLMs are taught not to insult so this will help decay pathways that block that over training.
    fp16 = True
    report_to = "none"

# Load Dataset ----------------------------------------------------------
def dataset_loader(config: Config):
    tokenizer = AutoTokenizer.from_pretrained(config.MODEL_NAME)
    if tokenizer.pad_token is None:
        tokenizer.pad_token = tokenizer.eos_token

    # Load dataset; each example now has one key "text"
    raw_dataset = load_dataset("json", data_files='training_data.json', split='train')
    print(raw_dataset[0])

    def tokenize(example):
        # Tokenize the batch of texts with offset mapping
        outputs = tokenizer(
            example["text"],
            padding="max_length",
            truncation=True,
            max_length=128,
            return_offsets_mapping=True  # Needed to locate the "Teacher:" part
        )
        
        new_labels = []
        # Iterate over each text, its input_ids, and corresponding offset_mapping in the batch
        for text, input_ids, offset_mapping in zip(example["text"], outputs["input_ids"], outputs["offset_mapping"]):
            teacher_pos = text.find("Teacher:")
            labels = []
            # Mask tokens that occur before the "Teacher:" prompt by setting their label to -100
            for token, offset in zip(input_ids, offset_mapping):
                if offset[0] < teacher_pos:
                    labels.append(-100)
                else:
                    labels.append(token)
            new_labels.append(labels)
        
        outputs["labels"] = new_labels
        outputs.pop("offset_mapping")
        return outputs

    return raw_dataset.map(tokenize, batched=True)

# Training Function -----------------------------------------------------
def train(config: Config):
    dataset = dataset_loader(config)
    
    model = AutoModelForCausalLM.from_pretrained(
        config.MODEL_NAME,
        torch_dtype=torch.float16,
        device_map={"": torch.cuda.current_device()},
    )
    model.gradient_checkpointing_enable()

    lora_config = LoraConfig(
        r=config.rank,
        lora_alpha=config.lora_alpha,
        lora_dropout=config.lora_dropout,
        target_modules=config.target_modules,
        bias=config.bias,
        task_type=config.task_type
    )

    model = get_peft_model(model, lora_config)
    model.print_trainable_parameters()

    training_args = TrainingArguments(
        output_dir=config.output_dir,
        per_device_train_batch_size=config.per_device_train_batch_size,  
        num_train_epochs=config.num_train_epochs,
        save_strategy=config.save_strategy,
        logging_steps=config.logging_steps,
        evaluation_strategy=config.evaluation_strategy,
        learning_rate=config.learning_rate,
        weight_decay=config.weight_decay,
        fp16=config.fp16,
        report_to=config.report_to,
        max_grad_norm=1.0,                                      # Important to clip gradients because I had training crash several time due to explosions
        gradient_accumulation_steps=4
    )

    trainer = Trainer(
        model=model,
        args=training_args,
        train_dataset=dataset
    )

    trainer.train()
    model.save_pretrained(config.output_dir)

if __name__ == "__main__":
    config = Config()
    train(config)
