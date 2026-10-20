package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/olgasafonova/miro-mcp-server/miro/oauth"
)

// authCommand is the signature for an auth subcommand handler.
type authCommand func(ctx context.Context, flow *oauth.AuthFlow, cfg *oauth.Config) error

// authCommands dispatches the auth subcommand name to its handler.
var authCommands = map[string]authCommand{
	"login":  runAuthLogin,
	"status": runAuthStatus,
	"logout": runAuthLogout,
}

// runAuthCommand handles auth subcommands (login, status, logout).
func runAuthCommand(args []string) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	oauthConfig := oauth.LoadConfigFromEnv()

	if !oauthConfig.IsConfigured() {
		printOAuthSetupHelp()
		os.Exit(1)
	}

	authFlow := oauth.NewAuthFlow(oauthConfig, logger)
	ctx := context.Background()

	if len(args) == 0 {
		printAuthUsage()
		return
	}

	handler, ok := authCommands[args[0]]
	if !ok {
		fmt.Printf("Unknown auth command: %s\n\n", args[0])
		printAuthUsage()
		os.Exit(1)
	}
	if err := handler(ctx, authFlow, oauthConfig); err != nil {
		log.Fatal(err)
	}
}

// runAuthLogin starts an interactive OAuth login flow.
func runAuthLogin(ctx context.Context, flow *oauth.AuthFlow, cfg *oauth.Config) error {
	fmt.Println("Starting OAuth login flow...")
	tokens, err := flow.Login(ctx)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	fmt.Printf("\nLogged in successfully!\n")
	fmt.Printf("User ID: %s\n", tokens.UserID)
	fmt.Printf("Team ID: %s\n", tokens.TeamID)
	fmt.Printf("Token expires: %s\n", tokens.ExpiresAt.Format(time.RFC3339))
	fmt.Printf("\nTokens saved to: %s\n", cfg.TokenStorePath)
	return nil
}

// runAuthStatus prints the current authentication status.
func runAuthStatus(ctx context.Context, flow *oauth.AuthFlow, _ *oauth.Config) error {
	tokens, err := flow.Status(ctx)
	if err != nil {
		fmt.Printf("Not logged in: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Authentication status: Logged in")
	fmt.Printf("User ID: %s\n", tokens.UserID)
	fmt.Printf("Team ID: %s\n", tokens.TeamID)
	if tokens.IsExpired() {
		fmt.Println("Token status: Expired (will refresh on next use)")
	} else {
		fmt.Printf("Token expires: %s\n", tokens.ExpiresAt.Format(time.RFC3339))
	}
	return nil
}

// runAuthLogout revokes saved tokens and reports success.
func runAuthLogout(ctx context.Context, flow *oauth.AuthFlow, _ *oauth.Config) error {
	if err := flow.Logout(ctx); err != nil {
		return fmt.Errorf("logout failed: %w", err)
	}
	fmt.Println("Logged out successfully.")
	return nil
}

func printOAuthSetupHelp() {
	fmt.Println("OAuth not configured. Set MIRO_CLIENT_ID and MIRO_CLIENT_SECRET.")
	fmt.Println("\nTo get OAuth credentials:")
	fmt.Println("1. Go to https://miro.com/app/settings/user-profile/apps")
	fmt.Println("2. Create a new app or edit an existing one")
	fmt.Println("3. Copy Client ID and Client Secret")
}

func printAuthUsage() {
	fmt.Println("Usage: miro-mcp-server auth <command>")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  login   Start OAuth login flow (opens browser)")
	fmt.Println("  status  Show current authentication status")
	fmt.Println("  logout  Revoke tokens and log out")
	fmt.Println("")
	fmt.Println("Environment variables:")
	fmt.Println("  MIRO_CLIENT_ID      OAuth client ID (required)")
	fmt.Println("  MIRO_CLIENT_SECRET  OAuth client secret (required)")
	fmt.Println("  MIRO_REDIRECT_URI   Callback URL (default: http://localhost:8089/callback)")
	fmt.Println("  MIRO_TOKEN_PATH     Token storage path (default: ~/.miro/tokens.json)")
}
