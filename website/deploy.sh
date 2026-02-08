#!/bin/sh

# Exit on error.
set -e

# Deploy via xmit (no build step — static HTML/CSS/JS).
xmit plugmy.ai
