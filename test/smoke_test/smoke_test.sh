#!/bin/bash
#
# This script is intended to be run from within the Docker "smoke_test" container
#
set -e

go run bcda_client.go decrypt_util.go -host=api:3000 -endpoint=ExplanationOfBenefit -encrypt=true
go run bcda_client.go decrypt_util.go -host=api:3000 -endpoint=Patient -encrypt=true
go run bcda_client.go decrypt_util.go -host=api:3000 -endpoint=ExplanationOfBenefit -encrypt=false
go run bcda_client.go decrypt_util.go -host=api:3000 -endpoint=Patient -encrypt=false
