package source

import (
	"context"
	"errors"
	"time"

	"github.com/Azure/iot-operations-sdks/go/wasm/internal/aio/connector/host"
	"github.com/Azure/iot-operations-sdks/go/wasm/internal/aio/connector/source"
	"github.com/Azure/iot-operations-sdks/go/wasm/net"
	"go.bytecodealliance.org/cm"
)

func Init(run func() error) {
	net.UseWASI()
	source.Exports.Run = func() cm.Result[string, struct{}, string] {
		if err := run(); err != nil {
			return cm.Err[cm.Result[string, struct{}, string]](err.Error())
		}
		return cm.OK[cm.Result[string, struct{}, string]](struct{}{})
	}
}

func Poll(ctx context.Context, freq time.Duration) <-chan Notification {
	ch := make(chan Notification)
	go func() {
		defer close(ch)
		for {
			n := host.Poll()
			if n.None() {
				select {
				case <-time.After(freq):
					continue
				case <-ctx.Done():
					return
				}
			}

			if n.Some().Shutdown() {
				return
			}

			select {
			case ch <- notification(n.Some()):
				continue
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch
}

func Send(kind, name string, data []byte) error {
	res := host.Send(kind, name, cm.ToList(data))
	if res.IsErr() {
		return errors.New(*res.Err())
	}
	return nil
}
