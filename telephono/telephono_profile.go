package telephono

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

type CallBuddyProfiles []*Profile

type Profile struct {
	Name  string
	Path  string
	State *CallBuddyState
}

func (profiles *CallBuddyProfiles) Init(dir string) (err error) {
	// First check for errors such as permissions or not existing...
	_, err = os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
		// Let's try and create the directory
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return
		}
	}

	profileGlob := filepath.Join(dir, "state-*.json")
	profileFilepaths, _ := filepath.Glob(profileGlob)

	var profileFilepathsModTime []time.Time
	var profileFilepathsCanRead []string
	for _, path := range profileFilepaths {
		stat, err := os.Stat(path)
		if err != nil {
			// FIXME DG: Hey maybe let's _not_ silently ignore bad profiles
			continue
		}
		profileFilepathsCanRead = append(profileFilepathsCanRead, path)
		profileFilepathsModTime = append(profileFilepathsModTime, stat.ModTime())
	}
	sort.SliceStable(profileFilepathsCanRead, func(i, j int) bool {
		return profileFilepathsModTime[i].Before(profileFilepathsModTime[j])
	})

	for _, path := range profileFilepathsCanRead {
		name, err := extractProfileName(path)
		if err != nil {
			// FIXME DG: Hey maybe let's _not_ silently ignore bad profile names
			continue
		}
		profile := Profile{
			Name:  name,
			Path:  path,
			State: &CallBuddyState{},
		}
		err = profile.State.Load(path)
		if err != nil {
			// FIXME DG: Hey maybe let's _not_ silently ignore the fact we can't load a profile?
			continue
		}
		*profiles = append(*profiles, &profile)
	}

	if len(*profiles) == 0 {
		_, err := profiles.New("default")
		if err != nil {
			// FIXME
			log.Fatalf("Failed to create default profile: %s\n", err)
		}
	}
	return
}

type InvalidProfileError struct {
	s string
}

func (e InvalidProfileError) Error() string {
	return e.s
}

func NewInvalidProfileError(name string) InvalidProfileError {
	return InvalidProfileError{"Not a valid profile name " + name}
}

func extractProfileName(path string) (name string, err error) {
	rest := path[len("state-"):]
	name = rest[:len(rest)-len(".json")+1]
	if !validProfileName(name) {
		return "", NewInvalidProfileError(name)
	}
	return
}

func validProfileName(name string) bool {
	var isValid = regexp.MustCompile(`^[a-z_A-Z0-9]+$`).MatchString
	return isValid(name)
}

func createProfilePath(name string) string {
	return "state-" + name + ".json"
}

func (profiles *CallBuddyProfiles) New(name string) (newProfile Profile, err error) {
	if !validProfileName(name) {
		err = NewInvalidProfileError(name)
		return
	}
	newState := InitNewState()
	newProfile = Profile{
		Name:  name,
		Path:  createProfilePath(name),
		State: &newState,
	}

	// FIXME DG: Collections is dead code. Sorry but not sorry it should be removed.
	newState.Collections = append(newState.Collections, CallBuddyCollection{
		Name: "Terminal Call-Buddy",
		RequestTemplates: []*RequestTemplate{
			{
				Method:  Get,
				Url:     "https://{vars.Host}",
				Headers: http.Header{},
				Body:    "Hello World"}},
	})
	*profiles = append(*profiles, &newProfile)
	return
}

func (profiles *CallBuddyProfiles) Rename(oldName string, newName string) (err error) {
	if !validProfileName(newName) {
		err = NewInvalidProfileError(newName)
		return
	}

	oldProfile, err := profiles.Get(oldName)
	if err != nil {
		return
	}

	oldPath := oldProfile.Path
	newPath := createProfilePath(newName)
	err = os.Rename(oldPath, newPath)
	return
}

func (profiles *CallBuddyProfiles) Get(name string) (profile Profile, err error) {
	for _, selected := range *profiles {
		if selected.Name == name {
			profile = *selected
			return
		}
	}
	err = NewInvalidProfileError(name)
	return
}

func (profiles *CallBuddyProfiles) CurrentState() *CallBuddyState {
	return (*profiles)[0].State
}

func (profiles *CallBuddyProfiles) Use(name string) (err error) {
	for i, selected := range *profiles {
		if selected.Name == name {
			// Move the selected one to the front. i.e.
			// 3, [1,2,3,4] -> [3, 1, 2, 4]
			newProfiles := []*Profile{}
			newProfiles = append(newProfiles, selected)
			for j, other := range *profiles {
				if i != j { // If not selected profile
					newProfiles = append(newProfiles, other)
				}
			}
			*profiles = newProfiles
			return
		}
	}
	return errors.New("No such profile " + name)
}

func (profiles *CallBuddyProfiles) List() []Profile {
	// FIXME:
	return []Profile{}
}

func (profiles *CallBuddyProfiles) Remove(name string) (err error) {
	for i, selected := range *profiles {
		if selected.Name == name {
			// Skip the selected one i.e.
			// 3, [1,2,3,4] -> [1, 2, 4]
			newProfiles := []*Profile{}
			for j, other := range *profiles {
				if i != j { // If not selected profile
					newProfiles = append(newProfiles, other)
				}
			}
			*profiles = newProfiles
			return
		}
	}
	return errors.New("No such profile " + name)
}

func (profiles *CallBuddyProfiles) Save(dir string) error {
	currentProfile := (*profiles)[0]
	return currentProfile.State.Save(currentProfile.Path)
}
