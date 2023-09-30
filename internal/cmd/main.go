package cmd

import (
	"context"
	"log"
	"os"
	"runtime/debug"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"go.alexhamlin.co/hfc/internal/config"
	"go.alexhamlin.co/hfc/internal/shelley"
	"go.alexhamlin.co/hfc/internal/state"
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:     "hfc",
	Short:   "Build and deploy serverless Go apps with AWS Lambda and CloudFormation",
	Version: getMainVersion(),
}

var (
	rootConfig config.Config
	rootState  state.State
	awsConfig  aws.Config
)

func initializePreRun(cmd *cobra.Command, args []string) {
	log.SetPrefix("[hfc] ")
	log.SetFlags(0)
	shelley.DefaultContext.DebugLogger = log.New(log.Writer(), "[hfc] $ ", 0)

	configPath, err := config.FindPath()
	if err != nil {
		log.Fatal(err)
	}
	rootConfig, err = config.Load()
	if err != nil {
		log.Fatal(err)
	}
	rootState, err = state.Get(configPath)
	if err != nil {
		log.Fatal(err)
	}

	awsConfig, err = awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(rootConfig.AWS.Region),
	)
	if err != nil {
		log.Fatal(err)
	}
}

func completeStackNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveDefault
	}

	rootConfig, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	names := lo.Map(rootConfig.Stacks, func(s config.StackConfig, _ int) string { return s.Name })
	names = lo.Filter(names, func(n string, _ int) bool { return strings.HasPrefix(n, toComplete) })
	return names, cobra.ShellCompDirectiveNoFileComp
}

func getMainVersion() string {
	const unknown = "v0.0.0-unknown"

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return unknown
	}

	// Use the module version if it's a real version. This is the case when
	// building using the module@version syntax, rather than from a local
	// checkout.
	if v := info.Main.Version; v != "(devel)" {
		return v
	}

	// For local checkouts, try to synthesize a pseudo-version from VCS metadata,
	// because that's just how much I like having this info at a glance. This
	// isn't bulletproof, but should work in most reasonable build scenarios.
	var vcstime, revision, modified string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.time":
			vcstime = strings.Map(digitsOnly, s.Value)
		case "vcs.revision":
			revision = s.Value
			if len(revision) > 12 {
				revision = revision[:12]
			}
		case "vcs.modified":
			if s.Value == "true" {
				modified = "+modified"
			}
		}
	}
	if vcstime != "" && revision != "" {
		return "v0.0.0-" + vcstime + "-" + revision + modified
	}

	return unknown
}

func digitsOnly(r rune) rune {
	if r >= '0' && r <= '9' {
		return r
	}
	return -1
}
