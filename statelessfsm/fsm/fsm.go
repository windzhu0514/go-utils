package statelessfsm

import "context"

// type State[T any] func(ctx context.Context, args T) (T, State[T], error)
type FSM interface {
	GetMachineId() string
}

type StateMachine struct{}

func NewFSM() *StateMachine {
	return &StateMachine{}
}

func (f *StateMachine) Fire(ctx context.Context, s State, e Event) State {
	return nil
}

func (f *StateMachines) CanFire() {
}
