package telephono

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
)

/*CallBuddyState is the full shippable state of call buddy
environments, call templates, possibly history, variables, etc. are all in here
It can be shipped to remote servers to be run
*/
type CallBuddyState struct {
	// The big 3.

	// Our collections of request templates
	Collections []CallBuddyCollection

	// The environments we source our variables from
	Environment CallBuddyEnvironment

	// The history of calls made (just during this session?)
	History CallBuddyHistory
}

// Save Saves the given call buddy state as JSON to the specififed file.
func (state *CallBuddyState) Save(filepath string) error {
	stateFile, err := os.OpenFile(filepath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open state file %s: %s\n", filepath, err)
		return err
	}
	defer stateFile.Close()

	log.Printf("Encoding state...")
	enc := json.NewEncoder(stateFile)
	if err := enc.Encode(&state); err != nil {
		log.Println("Failed to encode state: %s\n", err)
		return err
	}
	return nil
}

// Load Loads the call buddy state in JSON from the specififed file.
func (state *CallBuddyState) Load(filepath string) error {
	stateFile, err := os.Open(filepath)
	if err != nil {
		log.Printf("Failed to open state file %s: %s\n", filepath, err)
		return err
	}
	defer stateFile.Close()

	dec := json.NewDecoder(stateFile)
	if err := dec.Decode(&state); err != nil {
		log.Printf("Failed to decode state: %s\n", err)
		return err
	}
	return nil
}

//InitNewState creates a correctly initialized CallBuddyState with some defaults
func InitNewState() CallBuddyState {
	state := CallBuddyState{
		Collections: []CallBuddyCollection{},
		Environment: CallBuddyEnvironment{
			OS:   Environment{"Var", map[string]string{}},
			User: Environment{"User", map[string]string{}},
		},
		History: CallBuddyHistory{},
	}
	state.Environment.OS.PopulateFromEnviron()
	return state
}

type CallBuddyCollection struct {
	Name string
	// TODO AH: Should this really be pointer?
	RequestTemplates []*RequestTemplate
}

type CallBuddyEnvironment struct {
	OS   Environment
	User Environment
}

func (env *CallBuddyEnvironment) UnmarshalJSON(b []byte) error {
	// See MarshalJSON
	var userEnv Environment
	if err := json.Unmarshal(b, &userEnv); err != nil {
		return err
	}
	env.User = userEnv
	return nil
}

func (env *CallBuddyEnvironment) MarshalJSON() ([]byte, error) {
	// We only care about the user environment, not the OS one
	return json.Marshal(env.User)
}

// Expands the string in all the environments
func (env *CallBuddyEnvironment) Expand(content string) string {
	return env.OS.Expand(env.User.Expand(content))
}

type CallBuddyInternalState struct {
	client      *http.Client
	callContext context.Context

	freeFunc context.CancelFunc
}

var globalState CallBuddyInternalState

func init() {
	// TODO AH: Make this timeout longer and configurable. Maybe have a check for number of received bytes on each call
	//timeoutContext, cancelFunc := context.@WithTimeout(context.Background(), time.Minute*3)
	// goddamn I love garbage collection
	globalState.callContext = context.Background()
	globalState.freeFunc = func() {}

	// create the client
	globalState.client = &http.Client{
		Transport:     http.DefaultTransport,
		CheckRedirect: nil,
		Jar:           nil,
		Timeout:       0,
	}
}
