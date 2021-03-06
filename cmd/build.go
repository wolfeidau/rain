package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/aws-cloudformation/rain/cfn/format"
	"github.com/aws-cloudformation/rain/cfn/spec"
	"github.com/aws-cloudformation/rain/cfn/spec/builder"
	"github.com/aws-cloudformation/rain/console/text/colourise"
	"github.com/spf13/cobra"
)

var buildListFlag = false
var bareTemplate = false
var buildJSON = false

var buildCmd = &cobra.Command{
	Use:                   "build [<resource type>...]",
	Short:                 "Create CloudFormation templates",
	Long:                  "Outputs a CloudFormation template containing the named resource types.",
	Annotations:           templateAnnotation,
	DisableFlagsInUseLine: true,
	Run: func(cmd *cobra.Command, args []string) {
		if buildListFlag {
			types := make([]string, 0)
			for t := range spec.Cfn.ResourceTypes {
				types = append(types, t)
			}
			sort.Strings(types)
			fmt.Println(strings.Join(types, "\n"))

			return
		}

		if len(args) == 0 {
			cmd.Help()
			return
		}

		config := make(map[string]string)
		for _, typeName := range args {
			resourceName := "My" + strings.Split(typeName, "::")[2]
			config[resourceName] = typeName
		}

		b := builder.NewCfnBuilder(!bareTemplate, true)
		t, c := b.Template(config)

		options := format.Options{
			Comments: c,
		}

		if buildJSON {
			options.Style = format.JSON
		}

		out := format.Template(t, options)

		if !buildJSON {
			out = colourise.Yaml(out)
		}

		fmt.Println(out)
	},
}

func init() {
	buildCmd.Flags().BoolVarP(&buildListFlag, "list", "l", false, "List all CloudFormation resource types")
	buildCmd.Flags().BoolVarP(&bareTemplate, "bare", "b", false, "Produce a minimal template, omitting all optional resource properties")
	buildCmd.Flags().BoolVarP(&buildJSON, "json", "j", false, "Output the templates as JSON (default format: YAML)")
	Rain.AddCommand(buildCmd)
}
