//go:build !solution

package main

import (
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"gitlab.com/slon/shad-go/gitfame/parsing"

	"github.com/spf13/pflag"
)

//go:embed language_extensions.json
var langExtensData []byte

var (
	flagRepo     = pflag.String("repository", ".", "path to git repository")
	flagRev      = pflag.String("revision", "HEAD", "pointer to commit")
	flagOrder    = pflag.String("order-by", "lines", "sort key")
	flagUseCmtr  = pflag.Bool("use-committer", false, "replace author with committer")
	flagFormat   = pflag.String("format", "tabular", "output format")
	flagExtens   = pflag.StringSlice("extensions", nil, "list of extensions")
	flagLangs    = pflag.StringSlice("languages", nil, "list of languages")
	flagExclude  = pflag.StringSlice("exclude", nil, "glob patterns to exclude")
	flagRestrict = pflag.StringSlice("restrict-to", nil, "glob patterns to restrict to")
)

func main() {
	pflag.Parse()

	if *flagFormat != "tabular" && *flagFormat != "csv" && *flagFormat != "json" && *flagFormat != "json-lines" {
		fmt.Fprintf(os.Stderr, "invalid format %s\n", *flagFormat)
		os.Exit(1)
	}
	if *flagOrder != "lines" && *flagOrder != "commits" && *flagOrder != "files" {
		fmt.Fprintf(os.Stderr, "invalid orderby %s\n", *flagOrder)
		os.Exit(1)
	}

	opts := parsing.Options{
		Repo:     *flagRepo,
		Rev:      *flagRev,
		Order:    *flagOrder,
		UseCmtr:  *flagUseCmtr,
		Extens:   *flagExtens,
		Langs:    *flagLangs,
		Exclude:  *flagExclude,
		Restrict: *flagRestrict,
		LangData: langExtensData,
	}

	validExt := parsing.GetValidExt(opts)
	targets := parsing.GetTargetFiles(opts, validExt)
	stats := parsing.CollectStats(targets, opts)
	results := parsing.BuildResults(stats, opts)
	printResults(results)
}

func printResults(results []parsing.AuthorStat) {
	switch *flagFormat {
	case "csv":
		w := csv.NewWriter(os.Stdout)
		_ = w.Write([]string{"Name", "Lines", "Commits", "Files"})
		for _, r := range results {
			_ = w.Write([]string{r.Name, strconv.Itoa(r.Lines), strconv.Itoa(r.Commits), strconv.Itoa(r.Files)})
		}
		w.Flush()
	case "tabular":
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
		fmt.Fprintln(w, "Name\tLines\tCommits\tFiles")
		for _, r := range results {
			fmt.Fprintf(w, "%s\t%d\t%d\t%d\n", r.Name, r.Lines, r.Commits, r.Files)
		}
		w.Flush()
	case "json":
		data, _ := json.Marshal(results)
		fmt.Println(string(data))
	case "json-lines":
		for _, r := range results {
			data, _ := json.Marshal(r)
			fmt.Println(string(data))
		}
	}
}
