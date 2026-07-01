#!/bin/bash
set -e

# Change to project root relative to script
cd "$(dirname "$0")/../.."

mkdir -p data/raw
cd data/raw

echo "Downloading UNSW-NB15 train/test CSVs..."

# The original UNSW-NB15 dataset mirror
curl -LO https://raw.githubusercontent.com/CyberSecurityUP/UNSW-NB15-Dataset/master/UNSW_NB15_training-set.csv
curl -LO https://raw.githubusercontent.com/CyberSecurityUP/UNSW-NB15-Dataset/master/UNSW_NB15_testing-set.csv

echo "Verification:"
echo "Train rows: $(wc -l < UNSW_NB15_training-set.csv)"
echo "Test rows:  $(wc -l < UNSW_NB15_testing-set.csv)"
echo "Done."
