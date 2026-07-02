#!/bin/bash
set -e

# Change to project root relative to script
cd "$(dirname "$0")/../.."

mkdir -p data/raw
cd data/raw

echo "Downloading UNSW-NB15 train/test CSVs..."

# The correct UNSW-NB15 dataset mirror
curl -LO https://raw.githubusercontent.com/Nir-J/ML-Projects/master/UNSW-Network_Packet_Classification/UNSW_NB15_training-set.csv
curl -LO https://raw.githubusercontent.com/Nir-J/ML-Projects/master/UNSW-Network_Packet_Classification/UNSW_NB15_testing-set.csv

echo "Verification:"
TRAIN_ROWS=$(wc -l < UNSW_NB15_training-set.csv | tr -d ' ')
TEST_ROWS=$(wc -l < UNSW_NB15_testing-set.csv | tr -d ' ')

echo "Train rows: $TRAIN_ROWS"
echo "Test rows:  $TEST_ROWS"

if [ "$TRAIN_ROWS" -lt 175341 ] || [ "$TEST_ROWS" -lt 82332 ]; then
  echo "Error: Downloaded files have incorrect row counts. Expected at least 175341 train rows and 82332 test rows."
  exit 1
fi

echo "Done."
