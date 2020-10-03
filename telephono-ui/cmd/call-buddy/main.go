package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	t "github.com/call-buddy/call-buddy/telephono"
	"github.com/jroimartin/gocui"
)

var globalTelephonoState *t.CallBuddyState = nil
var userContributor t.SimpleContributor = t.NewSimpleContributor("User")

func init() {
	if f, err := os.OpenFile("tui.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0755); err != nil {
		panic(err.Error())
	} else {
		log.SetOutput(f)
	}

	log.Print("Starting up TCB")

	createdState := t.InitNewState()
	globalTelephonoState = &createdState
	globalTelephonoState.Collections = append(globalTelephonoState.Collections, t.CallBuddyCollection{
		Name: "The fake one FIXME up boys",
		RequestTemplates: []*t.RequestTemplate{
			{
				Method:         t.Get,
				Url:            t.NewExpandable("https://{vars.Host}"),
				Headers:        t.NewHeadersTemplate(),
				ExpandableBody: t.NewExpandable("Hello World")}},
	})
	globalTelephonoState.Environments = append(globalTelephonoState.Environments, t.CallBuddyEnvironment{userContributor})
}

func getCurrentRequestTemplate(state *t.CallBuddyState) *t.RequestTemplate {
	// DEMO AH: Obviously this is NOT ok
	return globalTelephonoState.Collections[0].RequestTemplates[0]
}

type TCBEditor struct {
	// This is super dumb that I have to embed this, no way to pass it in...
	gui               *gocui.Gui
	expectKeyModifier bool
}

func (editor *TCBEditor) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	if editor.expectKeyModifier && ch == 'Z' { // Tab for some some some some dumb dumb dumb reason
		switchPrevView(editor.gui, v)
		editor.expectKeyModifier = false
		return
	}

	switch {
	case ch != 0 && mod == 0:
		v.EditWrite(ch)
	case key == gocui.KeySpace:
		v.EditWrite(' ')
	case key == gocui.KeyBackspace || key == gocui.KeyBackspace2:
		v.EditDelete(true)
	case ch == '[' && mod == gocui.ModAlt:
		editor.expectKeyModifier = true
	}
	// FIXME Dylan: Call the parent gocui.Editor.Edit!
}

type viewProperties struct {
	name       string
	title      string
	wrap       bool
	editable   bool
	autoscroll bool
}

func (properties viewProperties) updateViewProperties(view *gocui.View) {
	view.Title = properties.title
	view.Wrap = properties.wrap
	view.Editable = properties.editable
	view.Autoscroll = properties.autoscroll
}

func (properties viewProperties) setToCurrentView(gui *gocui.Gui) {
	currView, _ := gui.SetCurrentView(properties.name)
	properties.updateViewProperties(currView)
	gui.SetViewOnTop(properties.name)
	gui.Cursor = true
}

const (
	TITLE_VIEW           = "title"
	METHOD_BODY_VIEW     = "method_body"
	REQUEST_HEADERS_VIEW = "request_headers"
	REQUEST_BODY_VIEW    = "request_body"
	COMMAND_LINE_VIEW    = "command_line"
	HISTORY_VIEW         = "history"
	RESPONSE_BODY_VIEW   = "response_body"
)

var views map[string]viewProperties = map[string]viewProperties{
	TITLE_VIEW: {
		name:       TITLE_VIEW,
		title:      "",
		wrap:       false,
		editable:   false,
		autoscroll: false,
	},
	METHOD_BODY_VIEW: {
		name:       METHOD_BODY_VIEW,
		title:      "Method Body",
		wrap:       false,
		editable:   false,
		autoscroll: false,
	},
	REQUEST_HEADERS_VIEW: {
		name:       REQUEST_HEADERS_VIEW,
		title:      "Request Headers",
		wrap:       true,
		editable:   true,
		autoscroll: true,
	},
	REQUEST_BODY_VIEW: {
		name:       REQUEST_BODY_VIEW,
		title:      "Request Body",
		wrap:       true,
		editable:   true,
		autoscroll: true,
	},
	COMMAND_LINE_VIEW: {
		name:       COMMAND_LINE_VIEW,
		title:      "",
		wrap:       false,
		editable:   true,
		autoscroll: false,
	},
	HISTORY_VIEW: {
		name:       HISTORY_VIEW,
		title:      "History",
		wrap:       false,
		editable:   false,
		autoscroll: false,
	},
	RESPONSE_BODY_VIEW: {
		name:       RESPONSE_BODY_VIEW,
		title:      "Response Body",
		wrap:       true,
		editable:   true,
		autoscroll: true,
	},
}

