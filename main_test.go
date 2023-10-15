package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDynamoDB is a mock implementation of the DynamoDB client
type MockDynamoDB struct {
	mock.Mock
}

func (m *MockDynamoDB) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*dynamodb.QueryOutput), args.Error(1)
}

func TestHandlePagination(t *testing.T) {
	tests := []struct {
		name             string
		queryParam       string
		expectedStatus   int
		expectedResponse Response
		mockOutput       *dynamodb.QueryOutput
		mockError        error
	}{
		{
			name:           "Successful Query",
			queryParam:     "key_condition=test",
			expectedStatus: http.StatusOK,
			expectedResponse: Response{
				Data: []Entry{
					{KeyCond: "test", SortKey: "item1"},
					{KeyCond: "test", SortKey: "item2"},
				},
				Page: 1,
				Size: 2,
			},
			mockOutput: &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{"key_cond": &types.AttributeValueMemberS{Value: "test"}, "sort_key": &types.AttributeValueMemberS{Value: "item1"}},
					{"key_cond": &types.AttributeValueMemberS{Value: "test"}, "sort_key": &types.AttributeValueMemberS{Value: "item2"}},
				},
				LastEvaluatedKey: nil,
			},
			mockError: nil,
		},
		{
			name:           "Successful Query Order By",
			queryParam:     "key_condition=test&orderby=-sort_key",
			expectedStatus: http.StatusOK,
			expectedResponse: Response{
				Data: []Entry{
					{KeyCond: "test", SortKey: "item2"},
					{KeyCond: "test", SortKey: "item1"},
				},
				Page: 1,
				Size: 2,
			},
			mockOutput: &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{"key_cond": &types.AttributeValueMemberS{Value: "test"}, "sort_key": &types.AttributeValueMemberS{Value: "item2"}},
					{"key_cond": &types.AttributeValueMemberS{Value: "test"}, "sort_key": &types.AttributeValueMemberS{Value: "item1"}},
				},
				LastEvaluatedKey: nil,
			},
			mockError: nil,
		},
		{
			name:           "Successful Query Search",
			queryParam:     "key_condition=test&search=1",
			expectedStatus: http.StatusOK,
			expectedResponse: Response{
				Data: []Entry{
					{KeyCond: "test", SortKey: "item1"},
				},
				Page: 1,
				Size: 1,
			},
			mockOutput: &dynamodb.QueryOutput{
				Items: []map[string]types.AttributeValue{
					{"key_cond": &types.AttributeValueMemberS{Value: "test"}, "sort_key": &types.AttributeValueMemberS{Value: "item1"}},
					{"key_cond": &types.AttributeValueMemberS{Value: "test"}, "sort_key": &types.AttributeValueMemberS{Value: "item2"}},
				},
				LastEvaluatedKey: nil,
			},
			mockError: nil,
		},
		{
			name:           "Invalid Key Condition",
			queryParam:     "",
			expectedStatus: http.StatusBadRequest,
			expectedResponse: Response{
				Data: nil,
				Page: 0,
				Size: 0,
			},
			mockOutput: nil,
			mockError:  nil,
		},
		{
			name:           "DynamoDB Query Error",
			queryParam:     "key_condition=test",
			expectedStatus: http.StatusInternalServerError,
			expectedResponse: Response{
				Data: nil,
				Page: 0,
				Size: 0,
			},
			mockOutput: nil,
			mockError:  errors.New("DynamoDB query error"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create a new instance of the mock DynamoDB client
			mockDynamoDB := new(MockDynamoDB)

			// Set up a test echo instance
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/paginate?"+test.queryParam, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set up the mock DynamoDB response
			if test.mockOutput != nil || test.mockError != nil {
				mockDynamoDB.On("Query", mock.Anything, mock.Anything).Return(test.mockOutput, test.mockError)
			}

			// Set up the handler with the mock DynamoDB client
			handler := &Handler{client: mockDynamoDB}

			// Call the handler
			_ = handler.handlePagination(c)

			assert.Equal(t, test.expectedStatus, rec.Code)

			if test.expectedStatus == http.StatusOK {
				// Unmarshal the response
				var response Response
				err := json.Unmarshal(rec.Body.Bytes(), &response)
				assert.NoError(t, err)

				// Check the response data
				assert.Equal(t, test.expectedResponse, response)
			}

			// Assert that the Query method on the mock DynamoDB client was called
			mockDynamoDB.AssertExpectations(t)
		})
	}
}
