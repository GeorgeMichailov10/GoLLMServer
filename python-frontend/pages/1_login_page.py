import streamlit as st
import requests
from streamlit_extras.switch_page_button import switch_page

st.set_page_config(page_title="Login", layout="centered")

st.title("Login Page")

username = st.text_input("Username")
password = st.text_input("Password", type="password")

if st.button("Login"):
    if username and password:
        payload = {"username": username, "password": password}
        try:
            response = requests.post("http://localhost:8080/login", json=payload)
            if response.status_code == 200:
                data = response.json()
                token = data.get("token")
                if token:
                    st.session_state["jwt_token"] = token
                    st.session_state["username"] = username
                    st.success("Login successful!")
                    switch_page("chat interface")
                else:
                    st.error("Login failed: Token not found in response.")
            else:
                error_message = response.json().get("error", "Login failed.")
                st.error(f"Login failed: {error_message}")
        except Exception as e:
            st.error(f"An error occurred: {str(e)}")
    else:
        st.error("Please enter both username and password.")

st.markdown("Don't have an account? [Register here](http://localhost:8501/register_page)")
