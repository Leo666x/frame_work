package powerai

import (
	"context"
	"github.com/gin-gonic/gin"
)

type Option struct {
	F func(o *Options)
}

type Options struct {
	PostRouters           map[string]gin.HandlerFunc
	GetRouters            map[string]gin.HandlerFunc
	OnShutDown            func(ctx context.Context)
	DefaultConfigs        map[string]*Config
	ConfigChangeCallbacks []func(k string)
	Decision              *Decision
}

func (o *Options) Apply(opts []Option) {
	for _, op := range opts {
		op.F(o)
	}
}

func WithCustomPostRouter(n string, f gin.HandlerFunc) Option {
	return Option{
		F: func(o *Options) {
			if o.PostRouters == nil {
				o.PostRouters = make(map[string]gin.HandlerFunc)
			}
			o.PostRouters[n] = f
		},
	}
}

func WithCustomGetRouter(n string, f gin.HandlerFunc) Option {
	return Option{
		F: func(o *Options) {
			if o.GetRouters == nil {
				o.GetRouters = make(map[string]gin.HandlerFunc)
			}
			o.GetRouters[n] = f
		},
	}
}
func WithSendMsgRouter(f gin.HandlerFunc) Option {
	return Option{
		F: func(o *Options) {
			if o.PostRouters == nil {
				o.PostRouters = make(map[string]gin.HandlerFunc)
			}
			o.PostRouters["send_msg"] = f
		},
	}
}

func WithOnShutDown(f func(ctx context.Context)) Option {
	return Option{
		F: func(o *Options) {
			o.OnShutDown = f
		},
	}
}

func WithDefaultConfigs(configs map[string]*Config) Option {
	return Option{
		F: func(o *Options) {
			o.DefaultConfigs = configs
		},
	}
}

func WithConfigChangeCallbacks(f ...func(k string)) Option {
	return Option{
		F: func(o *Options) {
			o.ConfigChangeCallbacks = append(o.ConfigChangeCallbacks, f...)
		},
	}
}

func newOptions(opts []Option) *Options {
	options := &Options{
		PostRouters: make(map[string]gin.HandlerFunc),
	}
	options.Apply(opts)
	return options
}
