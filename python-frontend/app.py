import streamlit as st

st.set_page_config(page_title="Main", page_icon="🏠")

# Check if user is authenticated
if "authenticated" not in st.session_state:
    st.session_state.authenticated = False

# Redirect to login if not authenticated
if not st.session_state.authenticated:
    st.experimental_set_query_params(page="Login/Register")
    st.experimental_rerun()

# If authenticated, show main page content
st.title("🏡 Welcome to the App")
st.write("🎉 You are successfully logged in!")

