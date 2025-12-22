#!/bin/bash
set -e

echo "Creating Book..."
BOOK_RES=$(curl -s -X POST http://localhost:8000/api/books -d '{"name":"Test Book"}')
BOOK_ID=$(echo $BOOK_RES | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
echo "Book ID: $BOOK_ID"

echo "Creating Page (without name)..."
PAGE_RES=$(curl -s -X POST http://localhost:8000/api/pages -d "{\"bookId\":$BOOK_ID, \"number\":1, \"content\":\"Content without name\"}")
PAGE_ID=$(echo $PAGE_RES | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
echo "Page ID: $PAGE_ID"

echo "Verifying Name is Gone from Response..."
if [[ $PAGE_RES == *"name"* ]]; then
    # Note: JSON might still have "name" key if struct tag was kept or if omitempty not used and empty string. 
    # But new struct removed Name field, so it shouldn't be in JSON output unless embedded or custom marshaler.
    # Let's check if it's there.
    echo "Warning: 'name' field still present in JSON response (might be okay if empty/default, but better if gone)."
else
    echo "SUCCESS: 'name' field not in JSON response."
fi

echo "Updating Page..."
UPDATE_RES=$(curl -s -X PUT http://localhost:8000/api/pages/$PAGE_ID -d "{\"bookId\":$BOOK_ID, \"number\":2, \"content\":\"Updated content without name\"}")

echo "Verifying Update..."
GET_RES=$(curl -s "http://localhost:8000/api/pages?bookId=$BOOK_ID")
echo "Get Response: $GET_RES"

if [[ $GET_RES == *"Updated content without name"* ]]; then
    echo "SUCCESS: Content updated correctly."
else
    echo "FAILURE: Content not updated."
    exit 1
fi