// saveResponseToFile Save response body to a file
func saveResponseToFile(contents, filepath string) {
	fd, _ := os.Create(filepath)
	defer fd.Close()
	fd.WriteString(contents)
}

func addUserEnvironmentVariable(kv string) {
	var splatted []string
	if splatted = strings.SplitN(kv, "=", 2); len(splatted) != 2 {
		return
	}
	userContributor.Set(splatted[0], splatted[1])
}

// appendHeaderToView Adds a key=value header to the header view
func appendHeaderToView(kv string, requestHeaderView *gocui.View) {
	var splatted []string
	if splatted = strings.SplitN(kv, "=", 2); len(splatted) != 2 {
		return
	}

	// We want to set a header
	headers := getCurrentRequestTemplate(globalTelephonoState).Headers
	headers.Set(splatted[0], splatted[1])

	var expandedHttpHeaders http.Header
	var expansionErr []error
	if expandedHttpHeaders, expansionErr = headers.ExpandAllAsHeader(globalTelephonoState.GenerateExpander()); len(expansionErr) != 0 {
		for _, err := range expansionErr {
			// TODO AH: Use fancy new wrapped errors using Printf
			log.Print("Had an error expanding a header: ", err.Error())
		}
	}
	updateRequestHeaderView(requestHeaderView, expandedHttpHeaders)
}

// responseToString Creates a "report" of the response
func responseToString(resp *http.Response) string {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err.Error()
	}
	var rspBodyStr string
	if len(resp.Header) > 1 {
		for key, value := range resp.Header {
			rspBodyStr += fmt.Sprintf("%s: %s\n", key, strings.Trim(strings.Join(value, " "), "[]"))
		}
		rspBodyStr += "\n"
	}
	rspBodyStr += string(body)
	return rspBodyStr
}

// TODO AH: args should probably get broken out into real parameters
func call(methodType, url, body string, headers http.Header) (response *http.Response, err error) {
	methodType = strings.ToLower(methodType)
	// TODO AH: Clean up documentation and other places
	//contentType := "text/plain"

	theTemplate := getCurrentRequestTemplate(globalTelephonoState)
	theTemplate.Method = t.HttpMethod(strings.ToUpper(methodType))
	theTemplate.ExpandableBody = t.NewExpandable(body)
	theTemplate.Url = t.NewExpandable(url)
	return theTemplate.ExecuteWithClientAndExpander(http.DefaultClient, globalTelephonoState.GenerateExpander())
}

func enterHistoryView(g *gocui.Gui) {
	g.Update(func(gui *gocui.Gui) error {
		gui.SetManagerFunc(histLayout)
		gui.Update(func(nGui *gocui.Gui) error {
			views[HISTORY_VIEW].setToCurrentView(g)
			return nil
		})
		gui.Update(setKeybindings)
		gui.Update(func(gui *gocui.Gui) error {
			histView, _ := gui.View(HISTORY_VIEW)
			updateHistoryView(histView)
			return nil
		})
		return nil
	})
}

func exitHistoryView(g *gocui.Gui) {
	g.Update(func(gui *gocui.Gui) error {
		gui.SetManagerFunc(layout)
		return nil
	})
	views[COMMAND_LINE_VIEW].setToCurrentView(g)
	g.Update(setKeybindings)
}

