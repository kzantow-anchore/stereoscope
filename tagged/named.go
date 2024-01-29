package tagged

type NamedValue[T any] struct {
	Name  string
	Value *T // must be a pointer for this struct to be comparable
}

func Named[T any](name string, value T) NamedValue[T] {
	return NamedValue[T]{
		Name:  name,
		Value: &value,
	}
}
