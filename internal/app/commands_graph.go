package app

import (
	"fmt"
	"os"

	"github.com/salman-frs/meridian/internal/graph"
	"github.com/salman-frs/meridian/internal/model"
	"github.com/spf13/cobra"
)

func newGraphCommand(global *GlobalOptions) *cobra.Command {
	var renderMode string
	var outPath string
	var view string
	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Build a pipeline graph for the collector config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(global)
			if err != nil {
				return err
			}
			renderFormat, err := parseGraphOutputRender(renderMode)
			if err != nil {
				return &model.ExitError{Code: 2, Err: err}
			}
			g := graph.Build(cfg)
			if view != "" {
				fmt.Fprintln(cmd.OutOrStdout(), graph.RenderTable(cfg))
			}
			if isJSONOutput(global) {
				return printJSON(g)
			}
			switch renderFormat {
			case graphOutputRenderNone:
				return nil
			case graphOutputRenderDOT:
				return writeGraphOutput(outPath, graph.RenderDOT(g), cmd)
			case graphOutputRenderSVG:
				svg, err := graph.RenderSVG(graph.RenderDOT(g))
				if err != nil {
					return &model.ExitError{Code: 3, Err: fmt.Errorf("graphviz dot is required for --render=svg: %w", err)}
				}
				target := outPath
				if target == "" {
					target = "graph.svg"
				}
				return os.WriteFile(target, svg, 0o644)
			default:
				return writeGraphOutput(outPath, graph.RenderMermaid(g), cmd)
			}
		},
	}
	cmd.Flags().StringVar(&renderMode, "render", "mermaid", "render mode: mermaid|dot|svg|none")
	cmd.Flags().StringVar(&outPath, "out", "", "write graph output to a file")
	cmd.Flags().StringVar(&view, "view", "table", "terminal view: ascii|table")
	return cmd
}

func writeGraphOutput(outPath string, rendered string, cmd *cobra.Command) error {
	if outPath != "" {
		return os.WriteFile(outPath, []byte(rendered), 0o644)
	}
	fmt.Fprintln(cmd.OutOrStdout(), rendered)
	return nil
}
