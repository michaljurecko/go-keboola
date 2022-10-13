package errors_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/keboola/go-utils/pkg/wildcards"
	"github.com/stretchr/testify/assert"

	. "github.com/keboola/keboola-as-code/internal/pkg/utils/errors"
)

func TestSingleError_Format(t *testing.T) {
	t.Parallel()
	e := NewMultiError()
	e.Append(fmt.Errorf("foo bar"))
	assert.Equal(t, "foo bar", e.Error())
}

func TestSingleError_FormatWithDebug(t *testing.T) {
	t.Parallel()
	e := NewMultiError()
	e.Append(fmt.Errorf("foo bar"))
	wildcards.Assert(t, "foo bar [%s/format_test.go:24]", FormatWithDebug(e))
}

func TestMultiError_Format(t *testing.T) {
	t.Parallel()
	expected := `
- error 1
- error with debug trace
- wrapped2: wrapped1: error 2
- my prefix:
  - abc
  - def
  - sub1:
    - x
    - y
  - sub2: z
  - sub3 with format:
    - this is a very long line from error message, it is printed on new line
  - sub4:
    - 1
    - 2
    - 3
- last error
`
	assert.Equal(t, strings.TrimSpace(expected), MultiErrorForTest().Error())
}

func TestMultiError_FormatWithDebug(t *testing.T) {
	t.Parallel()
	expected := `
- error 1 [%s/errors_test.go:13]
- error with debug trace [%s/errors_test.go:10]
- wrapped2: wrapped1: error 2 [%s/errors_test.go:17]:
  - *fmt.wrapError >>> wrapped1: error 2 [%s/errors_test.go:22]:
    - *fmt.wrapError >>> error 2 [%s/errors_test.go:22]
- my prefix [%s/errors_test.go:48]:
  - abc [%s/errors_test.go:27]
  - def [%s/errors_test.go:28]
  - sub1 [%s/errors_test.go:38]:
    - x [%s/errors_test.go:31]
    - y [%s/errors_test.go:32]
  - sub2 [%s/errors_test.go:39]:
    - z [%s/errors_test.go:34]
  - sub3 with format [%s/errors_test.go:40]:
    - this is a very long line from error message, it is printed on new line [%s/errors_test.go:36]
  - sub4 [%s/errors_test.go:41]:
    - 1 [%s/errors_test.go:42]
    - 2 [%s/errors_test.go:43]
    - 3 [%s/errors_test.go:44]
- last error [%s/errors_test.go:49]
`
	wildcards.Assert(t, strings.TrimSpace(expected), FormatWithDebug(MultiErrorForTest()))
}

func TestMultiError_CustomMessageFormatter_Format(t *testing.T) {
	t.Parallel()

	// Custom function to modify message
	f := NewFormatter().
		WithPrefixFormatter(func(prefix string) string {
			return prefix + " --->"
		}).
		WithMessageFormatter(func(msg string, _ StackTrace) string {
			return fmt.Sprintf("<<< %s >>>", msg)
		})

	expected := `
- <<< error 1 >>>
- <<< error with debug trace >>>
- <<< wrapped2: wrapped1: error 2 >>>
- <<< my prefix >>> --->
  - <<< abc >>>
  - <<< def >>>
  - <<< sub1 >>> --->
    - <<< x >>>
    - <<< y >>>
  - <<< sub2 >>> ---> <<< z >>>
  - <<< sub3 with format >>> --->
    - <<< this is a very long line from error message, it is printed on new line >>>
  - <<< sub4 >>> --->
    - <<< 1 >>>
    - <<< 2 >>>
    - <<< 3 >>>
- <<< last error >>>
`
	assert.Equal(t, strings.TrimSpace(expected), f.Format(MultiErrorForTest()))
}

func TestMultiError_CustomMessageFormatter_FormatWithDebug(t *testing.T) {
	t.Parallel()

	// Custom function to modify message
	f := NewFormatter().
		WithPrefixFormatter(func(prefix string) string {
			return prefix + " --->"
		}).
		WithMessageFormatter(func(msg string, _ StackTrace) string {
			return fmt.Sprintf("| %s |", msg)
		})

	expected := `
- | error 1 |
- | error with debug trace |
- | wrapped2: wrapped1: error 2 | --->
  - *fmt.wrapError >>> | wrapped1: error 2 | --->
    - *fmt.wrapError >>> | error 2 |
- | my prefix | --->
  - | abc |
  - | def |
  - | sub1 | --->
    - | x |
    - | y |
  - | sub2 | ---> | z |
  - | sub3 with format | --->
    - | this is a very long line from error message, it is printed on new line |
  - | sub4 | --->
    - | 1 |
    - | 2 |
    - | 3 |
- | last error |
`
	assert.Equal(t, strings.TrimSpace(expected), f.FormatWithDebug(MultiErrorForTest()))
}

