package app

import "context"

type addonApp interface {
	Run(<-chan struct{}) error
}

var _ Runnable = &addonRunnable{}

type addonRunnable struct {
	runner addonApp
}

func (k *addonRunnable) Run(ctx context.Context) error {
	return k.runner.Run(ctx.Done())
}
