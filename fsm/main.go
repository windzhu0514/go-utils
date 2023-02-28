package main

import (
	"github.com/windzhu0514/go-utils/fsm/fsm"
)

func main() {
	f := fsm.NewFSM([]fsm.StateInfo{
		{
			State: fsm.State{},
			Events: []fsm.EventInfo{
				{
					Name:            "event1",
					StateChangeInfo: fsm.StateFrom(fsm.State{}).To(fsm.State{}),
					EventHandler:    nil,
				},
				{
					Name:            "event2",
					StateChangeInfo: fsm.StateFrom(fsm.State{}).To(fsm.State{}),
					EventHandler:    nil,
				},
			},
		},
		{
			State: fsm.State{},
			Events: []fsm.EventInfo{
				{
					Name:            "event3",
					StateChangeInfo: fsm.StateFrom(fsm.State{}).To(fsm.State{}),
					EventHandler:    nil,
				},
			},
		},
	}, func(ctx fsm.StateContext) (fsm.State, error) {
		return fsm.State{}, nil
	})

	f.SendEvent(fsm.EventContext{})
}
