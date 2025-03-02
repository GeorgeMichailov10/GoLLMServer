import streamlit as st
import json
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

# ------------------ Delete Account Button (Top Right) ------------------
# Create two columns; left for title, right for the delete button.
col1, col2 = st.columns([8, 2])
with col1:
    st.title("George LLM Chat (Non-Streaming)")
with col2:
    if st.button("Delete Account"):
        try:
            del_response = requests.delete("http://localhost:8080/user", headers=headers)
            if del_response.status_code in [200, 204]:
                st.success("Account deleted successfully.")
                st.session_state.clear()
                switch_page("login page")
            else:
                st.error("Failed to delete account.")
        except Exception as e:
            st.error(f"An error occurred: {e}")
# ------------------ End Delete Account Button ------------------

# Function to load all chats.
def load_chats():
    try:
        response = requests.get("http://localhost:8080/user/chats", headers=headers)
        if response.status_code == 200:
            print(response.json().get("chats", {}))
            return response.json().get("chats", {})
        else:
            st.error(f"Failed to retrieve chats. {response}")
            return {}
    except Exception as e:
        st.error(f"An error occurred while retrieving chats: {str(e)}")
        return {}

# Initially load chats into session_state if not already loaded.
if "chats_data" not in st.session_state:
    st.session_state.chats_data = load_chats()

# Sidebar: add the "New Chat" button above the chat list.
st.sidebar.title("Your Chats")
if st.sidebar.button("New Chat"):
    st.session_state.selected_chat = None
    st.session_state.messages = []

# Display the user's chats as buttons in the sidebar.
if st.session_state.chats_data:
    for chat_id, title in reversed(list(st.session_state.chats_data.items())):
        if st.sidebar.button(f"Chat ID: {chat_id} - {title}", key=f"chat_{chat_id}"):
            st.session_state.selected_chat = chat_id
            # Build the URL for the selected chat.
            chat_url = f"http://localhost:8080/chat/{chat_id}"
            chat_response = requests.get(chat_url, headers=headers)
            if chat_response.status_code == 200:
                chat_data = chat_response.json()
                messages = []
                if "content" in chat_data and chat_data["content"]:
                    for interaction in chat_data["content"]:
                        if "user" in interaction and interaction["user"]:
                            messages.append({"role": "user", "content": interaction["user"]})
                        if "model" in interaction and interaction["model"]:
                            messages.append({"role": "assistant", "content": interaction["model"]})
                st.session_state.messages = messages
                st.sidebar.write(f"Selected chat {chat_id}")
            else:
                st.sidebar.error("Failed to retrieve chat content.")
else:
    st.sidebar.write("No chats available.")

# Set up an initial messages list if none exist.
if "messages" not in st.session_state:
    st.session_state.messages = []

# Render the conversation.
for message in st.session_state.messages:
    with st.chat_message(message["role"]):
        st.write(message["content"])

def generate_response(prompt_input):
    """
    Sends the user's prompt to the websocket server and collects tokens
    until the '[END]' marker is received. Returns the full response.
    """
    # Directly get the selected chat from session state.
    selected_chat = st.session_state.get("selected_chat", "")
    payload = json.dumps({
        "claims": {"username": st.session_state.username},
        "query": prompt_input,
        "chatid": selected_chat
    })
    ws = websocket.create_connection(
        "ws://localhost:8080/ws",
        header=["Authorization: Bearer " + st.session_state["jwt_token"]]
    )
    ws.send(payload)

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
    # Directly retrieve the selected chat from session state.
    selected_chat = st.session_state.get("selected_chat", "")
    st.session_state.messages.append({"role": "user", "content": prompt})
    with st.chat_message("user"):
        st.write(prompt)

    with st.spinner("Waiting for response..."):
        response = generate_response(prompt)
    st.session_state.messages.append({"role": "assistant", "content": response})
    with st.chat_message("assistant"):
        st.write(response)

    # Sneaky reload: if no selected chat was set (first interaction),
    # reload chats, assume the newest chat is the one just created,
    # and update the selected chat accordingly.
    print("Selected Chat id:", st.session_state.get("selected_chat", ""))
    if not st.session_state.get("selected_chat", ""):
        st.session_state.chats_data = load_chats()
        print(st.session_state.chats_data)
        if st.session_state.chats_data:
            newest_chat_id = list(st.session_state.chats_data.keys())[-1]
            chat_url = f"http://localhost:8080/chat/{newest_chat_id}"
            chat_response = requests.get(chat_url, headers=headers)
            print(chat_response)
            if chat_response.status_code == 200:
                chat_data = chat_response.json()
                messages = []
                if "content" in chat_data and chat_data["content"]:
                    for interaction in chat_data["content"]:
                        if "user" in interaction and interaction["user"]:
                            messages.append({"role": "user", "content": interaction["user"]})
                        if "model" in interaction and interaction["model"]:
                            messages.append({"role": "assistant", "content": interaction["model"]})
                st.session_state.messages = messages
                st.session_state.selected_chat = newest_chat_id
            else:
                st.error("Failed to retrieve the newest chat content.")
        else:
            st.error("No chats found after reloading.")
