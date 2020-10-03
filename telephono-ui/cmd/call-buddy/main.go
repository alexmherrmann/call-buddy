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

var theEditor TCBEditor

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

// ViewState Which view is active
type ViewState int

const (
	// CMD_LINE The command line view is active
	CMD_LINE ViewState = iota
	// MTD_BODY The method body view is active (GET, PUT, HEAD)
	MTD_BODY
	// RQT_HEAD The request header view is active
	RQT_HEAD
	// RQT_BODY The request body view is active
	RQT_BODY
	// HIST_BODY The history body is active
	HIST_BODY
	// RSP_BODY The response body view is active
	RSP_BODY
	// NO_STATE No state is selected
	NO_STATE
)
const (
	TTL_LINE_VIEW = "title_view"
	// CMD_LINE_VIEW The command line view string
	CMD_LINE_VIEW = "command"
	// MTD_BODY_VIEW The method body view string
	MTD_BODY_VIEW = "method_body"
	// RQT_HEAD_VIEW The request header view string
	RQT_HEAD_VIEW = "request_head"
	// RQT_BODY_VIEW The request body view string
	RQT_BODY_VIEW = "request_body"
	// RSP_BODY_VIEW The response body view string
	RSP_BODY_VIEW = "response_body"
	// HIST_VIEW The history body view string
	HIST_VIEW = "history_body"
)

type ioHijacker struct {
	backupFile *os.File
	pipe       *os.File
	channel    chan string
}

var currView ViewState = NO_STATE //needs a better name

func die(msg string) {
	os.Stderr.WriteString(msg)
	os.Exit(1)
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
			setView(nGui, HIST_VIEW, HIST_BODY)
			return nil
		})
		gui.Update(setKeybindings)
		return nil
	})
}

func exitHistoryView(g *gocui.Gui) {
	g.Update(func(gui *gocui.Gui) error {
		gui.SetManagerFunc(layout)
		return nil
	})
	setView(g, CMD_LINE_VIEW, CMD_LINE)
	g.Update(setKeybindings)
}

func evalCmdLine(g *gocui.Gui) {
	var err error
	var response *http.Response

	// FIXME: Deal with errors!
	cmdLineView, _ := g.View(CMD_LINE_VIEW)
	rspBodyView, _ := g.View(RSP_BODY_VIEW)
	rqtBodyView, err := g.View(RQT_BODY_VIEW)
	rqtHeaderView, _ := g.View(RQT_HEAD_VIEW)
	histView, _ := g.View(HIST_VIEW)

	requestBodyBuffer := rqtBodyView.Buffer()

	log.Print(err)

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
		methodBodyView, _ := g.View(MTD_BODY_VIEW)
		updateMethodBodyView(methodBodyView, response.Request.URL.String(), response.Request.Method)

		requestHeaderView, _ := g.View(RQT_HEAD_VIEW)
		updateRequestHeaderView(requestHeaderView, response.Request.Header)

		// Update the history and history view
		globalTelephonoState.History.AddFinishedCall(response)
		updateHistoryView(histView)
	}
}

func setView(gui *gocui.Gui, name string, state ViewState) {
	currView = state
	currViewPtr, _ := gui.SetCurrentView(name)
	// FIXME Dylan: This should be done only if the editor is editable
	currViewPtr.Editor = &theEditor
	gui.SetViewOnTop(name)
	gui.Cursor = true
}

func switchNextView(g *gocui.Gui, v *gocui.View) error {
	// FIXME: Properly handle errors
	// Round robben switching between views
	log.Print(currView)
	switch currView {
	case CMD_LINE:
		// -> method body
		setView(g, MTD_BODY_VIEW, MTD_BODY)
	case MTD_BODY:
		// -> request headers
		setView(g, RQT_HEAD_VIEW, RQT_HEAD)
	case RQT_HEAD:
		// -> request body
		setView(g, RQT_BODY_VIEW, RQT_BODY)
	case RQT_BODY:
		// -> reqponse body
		setView(g, RSP_BODY_VIEW, RSP_BODY)
	case RSP_BODY:
		// -> command line
		setView(g, CMD_LINE_VIEW, CMD_LINE)
	case HIST_BODY:
		exitHistoryView(g)
	default:
		log.Panicf("Got to a unknown view! %d\n", currView)
	}
	return nil
}

func switchPrevView(g *gocui.Gui, v *gocui.View) error {
	// FIXME: Properly handle errors

	// Round robben switching between views
	switch currView {
	case CMD_LINE:
		// -> method body
		setView(g, RSP_BODY_VIEW, RSP_BODY)
	case MTD_BODY:
		// -> request headers
		setView(g, CMD_LINE_VIEW, CMD_LINE)
	case RQT_HEAD:
		// -> request body
		setView(g, MTD_BODY_VIEW, MTD_BODY)
	case RQT_BODY:
		// -> reqponse body
		setView(g, RQT_HEAD_VIEW, RQT_HEAD)
	case RSP_BODY:
		// -> command line
		setView(g, RQT_BODY_VIEW, RQT_BODY)
	case HIST_BODY:
		exitHistoryView(g)
	default:
		log.Panicf("Got to a unknown view! %d\n", currView)
	}
	return nil
}

