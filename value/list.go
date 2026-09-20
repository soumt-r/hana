// Package value holds the runtime values that are more than a Go primitive.
package value

// List is a hana list. A list is an object: a variable, a parameter and a field that
// were given the same list see the same one, so a function that pushes onto the list
// it was passed changes the caller's list (Runtime spec 1.2). Items is replaced
// (never shared between two Lists) when the list grows, shrinks or is emptied.
type List struct {
	Items []interface{}
}

// NewList makes a list of items, which the list takes over.
func NewList(items []interface{}) *List { return &List{Items: items} }

// Push puts v at the front or the back of the list.
func (l *List) Push(v interface{}, front bool) {
	if front {
		l.Items = append([]interface{}{v}, l.Items...)
	} else {
		l.Items = append(l.Items, v)
	}
}

// Unpush takes back what Push just put at that end (for a push that has to be refused
// after it was made). It must follow the Push directly.
func (l *List) Unpush(front bool) { l.pop(front) }

// Pop removes the front or the back element ("front" or anything else for the back)
// and returns it. The list must not be empty.
func (l *List) Pop(position string) interface{} { return l.pop(position == "front") }

func (l *List) pop(front bool) interface{} {
	n := len(l.Items)
	if front {
		v := l.Items[0]
		l.Items = l.Items[1:]
		return v
	}
	v := l.Items[n-1]
	l.Items = l.Items[:n-1]
	return v
}
