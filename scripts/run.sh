#!/usr/bin/env bash

# Exit on any error
set -e

#####################################
# 1) Start the "second" Python file #
#####################################
echo "Starting the main model server (vllm_server.py)..."
cd ./go-server/model-service

# Start the model server in the background
python3 vllm_server.py &
MAIN_PY_SERVER_PID=$!
echo "Model server started with PID: $MAIN_PY_SERVER_PID"

sleep 30

#####################################
# 3) Start the Go server            #
#####################################
echo "Starting Go server..."
cd ..
go run server.go &
GO_SERVER_PID=$!
echo "Go server started with PID: $GO_SERVER_PID"

#####################################
# 4) Start the Streamlit front end  #
#####################################
echo "Starting Streamlit front end..."
cd ../python-frontend

streamlit run app.py --server.port=8501 &
FRONTEND_PID=$!
echo "Streamlit front end started with PID: $FRONTEND_PID"

#####################################
# Wait for user to CTRL+C to stop   #
#####################################
echo ""
echo "All services have started."
echo "Press CTRL+C to stop."
wait
