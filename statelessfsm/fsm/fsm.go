package statelessfsm

import "context"

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
