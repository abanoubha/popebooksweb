#!/bin/bash
set -e

echo "Creating Book..."
BOOK_RES=$(curl -s -X POST http://localhost:8000/api/books -d '{"name":"Original Book"}')
BOOK_ID=$(echo $BOOK_RES | grep -o '"id":[0-9]*' | head -1 | cut -d: -f2)
echo "Book ID: $BOOK_ID"

echo "Updating Book Name..."
UPDATE_RES=$(curl -s -X PUT http://localhost:8000/api/books/$BOOK_ID -d "{\"name\":\"Updated Book Name\"}")
echo "Update Response: $UPDATE_RES"

echo "Verifying Update..."
GET_RES=$(curl -s "http://localhost:8000/api/books")
echo "Get Response: $GET_RES"

if [[ $GET_RES == *"Updated Book Name"* ]]; then
    echo "SUCCESS: Book name updated correctly."
else
    echo "FAILURE: Book name not updated."
    exit 1
fi
