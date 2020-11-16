package telephono

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type CallBuddyProfiles []*Profile

type Profile struct {
	Name  string
	Path  string
	State *CallBuddyState
}

func (profiles *CallBuddyProfiles) Init(dir string) (ok bool, errs []error) {
	// First check for errors such as permissions or not existing...
	ok = true
	var err error

	_, err = os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			errs = append(errs, err)
			ok = false
			return
		}
		// Let's try and create the directory
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			errs = append(errs, err)
			ok = false
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
			errs = append(errs, err)
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
			errs = append(errs, err)
			continue
		}
		profile := Profile{
			Name:  name,
			Path:  path,
			State: &CallBuddyState{},
		}
		err = profile.State.Load(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		*profiles = append(*profiles, &profile)
	}

	if len(*profiles) == 0 {
		_, err := profiles.New(dir, "default")
		if err != nil {
			tempErr := fmt.Sprintf("Failed to create default profile: %s", err)
			errs = append(errs, errors.New(tempErr))
			ok = false
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
	return InvalidProfileError{"Not a valid profile name " + name + ". Can only contain a-z, 0-9 and underscores."}
}

func extractProfileName(path string) (name string, err error) {
	fileName := filepath.Base(path)
	rest := fileName[len("state-"):]
	name = rest[:len(rest)-len(".json")]
	if !validProfileName(name) {
		return "", NewInvalidProfileError(name)
	}
	return
}

func validProfileName(name string) bool {
	var isValid = regexp.MustCompile(`^[a-z_0-9]+$`).MatchString
	return isValid(name)
}

func createProfilePath(dir, name string) string {
	return filepath.Join(dir, "state-"+name+".json")
}

func (profiles *CallBuddyProfiles) New(dir, name string) (newProfile Profile, err error) {
	name = strings.ToLower(name)

	if !validProfileName(name) {
		err = NewInvalidProfileError(name)
		return
	}
	for _, profile := range *profiles {
		if name == profile.Name {
			return Profile{}, errors.New("Duplicate profile name " + name)
		}
	}

	newState := InitNewState()
	newProfile = Profile{
		Name:  name,
		Path:  createProfilePath(dir, name),
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
	profiles.Use(name)
	profiles.Save(dir)
	return
}

func (profiles *CallBuddyProfiles) Rename(oldName, newName string) (err error) {
	oldName = strings.ToLower(oldName)
	newName = strings.ToLower(newName)

	if !validProfileName(newName) {
		err = NewInvalidProfileError(newName)
		return
	}
	if oldName == newName {
		return errors.New("Duplicate profile name " + oldName)
	}

	oldProfile, err := profiles.Get(oldName)
	if err != nil {
		return
	}

	oldPath := oldProfile.Path
	newPath := createProfilePath(filepath.Dir(oldPath), newName)
	err = os.Rename(oldPath, newPath)
	return
}

func (profiles *CallBuddyProfiles) Get(name string) (profile Profile, err error) {
	name = strings.ToLower(name)
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

func (profiles *CallBuddyProfiles) Use(name string) (Profile, error) {
	name = strings.ToLower(name)

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
			return *selected, nil
		}
	}
	return Profile{}, errors.New("No such profile " + name)
}

func (profiles *CallBuddyProfiles) List() []Profile {
	var tempList []Profile
	for _, selected := range *profiles {
		tempList = append(tempList, *selected)
	}
	return tempList
}

func (profiles *CallBuddyProfiles) Remove(dir, name string) (err error) {
	name = strings.ToLower(name)

	var removed bool
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
			err = os.Remove(selected.Path)
			*profiles = newProfiles
			removed = true
		}
	}
	if removed {
		if len(*profiles) == 0 {
			_, err = profiles.New(dir, "default")
		}
	} else {
		err = errors.New("No such profile " + name)
	}
	return
}

func (profiles *CallBuddyProfiles) Save(dir string) error {
	currentProfile := (*profiles)[0]
	return currentProfile.State.Save(currentProfile.Path)
}
