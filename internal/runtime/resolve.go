package runtime

import "github.com/salman-frs/meridian/internal/model"

type ResolvedEngine struct {
	adapter engineAdapter
}

func Resolve(requested model.RuntimeEngine) (ResolvedEngine, error) {
	adapter, err := resolveEngine(requested)
	if err != nil {
		return ResolvedEngine{}, err
	}
	return ResolvedEngine{adapter: adapter}, nil
}

func ResolveEngine(requested model.RuntimeEngine) (engineAdapter, error) {
	return resolveEngine(requested)
}

func (r ResolvedEngine) Engine() model.RuntimeEngine {
	if r.adapter == nil {
		return ""
	}
	return r.adapter.Engine()
}

func (r ResolvedEngine) RuntimeBackend() string {
	if r.adapter == nil {
		return ""
	}
	return r.adapter.RuntimeBackend()
}

func (r ResolvedEngine) CaptureEndpoint(address string, capturePort int) string {
	if r.adapter == nil {
		return address
	}
	return r.adapter.CaptureEndpoint(address, capturePort)
}

func (r ResolvedEngine) adapterOrNil() engineAdapter {
	return r.adapter
}
