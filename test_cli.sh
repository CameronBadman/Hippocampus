#!/bin/bash

set -e  # Exit on error

echo "=================================================="
echo "Hippocampus CLI Test Suite"
echo "=================================================="
echo

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test database file
TEST_DB="test_tree.bin"

# Clean up any existing test database
rm -f "$TEST_DB"

echo -e "${BLUE}Step 1: Building CLI...${NC}"
make build-cli
echo -e "${GREEN}✓ Build complete${NC}"
echo

echo -e "${BLUE}Step 2: Testing INSERT command${NC}"
./bin/hippocampus insert -binary "$TEST_DB" \
  -key "user_preference_theme" \
  -text "User prefers dark mode for the interface"
echo -e "${GREEN}✓ Insert #1 complete${NC}"
echo

./bin/hippocampus insert -binary "$TEST_DB" \
  -key "user_preference_language" \
  -text "User prefers Python programming language"
echo -e "${GREEN}✓ Insert #2 complete${NC}"
echo

./bin/hippocampus insert -binary "$TEST_DB" \
  -key "allergy_shellfish" \
  -text "User is allergic to shellfish and seafood"
echo -e "${GREEN}✓ Insert #3 complete${NC}"
echo

./bin/hippocampus insert -binary "$TEST_DB" \
  -key "hobby_running" \
  -text "User enjoys running marathons and trains regularly"
echo -e "${GREEN}✓ Insert #4 complete${NC}"
echo

echo -e "${BLUE}Step 3: Testing SEARCH command (default parameters)${NC}"
./bin/hippocampus search -binary "$TEST_DB" \
  -text "What are the user's UI preferences?"
echo -e "${GREEN}✓ Search #1 complete${NC}"
echo

echo -e "${BLUE}Step 4: Testing SEARCH with custom parameters${NC}"
./bin/hippocampus search -binary "$TEST_DB" \
  -text "programming languages" \
  -epsilon 0.4 \
  -threshold 0.4 \
  -top-k 3
echo -e "${GREEN}✓ Search #2 complete${NC}"
echo

echo -e "${BLUE}Step 5: Testing SEARCH for allergies (high threshold)${NC}"
./bin/hippocampus search -binary "$TEST_DB" \
  -text "food allergies" \
  -epsilon 0.3 \
  -threshold 0.6 \
  -top-k 5
echo -e "${GREEN}✓ Search #3 complete${NC}"
echo

echo -e "${BLUE}Step 6: Creating test CSV file${NC}"
cat > test_data.csv << EOF
location_city,User lives in Seattle Washington
job_company,User works at Amazon Web Services
pet_dog,User has a golden retriever named Max
EOF
echo -e "${GREEN}✓ CSV file created${NC}"
echo

echo -e "${BLUE}Step 7: Testing INSERT-CSV command${NC}"
./bin/hippocampus insert-csv -binary "$TEST_DB" -csv test_data.csv
echo -e "${GREEN}✓ CSV insert complete${NC}"
echo

echo -e "${BLUE}Step 8: Verifying CSV data with search${NC}"
./bin/hippocampus search -binary "$TEST_DB" \
  -text "Where does the user live?" \
  -epsilon 0.3 \
  -threshold 0.5 \
  -top-k 3
echo -e "${GREEN}✓ CSV verification complete${NC}"
echo

echo -e "${BLUE}Step 9: Testing AGENT-CURATE command (medium importance)${NC}"
echo -e "${YELLOW}This will call AWS Bedrock - make sure you have credentials configured${NC}"
./bin/hippocampus agent-curate -binary "$TEST_DB" \
  -text "My name is Sarah Chen, I'm 34 years old, and I work as a software engineer at Google. I'm allergic to peanuts and I love hiking in the mountains." \
  -importance medium \
  -model "us.amazon.nova-lite-v1:0" \
  -bedrock-region "us-east-1" \
  -timeout-ms 100
echo -e "${GREEN}✓ Agent curation complete${NC}"
echo

echo -e "${BLUE}Step 10: Searching for curated memories${NC}"
./bin/hippocampus search -binary "$TEST_DB" \
  -text "Tell me about Sarah" \
  -epsilon 0.3 \
  -threshold 0.4 \
  -top-k 5
echo -e "${GREEN}✓ Curated memory search complete${NC}"
echo

echo -e "${BLUE}Step 11: Testing another search query${NC}"
./bin/hippocampus search -binary "$TEST_DB" \
  -text "What are Sarah's allergies?" \
  -epsilon 0.3 \
  -threshold 0.5 \
  -top-k 3
echo -e "${GREEN}✓ Final search complete${NC}"
echo

echo "=================================================="
echo -e "${GREEN}All tests passed! ✓${NC}"
echo "=================================================="
echo
echo "Test artifacts:"
echo "  - Database: $TEST_DB"
echo "  - CSV file: test_data.csv"
echo
echo "To clean up test files:"
echo "  rm $TEST_DB test_data.csv"
