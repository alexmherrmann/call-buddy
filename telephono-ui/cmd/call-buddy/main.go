package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/jroimartin/gocui"
)

var theEditor TCBEditor

type TCBEditor struct {
	// This is super dumb that I have to embed this, no way to pass it in...
	gui               *gocui.Gui
	expectKeyModifier bool
}

func (editor *TCBEditor) Edit(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	if editor.expectKeyModifier && ch == 'Z' { // Tab for some some some some dumb dumb dumb reason
		prevSwitchView(editor.gui, v)
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

var rspBodyStr string = ""

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
)

type ioHijacker struct {
	backupFile *os.File
	pipe       *os.File
	channel    chan string
}

var currView ViewState = NO_STATE

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

func responseToString(resp *http.Response) string {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err.Error()
	}
	if len(resp.Header) > 1 {
		for key, value := range resp.Header {
			rspBodyStr += fmt.Sprintf("%s: %s\n", key, strings.Trim(strings.Join(value, " "), "[]"))
		}
		rspBodyStr += "\n"
	}
	rspBodyStr += string(body)
	return rspBodyStr
}

func call(args []string) string {
	var r string
	// If we run into any issues here, rather than dying we can catch them with
	// the hijacker and print them out to the tui!
	hijack := hijackStderr()

	argLen := len(args)
	if argLen < 2 {
		return "Usage: <call-type> <url> [content-type]"
	}

	callType := strings.ToLower(args[0])
	url := args[1]
	contentType := "text/plain"
	if argLen > 2 {
		contentType = args[2]
	}
	switch callType {
	case "get":
		resp, err := http.Get(url)
		if err != nil {
			log.Print(err)
			break
		}
		defer resp.Body.Close()
		r = responseToString(resp)

	case "post":
		resp, err := http.Post(url, contentType, os.Stdin)
		if err != nil {
			log.Print(err)
			break
		}
		defer resp.Body.Close()
		r = responseToString(resp)

	case "head":
		resp, err := http.Head(url)
		if err != nil {
			log.Print(err)
			break
		}
		defer resp.Body.Close()
		r = responseToString(resp)

	case "delete":
		req, err := http.NewRequest("DELETE", url, nil)
		if err != nil {
			log.Print(err)
			break
		}
		req.Header.Add("Connection", "close")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Print(err)
			break
		}
		defer resp.Body.Close()
		r = responseToString(resp)

	case "put":
		req, err := http.NewRequest("PUT", url, os.Stdin)
		if err != nil {
			log.Print(err)
		}
		req.Header.Add("Connection", "close")
		req.Header.Add("Content-type", contentType)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Print(err)
			break
		}
		defer resp.Body.Close()
		r = responseToString(resp)

	default:
		r = "Invalid <call-type> given.\n"
	}
	stderr := unhijackStderr(hijack)
	if stderr != "" {
		r = stderr
	}

	return r
}

func evalCmdLine(g *gocui.Gui) {
	// FIXME: Deal with errors!
	cmdLineView, _ := g.View(CMD_LINE_VIEW)
	rspBodyView, _ := g.View(RSP_BODY_VIEW)
	commandStr := cmdLineView.ViewBuffer()
	commandStr = strings.TrimSpace(commandStr)
	args := strings.Split(commandStr, " ")

	if strings.HasPrefix(commandStr, ">") {
		if len(args) < 2 {
			return
		}
		outfile := args[1]
		fd, _ := os.Create(outfile)
		defer fd.Close()
		fd.WriteString(rspBodyView.ViewBuffer())
	} else {
		rspBodyView.Clear()
		responseStr := call(args)
		fmt.Fprint(rspBodyView, responseStr)
	}
}

func switchView(g *gocui.Gui, v *gocui.View) error {
	// FIXME: Properly handle errors
	switchViewAttrFunc := func(gui *gocui.Gui, next string) {
		currViewPtr, _ := gui.SetCurrentView(next)
		// FIXME Dylan: This should be done only if the editor is editable
		//              Also, why are we instantiating this every time we switch?!
		currViewPtr.Editor = &theEditor
		g.SetViewOnTop(next)
		g.Cursor = true
	}
	// Round robben switching between views
	switch currView {
	case CMD_LINE:
		// -> method body
		currView = MTD_BODY
		switchViewAttrFunc(g, MTD_BODY_VIEW)
	case MTD_BODY:
		// -> request headers
		currView = RQT_HEAD
		switchViewAttrFunc(g, RQT_HEAD_VIEW)
	case RQT_HEAD:
		// -> request body
		currView = RQT_BODY
		switchViewAttrFunc(g, RQT_BODY_VIEW)
	case RQT_BODY:
		// -> reqponse body
		currView = RSP_BODY
		switchViewAttrFunc(g, RSP_BODY_VIEW)
	case RSP_BODY:
		// -> command line
		currView = CMD_LINE
		switchViewAttrFunc(g, CMD_LINE_VIEW)
	default:
		log.Panicf("Got to a unknown view! %d\n", currView)
	}
	return nil
}

func prevSwitchView(g *gocui.Gui, v *gocui.View) error {
	// FIXME: Properly handle errors
	switchViewAttrFunc := func(gui *gocui.Gui, next string) {
		currViewPtr, _ := gui.SetCurrentView(next)
		// FIXME Dylan: This should be done only if the editor is editable
		//              Also, why are we instantiating this every time we switch?!
		currViewPtr.Editor = &theEditor
		g.SetViewOnTop(next)
		g.Cursor = true
	}
	// Round robben switching between views
	switch currView {
	case CMD_LINE:
		// -> method body
		currView = RSP_BODY
		switchViewAttrFunc(g, RSP_BODY_VIEW)
	case MTD_BODY:
		// -> request headers
		currView = CMD_LINE
		switchViewAttrFunc(g, CMD_LINE_VIEW)
	case RQT_HEAD:
		// -> request body
		currView = MTD_BODY
		switchViewAttrFunc(g, MTD_BODY_VIEW)
	case RQT_BODY:
		// -> reqponse body
		currView = RQT_HEAD
		switchViewAttrFunc(g, RQT_HEAD_VIEW)
	case RSP_BODY:
		// -> command line
		currView = RQT_BODY
		switchViewAttrFunc(g, RQT_BODY_VIEW)
	default:
		log.Panicf("Got to a unknown view! %d\n", currView)
	}
	return nil
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
		fmt.Fprintln(v, "> https://google.com")
		fmt.Fprintln(v)
		fmt.Fprintln(v, "[ ]"+"GET")
		fmt.Fprintln(v, "[ ]"+"POST")
		fmt.Fprintln(v, "[ ]"+"HEAD")
		fmt.Fprintln(v, "[ ]"+"PUT")
		fmt.Fprintln(v, "[ ]"+"DELETE")
		fmt.Fprintln(v, "[ ]"+"OPTIONS")
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

//This function will update the response body (currently) by pressing a variable
func onEnter(g *gocui.Gui, v *gocui.View) error {
	// FIXME: Deal with other views
	if currView == CMD_LINE {
		evalCmdLine(g)
	}
	return nil
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

	//Setting keybindings
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyEnter, gocui.ModNone, onEnter); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, switchView); err != nil {
		log.Panicln(err)
	}

	currView = CMD_LINE
	g.SetCurrentView(CMD_LINE_VIEW)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}
