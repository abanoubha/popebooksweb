#!/bin/bash
set -e

echo "Creating Book..."
BOOK_RES=$(curl -s -X POST http://localhost:8000/api/books -d '{"name":"Test Book"}')
BOOK_ID=$(echo $BOOK_RES | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
echo "Book ID: $BOOK_ID"

echo "Creating Page..."
PAGE_RES=$(curl -s -X POST http://localhost:8000/api/pages -d "{\"bookId\":$BOOK_ID, \"name\":\"Page 1\", \"number\":1, \"content\":\"Original Content\"}")
PAGE_ID=$(echo $PAGE_RES | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
echo "Page ID: $PAGE_ID"

echo "Updating Page..."
UPDATE_RES=$(curl -s -X PUT http://localhost:8000/api/pages/$PAGE_ID -d "{\"bookId\":$BOOK_ID, \"name\":\"Page 1 Updated\", \"number\":1, \"content\":\"Updated Content\"}")
echo "Update Response: $UPDATE_RES"

echo "Verifying Update..."
GET_RES=$(curl -s "http://localhost:8000/api/pages?bookId=$BOOK_ID")
echo "Get Response: $GET_RES"

if [[ $GET_RES == *"Updated Content"* ]]; then
    echo "SUCCESS: Content updated correctly."
else
    echo "FAILURE: Content not updated."
    exit 1
fi
