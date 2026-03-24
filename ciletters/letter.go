//go:build !solution

package ciletters

import (
	_ "embed"
	"slices"
	"strings"
	"text/template"
)

func cutHash(hash string) string {
	return hash[:8]
}

func lastLogs(logs string) []string {
	// returns last logs, up to 10
	// the 2nd int is cnt logs
	cnt := 10
	spliter := "\n"

	res := make([]string, 0)

	for range cnt {
		ind := strings.LastIndex(logs, spliter)
		if ind == -1 {
			res = append(res, logs)
			slices.Reverse(res)
			return res
		}

		res = append(res, logs[ind+1:])
		logs = logs[:ind]
	}

	slices.Reverse(res)
	return res
}

//go:embed ci.tmpl
var templateContent string

func MakeLetter(n *Notification) (string, error) {
	temp := template.New("ci_parser").Funcs(
		template.FuncMap{
			"cutHash":  cutHash,
			"lastLogs": lastLogs,
		})

	temp, err := temp.Parse(templateContent)
	if err != nil {
		return "", err
	}

	sb := strings.Builder{}
	err = temp.Execute(&sb, n)
	if err != nil {
		return "", err
	}
	return sb.String(), err
}
