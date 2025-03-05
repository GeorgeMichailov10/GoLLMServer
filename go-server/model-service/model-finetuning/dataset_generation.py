import json
import openai
import os
from tqdm import tqdm
import json
import re


# LMStudio API Endpoint
API_URL = "http://localhost:1234/v1"
client = openai.OpenAI(base_url=API_URL, api_key="lm-studio")

BATCH_SIZE = 5
TOTAL_SAMPLES = 5000
OUTPUT_FILE = "raw.json"

def load_existing_data():
    """Loads existing data from raw.json if available."""
    if os.path.exists(OUTPUT_FILE):
        with open(OUTPUT_FILE, "r", encoding="utf-8") as f:
            try:
                return json.load(f)
            except json.JSONDecodeError:
                print("raw.json is corrupted")
                return {} 
    return {}

def save_data(data):
    with open(OUTPUT_FILE, "w", encoding="utf-8") as f:
        json.dump(data, f, indent=4)

def generate_sarcastic_responses(raw_data):
    id_counter = len(raw_data) + 1

    prompt = (
        """
        You are an extremely mean professor who wants to make students feel bad for their dumb questions.
        Generate 5 mean tutoring exchanges about academic topics ranging from very basic to difficult.

        Example Format:
        Student: [Their question]
        Teacher: [Mean and correct response]

        Make sure the responses contain:
        - A mean insult
        - The correct answer in the teacher's response.

        Example:
        Student: What is 2+2? Is it 5?
        Teacher: A monkey could have learned faster than you that 2+2 is 4.
        Reminder: Answer only in
        Student:
        Teacher:
        Responses
        """
    )

    try:
        response = client.chat.completions.create(
            model="qwen2.5-coder-32b-instruct",
            messages=[{"role": "system", "content": prompt}],
            temperature=0.8,
            max_tokens=300
        )
        ai_text = response.choices[0].message.content.strip()
        raw_data[str(id_counter)] = ai_text
        save_data(raw_data)
        print(f"Saved batch {id_counter} to {OUTPUT_FILE}")
    except Exception as e:
        print(f"Error generating response: {e}")

def create_raw_dataset():
    raw_data = load_existing_data()
    for _ in tqdm(range(TOTAL_SAMPLES // BATCH_SIZE), desc="Generating Batches"):
        generate_sarcastic_responses(raw_data)
    print(f"Successfully generated {len(raw_data)} batches and saved to {OUTPUT_FILE}")

def create_training_dataset():
    SYSTEM_PROMPT = "You are an extremely mean professor who wants to make students feel bad for their dumb questions, but provides them with the correct answer. "
    training_data = []
    raw_data = load_existing_data()

    for id, raw_text in raw_data.items():
        student_matches = re.findall(r"Student:\s*(.+?)(?=\s*Teacher:|$)", raw_text, re.DOTALL)
        teacher_matches = re.findall(r"Teacher:\s*(.+?)(?=\s*Student:|$)", raw_text, re.DOTALL)

        if len(student_matches) != 5 or len(teacher_matches) != 5:
            continue

        for student_text, teacher_text in zip(student_matches, teacher_matches):
            student_text = student_text.strip()
            teacher_text = teacher_text.strip()
            training_data.append({
                "text": SYSTEM_PROMPT + f"Student: {student_text}\nTeacher:" + teacher_text
            })

    output_file = "training_data.json"
    with open(output_file, "w", encoding="utf-8") as f:
        json.dump(training_data, f, indent=4)

    print(f"Successfully prepared {len(training_data)} training examples and saved to {output_file}")
        
if __name__ == "__main__":
    #create_raw_dataset()
    create_training_dataset()
