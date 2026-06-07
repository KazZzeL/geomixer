package geoip

import (
	"net/http"
	"time"

	"github.com/KazZzeL/geomixer/internal/config"
	"github.com/KazZzeL/geomixer/internal/utils/runner"
)

type Runner struct {
	*runner.Runner
}

func NewRunner(cfg *config.Runner, outputDir string) *Runner {
	return &Runner{
		Runner: runner.NewRunner(cfg, outputDir, newParser, newGenerator),
	}
}

func newParser(cfg *config.Input, hc *http.Client, ht time.Duration) runner.Parser {
	return NewInput(cfg, hc, ht)
}

func newGenerator(cfg *config.Output, defaultDir string, parsers map[string]runner.Parser) runner.Generator {
	typedParsers := make(map[string]*Input, len(parsers))

	for name, p := range parsers {
		if input, ok := p.(*Input); ok {
			typedParsers[name] = input
		}
	}

	return NewOutput(cfg, defaultDir, typedParsers)
}