func evalCmdLine(g *gocui.Gui) {
	var err error
	var response *http.Response

	// FIXME: Deal with errors!
	cmdLineView, _ := g.View(COMMAND_LINE_VIEW)
	rspBodyView, _ := g.View(RESPONSE_BODY_VIEW)
	rqtBodyView, _ := g.View(REQUEST_BODY_VIEW)
	rqtHeaderView, _ := g.View(REQUEST_HEADERS_VIEW)

	requestBodyBuffer := rqtBodyView.Buffer()

	// Extract the command into an args list
	rawCommand := strings.TrimSpace(cmdLineView.Buffer())
	argv := strings.Split(rawCommand, " ")
	command := argv[0]

	switch {
	case strings.HasPrefix(command, ">"):
		if len(argv) < 2 {
			break
		}
		saveResponseToFile(rspBodyView.Buffer(), argv[1])

	case command == "env":
		if len(argv) < 2 {
			break
		}
		for _, kv := range argv[1:] {
			addUserEnvironmentVariable(kv)
		}

	case command == "history":
		enterHistoryView(g)

	case command == "header":
		if len(argv) < 2 {
			break
		}
		appendHeaderToView(argv[1], rqtHeaderView)

	default:
		// FIXME DG: Split out these calls into individual commands
		// Assume is a call
		if len(argv) < 2 {
			updateResponseBodyView(rspBodyView, "Invalid Usage: <call-type> <url>")
			break
		}
		url := argv[1]
		headers := getCurrentRequestTemplate(globalTelephonoState).Headers

		var expandedHttpHeaders http.Header
		var expansionErr []error
		if expandedHttpHeaders, expansionErr = headers.ExpandAllAsHeader(globalTelephonoState.GenerateExpander()); len(expansionErr) != 0 {
			for _, err := range expansionErr {
				// TODO AH: Use fancy new wrapped errors using Printf
				log.Print("Had an error expanding a header: ", err.Error())
			}
		}

		if response, err = call(command, url, requestBodyBuffer, expandedHttpHeaders); err != nil {
			// Print error out in place of response body
			updateResponseBodyView(rspBodyView, err.Error())
			return
		}
		defer response.Body.Close()

		// Print out new response
		responseBody := responseToString(response)
		updateResponseBodyView(rspBodyView, responseBody)

		// Update the request views
		methodBodyView, _ := g.View(METHOD_BODY_VIEW)
		updateMethodBodyView(methodBodyView, response.Request.URL.String(), response.Request.Method)

		requestHeaderView, _ := g.View(REQUEST_HEADERS_VIEW)
		updateRequestHeaderView(requestHeaderView, response.Request.Header)

		// Update the history
		globalTelephonoState.History.AddFinishedCall(response)
	}
}

func switchNextView(g *gocui.Gui, currView *gocui.View) error {
	// FIXME: Properly handle errors
	// Round robben switching between views
	if currView == nil {
		views[COMMAND_LINE_VIEW].setToCurrentView(g)
		return nil
	}

	switch currView.Name() {
	case COMMAND_LINE_VIEW:
		// -> method body
		views[METHOD_BODY_VIEW].setToCurrentView(g)
	case METHOD_BODY_VIEW:
		// -> request headers
		views[REQUEST_HEADERS_VIEW].setToCurrentView(g)
	case REQUEST_HEADERS_VIEW:
		// -> request body
		views[REQUEST_BODY_VIEW].setToCurrentView(g)
	case REQUEST_BODY_VIEW:
		// -> reqponse body
		views[RESPONSE_BODY_VIEW].setToCurrentView(g)
	case RESPONSE_BODY_VIEW:
		// -> command line
		views[COMMAND_LINE_VIEW].setToCurrentView(g)
	case HISTORY_VIEW:
		exitHistoryView(g)
	default:
		log.Panicf("Got to a unknown view! %s\n", currView.Name())
	}
	return nil
}

func switchPrevView(g *gocui.Gui, currView *gocui.View) error {
	// FIXME: Properly handle errors
	// Round robben switching between views
	if currView == nil {
		views[COMMAND_LINE_VIEW].setToCurrentView(g)
		return nil
	}

	switch currView.Name() {
	case COMMAND_LINE_VIEW:
		// -> response body
		views[RESPONSE_BODY_VIEW].setToCurrentView(g)
	case METHOD_BODY_VIEW:
		// -> command line
		views[COMMAND_LINE_VIEW].setToCurrentView(g)
	case REQUEST_HEADERS_VIEW:
		// -> method body
		views[METHOD_BODY_VIEW].setToCurrentView(g)
	case REQUEST_BODY_VIEW:
		// -> request header
		views[REQUEST_HEADERS_VIEW].setToCurrentView(g)
	case RESPONSE_BODY_VIEW:
		// -> request body
		views[REQUEST_BODY_VIEW].setToCurrentView(g)
	case HISTORY_VIEW:
		exitHistoryView(g)
	default:
		log.Panicf("Got to a unknown view! %s\n", currView.Name())
	}
	return nil
}

func setHistView(g *gocui.Gui, v *gocui.View) error {
	// FIXME CP: POSSIBLY SWITCH BACK TO LAST SELECTED VIEW?
	if v.Name() == HISTORY_VIEW {
		views[COMMAND_LINE_VIEW].setToCurrentView(g)
	} else {
		views[HISTORY_VIEW].setToCurrentView(g)
	}
	return nil
}

