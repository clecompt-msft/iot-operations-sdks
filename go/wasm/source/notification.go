package source

import "github.com/Azure/iot-operations-sdks/go/wasm/internal/aio/connector/host"

type (
	Notification interface {
		notification()
	}

	DatasetCreated struct {
		*Dataset
	}

	DatasetUpdated struct {
		*Dataset
	}

	DatasetRemoved struct {
		Name string
	}

	EventCreated struct {
		*Event
	}

	EventUpdated struct {
		*Event
	}

	EventRemoved struct {
		Name string
	}
)

func notification(n *host.Notification) Notification {
	if d := n.DatasetCreated(); d != nil {
		return DatasetCreated{d}
	}
	if d := n.DatasetUpdated(); d != nil {
		return DatasetUpdated{d}
	}
	if s := n.DatasetRemoved(); s != nil {
		return DatasetRemoved{*s}
	}
	if e := n.EventCreated(); e != nil {
		return EventCreated{e}
	}
	if e := n.EventUpdated(); e != nil {
		return EventUpdated{e}
	}
	if s := n.EventRemoved(); s != nil {
		return EventRemoved{*s}
	}
	return nil
}

func (DatasetCreated) notification() {}
func (DatasetUpdated) notification() {}
func (DatasetRemoved) notification() {}
func (EventCreated) notification()   {}
func (EventUpdated) notification()   {}
func (EventRemoved) notification()   {}
