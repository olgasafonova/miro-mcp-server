package resources

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	miroclient "github.com/olgasafonova/miro-mcp-server/miro"
)

// MockClient implements ResourceClient for testing
type MockClient struct {
	GetBoardSummaryFn func(ctx context.Context, args miroclient.GetBoardSummaryArgs) (miroclient.GetBoardSummaryResult, error)
	ListAllItemsFn    func(ctx context.Context, args miroclient.ListAllItemsArgs) (miroclient.ListAllItemsResult, error)
	ListItemsFn       func(ctx context.Context, args miroclient.ListItemsArgs) (miroclient.ListItemsResult, error)
}

func (m *MockClient) GetBoardSummary(ctx context.Context, args miroclient.GetBoardSummaryArgs) (miroclient.GetBoardSummaryResult, error) {
	if m.GetBoardSummaryFn != nil {
		return m.GetBoardSummaryFn(ctx, args)
	}
	return miroclient.GetBoardSummaryResult{
		ID:         args.BoardID,
		Name:       "Test Board",
		TotalItems: 10,
		ItemCounts: map[string]int{"sticky_note": 5, "shape": 3, "text": 2},
		Message:    "Board summary retrieved",
	}, nil
}

func (m *MockClient) ListAllItems(ctx context.Context, args miroclient.ListAllItemsArgs) (miroclient.ListAllItemsResult, error) {
	if m.ListAllItemsFn != nil {
		return m.ListAllItemsFn(ctx, args)
	}
	return miroclient.ListAllItemsResult{
		Items: []miroclient.ItemSummary{
			{ID: "item1", Type: "sticky_note", Content: "Test sticky"},
			{ID: "item2", Type: "shape", Content: "Test shape"},
		},
		Count:   2,
		Message: "Items retrieved",
	}, nil
}

func (m *MockClient) ListItems(ctx context.Context, args miroclient.ListItemsArgs) (miroclient.ListItemsResult, error) {
	if m.ListItemsFn != nil {
		return m.ListItemsFn(ctx, args)
	}
	return miroclient.ListItemsResult{
		Items: []miroclient.ItemSummary{
			{ID: "frame1", Type: "frame", Content: "Test Frame"},
		},
		Count: 1,
	}, nil
}

func TestExtractBoardID(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		prefix  string
		want    string
		wantErr bool
	}{
		{
			name:    "valid board URI",
			uri:     "miro://board/abc123",
			prefix:  "miro://board/",
			want:    "abc123",
			wantErr: false,
		},
		{
			name:    "board URI with items suffix",
			uri:     "miro://board/abc123/items",
			prefix:  "miro://board/",
			want:    "abc123/items",
			wantErr: false,
		},
		{
			name:    "invalid prefix",
			uri:     "other://board/abc123",
			prefix:  "miro://board/",
			want:    "",
			wantErr: true,
		},
		{
			name:    "missing board ID",
			uri:     "miro://board/",
			prefix:  "miro://board/",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractBoardID(tt.uri, tt.prefix)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractBoardID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractBoardID() = %v, want %v", got, tt.want)
			}
		})
	}
}

// resourceHandler is the shape shared by the board resource handlers.
type resourceHandler func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error)

// readResourceRequest builds a ReadResourceRequest for uri.
func readResourceRequest(uri string) *mcp.ReadResourceRequest {
	return &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: uri},
	}
}

// mustReadResource calls handler with uri, failing the test on error.
func mustReadResource(t *testing.T, handler resourceHandler, uri string) *mcp.ReadResourceResult {
	t.Helper()
	result, err := handler(context.Background(), readResourceRequest(uri))
	if err != nil {
		t.Fatalf("resource handler error = %v", err)
	}
	return result
}

// assertSingleJSONContent checks the result carries one JSON content item.
func assertSingleJSONContent(t *testing.T, result *mcp.ReadResourceResult) {
	t.Helper()
	if len(result.Contents) != 1 {
		t.Errorf("expected 1 content, got %d", len(result.Contents))
	}
	if result.Contents[0].MIMEType != "application/json" {
		t.Errorf("content MIMEType = %v, want application/json", result.Contents[0].MIMEType)
	}
}

// assertReadFails checks that handler surfaces an error for uri.
func assertReadFails(t *testing.T, handler resourceHandler, uri string) {
	t.Helper()
	if _, err := handler(context.Background(), readResourceRequest(uri)); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestHandleBoardResource(t *testing.T) {
	registry := &Registry{client: &MockClient{}}
	uri := "miro://board/test-board-123"

	result := mustReadResource(t, registry.handleBoardResource, uri)

	assertSingleJSONContent(t, result)
	if result.Contents[0].URI != uri {
		t.Errorf("content URI = %v, want %v", result.Contents[0].URI, uri)
	}
}

func TestHandleBoardItemsResource(t *testing.T) {
	registry := &Registry{client: &MockClient{}}

	result := mustReadResource(t, registry.handleBoardItemsResource, "miro://board/test-board-123/items")

	assertSingleJSONContent(t, result)
}

func TestHandleBoardFramesResource(t *testing.T) {
	registry := &Registry{client: &MockClient{}}

	result := mustReadResource(t, registry.handleBoardFramesResource, "miro://board/test-board-123/frames")

	if len(result.Contents) != 1 {
		t.Errorf("expected 1 content, got %d", len(result.Contents))
	}
}

func TestNewRegistry(t *testing.T) {
	mock := &MockClient{}
	registry := NewRegistry(mock)

	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}

	if registry.client != mock {
		t.Error("NewRegistry() client not set correctly")
	}
}

func TestHandleBoardResourceWithError(t *testing.T) {
	registry := &Registry{client: &MockClient{
		GetBoardSummaryFn: func(ctx context.Context, args miroclient.GetBoardSummaryArgs) (miroclient.GetBoardSummaryResult, error) {
			return miroclient.GetBoardSummaryResult{}, context.DeadlineExceeded
		},
	}}

	assertReadFails(t, registry.handleBoardResource, "miro://board/test-board-123")
}

func TestHandleBoardItemsResourceWithError(t *testing.T) {
	registry := &Registry{client: &MockClient{
		ListAllItemsFn: func(ctx context.Context, args miroclient.ListAllItemsArgs) (miroclient.ListAllItemsResult, error) {
			return miroclient.ListAllItemsResult{}, context.DeadlineExceeded
		},
	}}

	assertReadFails(t, registry.handleBoardItemsResource, "miro://board/test-board-123/items")
}

func TestHandleBoardFramesResourceWithError(t *testing.T) {
	registry := &Registry{client: &MockClient{
		ListItemsFn: func(ctx context.Context, args miroclient.ListItemsArgs) (miroclient.ListItemsResult, error) {
			return miroclient.ListItemsResult{}, context.DeadlineExceeded
		},
	}}

	assertReadFails(t, registry.handleBoardFramesResource, "miro://board/test-board-123/frames")
}
