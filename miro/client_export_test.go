package miro

// Tests for the board export job methods, split out of client_test.go.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newExportTestClient starts a test server with the given handler and returns
// a client pointed at it. The server is closed automatically at test end.
func newExportTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return newTestClientWithServer(server.URL)
}

// checkExportField fails the test when got differs from want.
func checkExportField[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func TestCreateExportJob_Success(t *testing.T) {
	client := newExportTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkExportField(t, "method", r.Method, http.MethodPost)
		checkExportField(t, "path", r.URL.Path, "/orgs/org123/boards/export/jobs")
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, map[string]interface{}{
			"id":        "job123",
			"status":    "pending",
			"requestId": "req456",
		})
	})

	result, err := client.CreateExportJob(context.Background(), CreateExportJobArgs{
		OrgID:    "org123",
		BoardIDs: []string{"board1", "board2"},
		Format:   "pdf",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkExportField(t, "JobID", result.JobID, "job123")
	checkExportField(t, "Status", result.Status, "pending")
}

// TestExportJob_ValidationErrors covers input validation across the export
// job methods; every case must fail with a message naming the problem.
func TestExportJob_ValidationErrors(t *testing.T) {
	client := NewClient(testConfig(), testLogger())
	ctx := context.Background()

	tests := []struct {
		name    string
		call    func() error
		errText string
	}{
		{"create with empty org_id", func() error {
			_, err := client.CreateExportJob(ctx, CreateExportJobArgs{BoardIDs: []string{"board1"}})
			return err
		}, "org_id is required"},
		{"create with empty board_ids", func() error {
			_, err := client.CreateExportJob(ctx, CreateExportJobArgs{OrgID: "org123", BoardIDs: []string{}})
			return err
		}, "board_ids is required"},
		{"create with too many boards", func() error {
			_, err := client.CreateExportJob(ctx, CreateExportJobArgs{OrgID: "org123", BoardIDs: make([]string, 51)})
			return err
		}, "maximum 50 boards"},
		{"create with invalid format", func() error {
			_, err := client.CreateExportJob(ctx, CreateExportJobArgs{OrgID: "org123", BoardIDs: []string{"board1"}, Format: "png"})
			return err
		}, "format must be pdf, svg, or html"},
		{"status with empty org_id", func() error {
			_, err := client.GetExportJobStatus(ctx, GetExportJobStatusArgs{JobID: "job123"})
			return err
		}, "org_id is required"},
		{"status with empty job_id", func() error {
			_, err := client.GetExportJobStatus(ctx, GetExportJobStatusArgs{OrgID: "org123"})
			return err
		}, "job_id is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("expected error containing %q, got: %v", tt.errText, err)
			}
		})
	}
}

func TestGetExportJobStatus_Success(t *testing.T) {
	client := newExportTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkExportField(t, "method", r.Method, http.MethodGet)
		checkExportField(t, "path", r.URL.Path, "/orgs/org123/boards/export/jobs/job456")
		writeJSON(w, map[string]interface{}{
			"id":             "job456",
			"status":         "in_progress",
			"progress":       50,
			"boardsTotal":    10,
			"boardsExported": 5,
		})
	})

	result, err := client.GetExportJobStatus(context.Background(), GetExportJobStatusArgs{
		OrgID: "org123",
		JobID: "job456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkExportField(t, "Status", result.Status, "in_progress")
	checkExportField(t, "Progress", result.Progress, 50)
	checkExportField(t, "BoardsExported", result.BoardsExported, 5)
}

func TestGetExportJobResults_Success(t *testing.T) {
	client := newExportTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		checkExportField(t, "method", r.Method, http.MethodGet)
		checkExportField(t, "path", r.URL.Path, "/orgs/org123/boards/export/jobs/job456/results")
		writeJSON(w, map[string]interface{}{
			"id":     "job456",
			"status": "completed",
			"data": []map[string]interface{}{
				{
					"boardId":     "board1",
					"boardName":   "Board One",
					"downloadUrl": "https://download.miro.com/exports/board1.pdf",
					"format":      "pdf",
				},
				{
					"boardId":     "board2",
					"boardName":   "Board Two",
					"downloadUrl": "https://download.miro.com/exports/board2.pdf",
					"format":      "pdf",
				},
			},
		})
	})

	result, err := client.GetExportJobResults(context.Background(), GetExportJobResultsArgs{
		OrgID: "org123",
		JobID: "job456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkExportField(t, "Status", result.Status, "completed")
	checkExportField(t, "Boards count", len(result.Boards), 2)
	checkExportField(t, "download URL", result.Boards[0].DownloadURL, "https://download.miro.com/exports/board1.pdf")
}

func TestGetExportJobResults_NotCompleted(t *testing.T) {
	client := newExportTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]interface{}{
			"id":     "job456",
			"status": "in_progress",
		})
	})

	result, err := client.GetExportJobResults(context.Background(), GetExportJobResultsArgs{
		OrgID: "org123",
		JobID: "job456",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checkExportField(t, "Status", result.Status, "in_progress")
	if result.Boards != nil {
		t.Error("expected nil Boards for incomplete job")
	}
	if !strings.Contains(result.Message, "not yet available") {
		t.Errorf("expected 'not yet available' message, got: %s", result.Message)
	}
}
