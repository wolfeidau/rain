package cmd

import (
	"fmt"
	"strings"

	"github.com/aws-cloudformation/rain/cfn"
	"github.com/aws-cloudformation/rain/cfn/graph"
	"github.com/aws-cloudformation/rain/cfn/parse"
	"github.com/aws-cloudformation/rain/console/text"
	"github.com/spf13/cobra"
)

var allLinks = false
var dotGraph = false
var twoWayTree = false

func printLinks(links []interface{}, typeFilter string) {
	names := make([]string, 0)
	for _, link := range links {
		to := link.(cfn.Element)
		if to.Type == typeFilter {
			names = append(names, to.Name)
		}
	}

	if len(names) == 0 {
		return
	}

	fmt.Printf("      %s:\n", typeFilter)
	for _, name := range names {
		fmt.Printf("        - %s\n", text.Orange(name))
	}
}

func printGraph(graph graph.Graph, typeFilter string) {
	elements := make([]cfn.Element, 0)
	fromLinks := make(map[cfn.Element][]interface{})
	toLinks := make(map[cfn.Element][]interface{})

	for _, item := range graph.Nodes() {
		el := item.(cfn.Element)
		if el.Type == typeFilter {
			elements = append(elements, el)
			froms := graph.Get(item)

			if allLinks || len(froms) > 0 {
				fromLinks[el] = froms
			}

			if twoWayTree {
				tos := graph.GetReverse(item)

				if allLinks || len(tos) > 0 {
					toLinks[el] = tos
				}
			}
		}
	}

	if len(fromLinks) == 0 && len(toLinks) == 0 {
		return
	}

	fmt.Printf("%s:\n", typeFilter)

	for _, el := range elements {
		if !allLinks && len(fromLinks[el]) == 0 && len(toLinks[el]) == 0 {
			continue
		}

		fmt.Printf("  %s:\n", text.Yellow(el.Name))

		if allLinks || len(fromLinks[el]) > 0 {
			if len(fromLinks[el]) == 0 {
				fmt.Println("    DependsOn: []")
			} else {
				fmt.Println("    DependsOn:")
				printLinks(fromLinks[el], "Parameters")
				printLinks(fromLinks[el], "Resources")
				printLinks(fromLinks[el], "Outputs")
			}
		}

		if twoWayTree && (allLinks || len(toLinks[el]) > 0) {
			if len(toLinks[el]) == 0 {
				fmt.Println("    UsedBy: []")
			} else {
				fmt.Println("    UsedBy:")
				printLinks(toLinks[el], "Parameters")
				printLinks(toLinks[el], "Resources")
				printLinks(toLinks[el], "Outputs")
			}
		}
	}

	fmt.Println()
}

var dotShapes = map[string]string{
	"Parameters": "diamond",
	"Resources":  "Mrecord",
	"Outputs":    "rectangle",
}

func printDot(graph graph.Graph) {
	out := strings.Builder{}

	out.WriteString("digraph {\n")
	out.WriteString("    rankdir=LR;\n")
	out.WriteString("    concentrate=true;\n")

	// First pass, group types
	doGroup := func(t string) {
		out.WriteString(fmt.Sprintf("    subgraph cluster_%s {\n", t))
		out.WriteString(fmt.Sprintf("        label=\"%s\";\n", t))
		for _, from := range graph.Nodes() {
			el := from.(cfn.Element)
			if el.Type == t {
				nodeName := fmt.Sprintf("%s: %s", el.Type, el.Name)

				out.WriteString(fmt.Sprintf("        \"%s\" [label=\"%s\" shape=%s];\n", nodeName, el.Name, dotShapes[el.Type]))
			}
		}
		out.WriteString("    }\n")
		out.WriteString("\n")
	}

	doGroup("Parameters")
	doGroup("Resources")
	doGroup("Outputs")

	for _, from := range graph.Nodes() {
		fromEl := from.(cfn.Element)
		fromStr := fmt.Sprintf("%s: %s", fromEl.Type, fromEl.Name)

		for _, to := range graph.Get(from) {
			toEl := to.(cfn.Element)
			toStr := fmt.Sprintf("%s: %s", toEl.Type, toEl.Name)

			out.WriteString(fmt.Sprintf("    \"%s\" -> \"%s\";\n", toStr, fromStr))
		}
	}

	out.WriteString("}")

	fmt.Println(out.String())
}

var graphCmd = &cobra.Command{
	Use:                   "tree [template]",
	Short:                 "Find dependencies of Resources and Outputs in a local template",
	Long:                  "Find and display the dependencies between Parameters, Resources, and Outputs in a CloudFormation template.",
	Args:                  cobra.ExactArgs(1),
	Aliases:               []string{"graph"},
	Annotations:           templateAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		fileName := args[0]

		t, err := parse.File(fileName)
		if err != nil {
			panic(fmt.Errorf("Unable to parse template '%s': %s", fileName, err))
		}

		graph := t.Graph()

		if dotGraph {
			printDot(graph)
		} else {
			printGraph(graph, "Parameters")
			printGraph(graph, "Resources")
			printGraph(graph, "Outputs")
		}
	},
}

func init() {
	graphCmd.Flags().BoolVarP(&allLinks, "all", "a", false, "Display all elements, even those without any dependencies")
	graphCmd.Flags().BoolVarP(&twoWayTree, "both", "b", false, "For each element, display both its dependencies and its dependents")
	graphCmd.Flags().BoolVarP(&dotGraph, "dot", "d", false, "Output the graph in GraphViz DOT format")
	Rain.AddCommand(graphCmd)
}
