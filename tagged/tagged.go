package tagged

import "reflect"

// Value holds an arbitrary value with associated tags
type Value[T any] struct {
	Value T
	Tags  []string
}

// HasTag indicates the Value has a tag matching one or more of the provided arguments
func (t Value[T]) HasTag(tags ...string) bool {
	for _, tag := range tags {
		for _, existing := range t.Tags {
			if tag == existing {
				return true
			}
		}
	}
	return false
}

// New returns a tagged value, that can be added to a Values collection
func New[T any](value T, tags ...string) Value[T] {
	return Value[T]{
		Value: value,
		Tags:  tags,
	}
}

// Values is a utility to handle a set of tagged items including basic filtering
type Values[T any] []Value[T]

// Collect returns a slice containing the values in the set
func (t Values[T]) Collect() []T {
	out := make([]T, len(t))
	for i, v := range t {
		out[i] = v.Value
	}
	return out
}

// HasTag indicates this set contains one or more items matching any of the provided tags
func (t Values[T]) HasTag(tags ...string) bool {
	for _, tagged := range t {
		if tagged.HasTag(tags...) {
			return true
		}
	}
	return false
}

// HasValue indicates at least one of the provided values is present in this set
func (t Values[T]) HasValue(value ...T) bool {
	for _, tagged := range t {
		for _, v := range value {
			if isEqual(tagged.Value, v) {
				return true
			}
		}
	}
	return false
}

// Select returns a new set of values matching any of the provided tags
func (t Values[T]) Select(tags ...string) Values[T] {
	if len(tags) == 0 {
		return Values[T]{}
	}
	out := make(Values[T], 0, len(t))
	for _, tagged := range t {
		if tagged.HasTag(tags...) {
			out = append(out, tagged)
		}
	}
	return out
}

// Remove returns a new set of values, excluding those with any of the provided tags
func (t Values[T]) Remove(tags ...string) Values[T] {
	if len(tags) == 0 {
		return t
	}
	out := make(Values[T], 0, len(t))
	for _, tagged := range t {
		if !tagged.HasTag(tags...) {
			out = append(out, tagged)
		}
	}
	return out
}

// Join returns a new set of values, combining this set and the provided values, omitting duplicates
func (t Values[T]) Join(taggedValues ...Value[T]) Values[T] {
	if len(taggedValues) == 0 {
		return t
	}
	out := make(Values[T], 0, len(t)+len(taggedValues))
	out = append(out, t...)
	for _, value := range taggedValues {
		if t.HasValue(value.Value) {
			continue
		}
		out = append(out, value)
	}
	return out
}

// Sort returns a new set of Values with the entries ordered by the provided tags
func (t Values[T]) Sort(tags ...string) Values[T] {
	if len(tags) == 0 {
		return t
	}
	out := make(Values[T], 0, len(t))
	for _, tag := range tags {
		for _, existing := range t {
			if existing.HasTag(tag) {
				if out.HasValue(existing.Value) {
					continue
				}
				out = append(out, existing)
			}
		}
	}
	for _, v := range t {
		if !v.HasTag(tags...) {
			out = append(out, v)
		}
	}
	return out
}

func isEqual(a, b any) bool {
	if a == b {
		return true
	}
	if reflect.ValueOf(a) == reflect.ValueOf(b) {
		return true
	}
	return false
}
