package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var tableName = "TableName"

// Params struct represents the pagination parameters
type Params struct {
	Page     int64  `json:"page"`
	PageSize int64  `json:"pagesize"`
	OrderBy  string `json:"orderby"`
	Search   string `json:"search"`
}

// Entry represents a DynamoDB item for the Entry table
type Entry struct {
	KeyCond string `dynamodbav:"key_cond" json:"key_cond"` 
	SortKey string `dynamodbav:"sort_key" json:"sort_key"`

}

type Response struct {
	Data []Entry
	Page int64
	Size int64
}

func main() {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal("Failed to load AWS configuration")
	}

	// Create a DynamoDB client
	client := dynamodb.NewFromConfig(cfg)

	h := Handler{client}
	// Create a new Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Route
	e.GET("/paginate", h.handlePagination)

	// Start the HTTP server
	e.Logger.Fatal(e.Start(":8080"))
}

type DynamoClient interface {
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

type Handler struct {
	client DynamoClient
}

func (h *Handler) extractParams(c echo.Context) Params {
	// Parse the query parameters to get Pagination parameters
	pageStr := c.QueryParam("page")
	pageSizeStr := c.QueryParam("pagesize")
	orderBy := c.QueryParam("orderby")
	search := c.QueryParam("search")

	page, err := strconv.ParseInt(pageStr, 10, 64)
	if err != nil {
		page = 1
	}

	if page <= 0 {
		page = 1
	}

	pageSize, err := strconv.ParseInt(pageSizeStr, 10, 64)
	if err != nil {
		pageSize = 10
	}

	if page <= 0 {
		page = 10
	}

	return Params{
		Page:     page,
		PageSize: pageSize,
		OrderBy:  orderBy,
		Search:   search,
	}
}

func (h *Handler) handlePagination(c echo.Context) error {
	keyCond := c.QueryParam("key_condition")
	if keyCond == "" {
		return c.String(http.StatusBadRequest, "Invalid key_condition parameter")
	}

	params := h.extractParams(c)

	// Pagination parameters
	limit := int32(params.PageSize)
	var pageNumber int64 = 1

	var lastEvaluatedKey map[string]types.AttributeValue
	var itemsForPage []Entry

	for {
		// Prepare the query input
		input := &dynamodb.QueryInput{
			Limit:                  &limit,
			TableName:              &tableName,
			KeyConditionExpression: aws.String("key_condition = :keyCond"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":keyCond": &types.AttributeValueMemberS{Value: keyCond},
			},
			ExclusiveStartKey: lastEvaluatedKey,
		}

		// Set the order by attribute if provided
		if params.OrderBy != "" {
			input.ScanIndexForward = aws.Bool(true) // Default to ascending order
			if params.OrderBy[0] == '-' {
				// If the attribute starts with '-', it indicates descending order
				input.ScanIndexForward = aws.Bool(false)
			}

		}

		// Perform the query
		result, err := h.client.Query(context.TODO(), input)
		if err != nil {
			c.Logger().Error(err)
			return c.String(http.StatusInternalServerError, "Error in DynamoDB query")
		}

		// Unmarshal DynamoDB items into DbPermission struct
		for _, item := range result.Items {
			var entry Entry
			err := attributevalue.UnmarshalMap(item, &entry)
			if err != nil {
				c.Logger().Error(err)
				return c.String(http.StatusInternalServerError, "Error unmarshalling DynamoDB item")
			}

			if params.Search != "" {
				// Add a condition here to filter items based on the "SortKey" attribute
				if strings.Contains(strings.ToLower(entry.SortKey), strings.ToLower(params.Search)) {
					itemsForPage = append(itemsForPage, entry)
				}
				continue
			}

			itemsForPage = append(itemsForPage, entry)
		}

		// Update lastEvaluatedKey for the next iteration
		lastEvaluatedKey = result.LastEvaluatedKey

		// Break the loop if there are no more pages or if we've reached the requested page
		if lastEvaluatedKey == nil || pageNumber >= params.Page {
			break
		}

		// Increment the page number
		pageNumber++
	}

	// Calculate the start and end indices for the requested page
	startIndex := int((pageNumber - 1) * params.PageSize)
	endIndex := int(pageNumber * params.PageSize)

	// Ensure the indices are within the range of the items
	if startIndex < 0 {
		startIndex = 0
	}
	if endIndex > len(itemsForPage) {
		endIndex = len(itemsForPage)
	}

	actualSize := int64(endIndex - startIndex)

	// Extract the items for the requested page
	pageItems := itemsForPage[startIndex:endIndex]

	res := Response{
		Data: pageItems,
		Page: pageNumber,
		Size: actualSize,
	}

	// Convert the items to JSON
	responseData, err := json.Marshal(res)
	if err != nil {
		c.Logger().Error(err)
		return c.String(http.StatusInternalServerError, "Error converting items to JSON")
	}

	// Respond with the paginated results for the requested page
	return c.JSONBlob(http.StatusOK, responseData)
}
