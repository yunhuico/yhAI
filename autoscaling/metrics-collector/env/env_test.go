package env

import (
	"github.com/bmizerany/assert"
	"os"
	"testing"
)

func TestToInt(t *testing.T) {
	os.Setenv("AGE", "30")

	age, err := Get("AGE").ToInt()
	assert.Equal(t, err, nil)
	assert.Equal(t, age, 30)

	os.Unsetenv("AGE")
}

func TestToBool(t *testing.T) {
	os.Setenv("MARRIED", "true")

	married, err := Get("MARRIED").ToBool()
	assert.Equal(t, err, nil)
	assert.Equal(t, married, true)

	os.Unsetenv("MARRIED")
}

func TestToFloat64(t *testing.T) {
	os.Setenv("PI", "3.14")

	pi, err := Get("PI").ToFloat64()
	assert.Equal(t, err, nil)
	assert.Equal(t, pi, 3.14)

	os.Unsetenv("PI")
}

func TestToString(t *testing.T) {
	os.Setenv("NAME", "tom")

	name, err := Get("NAME").ToString()
	assert.Equal(t, err, nil)
	assert.Equal(t, name, "tom")

	os.Unsetenv("NAME")
}

func TestToStringArr(t *testing.T) {
	os.Setenv("NAMES", "tom;bob;alice")

	names, err := Get("NAMES").ToStringArr(";")
	assert.Equal(t, err, nil)
	assert.Equal(t, names, []string{"tom", "bob", "alice"})

	os.Unsetenv("NAMES")
}

func TestString(t *testing.T) {
	os.Setenv("NAME", "tom")
	os.Setenv("AGE", "30")

	name := Get("NAME").String()
	assert.Equal(t, name, "NAME")

	age := Get("AGE").String()
	assert.Equal(t, age, "AGE")

	os.Unsetenv("NAME")
	os.Unsetenv("AGE")
}
