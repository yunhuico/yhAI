package common

import "fmt"

type (
	// AdapterClass alias string
	AdapterClass string
)

// SpecClass join adapter class and spec name.
func (c AdapterClass) SpecClass(name string) string {
	return fmt.Sprintf("%s#%s", c, name)
}
