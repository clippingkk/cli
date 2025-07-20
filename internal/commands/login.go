package commands

import (
	"fmt"
	"os"

	"github.com/clippingkk/cli/internal/config"
	"github.com/urfave/cli/v2"
)

// LoginCommand handles user authentication
var LoginCommand = &cli.Command{
	Name:    "login",
	Usage:   "Authenticate with ClippingKK service",
	Description: `Login to ClippingKK service using your API token.

Visit https://clippingkk.annatarhe.com, login to your account, 
navigate to your profile page and open the 'API Token' dialog.
Copy the token and use it with this command.

Example:
  ck-cli login --token YOUR_API_TOKEN
  ck-cli --token YOUR_API_TOKEN login`,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "token",
			Aliases:  []string{"t"},
			Usage:    "API token from ClippingKK profile page",
			Required: false,
		},
	},
	Action: loginAction,
}

func loginAction(c *cli.Context) error {
	// Get token from flag or global flag
	token := c.String("token")
	if token == "" {
		token = c.String("token") // Try global flag
	}
	
	if token == "" {
		fmt.Fprintf(os.Stderr, "❌ Token not found\n\n")
		fmt.Fprintf(os.Stderr, "Visit https://clippingkk.annatarhe.com and login\n")
		fmt.Fprintf(os.Stderr, "Then navigate to your profile page and open 'API Token' dialog.\n")
		fmt.Fprintf(os.Stderr, "Copy the token and run:\n")
		fmt.Fprintf(os.Stderr, "  ck-cli login --token YOUR_TOKEN\n\n")
		os.Exit(1)
	}

	// Load or create config
	configPath, err := config.GetConfigPath(c.String("config"))
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Update token
	cfg.UpdateToken(token)

	// Save config
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("✅ Successfully logged in!\n\n")
	fmt.Printf("You can now synchronize your Kindle clippings by running:\n")
	fmt.Printf("  ck-cli parse --input /path/to/My\\ Clippings.txt --output http\n\n")

	return nil
}