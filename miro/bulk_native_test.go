package miro

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// bulkProbe records every request the client made, so a test can assert both
// how many calls went out and where they went.
type bulkProbe struct {
	mu     sync.Mutex
	paths  []string
	bodies []string
}

func (p *bulkProbe) record(r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.paths = append(p.paths, r.URL.Path)
	p.bodies = append(p.bodies, string(body))
}

func (p *bulkProbe) calls() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.paths...)
}

func threeStickies() []BulkCreateItem {
	return []BulkCreateItem{
		{Type: "sticky_note", Content: "one", Color: "yellow"},
		{Type: "sticky_note", Content: "two"},
		{Type: "sticky_note", Content: "three"},
	}
}

// bulkCreatedResponse is the native endpoint's 201 envelope.
func bulkCreatedResponse(ids ...string) map[string]any {
	data := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		data = append(data, map[string]any{"id": id})
	}
	return map[string]any{"type": "bulk-list", "data": data}
}

// TestBulkCreate_UsesOneRequest is the point of the change: N items used to
// cost N requests, and now cost one.
func TestBulkCreate_UsesOneRequest(t *testing.T) {
	probe := &bulkProbe{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		probe.record(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(bulkCreatedResponse("i1", "i2", "i3"))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.BulkCreate(context.Background(), BulkCreateArgs{
		BoardID: "board123",
		Items:   threeStickies(),
	})
	if err != nil {
		t.Fatalf("BulkCreate: %v", err)
	}

	calls := probe.calls()
	if len(calls) != 1 {
		t.Fatalf("made %d requests for 3 items, want 1: %v", len(calls), calls)
	}
	if !strings.HasSuffix(calls[0], "/items/bulk") {
		t.Errorf("posted to %q, want /items/bulk", calls[0])
	}
	if result.Created != 3 || len(result.ItemIDs) != 3 {
		t.Errorf("Created=%d IDs=%v, want 3", result.Created, result.ItemIDs)
	}
	if len(result.ItemURLs) != 3 {
		t.Errorf("ItemURLs = %v, want one per created item", result.ItemURLs)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want none", result.Errors)
	}
}

// TestBulkCreate_ReusesStickyColorVocabulary pins the reason bulkCreateItemBody
// calls the same builder the typed create does. Sticky fill colours are a named
// enum, not hex, and a hand-rolled bulk mapping would be the obvious place for
// that to drift.
func TestBulkCreate_ReusesStickyColorVocabulary(t *testing.T) {
	probe := &bulkProbe{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		probe.record(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(bulkCreatedResponse("i1"))
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	if _, err := client.BulkCreate(context.Background(), BulkCreateArgs{
		BoardID: "board123",
		Items:   []BulkCreateItem{{Type: "sticky_note", Content: "hi", Color: "yellow"}},
	}); err != nil {
		t.Fatalf("BulkCreate: %v", err)
	}

	var sent []map[string]any
	if err := json.Unmarshal([]byte(probe.bodies[0]), &sent); err != nil {
		t.Fatalf("decode sent body: %v", err)
	}
	if sent[0]["type"] != "sticky_note" {
		t.Errorf("type = %v, want sticky_note", sent[0]["type"])
	}
	style, ok := sent[0]["style"].(map[string]any)
	if !ok {
		t.Fatalf("no style block sent: %v", sent[0])
	}
	if got := style["fillColor"]; got != normalizeStickyColor("yellow") {
		t.Errorf("fillColor = %v, want %v (normalized)", got, normalizeStickyColor("yellow"))
	}
}

// TestBulkCreate_FallsBackOnValidationRejection covers the safe half of the
// asymmetry. A 400 means the batch was rejected on validation, and the
// endpoint's transactionality guarantees nothing was created — so re-running
// per item cannot double-create, and is the only way to report WHICH item was
// bad. The caller keeps the per-item contract at the cost of one extra request.
func TestBulkCreate_FallsBackOnValidationRejection(t *testing.T) {
	probe := &bulkProbe{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		probe.record(r)
		w.Header().Set("Content-Type", "application/json")

		if strings.HasSuffix(r.URL.Path, "/items/bulk") {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "invalid item"})
			return
		}
		// Per-item fan-out: every sticky succeeds individually.
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "fanned"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.BulkCreate(context.Background(), BulkCreateArgs{
		BoardID: "board123",
		Items:   threeStickies(),
	})
	if err != nil {
		t.Fatalf("BulkCreate: %v", err)
	}

	calls := probe.calls()
	if len(calls) != 4 {
		t.Fatalf("made %d requests, want 4 (1 bulk + 3 fan-out): %v", len(calls), calls)
	}
	fanned := 0
	for _, p := range calls {
		if strings.HasSuffix(p, "/sticky_notes") {
			fanned++
		}
	}
	if fanned != 3 {
		t.Errorf("fan-out issued %d per-item calls, want 3: %v", fanned, calls)
	}
	if result.Created != 3 {
		t.Errorf("Created = %d, want 3 (fallback should still land every valid item)", result.Created)
	}
}

// TestBulkCreate_DoesNotFallBackOnUnknownOutcome covers the dangerous half. A
// 429 or 5xx leaves the outcome UNKNOWN — the transaction may have committed
// with the response lost — so a fan-out here could create every item twice.
//
// This is the assertion that would catch someone "simplifying" the fallback to
// fire on any error. The count is the whole test: exactly one request leaves.
func TestBulkCreate_DoesNotFallBackOnUnknownOutcome(t *testing.T) {
	for name, status := range map[string]int{
		"rate limited": http.StatusTooManyRequests,
		"server error": http.StatusInternalServerError,
		"bad gateway":  http.StatusBadGateway,
	} {
		t.Run(name, func(t *testing.T) {
			probe := &bulkProbe{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				probe.record(r)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(map[string]any{"message": "nope"})
			}))
			defer server.Close()

			client := newTestClientWithServer(server.URL)
			result, err := client.BulkCreate(context.Background(), BulkCreateArgs{
				BoardID: "board123",
				Items:   threeStickies(),
			})
			if err != nil {
				t.Fatalf("BulkCreate returned a hard error, want a per-item result: %v", err)
			}

			for _, p := range probe.calls() {
				if strings.HasSuffix(p, "/sticky_notes") {
					t.Fatalf("fell back to the fan-out after an unknown-outcome failure; "+
						"this can double-create. calls: %v", probe.calls())
				}
			}

			if result.Created != 0 {
				t.Errorf("Created = %d, want 0", result.Created)
			}
			if len(result.FailedItems) != 3 {
				t.Errorf("FailedItems = %d, want 3 (every item accounted for)", len(result.FailedItems))
			}
			if len(result.RetriableIDs) != 3 {
				t.Errorf("RetriableIDs = %v, want all three retriable on %s", result.RetriableIDs, name)
			}
			// The caller must be told the outcome is not provable, or they will
			// retry blind and duplicate.
			if !strings.Contains(result.Message, "lost in transit") {
				t.Errorf("Message = %q, want it to flag the unprovable outcome", result.Message)
			}
		})
	}
}

// TestBulkCreate_UnsupportedTypeFallsBack pins that a locally unmappable item
// takes the fan-out path. Nothing was sent, so nothing was created, and the
// fan-out reports the bad item by index while still landing the good ones.
func TestBulkCreate_UnsupportedTypeFallsBack(t *testing.T) {
	probe := &bulkProbe{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		probe.record(r)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "ok1"})
	}))
	defer server.Close()

	client := newTestClientWithServer(server.URL)
	result, err := client.BulkCreate(context.Background(), BulkCreateArgs{
		BoardID: "board123",
		Items: []BulkCreateItem{
			{Type: "sticky_note", Content: "fine"},
			{Type: "hologram", Content: "not a thing"},
		},
	})
	if err != nil {
		t.Fatalf("BulkCreate: %v", err)
	}

	for _, p := range probe.calls() {
		if strings.HasSuffix(p, "/items/bulk") {
			t.Errorf("sent an unmappable batch to the native endpoint: %v", probe.calls())
		}
	}
	if result.Created != 1 {
		t.Errorf("Created = %d, want 1 (the valid item should still land)", result.Created)
	}
	if len(result.FailedItems) != 1 {
		t.Fatalf("FailedItems = %d, want 1", len(result.FailedItems))
	}
	if !strings.Contains(strings.Join(result.Errors, " "), "hologram") {
		t.Errorf("Errors = %v, want the offending type named", result.Errors)
	}
}

// TestNativeBulkRejectedBatch pins the status predicate directly, since the
// safety of the fallback rests entirely on it.
func TestNativeBulkRejectedBatch(t *testing.T) {
	for status, want := range map[int]bool{
		http.StatusBadRequest:          true,
		http.StatusTooManyRequests:     false,
		http.StatusInternalServerError: false,
		http.StatusForbidden:           false,
		http.StatusNotFound:            false,
	} {
		if got := nativeBulkRejectedBatch(&APIError{StatusCode: status}); got != want {
			t.Errorf("nativeBulkRejectedBatch(%d) = %v, want %v", status, got, want)
		}
	}
	if nativeBulkRejectedBatch(nil) {
		t.Error("nativeBulkRejectedBatch(nil) = true, want false")
	}
}
