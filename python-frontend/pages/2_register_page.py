import streamlit as st
import requests
from streamlit_extras.switch_page_button import switch_page

st.set_page_config(page_title="Register", page_icon="📝", layout="centered")

st.title("Registration Page")

new_username = st.text_input("New Username")
new_password = st.text_input("New Password", type="password")
confirm_password = st.text_input("Confirm Password", type="password")

if st.button("Register"):
    if not new_username or not new_password:
        st.error("Please fill out all fields.")
    elif new_password != confirm_password:
        st.error("Passwords do not match!")
    else:
        payload = {"username": new_username, "password": new_password}
        try:
            response = requests.post("http://localhost:8080/register", json=payload)
            if response.status_code in (200, 201):
                st.success("Registration successful! Redirecting to login page...")
                switch_page("login page")
            else:
                error_message = response.json().get("error", "Registration failed.")
                st.error(f"Registration failed: {error_message}")
        except Exception as e:
            st.error(f"An error occurred: {str(e)}")

st.markdown("Already have an account? [Login here](http://localhost:8501/login_page)")
