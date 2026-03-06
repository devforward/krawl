package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/devforward/krawl/internal/fetcher"
	"github.com/devforward/krawl/internal/parser"
)

var linksCmd = &cobra.Command{
	Use:   "links [url]",
	Short: "Check all links on a page for broken URLs",
	Long:  `Fetches a page, extracts all internal and external links, and checks each one for availability.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runLinks,
}

func init() {
	rootCmd.AddCommand(linksCmd)
	linksCmd.Flags().IntP("concurrency", "c", 10, "Number of concurrent link checks")
	linksCmd.Flags().BoolP("json", "j", false, "Output results as JSON")
}

type linkResult struct {
	URL        string `json:"url"`
	Text       string `json:"text,omitempty"`
	StatusCode int    `json:"status_code"`
	Error      string `json:"error,omitempty"`
}

type linksOutput struct {
	Page     string       `json:"page"`
	Internal []linkResult `json:"internal"`
	External []linkResult `json:"external"`
	Summary  linksSummary `json:"summary"`
}

type linksSummary struct {
	Internal    int `json:"internal"`
	External    int `json:"external"`
	Total       int `json:"total"`
	OK          int `json:"ok"`
	Broken      int `json:"broken"`
	Redirected  int `json:"redirected"`
}

func runLinks(cmd *cobra.Command, args []string) error {
	pageURL := args[0]

	timeout := viper.GetDuration("timeout")
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	userAgent := viper.GetString("user-agent")
	if userAgent == "" {
		userAgent = "krawl/1.0"
	}
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	result, err := fetcher.Fetch(pageURL, timeout, userAgent)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", pageURL, err)
	}

	links, err := parser.ParseLinks(result.Body, result.FinalURL)
	if err != nil {
		return fmt.Errorf("failed to parse links: %w", err)
	}

	var internal, external []parser.PageLink
	for _, link := range links {
		if parser.IsInternal(link.Href, result.FinalURL) {
			internal = append(internal, link)
		} else {
			external = append(external, link)
		}
	}

	if !jsonOutput {
		bold := color.New(color.Bold)
		fmt.Println()
		bold.Println("╔══════════════════════════════════════════════════════════════════╗")
		bold.Printf("║  Link Check: %-52s║\n", truncateStr(pageURL, 52))
		bold.Println("╚══════════════════════════════════════════════════════════════════╝")
		fmt.Printf("\n  Found %d links (%d internal, %d external). Checking...\n",
			len(links), len(internal), len(external))
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	checkLink := func(link parser.PageLink) linkResult {
		lr := linkResult{URL: link.Href, Text: link.Text}
		req, err := http.NewRequest("HEAD", link.Href, nil)
		if err != nil {
			lr.Error = err.Error()
			return lr
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := client.Do(req)
		if err != nil {
			// Retry with GET — some servers reject HEAD
			req.Method = "GET"
			resp, err = client.Do(req)
			if err != nil {
				lr.Error = err.Error()
				return lr
			}
		}
		resp.Body.Close()
		lr.StatusCode = resp.StatusCode
		return lr
	}

	allLinks := make([]parser.PageLink, 0, len(internal)+len(external))
	allLinks = append(allLinks, internal...)
	allLinks = append(allLinks, external...)

	results := make([]linkResult, len(allLinks))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, link := range allLinks {
		wg.Add(1)
		go func(i int, link parser.PageLink) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = checkLink(link)
		}(i, link)
	}
	wg.Wait()

	internalResults := results[:len(internal)]
	externalResults := results[len(internal):]

	output := linksOutput{
		Page:     pageURL,
		Internal: internalResults,
		External: externalResults,
	}

	for _, r := range results {
		output.Summary.Total++
		if r.StatusCode >= 200 && r.StatusCode < 300 {
			output.Summary.OK++
		} else if r.StatusCode >= 300 && r.StatusCode < 400 {
			output.Summary.Redirected++
		} else {
			output.Summary.Broken++
		}
	}
	output.Summary.Internal = len(internalResults)
	output.Summary.External = len(externalResults)

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	printLinkGroup("Internal Links", internalResults)
	printLinkGroup("External Links", externalResults)

	// Summary
	bold := color.New(color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)

	fmt.Println()
	bold.Println("────────────────────────────────────────────────────────────────────")
	fmt.Printf("  Summary: %s ok  %s redirected  %s broken  (%d total)\n",
		green.Sprintf("%d", output.Summary.OK),
		yellow.Sprintf("%d", output.Summary.Redirected),
		red.Sprintf("%d", output.Summary.Broken),
		output.Summary.Total,
	)
	bold.Println("────────────────────────────────────────────────────────────────────")
	fmt.Println()

	return nil
}

func printLinkGroup(title string, results []linkResult) {
	bold := color.New(color.Bold, color.FgWhite)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)
	dim := color.New(color.Faint)
	cyan := color.New(color.FgCyan)

	fmt.Println()
	bold.Printf("┌─ %s (%d) ", title, len(results))
	bold.Println(repeatStr("─", max(0, 56-len(title))))

	if len(results) == 0 {
		cyan.Print("│ ")
		dim.Println("  (none)")
		return
	}

	for _, r := range results {
		var icon string
		var status string
		if r.Error != "" {
			icon = red.Sprint("  ✗ ")
			status = red.Sprintf("ERR  ")
		} else if r.StatusCode >= 200 && r.StatusCode < 300 {
			icon = green.Sprint("  ✓ ")
			status = green.Sprintf("%-5d", r.StatusCode)
		} else if r.StatusCode >= 300 && r.StatusCode < 400 {
			icon = yellow.Sprint("  → ")
			status = yellow.Sprintf("%-5d", r.StatusCode)
		} else {
			icon = red.Sprint("  ✗ ")
			status = red.Sprintf("%-5d", r.StatusCode)
		}

		url := r.URL
		if len(url) > 60 {
			url = url[:57] + "..."
		}

		text := r.Text
		if text == "" {
			text = "-"
		} else if len(text) > 30 {
			text = text[:27] + "..."
		}

		fmt.Printf("%s%s %-60s %s\n", icon, status, url, dim.Sprint(text))

		if r.Error != "" {
			errMsg := r.Error
			if len(errMsg) > 80 {
				errMsg = errMsg[:77] + "..."
			}
			red.Printf("         %s\n", errMsg)
		}
	}
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func repeatStr(s string, n int) string {
	if n <= 0 {
		return ""
	}
	result := ""
	for range n {
		result += s
	}
	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