func updateMethodBodyView(view *gocui.View, url, method string) {
	view.Clear()

	fmt.Fprintln(view, url)
	fmt.Fprintln(view)
	allMethods := []string{"get", "post", "head", "put", "delete", "options"}
	for _, possibleMethod := range allMethods {
		x := " "
		if possibleMethod == strings.ToLower(method) {
			x = "x"
		}
		fmt.Fprintf(view, "[%s] %s\n", x, strings.ToUpper(possibleMethod))
	}
}

func updateRequestHeaderView(view *gocui.View, headers http.Header) {
	view.Clear()

	// For some reason golang stores the value as a list... I dunno why
	for key, values := range headers {
		fmt.Fprintf(view, "%s: %s\n", key, strings.Join(values, " "))
	}
}

func updateRequestBodyView(view *gocui.View, body string) {
	view.Clear()
	fmt.Fprint(view, body)
}

func updateResponseBodyView(view *gocui.View, body string) {
	view.Clear()
	fmt.Fprint(view, body)
}

func updateHistoryView(view *gocui.View) {
	view.Clear()
	histFormat := globalTelephonoState.History.GetSimpleWholeHistoryReport()
	fmt.Fprint(view, histFormat)
}

func updateTitleView(view *gocui.View, title string) {
	view.Clear()
	fmt.Fprint(view, title)
}

//Setting the manager
func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	realMaxX, realMaxY := maxX-1, maxY-1
	verticalSplitX := 50         // Defines the vertical split down to the command line
	horizontalSplitY := maxY - 4 // Defines the horizontal command line split

	// Call-Buddy Title
	titleYStart := 0
	titleYEnd := titleYStart + 2
	if v, err := g.SetView(TITLE_VIEW, 0, titleYStart, verticalSplitX, titleYEnd); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editor = &TCBEditor{g, false}
		views[TITLE_VIEW].updateViewProperties(v)
		updateTitleView(v, "\u001b[32mTerminal "+"\u001b[29mCall "+"\u001b[29mBuddy")
	}

	// Response Body (e.g. html)
	if v, err := g.SetView(RESPONSE_BODY_VIEW, verticalSplitX+1, titleYStart, realMaxX, horizontalSplitY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Response Body"
		v.Wrap = true
		v.Autoscroll = true
		v.Editable = true
	}

	// Method Body (e.g. GET, PUT, HEAD...)
	methodBodyYStart := titleYEnd + 1
	methodBodyYEnd := methodBodyYStart + 10
	if v, err := g.SetView(METHOD_BODY_VIEW, 0, methodBodyYStart, verticalSplitX, methodBodyYEnd); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Method Body"
		updateMethodBodyView(v, "http://", "get")
	}

	// Request Headers (e.g. Content-type: text/json)
	requestHeadersYStart := methodBodyYEnd + 1
	requestHeadersYEnd := requestHeadersYStart + 6
	if v, err := g.SetView(REQUEST_HEADERS_VIEW, 0, requestHeadersYStart, verticalSplitX, requestHeadersYEnd); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Request Headers"
		v.Wrap = true
		v.Autoscroll = true
		v.Editable = true
	}

	// Request Body (e.g. json: {})
	requestBodyYStart := requestHeadersYEnd + 1
	if v, err := g.SetView(REQUEST_BODY_VIEW, 0, requestBodyYStart, verticalSplitX, horizontalSplitY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Request Body"
		v.Wrap = true
		v.Autoscroll = true
		v.Editable = true
	}

	// Command Line (e.g. :get http://httpbin.org/get)
	if v, err := g.SetView(COMMAND_LINE_VIEW, 0, horizontalSplitY+1, realMaxX, realMaxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Wrap = false
		v.Editable = true
		v.Autoscroll = false
	}

	return nil
}

