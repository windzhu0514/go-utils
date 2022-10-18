package fsm

import (
	"context"
	"errors"
)

// TODO:
// 事件并发处理

type State struct {
	Name string
	ID   string
}

type EventContext struct {
	Name string
	Data interface{}
}

type EventInfo struct {
	Name            string // 事件名
	StateChangeInfo StateChangeInfo
	EventHandler    EventProcessor
}

type StateInfo struct {
	State State
	// StateChange StateChange
	Events []EventInfo
}

type StateChangeInfo struct {
	srcState State
	dstState State
}

type FromState struct {
	state State
}

func StateFrom(state State) *FromState {
	return &FromState{state: state}
}

func (f *FromState) To(state State) StateChangeInfo {
	return StateChangeInfo{
		srcState: f.state,
		dstState: state,
	}
}

type FSM struct {
	States      []StateInfo
	Events      map[string]map[string]EventInfo
	StateSetter func(ctx StateContext) error          // 设置状态
	StateGetter func(ctx StateContext) (State, error) // 获取状态
}

func NewFSM(states []StateInfo, stateGetter func(ctx StateContext) (State, error)) *FSM {
	return &FSM{
		States:      states,
		Events:      make(map[string]map[string]EventInfo),
		StateGetter: stateGetter,
	}
}

type StateContext struct {
	ctx context.Context
}

type EventProcessor interface {
	Initialize() error
	BeforeEnter(ctx context.Context) error
	Action(state State) error
	BeforeLeave(ctx context.Context) error
}

func (f *FSM) SendEvent(e EventContext) {
}

func (f *FSM) transition(e EventContext) error {
	// 获取当前状态
	f.StateGetter(StateContext{})
	var state State
	// 查找当前状态的处理事件
	events, ok := f.Events[state.Name]
	if !ok {
		return errors.New("no state events")
	}

	// 查找事件
	event, ok := events[e.Name]
	if !ok {
		return errors.New("no event")
	}

	// 参数检查
	err := event.EventHandler.BeforeEnter(context.Background())
	if err != nil {
		return err
	}

	// 执行逻辑
	err = event.EventHandler.Action(state)
	if err != nil {
		return err
	}

	// 收尾
	err = event.EventHandler.BeforeLeave(context.Background())
	if err != nil {
		return err
	}

	err = f.StateSetter(StateContext{})
	if err != nil {
		return err
	}

	return nil
}
