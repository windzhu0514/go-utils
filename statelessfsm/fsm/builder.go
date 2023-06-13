package statelessfsm

type Builer interface{}

type builer struct{}

type (
	State  interface{}
	Event  interface{}
	Action interface{}
)

func NewTransitionBuilder() Builer {
	return &builer{}
}

type ExternalTransition struct {
	source State
	target State
	on     Event
	action Action
}

func (t *ExternalTransition) From(s State) *ExternalTransition {
	t.source = s
	return t
}

func (t *ExternalTransition) To(s State) *ExternalTransition {
	t.target = s
	return t
}

func (t *ExternalTransition) On(e Event) *ExternalTransition {
	t.on = e
	return t
}

func (t *ExternalTransition) Action(a Action) *ExternalTransition {
	t.action = a
	return t
}

type ExternalTransitions struct {
	ExternalTransition
	sources []State
}

func (t *ExternalTransitions) FromAmong(s ...State) *ExternalTransitions {
	t.sources = s
	return t
}

type InternalTransition struct{}

func (*builer) ExternalTransitios() {
}

func (*builer) InternalTransition() {
}
