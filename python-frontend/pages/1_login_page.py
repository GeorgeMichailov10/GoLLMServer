import streamlit as st

st.set_page_config(page_title="Login Page", page_icon="🔐", layout="centered")

st.title("🔒 Login/Register")

# Initialize authentication state
if "authenticated" not in st.session_state:
    st.session_state.authenticated = False

# Login form
with st.form("login_form"):
    st.subheader("Please enter your credentials")
    username = st.text_input("Username", placeholder="Enter your username")
    password = st.text_input("Password", type="password", placeholder="Enter your password")
    submit_button = st.form_submit_button("Login")

if submit_button:
    # Simple authentication placeholder (replace with real logic)
    if username == "admin" and password == "password":
        st.session_state.authenticated = True
        st.success("✅ Login successful! Redirecting to Home...")
        st.query_params(page="Chat")  # Redirect
        st.experimental_rerun()
    else:
        st.error("❌ Invalid credentials. Please try again.")