//Possible layout for when history is popped up.
func histLayout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	realMaxX, realMaxY := maxX-1, maxY-1
	verticalSplitX := 50         // Defines the vertical split down to the command line
	horizontalSplitY := maxY - 4 // Defines the horizontal command line split

	// Call-Buddy Title
	titleYStart := 0
	titleYEnd := titleYStart + 2
	if v, err := g.SetView(TITLE_VIEW, 0, titleYStart, verticalSplitX, titleYEnd); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprint(v, "\u001b[32mTerminal "+"\u001b[29mCall "+"\u001b[29mBuddy")
	}

	historyYEnd := titleYStart + 6
	// History View
	if v, err := g.SetView(HISTORY_VIEW, verticalSplitX+1, titleYStart, realMaxX, historyYEnd); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editor = &TCBEditor{g, false}
		views[HISTORY_VIEW].updateViewProperties(v)
	}

	// Response Body (e.g. html)
	if v, err := g.SetView(RESPONSE_BODY_VIEW, verticalSplitX+1, historyYEnd+1, realMaxX, horizontalSplitY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editor = &TCBEditor{g, false}
		views[RESPONSE_BODY_VIEW].updateViewProperties(v)
	}

	// Method Body (e.g. GET, PUT, HEAD...)
	methodBodyYStart := titleYEnd + 1
	methodBodyYEnd := methodBodyYStart + 10
	if v, err := g.SetView(METHOD_BODY_VIEW, 0, methodBodyYStart, verticalSplitX, methodBodyYEnd); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editor = &TCBEditor{g, false}
		views[METHOD_BODY_VIEW].updateViewProperties(v)
		updateMethodBodyView(v, "http://", "get")
	}

	// Request Headers (e.g. Content-type: text/json)
	requestHeadersYStart := methodBodyYEnd + 1
	requestHeadersYEnd := requestHeadersYStart + 6
	if v, err := g.SetView(REQUEST_HEADERS_VIEW, 0, requestHeadersYStart, verticalSplitX, requestHeadersYEnd); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editor = &TCBEditor{g, false}
		views[REQUEST_HEADERS_VIEW].updateViewProperties(v)
	}

	// Request Body (e.g. json: {})
	requestBodyYStart := requestHeadersYEnd + 1
	if v, err := g.SetView(REQUEST_BODY_VIEW, 0, requestBodyYStart, verticalSplitX, horizontalSplitY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editor = &TCBEditor{g, false}
		views[REQUEST_BODY_VIEW].updateViewProperties(v)
	}

	// Command Line (e.g. :get http://httpbin.org/get)
	if v, err := g.SetView(COMMAND_LINE_VIEW, 0, horizontalSplitY+1, realMaxX, realMaxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editor = &TCBEditor{g, false}
		views[COMMAND_LINE_VIEW].updateViewProperties(v)
	}

	return nil
}

//This is the function to QUIT out of the TUI
func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func setKeybindings(g *gocui.Gui) error {
	// Global Keybindings
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, switchNextView); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlY, gocui.ModNone, setHistView); err != nil {
		log.Panicln(err)
	}

	// View-Specific Keybindings:

	// Command View Keybindings
	if err := g.SetKeybinding(COMMAND_LINE_VIEW, gocui.KeyEnter, gocui.ModNone, cmdOnEnter); err != nil {
		log.Panicln(err)
	}

	// History View Keybindings
	if err := g.SetKeybinding(HISTORY_VIEW, gocui.KeyArrowDown, gocui.ModNone, histArrowDown); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding(HISTORY_VIEW, gocui.KeyArrowUp, gocui.ModNone, histArrowUp); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding(HISTORY_VIEW, gocui.KeyEnter, gocui.ModNone, histOnEnter); err != nil {
		log.Panicln(err)
	}
	return nil

}

func main() {
	//Setting up a new TUI
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Highlight = true
	g.Cursor = true
	g.SelFgColor = gocui.ColorGreen

	//Setting a manager, sets the view (defined as another function above)
	g.SetManagerFunc(layout)

	setKeybindings(g)

	g.SetCurrentView(COMMAND_LINE_VIEW)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func histArrowUp(gui *gocui.Gui, view *gocui.View) error {
	curX, curY := view.Cursor()
	if curY > 0 && globalTelephonoState.History.Size() > 0 {
		curY -= 1
	}
	view.SetCursor(curX, curY)
	return nil
}

func histArrowDown(gui *gocui.Gui, view *gocui.View) error {
	curX, curY := view.Cursor()
	if curY < globalTelephonoState.History.Size()-1 {
		curY += 1
	}
	view.SetCursor(curX, curY)
	return nil
}

// cmdOnEnter Evaluates the command line
func cmdOnEnter(g *gocui.Gui, v *gocui.View) error {
	evalCmdLine(g)
	return nil
}

// histOnEnter Populates the history with the currently selected history item
func histOnEnter(g *gocui.Gui, v *gocui.View) error {
	defer views[COMMAND_LINE_VIEW].setToCurrentView(g)

	_, curY := v.Cursor()
	cmd, err := globalTelephonoState.History.GetNthCommand(curY)
	if err != nil {
		log.Fatalf("pos: %d\n", curY)
		return err
	}
	cmdView, _ := g.View(COMMAND_LINE_VIEW)
	cmdView.Clear()
	fmt.Fprint(cmdView, cmd)
	cmdView.SetCursor(len(cmd), 0)
	return nil
}
