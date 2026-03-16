package app

import (
	"errors"

	"github.com/salman-frs/meridian/internal/model"
	"github.com/spf13/cobra"
)

// NewRootCommand builds the Meridian CLI.
func NewRootCommand() *cobra.Command {
	opts := &GlobalOptions{}
	runtimeOpts := newRuntimeOptions()

	root := &cobra.Command{
		Use:           "meridian",
		Short:         "Review and runtime-test OpenTelemetry Collector configs",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.PersistentFlags().StringArrayVarP(&opts.ConfigPaths, "config", "c", nil, "collector config source; repeatable and may be a file path or collector config URI")
	root.PersistentFlags().StringVar(&opts.ConfigDir, "config-dir", "", "path to a rendered collector config directory")
	root.PersistentFlags().StringVar(&opts.EnvFile, "env-file", "", "dotenv file used for config interpolation")
	root.PersistentFlags().StringArrayVar(&opts.EnvInline, "env", nil, "inline KEY=VALUE env vars")
	root.PersistentFlags().StringVar(&opts.Format, "format", "human", "output format: human|json")
	root.PersistentFlags().StringVar(&opts.Output, "output", defaultOutputDir, "artifact output directory")
	root.PersistentFlags().StringVar(&opts.CollectorBinary, "collector-binary", "", "path to a collector binary used for semantic validation")
	root.PersistentFlags().BoolVar(&opts.Quiet, "quiet", false, "suppress human progress output")
	root.PersistentFlags().BoolVar(&opts.Verbose, "verbose", false, "enable verbose output")
	root.PersistentFlags().BoolVar(&opts.NoColor, "no-color", false, "disable colorized output")
	_ = root.PersistentFlags().MarkHidden("output")
	_ = root.PersistentFlags().MarkHidden("no-color")
	_ = root.PersistentFlags().MarkDeprecated("no-color", "colorized output is not implemented yet")

	root.AddCommand(newValidateCommand(opts))
	root.AddCommand(newGraphCommand(opts))
	root.AddCommand(newDiffCommand(opts))
	root.AddCommand(newTestCommand(opts, runtimeOpts))
	root.AddCommand(newCheckCommand(opts, runtimeOpts))
	root.AddCommand(newCICommand(opts, runtimeOpts))
	root.AddCommand(newDebugCommand(opts))
	root.AddCommand(newVersionCommand())
	root.AddCommand(newCompletionCommand(root))
	return root
}

// ExitCode maps Meridian errors to process exit codes.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *model.ExitError
	if errors.As(err, &exitErr) && exitErr != nil {
		return exitErr.Code
	}
	return 1
}
