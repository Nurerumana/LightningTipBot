package intercept

import (
	"context"
	"github.com/google/uuid"

	log "github.com/sirupsen/logrus"
	tb "gopkg.in/lightningtipbot/telebot.v3"
)

// Context bundles tb.Context and context.Context
type Context struct {
	context.Context
	TeleContext
}

// TeleContext is a proxy struct for tb.Context
type TeleContext struct {
	tb.Context
}

// Func is the custom intercept handler func type. This will accept and return the bundled context
type Func func(ctx Context) (Context, error)

// handlerWithInterceptor is the custom handler func containing a Chain of intercept's
type handlerWithInterceptor struct {
	handler Func
	before  Chain
	after   Chain
	onDefer Chain
	fields  log.Fields
}

// Chain are multiple custom intercept Func
type Chain []Func

// Option for building custom intercept Chain
type Option func(*handlerWithInterceptor)

func WithBefore(chain ...Func) Option {
	return func(a *handlerWithInterceptor) {
		a.before = chain
	}
}
func WithAfter(chain ...Func) Option {
	return func(a *handlerWithInterceptor) {
		a.after = chain
	}
}
func WithDefer(chain ...Func) Option {
	return func(a *handlerWithInterceptor) {
		a.onDefer = chain
	}
}

func WithDefaultLogFields(f log.Fields) Option {
	return func(a *handlerWithInterceptor) {
		a.fields = f
	}
}

// intercept is where all intercept's from single Chain are executed
func intercept(ctx Context, chain Chain) (Context, error) {
	if chain != nil {
		var err error
		for _, m := range chain {
			ctx, err = m(ctx)
			if err != nil {
				return ctx, err
			}
		}
	}
	return ctx, nil
}

// WithHandler returns a tb.HandlerFunc and which will proxy the real handler.
func WithHandler(handler Func, option ...Option) tb.HandlerFunc {
	// create handlerWithInterceptor
	hm := &handlerWithInterceptor{handler: handler}
	for _, opt := range option {
		opt(hm)
	}
	// return proxy handler
	return func(c tb.Context) error {
		h := Context{TeleContext: TeleContext{Context: c}, Context: context.Background()}
		// add default log fields from handler to context
		if len(hm.fields) > 0 {
			h.Context = context.WithValue(h.Context, "fields", hm.fields)
		}
		// add uuid for log tracing to context
		h.Context = context.WithValue(h.Context, "uuid", uuid.New().String())
		// run all before intercepts
		h, err := intercept(h, hm.before)
		if err != nil {
			log.Traceln(err)
			return err
		}
		// make sure to run onDefer intercepts
		defer intercept(h, hm.onDefer)
		// run the real handler
		h, err = hm.handler(h)
		if err != nil {
			log.Traceln(err)
			return err
		}
		// run all after intercepts
		_, err = intercept(h, hm.after)
		if err != nil {
			log.Traceln(err)
			return err
		}
		return nil
	}
}
