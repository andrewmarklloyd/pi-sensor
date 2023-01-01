#!/bin/bash

set -euo pipefail

cd frontend
npm install
npm run build
