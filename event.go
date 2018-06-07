package rotatelogs

func (h HandlerFunc) Handle(e Event) {
	h(e)
}

func (e *RotateEvent) Type() EventType {
	return FileRotatedEvent
}

func (e *RotateEvent) PreviousFile() string {
	return e.prev
}

func (e *RotateEvent) CurrentFile() string {
	return e.current
}
