package main

import (
	"fmt"

	"github.com/looplab/fsm"
)

func main() {
	fsm := fsm.NewFSM(
		"closed",
		fsm.Events{
			{Name: "open", Src: []string{"closed"}, Dst: "open"},
			{Name: "close", Src: []string{"open"}, Dst: "closed"},
		},
		fsm.Callbacks{
			"before_event": func(e *fsm.Event) {
				fmt.Println("before_event:src", e.Src, " dst:", e.Dst)
			},
			"leave_state": func(e *fsm.Event) {
				fmt.Println("leave_state:src", e.Src, " dst:", e.Dst)
			},
			"enter_state": func(e *fsm.Event) {
				fmt.Println("enter_state:src", e.Src, " dst:", e.Dst)
			},
			"after_event": func(e *fsm.Event) {
				fmt.Println("after_event:src", e.Src, " dst:", e.Dst)
			},
		},
	)

	fmt.Println(fsm.Current())

	err := fsm.Event("open")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(fsm.Current())

	err = fsm.Event("close")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(fsm.Current())
}
