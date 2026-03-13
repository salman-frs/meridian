package graph

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/salman-frs/meridian/internal/model"
)

func Build(cfg model.ConfigModel) model.GraphModel {
	nodes := []model.GraphNode{}
	edges := []model.GraphEdge{}
	seen := map[string]struct{}{}
	addNode := func(id, label, kind string, signal model.SignalType) {
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		nodes = append(nodes, model.GraphNode{ID: id, Label: label, Kind: kind, Signal: signal})
	}
	for _, name := range cfg.PipelineNames() {
		pipeline := cfg.Pipelines[name]
		pipelineID := "pipeline:" + name
		addNode(pipelineID, name, "pipeline", pipeline.Signal)
		prev := pipelineID
		for _, receiver := range pipeline.Receivers {
			id := "receiver:" + receiver
			kind := "receiver"
			if _, ok := cfg.Connectors[receiver]; ok {
				kind = "connector"
			}
			addNode(id, receiver, kind, pipeline.Signal)
			edges = append(edges, model.GraphEdge{From: id, To: prev, Signal: string(pipeline.Signal)})
		}
		for _, processor := range pipeline.Processors {
			id := "processor:" + processor
			addNode(id, processor, "processor", pipeline.Signal)
			edges = append(edges, model.GraphEdge{From: prev, To: id, Signal: string(pipeline.Signal)})
			prev = id
		}
		for _, exporter := range pipeline.Exporters {
			id := "exporter:" + exporter
			kind := "exporter"
			if _, ok := cfg.Connectors[exporter]; ok {
				kind = "connector"
			}
			addNode(id, exporter, kind, pipeline.Signal)
			edges = append(edges, model.GraphEdge{From: prev, To: id, Signal: string(pipeline.Signal)})
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return model.GraphModel{Nodes: nodes, Edges: edges}
}

func RenderMermaid(graph model.GraphModel) string {
	var b strings.Builder
	b.WriteString("flowchart LR\n")
	for _, node := range graph.Nodes {
		b.WriteString(fmt.Sprintf("  %s[%q]\n", sanitize(node.ID), node.Label))
	}
	for _, edge := range graph.Edges {
		b.WriteString(fmt.Sprintf("  %s --> %s\n", sanitize(edge.From), sanitize(edge.To)))
	}
	return b.String()
}

func RenderDOT(graph model.GraphModel) string {
	var b strings.Builder
	b.WriteString("digraph meridian {\n")
	b.WriteString("  rankdir=LR;\n")
	for _, node := range graph.Nodes {
		b.WriteString(fmt.Sprintf("  %s [label=%q];\n", sanitize(node.ID), node.Label))
	}
	for _, edge := range graph.Edges {
		b.WriteString(fmt.Sprintf("  %s -> %s;\n", sanitize(edge.From), sanitize(edge.To)))
	}
	b.WriteString("}\n")
	return b.String()
}

func RenderTable(cfg model.ConfigModel) string {
	lines := []string{"PIPELINE | RECEIVERS | PROCESSORS | EXPORTERS"}
	for _, name := range cfg.PipelineNames() {
		p := cfg.Pipelines[name]
		lines = append(lines, fmt.Sprintf("%s | %s | %s | %s", name, csv(p.Receivers), csv(p.Processors), csv(p.Exporters)))
	}
	return strings.Join(lines, "\n")
}

func RenderSVG(dot string) ([]byte, error) {
	if _, err := exec.LookPath("dot"); err != nil {
		return nil, err
	}
	cmd := exec.Command("dot", "-Tsvg")
	cmd.Stdin = strings.NewReader(dot)
	return cmd.Output()
}

func sanitize(id string) string {
	replacer := strings.NewReplacer(":", "_", "/", "_", "-", "_", ".", "_")
	return replacer.Replace(id)
}

func csv(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	return strings.Join(items, ",")
}
