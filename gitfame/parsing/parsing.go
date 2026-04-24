package parsing

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const gitHashLen = 40

type Options struct {
	Repo     string
	Rev      string
	Order    string
	UseCmtr  bool
	Extens   []string
	Langs    []string
	Exclude  []string
	Restrict []string
	LangData []byte
}

type langExtens struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	Extensions []string `json:"extensions"`
}

type AuthorStat struct {
	Name    string `json:"name"`
	Lines   int    `json:"lines"`
	Commits int    `json:"commits"`
	Files   int    `json:"files"`
}

type StatsData struct {
	Lines   int
	Commits map[string]bool
	Files   map[string]bool
}

func GetValidExt(opts Options) map[string]bool {
	validExt := make(map[string]bool)
	for _, ext := range opts.Extens {
		validExt[ext] = true
	}
	if len(opts.Langs) == 0 {
		return validExt
	}
	var mapping []langExtens
	if err := json.Unmarshal(opts.LangData, &mapping); err != nil {
		return validExt
	}
	langSet := make(map[string]bool, len(opts.Langs))
	for _, l := range opts.Langs {
		langSet[strings.ToLower(l)] = true
	}

	for _, l := range mapping {
		if !langSet[strings.ToLower(l.Name)] {
			continue
		}
		for _, ext := range l.Extensions {
			validExt[ext] = true
		}
	}

	return validExt
}

func matchAny(file string, patterns []string) bool {
	for _, pat := range patterns {
		if match, _ := filepath.Match(pat, file); match {
			return true
		}
	}
	return false
}

func GetTargetFiles(opts Options, validExt map[string]bool) []string {
	cmdLs := exec.Command("git", "ls-tree", "-r", "--name-only", opts.Rev)
	cmdLs.Dir = opts.Repo
	lsOut, err := cmdLs.Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "git ls-tree failed %v\n", err)
		os.Exit(1)
	}

	files := strings.Split(strings.TrimSpace(string(lsOut)), "\n")
	var targets []string
	for _, file := range files {
		if file == "" {
			continue
		}
		if len(opts.Extens) > 0 || len(opts.Langs) > 0 {
			ext := filepath.Ext(file)
			if !validExt[ext] {
				continue
			}
		}
		if matchAny(file, opts.Exclude) {
			continue
		}
		if len(opts.Restrict) > 0 && !matchAny(file, opts.Restrict) {
			continue
		}
		targets = append(targets, file)
	}
	return targets
}

func CollectStats(targets []string, opts Options) map[string]*StatsData {
	stats := make(map[string]*StatsData)
	var wg sync.WaitGroup
	var mu sync.Mutex
	sem := make(chan struct{}, 30)

	for _, file := range targets {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			processFile(file, opts, stats, &mu)
		}()
	}

	wg.Wait()
	return stats
}

func processFile(file string, opts Options, stats map[string]*StatsData, mu *sync.Mutex) {
	cmdBlame := exec.Command("git", "blame", "--porcelain", opts.Rev, "--", file)
	cmdBlame.Dir = opts.Repo
	blameOut, err := cmdBlame.Output()
	if err != nil {
		return
	}

	lines := strings.Split(string(blameOut), "\n")
	var curCommit string
	authors := make(map[string]string)
	hasLines := false

	for _, line := range lines {
		curCommit, hasLines = parseLine(line, file, curCommit, authors, opts, stats, mu, hasLines)
	}

	if !hasLines {
		processEmptyFile(file, opts, stats, mu)
	}
}

func parseLine(line, file, curCommit string, authors map[string]string, opts Options, stats map[string]*StatsData, mu *sync.Mutex, hasLines bool) (string, bool) {
	if len(line) >= gitHashLen && !strings.HasPrefix(line, "\t") && !strings.Contains(line[:gitHashLen], " ") {
		parts := strings.SplitN(line, " ", 4)
		return parts[0], hasLines
	}
	if strings.HasPrefix(line, "author ") && !opts.UseCmtr {
		authors[curCommit] = strings.TrimPrefix(line, "author ")
	} else if strings.HasPrefix(line, "committer ") && opts.UseCmtr {
		authors[curCommit] = strings.TrimPrefix(line, "committer ")
	} else if strings.HasPrefix(line, "\t") {
		hasLines = true
		author := authors[curCommit]
		addStat(author, curCommit, file, stats, mu, true)
	}
	return curCommit, hasLines
}

func processEmptyFile(file string, opts Options, stats map[string]*StatsData, mu *sync.Mutex) {
	logFmt := "%H%x00%an"
	if opts.UseCmtr {
		logFmt = "%H%x00%cn"
	}
	cmdLog := exec.Command("git", "log", "-1", "--format="+logFmt, opts.Rev, "--", file)
	cmdLog.Dir = opts.Repo
	logOut, err := cmdLog.Output()
	if err == nil && len(logOut) > 0 {
		parts := strings.SplitN(strings.TrimRight(string(logOut), "\r\n"), "\x00", 2)
		if len(parts) == 2 {
			addStat(parts[1], parts[0], file, stats, mu, false)
		}
	}
}

func addStat(author, commit, file string, stats map[string]*StatsData, mu *sync.Mutex, isLine bool) {
	mu.Lock()
	defer mu.Unlock()
	if stats[author] == nil {
		stats[author] = &StatsData{
			Commits: make(map[string]bool),
			Files:   make(map[string]bool),
		}
	}
	if isLine {
		stats[author].Lines++
	}
	stats[author].Commits[commit] = true
	stats[author].Files[file] = true
}

func cmpStats(s1, s2 AuthorStat, order string) bool {
	switch order {
	case "lines":
		if s1.Lines != s2.Lines {
			return s1.Lines > s2.Lines
		}
		if s1.Commits != s2.Commits {
			return s1.Commits > s2.Commits
		}
		if s1.Files != s2.Files {
			return s1.Files > s2.Files
		}
	case "commits":
		if s1.Commits != s2.Commits {
			return s1.Commits > s2.Commits
		}
		if s1.Lines != s2.Lines {
			return s1.Lines > s2.Lines
		}
		if s1.Files != s2.Files {
			return s1.Files > s2.Files
		}
	case "files":
		if s1.Files != s2.Files {
			return s1.Files > s2.Files
		}
		if s1.Lines != s2.Lines {
			return s1.Lines > s2.Lines
		}
		if s1.Commits != s2.Commits {
			return s1.Commits > s2.Commits
		}
	}
	return s1.Name < s2.Name
}

func BuildResults(stats map[string]*StatsData, opts Options) []AuthorStat {
	results := make([]AuthorStat, 0, len(stats))
	for name, data := range stats {
		results = append(results, AuthorStat{
			Name:    name,
			Lines:   data.Lines,
			Commits: len(data.Commits),
			Files:   len(data.Files),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return cmpStats(results[i], results[j], opts.Order)
	})
	return results
}
