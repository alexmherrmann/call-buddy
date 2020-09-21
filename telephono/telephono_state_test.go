package telephono_test

import (
	"encoding/json"
	. "github.com/call-buddy/call-buddy/telephono"
	. "github.com/onsi/gomega"
	"strings"
	"testing"
)

func testSerializeState(t *testing.T) {
	// Environment 1 setup
	simpleContributor1 := NewSimpleContributor("Var")
	env1 := CallBuddyEnvironment{simpleContributor1}
	simpleContributor1.Set("Environment", "Env1")

	// Environment 2 setup
	simpleContributor2 := NewSimpleContributor("Var")
	env2 := CallBuddyEnvironment{simpleContributor2}
	simpleContributor2.Set("Environment", "Env2")

	// Set up a collection
	headers := NewHeadersTemplate()
	headers.Set("BigBad", "Wolf")
	coll1 := CallBuddyCollection{
		RequestTemplates: []RequestTemplate{
			{
				Method:         Post,
				Url:            NewExpandable("https://google.com/"),
				Headers:        headers,
				ExpandableBody: NewExpandable("{{Var.Environment}}"),
			},
		}}

	state := CallBuddyState{
		Collections:  []CallBuddyCollection{coll1},
		Environments: []CallBuddyEnvironment{env1, env2},
	}

	var marshaledBytes []byte
	var marshaledString string
	t.Run("Marshal State", func(t *testing.T) {
		if tempMarshaledBytes, marshalErr := json.MarshalIndent(state, "", "\t"); marshalErr == nil {
			marshaledBytes = tempMarshaledBytes
			marshaledString = string(tempMarshaledBytes)
			t.Log("Successfully marshaled the big fat state")
			t.Log(marshaledString)
		} else {
			t.Fatal("Didn't marshal correctly!: " + marshalErr.Error())
		}
	})

	t.Run("Check Marshaled string", func(t *testing.T) {
		RegisterFailHandler(func(message string, callerSkip ...int) {
			t.Fatal(message)
		})

		Expect(marshaledString).Should(ContainSubstring("google.com"))
		Expect(marshaledString).Should(ContainSubstring("Env1"))
		Expect(marshaledString).Should(ContainSubstring("Env2"))
		Expect(marshaledString).Should(ContainSubstring("BigBad\""))
		Expect(marshaledString).Should(ContainSubstring("Wolf\""))
		Expect(marshaledString).Should(ContainSubstring("{{Var.Environment}}"))
	})

	unmarshalled := CallBuddyState{}
	t.Run("Deserialize string", func(t *testing.T) {
		RegisterFailHandler(func(message string, callerSkip ...int) {
			t.Fatal(message)
		})
		Expect(json.Unmarshal(marshaledBytes, &unmarshalled)).ShouldNot(HaveOccurred())
	})

	t.Run("Check Deserialized structure", func(t *testing.T) {
		RegisterFailHandler(func(message string, callerSkip ...int) {
			t.Fatal(message)
		})

		Expect(unmarshalled.Collections).Should(HaveLen(1))
		Expect(unmarshalled.Environments).Should(HaveLen(2))
		Expect(env1).Should(BeElementOf(unmarshalled.Environments))
		Expect(env2).Should(BeElementOf(unmarshalled.Environments))
		Expect(coll1).Should(BeElementOf(unmarshalled.Collections))
	})
}

func TestExpander_Expand(t *testing.T) {
	under := Expander{}
	var alex = NewSimpleContributor("Alex")
	var cooper = NewSimpleContributor("Cooper")

	t.Run("Adding Contributors", func(s *testing.T) {
		under.AddContributor(EnvironmentContributor{})
		under.AddContributor(alex)
		under.AddContributor(cooper)
	})

	t.Run("Resolve environment variable $PATH", func(sub *testing.T) {
		//language=Mustache
		rendered, renderErr := under.Expand("{{Env.PATH}}")

		// Check if we errored...
		if renderErr != nil {
			sub.Error("Couldn't render environment path: " + renderErr.Error())
		}

		// Check if we rendered the PATH correctly
		if len(rendered) < 2 {
			sub.Error("Rendered string is too small (Should be at least two characters)")
		}

		sub.Log("Rendered $PATH too: " + rendered)
	})

	t.Run("Using Simple Contributor", func(sub *testing.T) {
		alex.Set("iscool", "yes")
		cooper.Set("iscool", "no")

		rendered, renderErr := under.Expand(`
{{#Alex}}
{{iscool}}
{{/Alex}}
{{#Cooper}}
{{iscool}}
{{/Cooper}}`[1:])

		// Check if we errored...
		if renderErr != nil {
			sub.Error("Couldn't render environment path: " + renderErr.Error())
		}

		if !(strings.Contains(rendered, "yes") && strings.Contains(rendered, "no")) {
			sub.Error("Should have contained both yes and no:\n", rendered)
		}

		sub.Log("Rendered:\n", rendered)
	})
}
