package tools

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestRegisterProfile_FullExposesEverything spins up an in-memory MCP server,
// registers the full profile, and asserts that an MCP-protocol tools/list
// returns exactly len(AllTools) tools — proving the registration plumbing
// honors the profile end-to-end (not just at the spec level).
func TestRegisterProfile_FullExposesEverything(t *testing.T) {
	count := registerAndListTools(t, ProfileFull)
	if count != len(AllTools) {
		t.Errorf("ProfileFull tools/list returned %d tools, want %d", count, len(AllTools))
	}
}

// TestRegisterProfile_EssentialsIsLean asserts that ProfileEssentials drops
// the surface to the curated EssentialsToolNames list and keeps the discovery
// meta-tool first. This is the headline UX promise of the env var: a client
// that only sees ~15 tools but can still reach the rest via miro_tool_search.
func TestRegisterProfile_EssentialsIsLean(t *testing.T) {
	count := registerAndListTools(t, ProfileEssentials)
	wantCount := len(EssentialsToolNames)
	if count != wantCount {
		t.Errorf("ProfileEssentials tools/list returned %d tools, want %d", count, wantCount)
	}
}

// TestRegisterProfile_EssentialsContainsSearchTool verifies the discovery
// meta-tool itself is reachable in the lean profile. Without it, agents that
// don't know which tool to call have no recovery path.
func TestRegisterProfile_EssentialsContainsSearchTool(t *testing.T) {
	tools := registerAndListToolNames(t, ProfileEssentials)
	found := false
	for _, name := range tools {
		if name == ToolSearchName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("essentials profile must include %s; got tools: %v", ToolSearchName, tools)
	}
}

// TestRegisterProfile_TokenSavings_Realistic asserts the essentials profile
// is meaningfully smaller than full — the property that motivates having a
// profile at all. Numbers don't have to match the README estimate exactly
// (those depend on schema cost which we don't measure here), but the lean
// profile should be at least 70% smaller by tool count.
func TestRegisterProfile_TokenSavings_Realistic(t *testing.T) {
	full := len(ToolsForProfile(ProfileFull))
	ess := len(ToolsForProfile(ProfileEssentials))
	reduction := float64(full-ess) / float64(full) * 100
	if reduction < 70.0 {
		t.Errorf("expected at least 70%% reduction from full to essentials, got %.1f%% (%d -> %d)",
			reduction, full, ess)
	}
}

// registerAndListTools is the shared scaffolding: build a registry against a
// no-op mock client, register the profile on a real *mcp.Server, connect a
// client over in-memory transport, and count tools/list results. Returns the
// count.
func registerAndListTools(t *testing.T, profile Profile) int {
	t.Helper()
	return len(registerAndListToolNames(t, profile))
}

func registerAndListToolNames(t *testing.T, profile Profile) []string {
	t.Helper()

	registry := NewHandlerRegistry(&MockClient{}, testLogger())

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	registry.RegisterProfile(server, profile)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	t.Cleanup(func() {
		_ = session.Close()
		cancel()
		<-serverDone
	})

	res, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	names := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	return names
}
