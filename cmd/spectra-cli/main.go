package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	spectra "github.com/spectra-browser/spectra/client/go"
	"github.com/spf13/cobra"
)

var (
	serverURL string
	apiKey    string
	output    string
)

func main() {
	root := &cobra.Command{
		Use:   "spectra-cli",
		Short: "🔮 Spectra CLI — headless browser from your terminal",
	}

	root.PersistentFlags().StringVar(&serverURL, "server", "http://localhost:3000", "Spectra server URL")
	root.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key")
	root.PersistentFlags().StringVarP(&output, "output", "o", "", "Output file (default: stdout)")

	root.AddCommand(screenshotCmd(), pdfCmd(), scrapeCmd(), pluginsCmd(), healthCmd(), execCmd())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func newClient() *spectra.Client {
	opts := []spectra.Option{}
	if apiKey != "" {
		opts = append(opts, spectra.WithAPIKey(apiKey))
	}
	return spectra.NewClient(serverURL, opts...)
}

func outputResult(data json.RawMessage) {
	if output != "" {
		os.WriteFile(output, data, 0644)
		fmt.Fprintf(os.Stderr, "Written to %s\n", output)
		return
	}
	var pretty bytes.Buffer
	json.Indent(&pretty, data, "", "  ")
	fmt.Println(pretty.String())
}

func screenshotCmd() *cobra.Command {
	var width, height, quality int
	var fullPage bool

	cmd := &cobra.Command{
		Use:   "screenshot <url>",
		Short: "Take a screenshot",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()
			result, err := client.Screenshot(context.Background(), map[string]interface{}{
				"url": args[0], "width": width, "height": height,
				"full_page": fullPage, "quality": quality,
			})
			if err != nil {
				return err
			}
			outputResult(result)
			return nil
		},
	}
	cmd.Flags().IntVar(&width, "width", 1280, "Viewport width")
	cmd.Flags().IntVar(&height, "height", 720, "Viewport height")
	cmd.Flags().IntVar(&quality, "quality", 90, "Image quality")
	cmd.Flags().BoolVar(&fullPage, "full-page", false, "Full page screenshot")
	return cmd
}

func pdfCmd() *cobra.Command {
	var landscape bool

	cmd := &cobra.Command{
		Use:   "pdf <url>",
		Short: "Generate PDF",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()
			result, err := client.PDF(context.Background(), map[string]interface{}{
				"url": args[0], "landscape": landscape,
			})
			if err != nil {
				return err
			}
			outputResult(result)
			return nil
		},
	}
	cmd.Flags().BoolVar(&landscape, "landscape", false, "Landscape orientation")
	return cmd
}

func scrapeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scrape <url>",
		Short: "Scrape a web page",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()
			result, err := client.Scrape(context.Background(), map[string]interface{}{"url": args[0]})
			if err != nil {
				return err
			}
			outputResult(result)
			return nil
		},
	}
}

func pluginsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plugins",
		Short: "List available plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()
			result, err := client.Plugins(context.Background())
			if err != nil {
				return err
			}
			outputResult(result)
			return nil
		},
	}
}

func healthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check server health",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()
			if err := client.Health(context.Background()); err != nil {
				fmt.Println("❌ Server unhealthy:", err)
				return err
			}
			fmt.Println("✅ Server healthy")
			return nil
		},
	}
}

func execCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "exec <plugin> <method> [json-params]",
		Short: "Execute any plugin method",
		Args:  cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newClient()
			var params interface{}
			if len(args) == 3 {
				json.Unmarshal([]byte(args[2]), &params)
			}
			result, err := client.Execute(context.Background(), args[0], args[1], params)
			if err != nil {
				return err
			}
			outputResult(result)
			return nil
		},
	}
}
