package discovery

import (
	"os"
	"regexp"

	"github.com/prometheus/prometheus/model/rulefmt"
	"gopkg.in/yaml.v3"

	"github.com/cloudflare/pint/internal/parser"

	"github.com/rs/zerolog/log"
)

const (
	FileOwnerComment = "file/owner"
	RuleOwnerComment = "rule/owner"
)

type RuleFinder interface {
	Find() ([]Entry, error)
}

type Entry struct {
	Path          string
	PathError     error
	ModifiedLines []int
	Rule          parser.Rule
	Owner         string
}

func readFile(path string, isStrict bool) (entries []Entry, err error) {
	p := parser.NewParser()

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	content, err := parser.ReadContent(f)
	f.Close()
	if err != nil {
		return nil, err
	}

	fileOwner, _ := parser.GetComment(string(content), FileOwnerComment)

	if isStrict {
		var r rulefmt.RuleGroups
		if err = yaml.Unmarshal(content, &r); err != nil {
			log.Error().Str("path", path).Err(err).Msg("Failed to parse file content")
			entries = append(entries, Entry{
				Path:      path,
				PathError: err,
				Owner:     fileOwner,
			})
			return entries, nil
		}
	}

	rules, err := p.Parse(content)
	if err != nil {
		log.Error().Str("path", path).Err(err).Msg("Failed to parse file content")
		entries = append(entries, Entry{
			Path:      path,
			PathError: err,
		})
		return entries, nil
	}

	for _, rule := range rules {
		owner, ok := rule.GetComment(RuleOwnerComment)
		if !ok {
			owner = fileOwner
		}
		entries = append(entries, Entry{
			Path:  path,
			Rule:  rule,
			Owner: owner,
		})
	}

	log.Info().Str("path", path).Int("rules", len(entries)).Msg("File parsed")
	return entries, nil
}

func matchesAny(re []*regexp.Regexp, s string) bool {
	for _, r := range re {
		if v := r.MatchString(s); v {
			return true
		}
	}
	return false
}
