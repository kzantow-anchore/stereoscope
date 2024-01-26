package tagged

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_sel(t *testing.T) {
	tag := func(name string, tags ...string) Value[string] { return New(name, append(tags, name)...) }
	all := Values[string]{
		tag("js-1", "i", "js"),
		tag("js-2", "d", "js"),
		tag("js-3", "i", "d", "js"),
		tag("jv-1", "i", "d", "jv"),
		tag("jv-2", "i", "d", "jv"),
		tag("py-1", "i", "d", "py"),
		tag("py-2", "d", "py"),
		tag("py-3", "i", "d", "py"),
		tag("py-4", "i", "py"),
		tag("sc-1"),
	}

	tests := []struct {
		name     string
		req      req
		expected []string
	}{
		{
			name: "--override-default-catalogers i",
			req: req{
				Base: arr("i"),
			},
			expected: arr("js-1", "js-3", "jv-1", "jv-2", "py-1", "py-3", "py-4"),
		},
		{
			name: "--override-default-catalogers i,js",
			req: req{
				Base: arr("i", "js"),
			},
			expected: arr("js-1", "js-2", "js-3", "jv-1", "jv-2", "py-1", "py-3", "py-4"),
		},
		{
			name: "--select-catalogers “+javascript”  [ERROR]",
			req: req{
				Base: arr("i"),
				Add:  arr("js"),
			},
			expected: arr("js-1", "js-3", "jv-1", "jv-2", "py-1", "py-3", "py-4", "js-2"),
		},
		{
			name: "--select-catalogers +sc-1",
			req: req{
				Base: arr("i"),
				Add:  arr("sc-1"),
			},
			expected: arr("js-1", "js-3", "jv-1", "jv-2", "py-1", "py-3", "py-4", "sc-1"),
		},
		{
			name: "--select-catalogers -py-1",
			req: req{
				Base:   arr("i"),
				Remove: arr("py-1"),
			},
			expected: arr("js-1", "js-3", "jv-1", "jv-2", "py-3", "py-4"),
		},
		{
			name: "--select-catalogers js",
			req: req{
				Base:   arr("i"),
				Select: arr("js"),
			},
			expected: arr("js-1", "js-3"),
		},
		{
			name: "--override-default-catalogers d --select-catalogers py,js",
			req: req{
				Base:   arr("d"),
				Select: arr("py", "js"),
			},
			expected: arr("js-2", "js-3", "py-1", "py-2", "py-3"),
		},
		{
			name: "--select-catalogers -py-1,-py-2,+sc-1,+js-2",
			req: req{
				Base:   arr("i"),
				Add:    arr("sc-1", "js-2"),
				Remove: arr("py-1", "py-2"),
			},
			expected: arr("js-1", "js-3", "jv-1", "jv-2", "py-3", "py-4", "js-2", "sc-1"),
		},
		{
			name: "--select-catalogers js,py",
			req: req{
				Base:   arr("i"),
				Select: arr("js", "py"),
			},
			expected: arr("js-1", "js-3", "py-1", "py-3", "py-4"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := sel(all, test.req)
			require.ElementsMatch(t, test.expected, got)
		})
	}
}

type req struct {
	Base   []string
	Select []string
	Remove []string
	Add    []string
}

func sel[T any](allValues Values[T], r req) []T {
	values := allValues
	if len(r.Base) > 0 {
		values = values.Select(r.Base...)
	}
	if len(r.Select) > 0 {
		values = values.Select(r.Select...)
	}
	if len(r.Remove) > 0 {
		values = values.Remove(r.Remove...)
	}
	if len(r.Add) > 0 {
		values = values.Join(
			allValues.Select(r.Add...)...,
		)
	}
	return values.Collect()
}
