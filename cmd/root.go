package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/kmazanec/seocheck/internal/display"
	"github.com/kmazanec/seocheck/internal/fetcher"
	"github.com/kmazanec/seocheck/internal/parser"
	"github.com/kmazanec/seocheck/internal/rules"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "seocheck [url]",
	Short: "Fetch a page and evaluate its SEO metadata",
	Long:  `seocheck fetches a URL, displays HTTP response details, parses SEO-relevant meta tags, and evaluates them against standard SEO best practices.`,
	Args:  cobra.ExactArgs(1),
	RunE:  run,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.seocheck.yaml)")
	rootCmd.Flags().DurationP("timeout", "t", 30*time.Second, "HTTP request timeout")
	rootCmd.Flags().StringP("user-agent", "u", "seocheck/1.0", "User-Agent header for the request")
	rootCmd.Flags().Bool("no-audit", false, "Skip the SEO audit rules (only show metadata)")
	rootCmd.Flags().Bool("no-meta", false, "Skip the metadata display (only show audit)")
	rootCmd.Flags().Bool("json", false, "Output results as JSON")

	viper.BindPFlag("timeout", rootCmd.Flags().Lookup("timeout"))
	viper.BindPFlag("user-agent", rootCmd.Flags().Lookup("user-agent"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			viper.AddConfigPath(home)
		}
		viper.AddConfigPath(".")
		viper.SetConfigName(".seocheck")
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("SEOCHECK")
	viper.AutomaticEnv()
	viper.ReadInConfig()
}

func run(cmd *cobra.Command, args []string) error {
	url := args[0]

	timeout := viper.GetDuration("timeout")
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	userAgent := viper.GetString("user-agent")
	if userAgent == "" {
		userAgent = "seocheck/1.0"
	}

	result, err := fetcher.Fetch(url, timeout, userAgent)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", url, err)
	}

	display.PrintHTTPInfo(result)

	seoData, err := parser.Parse(result.Body)
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %w", err)
	}

	noMeta, _ := cmd.Flags().GetBool("no-meta")
	noAudit, _ := cmd.Flags().GetBool("no-audit")

	if !noMeta {
		display.PrintSEOData(seoData)
	}

	if !noAudit {
		auditResults := rules.Evaluate(seoData)
		display.PrintRules(auditResults)
	}

	return nil
}
