package runner

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/KazZzeL/geomixer/internal/config"
)

type Parser interface {
	fmt.Stringer
	Parse(ctx context.Context) error
}

type Generator interface {
	fmt.Stringer
	Generate(ctx context.Context) error
}

type Runner struct {
	outputDir    string
	inputs       []*config.Input
	outputs      []*config.Output
	newParser    func(cfg *config.Input, hc *http.Client, ht time.Duration) Parser
	newGenerator func(cfg *config.Output, d string, p map[string]Parser) Generator
	mtx          sync.Mutex
}

func NewRunner(
	cfg *config.Runner,
	outputDir string,
	newParser func(cfg *config.Input, hc *http.Client, ht time.Duration) Parser,
	newGenerator func(cfg *config.Output, d string, p map[string]Parser) Generator,
) *Runner {
	return &Runner{
		outputDir:    outputDir,
		inputs:       cfg.Inputs,
		outputs:      cfg.Outputs,
		newParser:    newParser,
		newGenerator: newGenerator,
	}
}

func (r *Runner) Run(ctx context.Context, hc *http.Client, ht time.Duration) error {
	parsers, err := r.runParsers(ctx, hc, ht)
	if err != nil {
		return fmt.Errorf("run inputs: %w", err)
	}

	if err := r.runGenerators(ctx, parsers); err != nil {
		return fmt.Errorf("run outputs: %w", err)
	}

	return nil
}

func (r *Runner) runParsers(ctx context.Context, hc *http.Client, ht time.Duration) (map[string]Parser, error) {
	parsers := make(map[string]Parser, len(r.inputs))

	eg, egCtx := errgroup.WithContext(ctx)
	for _, cfg := range r.inputs {
		select {
		case <-egCtx.Done():
			return nil, fmt.Errorf("ctx: %w", egCtx.Err())
		default:
			eg.Go(func() error {
				parser := r.newParser(cfg, hc, ht)

				r.mtx.Lock()
				parsers[parser.String()] = parser
				r.mtx.Unlock()

				if err := parser.Parse(egCtx); err != nil {
					return fmt.Errorf("input %s: %w", parser.String(), err)
				}

				return nil
			})
		}
	}

	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	return parsers, nil
}

func (r *Runner) runGenerators(ctx context.Context, parsers map[string]Parser) error {
	eg, egCtx := errgroup.WithContext(ctx)
	for _, cfg := range r.outputs {
		select {
		case <-egCtx.Done():
			return fmt.Errorf("ctx: %w", egCtx.Err())
		default:
			eg.Go(func() error {
				generator := r.newGenerator(cfg, r.outputDir, parsers)

				if err := generator.Generate(egCtx); err != nil {
					return fmt.Errorf("output %s: %w", generator.String(), err)
				}

				return nil
			})
		}
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	return nil
}
