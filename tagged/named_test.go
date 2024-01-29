package tagged

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_NamedValueFunc(t *testing.T) {
	var taggedFunc1 = Named[testFunc]("one", func1)
	var taggedFunc2 = Named[testFunc]("two", func2)

	var values = Values[NamedValue[testFunc]]{
		New(taggedFunc1, "tagged"),
		New(Named[testFunc]("two", func2), "tagged", "twice"),
	}

	require.True(t, values.HasValue(taggedFunc1))
	require.False(t, values.HasValue(taggedFunc2))
}

func Test_NamedValuesInterface(t *testing.T) {
	var named1 = Named[testInterface]("one", &testStruct{
		fn: func1,
	})

	var named2 = Named[testInterface]("two", &testStruct{
		fn: func2,
	})

	var values = Values[NamedValue[testInterface]]{
		New(named1, "tagged"),
		New(Named[testInterface]("two", &testStruct{func2}), "tagged", "twice"),
	}

	require.True(t, values.HasValue(named1))
	require.False(t, values.HasValue(named2))
}

type testFunc func(t *testing.T)

type testInterface interface {
	Fn(t *testing.T)
}

type testStruct struct {
	fn func(t *testing.T)
}

func (s *testStruct) Fn(_ *testing.T) {
}

var _ testInterface = (*testStruct)(nil)

func func1(_ *testing.T) {
}

func func2(_ *testing.T) {
}
