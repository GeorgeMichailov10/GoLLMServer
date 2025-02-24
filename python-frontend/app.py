import streamlit as st
import websocket
from langchain.memory import ConversationBufferWindowMemory

st.set_page_config(page_title="George LLM Chat")

with st.sidebar:
    st.title("This is where\nthe convo history\nwill be")

if "memory" not in st.session_state:
    st.session_state.memory = ConversationBufferWindowMemory(k=5, return_messages=True)

if "messages" not in st.session_state:
    st.session_state.messages = [{"role": "assistant", "content": "How may I help you?"}]

# Display existing messages
for message in st.session_state.messages:
    with st.chat_message(message["role"]):
        st.write(message["content"])

def generate_response(prompt_input, context):
    """
    Build the conversation history using the context and the current prompt,
    then send it to the websocket and yield tokens until the "[END]" marker is received.
    """
    conversation_history = ""
    for msg in context:
        conversation_history += f"{msg['role']}: {msg['content']}\n"
    full_prompt = conversation_history + f"user: {prompt_input}\nassistant:"
    
    ws = websocket.create_connection("ws://localhost:8080/ws")
    ws.send(full_prompt)
    
    # Yield tokens as they arrive until we get the termination marker.
    while True:
        token = ws.recv()
        if token == "[END]":
            break
        yield token
    ws.close()

# When the user provides a prompt
if prompt := st.chat_input():
    # Append and display the user's prompt
    st.session_state.messages.append({"role": "user", "content": prompt})
    with st.chat_message("user"):
        st.write(prompt)

    # Build context from memory: convert stored messages into a list of dicts.
    context = []
    for msg in st.session_state.memory.chat_memory.messages:
        role = "user" if msg.type.lower() == "human" else "assistant"
        context.append({"role": role, "content": msg.content})
    # Include the current prompt in context
    context.append({"role": "user", "content": prompt})
    # Use only the last 5 messages for context
    recent_context = context[-5:]
    
    full_response = ""
    with st.chat_message("assistant"):
        # Create a placeholder for streaming tokens.
        message_placeholder = st.empty()
        with st.spinner("Thinking..."):
            for token in generate_response(prompt, recent_context):
                full_response += token
                message_placeholder.write(full_response)
    
    # Save the assistant's full response for future context
    st.session_state.messages.append({"role": "assistant", "content": full_response})
    st.session_state.memory.save_context({"input": prompt}, {"output": full_response})
