import streamlit as st
import websocket

st.set_page_config(page_title="George LLM Chat - Non-Streaming")

st.title("George LLM Chat (Non-Streaming)")

if "messages" not in st.session_state:
    st.session_state.messages = [
        {"role": "assistant", "content": "How may I help you?"}
    ]

for message in st.session_state.messages:
    with st.chat_message(message["role"]):
        st.write(message["content"])

def generate_response(prompt_input):
    """
    Sends the user's prompt to the websocket server and collects tokens
    until the '[END]' marker is received. Returns the full response.
    """
    full_prompt = f"user: {prompt_input}\nassistant:"
    ws = websocket.create_connection("ws://localhost:8080/ws")
    ws.send(full_prompt)

    full_response = ""
    while True:
        token = ws.recv()
        if token == "[END]":
            break
        full_response += token
    ws.close()
    return full_response

if prompt := st.chat_input("Enter your message:"):
    st.session_state.messages.append({"role": "user", "content": prompt})
    with st.chat_message("user"):
        st.write(prompt)


    with st.spinner("Waiting for response..."):
        response = generate_response(prompt)
    st.session_state.messages.append({"role": "assistant", "content": response})
    with st.chat_message("assistant"):
        st.write(response)
