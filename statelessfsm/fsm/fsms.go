package statelessfsm

import (
	"sync"
)

type fsms struct {
	fsms sync.Map
}

var fsmsSet = &fsms{
	fsms: sync.Map{},
}

func regiter(fsm FSM) {
	fsmsSet.fsms.Store(fsm.GetMachineId(), fsm)
}

func deregiter(fsm FSM) {
	fsmsSet.fsms.Delete(fsm.GetMachineId())
}

func Get(fsm FSM) FSM {
	v, ok := fsmsSet.fsms.Load(fsm.GetMachineId())
	if !ok {
		return nil
	}

	return v.(FSM)
}
