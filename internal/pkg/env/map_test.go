package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnvMap(t *testing.T) {
	m := Empty()
	assert.Len(t, m.Keys(), 0)

	// Set
	m.Set(`abc_def`, `123`)
	assert.Equal(t, []string{`ABC_DEF`}, m.Keys())
	assert.Equal(t, `123`, m.Get(`abc_def`))
	assert.Equal(t, `123`, m.MustGet(`abc_def`))
	assert.Equal(t, `123`, m.Get(`ABC_DEF`))
	v, found := m.Lookup(`ABC_def`)
	assert.Equal(t, `123`, v)
	assert.True(t, found)

	// Overwrite
	m.Set(`abc_DEF`, `456`)
	assert.Equal(t, []string{`ABC_DEF`}, m.Keys())
	assert.Equal(t, `456`, m.Get(`abc_DEF`))
	assert.Equal(t, `456`, m.MustGet(`abc_DEF`))
	assert.Equal(t, `456`, m.Get(`ABC_DEF`))
	v, found = m.Lookup(`abc_def`)
	assert.Equal(t, `456`, v)
	assert.True(t, found)

	// Missing key
	assert.Equal(t, []string{`ABC_DEF`}, m.Keys())
	assert.Equal(t, ``, m.Get(`foo`))
	assert.PanicsWithError(t, `missing ENV variable "FOO"`, func() {
		m.MustGet(`foo`)
	})
	m.Unset(`foo`)
	assert.Equal(t, []string{`ABC_DEF`}, m.Keys())

	// Unset
	m.Unset(`ABC_def`)
	assert.Len(t, m.Keys(), 0)
	str, err := m.ToString()
	assert.NoError(t, err)
	assert.Equal(t, ``, str)

	// ToString
	m.Set(`A`, `123`)
	m.Set(`X`, `Y`)
	str, err = m.ToString()
	assert.NoError(t, err)
	assert.Equal(t, "A=\"123\"\nX=\"Y\"", str)
}

func TestEnvMapFromOs(t *testing.T) {
	assert.NoError(t, os.Setenv(`Foo`, `bar`))
	m, err := FromOs()
	assert.NotNil(t, m)
	assert.NoError(t, err)
	str, err := m.ToString()
	assert.NoError(t, err)
	assert.Contains(t, str, `FOO="bar"`)
}

func TestEnvMapMerge(t *testing.T) {
	m1 := Empty()
	m2 := Empty()

	m1.Set(`A`, `1`)
	m1.Set(`B`, `2`)

	m2.Set(`B`, `20`)
	m2.Set(`C`, `30`)

	m1.Merge(m2, false) // overwrite = false
	str, err := m1.ToString()

	assert.NoError(t, err)
	assert.Equal(t, "A=\"1\"\nB=\"2\"\nC=\"30\"", str)
}

func TestEnvMapMergeOverwrite(t *testing.T) {
	m1 := Empty()
	m2 := Empty()

	m1.Set(`A`, `1`)
	m1.Set(`B`, `2`)

	m2.Set(`B`, `20`)
	m2.Set(`C`, `30`)

	m1.Merge(m2, true) // overwrite = true
	str, err := m1.ToString()

	assert.NoError(t, err)
	assert.Equal(t, "A=\"1\"\nB=\"20\"\nC=\"30\"", str)
}

