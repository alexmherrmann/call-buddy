package telephono_test

import (
	tp "github.com/call-buddy/call-buddy/telephono"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestSimpleState(t *testing.T) {
	setUpServer()
	// Environment 1 setup
	simpleContributor1 := tp.NewSimpleContributor("simple1")
	env1 := tp.CallBuddyEnvironment{simpleContributor1}
	simpleContributor1.Set("Environment", "Env1")

	// Environment 2 setup
	simpleContributor2 := tp.NewSimpleContributor("test")
	env2 := tp.CallBuddyEnvironment{simpleContributor2}
	simpleContributor2.Set("Environment", "Env2")
	simpleContributor2.Set("Host", GlobalTestState.getPrefix())

	// Set up a collection
	headers := tp.NewHeadersTemplate()
	headers.Set("BigBad", "Wolf")

	coll1 := tp.CallBuddyCollection{
		RequestTemplates: []tp.RequestTemplate{
			{
				Method:         tp.Post,
				Url:            tp.NewExpandable("{{test.Host}}/postieboy"),
				Headers:        headers,
				ExpandableBody: tp.NewExpandable("Env: {{simple1.Environment}}"),
			},
		}}

	state := tp.CallBuddyState{
		Collections:  []tp.CallBuddyCollection{coll1},
		Environments: []tp.CallBuddyEnvironment{env1, env2},
		History:      tp.CallBuddyHistory{},
	}

	var client = &http.Client{}
	var resolved tp.CallResponse
	var err error

	if resolved, err = state.Collections[0].RequestTemplates[0].ExecuteWithClientAndExpander(client, state.GenerateExpander()); err != nil {
		t.Fatal(err.Error())
	}

	////
	// Just some
	////
	var all []byte

	if all, err = ioutil.ReadAll(resolved.Body); err != nil {
		t.Fatal(err.Error())
	}

	t.Log("Response body", string(all))
}
