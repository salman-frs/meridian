package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/salman-frs/meridian/internal/app"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	root := app.NewRootCommand()
	disableAutoGen(root)

	outputDir := filepath.Join("docs", "reference", "cli")
	if err := os.RemoveAll(outputDir); err != nil {
		fail(err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fail(err)
	}
	if err := writeIndex(root, outputDir); err != nil {
		fail(err)
	}

	filePrepender := func(filename string) string {
		title := strings.ReplaceAll(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)), "_", " ")
		return fmt.Sprintf("---\ntitle: %s\n---\n\n", title)
	}

	if err := doc.GenMarkdownTreeCustom(root, outputDir, filePrepender, func(name string) string { return name }); err != nil {
		fail(err)
	}
}

func disableAutoGen(cmd *cobra.Command) {
	cmd.DisableAutoGenTag = true
	for _, child := range cmd.Commands() {
		disableAutoGen(child)
	}
}

func writeIndex(root *cobra.Command, outputDir string) error {
	var lines []string
	lines = append(lines,
		"# CLI Reference",
		"",
		"These pages are generated from the Cobra command tree with `go run ./cmd/meridian-docs`.",
		"Do not hand-edit the generated command pages. Update command help text in Go code, then regenerate.",
		"",
		"## Commands",
		"",
	)
	visit := func(cmd *cobra.Command) {}
	visit = func(cmd *cobra.Command) {
		if !cmd.IsAvailableCommand() || cmd.IsAdditionalHelpTopicCommand() {
			return
		}
		name := strings.ReplaceAll(commandPath(cmd), " ", "_") + ".md"
		lines = append(lines, fmt.Sprintf("- [`%s`](%s): %s", commandPath(cmd), name, cmd.Short))
		children := visibleChildren(cmd)
		for _, child := range children {
			visit(child)
		}
	}
	visit(root)
	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(filepath.Join(outputDir, "index.md"), []byte(content), 0o644)
}

func visibleChildren(cmd *cobra.Command) []*cobra.Command {
	children := make([]*cobra.Command, 0, len(cmd.Commands()))
	for _, child := range cmd.Commands() {
		if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() {
			continue
		}
		children = append(children, child)
	}
	sort.Slice(children, func(i, j int) bool {
		return commandPath(children[i]) < commandPath(children[j])
	})
	return children
}

func commandPath(cmd *cobra.Command) string {
	return strings.TrimSpace(cmd.CommandPath())
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