func TestMultiError_Flatten(t *testing.T) {
	t.Parallel()
	a := NewMultiError()
	a.Append(fmt.Errorf("A 1"))
	a.Append(fmt.Errorf("A 2"))

	b := NewMultiError()
	b.Append(fmt.Errorf("B 1"))
	b.Append(fmt.Errorf("B 2"))

	c := NewMultiError()
	c.Append(fmt.Errorf("C 1"))
	c.Append(fmt.Errorf("C 2"))

	merged := NewMultiError()
	merged.Append(a)
	merged.Append(b)
	merged.AppendWithPrefix(c, "Prefix")
	assert.Equal(t, 5, merged.Len())

	expected := `
- A 1
- A 2
- B 1
- B 2
- Prefix:
  - C 1
  - C 2
`
	assert.Equal(t, strings.TrimSpace(expected), merged.Error())
}

func TestNestedError_Format_1(t *testing.T) {
	t.Parallel()

	sub2 := NewMultiError()
	sub2.Append(fmt.Errorf("a"))
	sub2.Append(fmt.Errorf("b"))
	sub2.Append(fmt.Errorf("c"))

	sub1 := NewNestedError(fmt.Errorf("reason"), sub2)
	err := NewNestedError(fmt.Errorf("error"), sub1)
	expected := `
error:
- reason:
  - a
  - b
  - c
`
	assert.Equal(t, strings.TrimSpace(expected), err.Error())
}

func TestNestedError_Format_2(t *testing.T) {
	t.Parallel()

	sub2 := NewMultiError()
	sub2.Append(fmt.Errorf("a lorem impsum"))
	sub2.Append(fmt.Errorf("b lorem impsum"))
	sub2.Append(fmt.Errorf("c lorem impsum"))

	sub1 := NewNestedError(fmt.Errorf("reason"), sub2)
	err1 := PrefixError(sub1, "error1")
	err2 := PrefixError(err1, "error2")
	expected := `
error2:
- error1:
  - reason:
    - a lorem impsum
    - b lorem impsum
    - c lorem impsum
`
	assert.Equal(t, strings.TrimSpace(expected), err2.Error())
}

func TestNestedError_Format_3(t *testing.T) {
	t.Parallel()

	sub2 := NewMultiError()
	sub2.Append(fmt.Errorf("lorem ipsum"))
	sub1 := NewNestedError(fmt.Errorf("reason"), sub2)
	err := NewNestedError(fmt.Errorf("error"), sub1)
	expected := `
error: reason: lorem ipsum
`
	assert.Equal(t, strings.TrimSpace(expected), err.Error())
}

func TestNestedError_Format_4(t *testing.T) {
	t.Parallel()

	sub2 := NewMultiError()
	sub2.Append(fmt.Errorf("lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor"))
	sub1 := NewNestedError(fmt.Errorf("reason"), sub2)
	err := NewNestedError(fmt.Errorf("error"), sub1)
	expected := `
error:
- reason:
  - lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor
`
	assert.Equal(t, strings.TrimSpace(expected), err.Error())
}

type customError struct {
	error
}

func (e customError) WriteError(w Writer, level int, _ StackTrace) {
	w.Write(fmt.Sprintf("this is a custom error message (%s)", e.error.Error()))
	w.WriteNewLine()

	w.WriteIndent(level)
	w.WriteBullet()
	w.Write("foo")
	w.WriteNewLine()

	w.WriteIndent(level)
	w.WriteBullet()
	w.Write("bar")
}

func TestCustom_WriteError(t *testing.T) {
	t.Parallel()

	sub2 := NewMultiError()
	sub2.Append(fmt.Errorf("lorem ipsum"))
	sub2.Append(customError{New("underlying error")})
	sub1 := NewNestedError(fmt.Errorf("reason"), sub2)
	err := NewNestedError(fmt.Errorf("error"), sub1)
	expected := `
error:
- reason:
  - lorem ipsum
  - this is a custom error message (underlying error)
    - foo
    - bar
`
	assert.Equal(t, strings.TrimSpace(expected), err.Error())
}

func TestCustomMultiLineError_1(t *testing.T) {
	t.Parallel()

	sub2 := NewMultiError()
	sub2.Append(fmt.Errorf("lorem ipsum"))
	sub2.Append(New("* A\n* B\n* C"))
	sub1 := NewNestedError(fmt.Errorf("reason"), sub2)
	err := NewNestedError(fmt.Errorf("error"), sub1)
	expected := `
error:
- reason:
  - lorem ipsum
  - * A
    * B
    * C
`
	assert.Equal(t, strings.TrimSpace(expected), err.Error())
}

func TestCustomMultiLineError_2(t *testing.T) {
	t.Parallel()

	sub := NewMultiError()
	sub.Append(fmt.Errorf("lorem ipsum"))
	sub.Append(New("* A\n* B\n* C"))
	err := NewNestedError(fmt.Errorf("error"), sub)
	expected := `
error:
- lorem ipsum
- * A
  * B
  * C
`
	assert.Equal(t, strings.TrimSpace(expected), err.Error())
}

func TestCustomMultiLineError_3(t *testing.T) {
	t.Parallel()
	err := NewNestedError(fmt.Errorf("error"), New("* A\n* B\n* C"))
	expected := `
error:
- * A
  * B
  * C
`
	assert.Equal(t, strings.TrimSpace(expected), err.Error())
}
