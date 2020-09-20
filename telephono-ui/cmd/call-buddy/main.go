package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	t "github.com/call-buddy/call-buddy/telephono"
	"github.com/jroimartin/gocui"
)

var globalTelephonoState t.CallBuddyState

func init() {
	globalTelephonoState = t.InitNewState()
	globalTelephonoState.Collections = append(globalTelephonoState.Collections, t.CallBuddyCollection{
		Name: "The fake one FIXME up boys",
		RequestTemplates: []t.RequestTemplate{
			{
				Method: t.Get, Url: t.NewExpandable("https://{vars.Host}"),
				Headers:        t.NewHeadersTemplate(),
				ExpandableBody: t.NewExpandable("Hello World")}},
	})
}

func getCurrentRequestTemplate(state *t.CallBuddyState) t.RequestTemplate {
	// DEMO AH: Obviously this is NOT ok
	return globalTelephonoState.Collections[0].RequestTemplates[0]
}

var theEditor TCBEditor
var historyRowSelected int

type TCBEditor struct {
	// This is super dumb that I have to embed this, no way to pass it in...
	gui               *gocui.Gui
	expectKeyModifier bool
}

//What will comprise each history entry
type historyEntry struct {
	method     string
	url        string
	requestMap map[string]string
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

// This function hijacks stderr and makes the errors go to a buffer (rather
// the screen)
func hijackStderr() *ioHijacker {
	// Backup of the real stderr so we can restore it later
	stderr := os.Stderr
	rpipe, wpipe, _ := os.Pipe()
	os.Stderr = wpipe
	log.SetOutput(wpipe)

	hijackChannel := make(chan string)
	// Copy the stderr in a separate goroutine we don't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, rpipe)
		hijackChannel <- buf.String()
	}()

	return &ioHijacker{
		backupFile: stderr,
		pipe:       wpipe,
		channel:    hijackChannel,
	}
}

// Returns a string of any errors that were supposed to go to stderr
func unhijackStderr(hijacker *ioHijacker) string {
	hijacker.pipe.Close()
	// Restore the real stderr
	os.Stderr = hijacker.backupFile
	log.SetOutput(os.Stderr)
	return <-hijacker.channel
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

func call(args []string) (response *http.Response, err error) {
	// If we run into any issues here, rather than dying we can catch them with
	// the hijacker and print them out to the tui!
	hijack := hijackStderr()

	argLen := len(args)
	if argLen < 2 {
		return nil, errors.New("Invalid Usage: <call-type> <url> [content-type]")
	}

	methodType := strings.ToLower(args[0])
	url := args[1]
	contentType := "text/plain"
	if argLen > 2 {
		contentType = args[2]
	}

	switch methodType {
	case "get":
		if response, err = http.Get(url); err != nil {
			return nil, err
		}

	case "post":
		if response, err = http.Get(url); err != nil {
			return nil, err
		}

	case "head":
		if response, err = http.Get(url); err != nil {
			return nil, err
		}

	case "delete":
		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Add("Connection", "close")
		if response, err = http.Get(url); err != nil {
			return nil, err
		}

	case "put":
		req, err := http.NewRequest("PUT", url, os.Stdin)
		if err != nil {
			log.Print(err)
		}
		req.Header.Add("Connection", "close")
		req.Header.Add("Content-type", contentType)
		response, err = http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.New("Invalid <call-type> given")
	}
	stderr := unhijackStderr(hijack)
	if stderr != "" {
		return nil, errors.New("Unknown error: " + stderr)
	}
	return response, err
}

func evalCmdLine(g *gocui.Gui) {
	var err error
	var response *http.Response

	// FIXME: Deal with errors!
	cmdLineView, _ := g.View(CMD_LINE_VIEW)
	rspBodyView, _ := g.View(RSP_BODY_VIEW)
	histView, _ := g.View(HIST_VIEW)

	// Extract the command into an args list
	commandStr := cmdLineView.ViewBuffer()
	commandStr = strings.TrimSpace(commandStr)
	args := strings.Split(commandStr, " ")

	if strings.HasPrefix(commandStr, ">") {
		// Save response body to a file
		if len(args) < 2 {
			return
		}
		outfile := args[1]
		fd, _ := os.Create(outfile)
		defer fd.Close()
		fd.WriteString(rspBodyView.ViewBuffer())
	} else if commandStr == "history" {
		setView(g, HIST_VIEW, HIST_BODY)
	} else {
		if response, err = call(args); err != nil {
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

		requestBodyView, _ := g.View(RQT_BODY_VIEW)
		updateRequestBodyView(requestBodyView, "")

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
	default:
		log.Panicf("Got to a unknown view! %d\n", currView)
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

func updateRequestHeaderView(view *gocui.View, headers map[string][]string) {
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
	verticalSplitX := 27         // Defines the vertical split down to the command line
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
		v.Autoscroll = true
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
	return nil
}

//This is the function to QUIT out of the TUI
func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func main() {
	// Switching to stderr since we do some black magic with catching that to
	// prevent errors from hitting the tui (see hijackStderr)

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

	// Global Keybindings
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, switchNextView); err != nil {
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

	currView = CMD_LINE
	g.SetCurrentView(CMD_LINE_VIEW)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func histArrowUp(gui *gocui.Gui, view *gocui.View) error {
	curX, curY := view.Cursor()
	if curY > 0 {
		curY -= 1
		historyRowSelected--
	}
	view.SetCursor(curX, curY)
	return nil
}

func histArrowDown(gui *gocui.Gui, view *gocui.View) error {
	curX, curY := view.Cursor()
	view.SetCursor(curX, curY+1)
	historyRowSelected++
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
	cmd, err := globalTelephonoState.History.GetLastCommand()
	if err != nil {
		return err
	}
	cmdView, _ := g.View(CMD_LINE_VIEW)
	cmdView.Clear()
	fmt.Fprint(cmdView, cmd)
	cmdView.SetCursor(len(cmd), 0)
	return nil
}
