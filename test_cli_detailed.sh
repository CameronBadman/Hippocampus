#!/bin/bash

set -e  # Exit on error

echo "=================================================="
echo "Hippocampus CLI Test Suite with Timing"
echo "Using AWS Region: ap-southeast-2"
echo "=================================================="
echo

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test database file
TEST_DB="test_tree.bin"

# Clean up any existing test database
rm -f "$TEST_DB"

# Function to time commands
time_command() {
    local description="$1"
    shift
    echo -e "${CYAN}⏱ Timing: $description${NC}"
    echo -e "${YELLOW}Command: $*${NC}"
    echo
    TIMEFORMAT='⏱ Execution time: %R seconds'
    time {
        "$@"
        echo
    }
    echo -e "${GREEN}✓ Complete${NC}"
    echo "=================================================="
    echo
}

echo -e "${BLUE}Step 1: Building CLI...${NC}"
time_command "Build CLI" make build-cli

echo -e "${BLUE}Step 2: Testing INSERT commands${NC}"
time_command "Insert #1 (dark mode preference)" \
    ./bin/hippocampus insert -binary "$TEST_DB" \
    -key "user_preference_theme" \
    -text "User prefers dark mode for the interface"

time_command "Insert #2 (Python preference)" \
    ./bin/hippocampus insert -binary "$TEST_DB" \
    -key "user_preference_language" \
    -text "User prefers Python programming language"

time_command "Insert #3 (shellfish allergy)" \
    ./bin/hippocampus insert -binary "$TEST_DB" \
    -key "allergy_shellfish" \
    -text "User is allergic to shellfish and seafood"

time_command "Insert #4 (running hobby)" \
    ./bin/hippocampus insert -binary "$TEST_DB" \
    -key "hobby_running" \
    -text "User enjoys running marathons and trains regularly"

time_command "Insert #5 (coffee preference)" \
    ./bin/hippocampus insert -binary "$TEST_DB" \
    -key "preference_coffee" \
    -text "User drinks coffee every morning, prefers espresso"

echo -e "${BLUE}Step 3: Testing SEARCH commands${NC}"
time_command "Search #1 (UI preferences with defaults)" \
    ./bin/hippocampus search -binary "$TEST_DB" \
    -text "What are the user's UI preferences?"

time_command "Search #2 (programming with custom params)" \
    ./bin/hippocampus search -binary "$TEST_DB" \
    -text "programming languages" \
    -epsilon 0.4 \
    -threshold 0.4 \
    -top-k 3

time_command "Search #3 (food allergies, high threshold)" \
    ./bin/hippocampus search -binary "$TEST_DB" \
    -text "food allergies" \
    -epsilon 0.3 \
    -threshold 0.6 \
    -top-k 5

time_command "Search #4 (hobbies and activities)" \
    ./bin/hippocampus search -binary "$TEST_DB" \
    -text "What does the user do for exercise?" \
    -epsilon 0.3 \
    -threshold 0.5 \
    -top-k 3

echo -e "${BLUE}Step 4: Creating and testing CSV bulk insert${NC}"
cat > test_data.csv << EOF
location_city,User lives in Seattle Washington
job_company,User works at Amazon Web Services in cloud infrastructure
pet_dog,User has a golden retriever named Max who is 3 years old
education_university,User graduated from MIT with a computer science degree
hobby_photography,User enjoys landscape photography on weekends
food_favorite,User loves Italian food especially pasta carbonara
travel_destination,User wants to visit Japan and explore Tokyo
book_preference,User reads science fiction novels by authors like Ted Chiang
music_genre,User listens to jazz and classical music while coding
workout_routine,User goes to the gym three times per week
EOF
echo -e "${GREEN}✓ CSV file created with 10 entries${NC}"
echo

time_command "Bulk CSV insert (10 entries)" \
    ./bin/hippocampus insert-csv -binary "$TEST_DB" -csv test_data.csv

time_command "Search #5 (verify CSV data - location)" \
    ./bin/hippocampus search -binary "$TEST_DB" \
    -text "Where does the user live?" \
    -epsilon 0.3 \
    -threshold 0.5 \
    -top-k 3

time_command "Search #6 (verify CSV data - work)" \
    ./bin/hippocampus search -binary "$TEST_DB" \
    -text "What company does the user work for?" \
    -epsilon 0.3 \
    -threshold 0.5 \
    -top-k 3

echo -e "${BLUE}Step 5: Testing AGENT-CURATE (AI decomposition)${NC}"
echo -e "${YELLOW}⚠ This requires AWS Bedrock access with Nova Lite model${NC}"
echo

time_command "Agent curation (complex biographical text)" \
    ./bin/hippocampus agent-curate -binary "$TEST_DB" \
    -text "My name is Sarah Chen, I'm 34 years old, and I work as a senior software engineer at Google in Mountain View. I specialize in distributed systems and have 10 years of experience. I'm allergic to peanuts and latex. I love hiking in the Sierra Nevada mountains and have climbed Half Dome twice. I speak Mandarin fluently and am learning Spanish." \
    -importance high \
    -model "us.amazon.nova-lite-v1:0" \
    -bedrock-region "ap-southeast-2" \
    -timeout-ms 50

echo -e "${BLUE}Step 6: Testing searches on curated data${NC}"
time_command "Search #7 (personal info about Sarah)" \
    ./bin/hippocampus search -binary "$TEST_DB" \
    -text "Tell me about Sarah Chen" \
    -epsilon 0.3 \
    -threshold 0.4 \
    -top-k 5

time_command "Search #8 (Sarah's allergies)" \
    ./bin/hippocampus search -binary "$TEST_DB" \
    -text "What allergies does Sarah have?" \
    -epsilon 0.3 \
    -threshold 0.5 \
    -top-k 3

time_command "Search #9 (Sarah's hobbies)" \
    ./bin/hippocampus search -binary "$TEST_DB" \
    -text "outdoor activities" \
    -epsilon 0.35 \
    -threshold 0.45 \
    -top-k 5

time_command "Search #10 (Sarah's work)" \
    ./bin/hippocampus search -binary "$TEST_DB" \
    -text "What does Sarah do professionally?" \
    -epsilon 0.3 \
    -threshold 0.5 \
    -top-k 3

echo "=================================================="
echo -e "${GREEN}✓✓✓ All tests passed! ✓✓✓${NC}"
echo "=================================================="
echo
echo -e "${CYAN}Test Summary:${NC}"
echo "  - Total inserts: 15+ (5 manual + 10 CSV + agent-curated)"
echo "  - Total searches: 10"
echo "  - Database file: $TEST_DB"
echo "  - CSV file: test_data.csv"
echo
ls -lh "$TEST_DB" | awk '{print "  - Database size: " $5}'
echo
echo -e "${YELLOW}To clean up:${NC}"
echo "  rm $TEST_DB test_data.csv"
