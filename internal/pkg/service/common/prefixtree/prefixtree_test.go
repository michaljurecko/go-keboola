package prefixtree_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/keboola/keboola-as-code/internal/pkg/service/common/prefixtree"
)

type value struct {
	field string
}

func TestPrefixTree(t *testing.T) {
	t.Parallel()
	tree := New[value]()

	// Get - not found
	_, found := tree.Get("key/1")
	assert.False(t, found)

	// AllFromPrefix - no value
	assert.Len(t, tree.AllFromPrefix("key"), 0)
	_, found = tree.FirstFromPrefix("key")
	assert.False(t, found)
	_, found = tree.LastFromPrefix("key")
	assert.False(t, found)

	// -----
	tree.Insert("key/1", value{field: "value1"})
	tree.Insert("key/2", value{field: "value2"})

	// Get - found
	v, found := tree.Get("key/1")
	assert.True(t, found)
	assert.Equal(t, value{field: "value1"}, v)
	v, found = tree.Get("key/2")
	assert.True(t, found)
	assert.Equal(t, value{field: "value2"}, v)

	// AllFromPrefix / FirstFromPrefix / LastFromPrefix - 2 items
	assert.Len(t, tree.AllFromPrefix("key"), 2)
	v, found = tree.FirstFromPrefix("key")
	assert.True(t, found)
	assert.Equal(t, value{field: "value1"}, v)
	v, found = tree.LastFromPrefix("key")
	assert.True(t, found)
	assert.Equal(t, value{field: "value2"}, v)

	// -----
	tree.Delete("key/2")

	// Get - found
	v, found = tree.Get("key/1")
	assert.True(t, found)
	assert.Equal(t, value{field: "value1"}, v)

	// Get - not found
	_, found = tree.Get("key/2")
	assert.False(t, found)

	// AllFromPrefix / FirstFromPrefix / LastFromPrefix - 1 item
	assert.Len(t, tree.AllFromPrefix("key"), 1)
	v, found = tree.FirstFromPrefix("key")
	assert.True(t, found)
	assert.Equal(t, value{field: "value1"}, v)
	v, found = tree.LastFromPrefix("key")
	assert.True(t, found)
	assert.Equal(t, value{field: "value1"}, v)

	// Atomic
	tree.Atomic(func(t *Tree[value]) {
		t.Delete("key/1")
		t.Delete("key/2")
		t.Insert("key/3", value{field: "foo"})
		t.Insert("key/4", value{field: "bar"})
	})
	tree.AtomicReadOnly(func(ro TreeReadOnly[value]) {
		_, found = ro.Get("key/1")
		assert.False(t, found)
		_, found = ro.Get("key/2")
		assert.False(t, found)
		_, found = ro.Get("key/3")
		assert.True(t, found)
		_, found = ro.Get("key/4")
		assert.True(t, found)
	})

	// ToMap
	assert.Equal(t, map[string]value{
		"key/3": {field: "foo"},
		"key/4": {field: "bar"},
	}, tree.ToMap())

	// Reset
	tree.Reset()
	assert.Empty(t, tree.AllFromPrefix(""))

	// DeletePrefix
	tree.Insert("key/1", value{field: "value1"})
	tree.Insert("key/2", value{field: "value2"})
	tree.DeletePrefix("key/")
	assert.Empty(t, tree.AllFromPrefix(""))
}
