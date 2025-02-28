import streamlit as st
import requests
import websocket
from streamlit_extras.switch_page_button import switch_page

st.set_page_config(page_title="Chat Interface", page_icon="💬", layout="wide")

# Redirect to login page if user is not logged in.
if "jwt_token" not in st.session_state or "username" not in st.session_state:
    st.error("You are not logged in. Redirecting to login page...")
    switch_page("login page")
    st.stop()

jwt_token = st.session_state["jwt_token"]
username = st.session_state["username"]

headers = {"Authorization": f"Bearer {jwt_token}"}

def load_chats():
    try:
        response = requests.get("http://localhost:8080/user/chats", headers=headers)
        if response.status_code == 200:
            return response.json().get("chats", {})
        else:
            st.error("Failed to retrieve chats.")
            return {}
    except Exception as e:
        st.error(f"An error occurred while retrieving chats: {str(e)}")
        return {}

# Initially load chats into session_state if not already loaded.
if "chats_data" not in st.session_state:
    st.session_state.chats_data = load_chats()

# Sidebar: add the "Reload Chats" button above the chat list.
st.sidebar.title("Your Chats")
if st.sidebar.button("Reload Chats"):
    post_response = requests.post("http://localhost:8080/user/chats", headers=headers)
    if post_response.status_code in [200, 201]:
        st.session_state.chats_data = load_chats()
    else:
        st.error("Failed to update chats from POST request.")

# Display the user's chats from session_state.
if st.session_state.chats_data:
    for chat_id, title in st.session_state.chats_data.items():
        st.sidebar.write(f"Chat ID: {chat_id} - {title}")
else:
    st.sidebar.write("No chats available.")

st.title("George LLM Chat (Non-Streaming)")

# Set up an initial message if none exist.
if "messages" not in st.session_state:
    st.session_state.messages = [
        {"role": "assistant", "content": "How may I help you?"}
    ]

# Render the conversation.
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

# Chat input and response generation.
if prompt := st.chat_input("Enter your message:"):
    st.session_state.messages.append({"role": "user", "content": prompt})
    with st.chat_message("user"):
        st.write(prompt)

    with st.spinner("Waiting for response..."):
        response = generate_response(prompt)
    st.session_state.messages.append({"role": "assistant", "content": response})
    with st.chat_message("assistant"):
        st.write(response)
