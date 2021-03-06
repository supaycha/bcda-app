#!/bin/bash
#
# This script is intended to be run from within the Docker "smoke_test" container
#
set -e
echo "Running Patient All"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient &
echo "Running Group All"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Group/all &
wait
