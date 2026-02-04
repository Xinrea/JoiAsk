#!/bin/bash

# Start the Go backend
./jask &

# Start Nginx
nginx -g 'daemon off;' &

# Wait for all background processes
wait
