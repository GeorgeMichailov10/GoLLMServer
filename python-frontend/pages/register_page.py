import streamlit as st

st.set_page_config(page_title="Register Page", page_icon="Register", layout="centered")

st.title("Login/Register")

with st.form("login_form"):
    st.subheader("Please enter your credentials")
    username = st.text_input("Username", placeholder="Enter your username")
    password = st.text_input("Password", type="password", placeholder="Enter your password")
    submit_button = st.form_submit_button("Login")


if submit_button:
    st.info("Authentication logic will be added here.")