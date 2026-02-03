#!/bin/bash

# Start the Go backend
./jask &

# Start the frontend
cd /work/frontend && npm start &

# Start the admin panel
cd /work/admin && npm start -- -p 3001 &

# Start Nginx
nginx -g 'daemon off;' &

# Wait for all background processes
wait
