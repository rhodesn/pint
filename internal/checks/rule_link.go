package checks

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"golang.org/x/exp/slices"

	"github.com/cloudflare/pint/internal/discovery"
	"github.com/cloudflare/pint/internal/parser"
)

const (
	RuleLinkCheckName = "rule/link"
)

func NewRuleLinkCheck(re *TemplatedRegexp, uriRewrite string, timeout time.Duration, headers map[string]string, s Severity) RuleLinkCheck {
	return RuleLinkCheck{
		scheme:     []string{"http", "https"},
		re:         re,
		uriRewrite: uriRewrite,
		timeout:    timeout,
		headers:    headers,
		severity:   s,
	}
}

type RuleLinkCheck struct {
	scheme     []string
	re         *TemplatedRegexp
	uriRewrite string
	timeout    time.Duration
	headers    map[string]string
	severity   Severity
}

func (c RuleLinkCheck) Meta() CheckMeta {
	return CheckMeta{IsOnline: true}
}

func (c RuleLinkCheck) String() string {
	return fmt.Sprintf("%s(%s)", RuleLinkCheckName, c.re.anchored)
}

func (c RuleLinkCheck) Reporter() string {
	return RuleLinkCheckName
}

func (c RuleLinkCheck) Check(ctx context.Context, _ string, rule parser.Rule, _ []discovery.Entry) (problems []Problem) {
	if rule.AlertingRule == nil || rule.AlertingRule.Annotations == nil {
		return nil
	}

	var u *url.URL
	var err error
	var uri string
	var re *regexp.Regexp
	for _, ann := range rule.AlertingRule.Annotations.Items {
		u, err = url.Parse(ann.Value.Value)
		if err != nil {
			continue
		}

		if !slices.Contains(c.scheme, u.Scheme) {
			continue
		}

		re = c.re.MustExpand(rule)
		if !re.MatchString(u.String()) {
			continue
		}

		uri = u.String()
		slog.Debug("Found link to check", slog.String("link", uri))
		if c.uriRewrite != "" {
			var result []byte
			for _, submatches := range re.FindAllStringSubmatchIndex(uri, -1) {
				result = re.ExpandString(result, c.uriRewrite, uri, submatches)
			}
			uri = string(result)
			slog.Debug("Link URI rewritten by rule", slog.String("link", u.String()), slog.String("uri", uri))
		}

		rctx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()

		req, _ := http.NewRequestWithContext(rctx, http.MethodGet, uri, nil)

		for k, v := range c.headers {
			req.Header.Set(k, v)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			problems = append(problems, Problem{
				Fragment: ann.Value.Value,
				Lines:    ann.Lines(),
				Reporter: c.Reporter(),
				Text:     fmt.Sprintf("GET request for %s returned an error: %s", uri, err),
				Severity: c.severity,
			})
			slog.Debug("Link request returned an error", slog.String("uri", uri), slog.Any("err", err))
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			problems = append(problems, Problem{
				Fragment: ann.Value.Value,
				Lines:    ann.Lines(),
				Reporter: c.Reporter(),
				Text:     fmt.Sprintf("GET request for %s returned invalid status code: %s", uri, resp.Status),
				Severity: c.severity,
			})
			slog.Debug("Link request returned invalid status code", slog.String("uri", uri), slog.String("status", resp.Status))
			continue
		}
		slog.Debug("Link request returned a valid status code", slog.String("uri", uri), slog.String("status", resp.Status))
	}

	return problems
}
