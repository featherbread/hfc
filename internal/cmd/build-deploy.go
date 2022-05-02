package cmd

import "github.com/spf13/cobra"

var buildDeployCmd = &cobra.Command{
	Use:   "build-deploy [flags] stack [parameters]",
	Short: "Build, upload, and deploy a binary all at once",
	Run:   runBuildDeploy,
	Args:  cobra.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(buildDeployCmd)
}

func runBuildDeploy(cmd *cobra.Command, args []string) {
	runBuild(cmd, args)
	runUpload(cmd, args)
	runDeploy(cmd, args)
}
