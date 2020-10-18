module github.com/call-buddy/call-buddy/telephono/ui

go 1.14

replace github.com/call-buddy/call-buddy/telephono => ../telephono

//replace github.com/call-buddy/gocui => /Users/dylngg/Local/go/src/github.com/call-buddy/gocui

require (
	github.com/call-buddy/call-buddy/telephono v0.0.2
	github.com/call-buddy/gocui v0.4.1-beta
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/nsf/termbox-go v0.0.0-20200418040025-38ba6e5628f1 // indirect
)
