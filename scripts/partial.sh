#!/usr/bin/env bash

# Exit on any error
set -e

#####################################
# Start the Go server               #
#####################################
echo "Starting Go server..."
cd ./go-server
go run *.go &
GO_SERVER_PID=$!
echo "Go server started with PID: $GO_SERVER_PID"

#####################################
# Start the Streamlit front end    #
#####################################
echo "Starting Streamlit front end..."
cd ../python-frontend
streamlit run login_page.py --server.port=8501 &
FRONTEND_PID=$!
echo "Streamlit front end started with PID: $FRONTEND_PID"

#####################################
# Wait for user to CTRL+C to stop   #
#####################################
echo ""
echo "All services have started."
echo "Press CTRL+C to stop."
wait
