package telephono

import (
	"os"
	"strings"

	"github.com/cbroglie/mustache"
)

type Environment struct {
	Name    string
	Mapping map[string]string
}

// Expands the environment variables in the given string and returns the result.
func (env *Environment) Expand(content string) (rendered string) {
	rendered = content

	var compiled *mustache.Template
	var err error
	if compiled, err = mustache.ParseString(content); err != nil {
		// FIXME DG: Don't just return the original content, make the incorrect
		//           template string in content blank
		return
	}
	contexts := make(map[string]interface{})
	contexts[env.Name] = env.Mapping
	if rendered, err = compiled.Render(contexts); err != nil {
		// FIXME DG: Same here
		return
	}
	return
}

// Set Sets the key=value pair in the given environment
func (env *Environment) Set(key, value string) {
	env.Mapping[key] = value
}

// PopulateFromEnviron Pulls in key=value pairs from the OS environment and
// populates the given environment
func (env *Environment) PopulateFromEnviron() {
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		env.Mapping[parts[0]] = parts[1]
	}
}
