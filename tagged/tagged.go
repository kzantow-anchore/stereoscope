package tagged

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

// HasTag indicates one or more Values within this set has a tag matching one or more of the provided arguments
func (t Values[T]) HasTag(tags ...string) bool {
	for _, tagged := range t {
		if tagged.HasTag(tags...) {
			return true
		}
	}
	return false
}

// Keep returns a new set of Values matching any of the provided tags or all values if no tags provided
func (t Values[T]) Keep(tags ...string) Values[T] {
	if len(tags) == 0 {
		return t
	}
	out := make(Values[T], 0, len(t))
	for _, tagged := range t {
		if tagged.HasTag(tags...) {
			out = append(out, tagged)
		}
	}
	return out
}

// Remove returns a new set of Values that do not match any of the provided tags
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

// Collect returns a slice containing the values in the set
func (t Values[T]) Collect() []T {
	out := make([]T, len(t))
	for i, v := range t {
		out[i] = v.Value
	}
	return out
}

// Add adds the tagged sets to a new set and returns the new set
func (t Values[T]) Add(tagged Values[T]) Values[T] {
	if len(tagged) == 0 {
		return t
	}
	out := make(Values[T], 0, len(t)+len(tagged))
	out = append(out, t...)
next:
	for _, value := range tagged {
		// check if already present and skip if so
		for _, existing := range t {
			if isEqual(existing.Value, value.Value) {
				continue next
			}
		}
		out = append(out, value)
	}
	return out
}

func isEqual(a, b any) bool {
	return a == b
}