func setHistView(g *gocui.Gui, v *gocui.View) error {
	// FIXME: POSSIBLY SWITCH BACK TO LAST SELECTED VIEW?
	if currView == HIST_BODY {
		setView(g, CMD_LINE_VIEW, CMD_LINE)
	} else {
		setView(g, HIST_VIEW, HIST_BODY)
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

//Setting the manager
func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	realMaxX, realMaxY := maxX-1, maxY-1
	verticalSplitX := 50         // Defines the vertical split down to the command line
	horizontalSplitY := maxY - 4 // Defines the horizontal command line split

	// Call-Buddy Title
	titleYStart := 0
	titleYEnd := titleYStart + 2
	if v, err := g.SetView(TTL_LINE_VIEW, 0, titleYStart, verticalSplitX, titleYEnd); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprint(v, "\u001b[32mTerminal "+"\u001b[29mCall "+"\u001b[29mBuddy")
	}

	// Response Body (e.g. html)
	if v, err := g.SetView(RSP_BODY_VIEW, verticalSplitX+1, titleYStart, realMaxX, horizontalSplitY); err != nil {
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
	if v, err := g.SetView(MTD_BODY_VIEW, 0, methodBodyYStart, verticalSplitX, methodBodyYEnd); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Method Body"
		updateMethodBodyView(v, "http://", "get")
	}

	// Request Headers (e.g. Content-type: text/json)
	requestHeadersYStart := methodBodyYEnd + 1
	requestHeadersYEnd := requestHeadersYStart + 6
	if v, err := g.SetView(RQT_HEAD_VIEW, 0, requestHeadersYStart, verticalSplitX, requestHeadersYEnd); err != nil {
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
	if v, err := g.SetView(RQT_BODY_VIEW, 0, requestBodyYStart, verticalSplitX, horizontalSplitY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Request Body"
		v.Wrap = true
		v.Autoscroll = true
		v.Editable = true
	}

	// Command Line (e.g. :get http://httpbin.org/get)
	if v, err := g.SetView(CMD_LINE_VIEW, 0, horizontalSplitY+1, realMaxX, realMaxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Wrap = false
		v.Editable = true
		v.Autoscroll = false
	}

	//setKeybindings(g)

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
	if v, err := g.SetView(TTL_LINE_VIEW, 0, titleYStart, verticalSplitX, titleYEnd); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprint(v, "\u001b[32mTerminal "+"\u001b[29mCall "+"\u001b[29mBuddy")
	}

	historyYEnd := titleYStart + 6
	//History View
	if v, err := g.SetView(HIST_VIEW, verticalSplitX+1, titleYStart, realMaxX, historyYEnd); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "History"
		v.Wrap = false
		v.Editable = false
		v.Autoscroll = false
	}

	// Response Body (e.g. html)
	if v, err := g.SetView(RSP_BODY_VIEW, verticalSplitX+1, historyYEnd+1, realMaxX, horizontalSplitY); err != nil {
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
	if v, err := g.SetView(MTD_BODY_VIEW, 0, methodBodyYStart, verticalSplitX, methodBodyYEnd); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Method Body"
		updateMethodBodyView(v, "http://", "get")
	}

	// Request Headers (e.g. Content-type: text/json)
	requestHeadersYStart := methodBodyYEnd + 1
	requestHeadersYEnd := requestHeadersYStart + 6
	if v, err := g.SetView(RQT_HEAD_VIEW, 0, requestHeadersYStart, verticalSplitX, requestHeadersYEnd); err != nil {
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
	if v, err := g.SetView(RQT_BODY_VIEW, 0, requestBodyYStart, verticalSplitX, horizontalSplitY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Request Body"
		v.Wrap = true
		v.Autoscroll = true
		v.Editable = true
	}

	// Command Line (e.g. :get http://httpbin.org/get)
	if v, err := g.SetView(CMD_LINE_VIEW, 0, horizontalSplitY+1, realMaxX, realMaxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Wrap = false
		v.Editable = true
		v.Autoscroll = false
	}

	//setKeybindings(g)

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
	if err := g.SetKeybinding(CMD_LINE_VIEW, gocui.KeyEnter, gocui.ModNone, cmdOnEnter); err != nil {
		log.Panicln(err)
	}

	// History View Keybindings
	if err := g.SetKeybinding(HIST_VIEW, gocui.KeyArrowDown, gocui.ModNone, histArrowDown); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding(HIST_VIEW, gocui.KeyArrowUp, gocui.ModNone, histArrowUp); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding(HIST_VIEW, gocui.KeyEnter, gocui.ModNone, histOnEnter); err != nil {
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

	theEditor = TCBEditor{g, false}
	g.Highlight = true
	g.Cursor = true
	g.SelFgColor = gocui.ColorGreen

	//Setting a manager, sets the view (defined as another function above)
	g.SetManagerFunc(layout)

	setKeybindings(g)

	currView = CMD_LINE
	g.SetCurrentView(CMD_LINE_VIEW)

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
	defer setView(g, CMD_LINE_VIEW, CMD_LINE)
	_, curY := v.Cursor()
	cmd, err := globalTelephonoState.History.GetNthCommand(curY)
	if err != nil {
		log.Fatalf("pos: %d\n", curY)
		return err
	}
	cmdView, _ := g.View(CMD_LINE_VIEW)
	cmdView.Clear()
	fmt.Fprint(cmdView, cmd)
	cmdView.SetCursor(len(cmd), 0)
	return nil
}
