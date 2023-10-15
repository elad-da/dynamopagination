# DynamoDB Pagination, Order By, and Free Search Sample

## Overview

This repository contains a sample code demonstrating how to implement pagination, ordering, and free-text search on the sort key in DynamoDB using Go and the Echo web framework.


## Connecting to a Real DynamoDB

1. **Set Up a DynamoDB Table:**
    Create a DynamoDB table with the desired structure.

2. **Update Configuration:**
    Open main.go and update the tableName variable with the name of your DynamoDB table.
    ```go
    var tableName = "YourTableName"
    ```
3. **Update Attribute Mapping:**
    Ensure that the attributes in the `Entry` struct match the attributes in your DynamoDB table.
    ```go
    type Entry struct {
        KeyCond string `dynamodbav:"key_condition" json:"key_condition"` 
        SortKey string `dynamodbav:"sort_key" json:"sort_key"`          
    }
    ```

## Usage

1. **Run the Application:**

   ```bash
    go run main.go
    ```
    The server will be accessible at `http://localhost:8080`.

2. **Make HTTP Requests:**

    Make requests to the `/paginate` endpoint with query parameters to test pagination, ordering, and free-text search.
   ```bash
    curl "http://localhost:8080/paginate?key_condition=test&page=1&pagesize=10&orderby=sort_key&search=example"
    ```