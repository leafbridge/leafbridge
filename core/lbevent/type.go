package lbevent

import "strings"

// Type uniquely identifies the type of an event. It is in the form
// "component:event".
type Type string

// Component returns the component that generates the event.
func (t Type) Component() string {
	component, _ := t.Split()
	return component
}

// Name returns the name of the event, such as "started" or "stopped".
func (t Type) Name() string {
	_, name := t.Split()
	return name
}

// Split splits the event type into its component and name.
func (t Type) Split() (component, name string) {
	parts := strings.SplitN(string(t), ":", 2)
	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return parts[0], ""
	default:
		return parts[0], parts[1]
	}
}
