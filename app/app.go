// Package app is the main UI that used rivo/tview
package app

import (
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"

	"github.com/joecroninallen/logsync/filechunk"
)

// fileView represents a TextView (aka text box) in the
// the UI that is used to view into one of the files.
// Each fileView is assigned one log file,
// and it is responsible for showing the appropriate section of the
// file to the user.
type fileView struct {
	*tview.TextView                      // The TextView is the text box widget from rivo/tview
	file            *os.File             // The file that this fileView is responsible for viewing
	headChunk       *filechunk.FileChunk // The headChunk is stored to allow for easy jumping to head of file
	tailChunk       *filechunk.FileChunk // The tailChunk is stored to allow for easy jumping to tail of file
	currChunk       *filechunk.FileChunk // The currentChunk is the current chunk being viewed on the screen
	index           int                  // This is the index of this fileView out of the list of all files being viewed
	lastScrollTime  int64                // Stores the last time this file was scrolled. Used to break ties when the timestamps are the same
	allFileViews    []fileView           // Stores a pointer to all the other fileViews including our own
}

// AdvanceNextFileViewForward figures out which fileview is next
// in line and advances its current chunk.
// This is based on who has the most recent timestamp and it is called
// when navigating forward. This advances one step, so we choose one file
// to advance and advance it by one timestamped log line
func AdvanceNextFileViewForward(fileViews []fileView) int {
	var currMinTime int64 = math.MaxInt64
	var currMinLastScrollTime int64 = math.MaxInt64
	var currMinChunk *filechunk.FileChunk
	var minIndex int = -1
	for i := range fileViews {
		nextChunk := fileViews[i].currChunk.GetNextFileChunk()
		if nextChunk == nil {
			continue
		}
		var isNewMin bool = false
		if nextChunk.LineTimeStamp < currMinTime {
			isNewMin = true
		} else if nextChunk.LineTimeStamp == currMinTime {
			if fileViews[i].lastScrollTime < currMinLastScrollTime {
				isNewMin = true
			}
		}

		if isNewMin {
			currMinTime = nextChunk.LineTimeStamp
			minIndex = i
			currMinChunk = nextChunk
			currMinLastScrollTime = fileViews[i].lastScrollTime
		}
	}
	if minIndex > -1 {
		fileViews[minIndex].currChunk = currMinChunk
		fileViews[minIndex].lastScrollTime = time.Now().Unix()
	}
	return minIndex
}

// AdvancePrevFileViewBackward figures out which fileview is the closest previous log
// line and advances its current chunk backward.
// This is based on who has the latest previous time and it is called
// when navigating backward. This advances one step backward, so we choose one file
// to advance backward and advance back it by one timestamped log line
func AdvancePrevFileViewBackward(fileViews []fileView) int {
	var currMaxTime int64 = -2
	var currMinLastScrollTime int64 = math.MaxInt64
	var currMaxChunk *filechunk.FileChunk
	var maxIndex int = -1
	for i := range fileViews {
		prevChunk := fileViews[i].currChunk.GetPrevFileChunk()
		if prevChunk == nil {
			continue
		}
		var isNewMax bool = false
		if prevChunk.LineTimeStamp > currMaxTime {
			isNewMax = true
		} else if prevChunk.LineTimeStamp == currMaxTime {
			if fileViews[i].lastScrollTime < currMinLastScrollTime {
				isNewMax = true
			}
		}

		if isNewMax {
			currMaxTime = prevChunk.LineTimeStamp
			maxIndex = i
			currMaxChunk = prevChunk
			currMinLastScrollTime = fileViews[i].lastScrollTime
		}
	}
	if maxIndex > -1 {
		fileViews[maxIndex].currChunk = currMaxChunk
		fileViews[maxIndex].lastScrollTime = time.Now().Unix()
	}
	return maxIndex
}

// MoveAllToBeginning moves all log lines to their respective head log line
// at the beginning of the file
func MoveAllToBeginning(fileViews []fileView) {
	for i := range fileViews {
		fileViews[i].currChunk = fileViews[i].headChunk
		fileViews[i].SetDisplayText()
	}
}

// MoveAllToEnd moves all log lines to their respective tail log line
// at the end of the file.
func MoveAllToEnd(fileViews []fileView) {
	for i := range fileViews {
		fileViews[i].currChunk = fileViews[i].tailChunk
		fileViews[i].SetDisplayText()
	}
}

// MoveAllToTime finds the closest log line to the searchTime and
// moves all the log lines such that they are at the log just before
// the searchTime. This allows us to search based on time and have all
// the logs jump to that spot.
func MoveAllToTime(fileViews []fileView, searchTime int64) {
	for i := range fileViews {
		closestChunk := fileViews[i].currChunk.GetFileChunkClosestToTime(searchTime)
		if closestChunk != nil {
			fileViews[i].currChunk = closestChunk
			fileViews[i].SetDisplayText()
		}
	}
}

// This updates the display for the fileView based on the currentChunk.
// For now, we show the currentChunk highlighted and then the previous and
// next chunks for context.
func (fv *fileView) SetDisplayText() {
	currStr := "[\"curr\"]" + string(fv.currChunk.FileChunkBytes) + "[\"\"]"
	nextChunk := fv.currChunk.GetNextFileChunk()
	prevChunk := fv.currChunk.GetPrevFileChunk()

	var prevStr string
	var nextStr string

	if nextChunk != nil {
		nextStr = string(nextChunk.FileChunkBytes)
	}

	if prevChunk != nil {
		prevStr = string(prevChunk.FileChunkBytes)
	}

	fv.Highlight("curr")
	fv.ScrollToHighlight()
	fv.SetText(prevStr + currStr + nextStr)
}

// LoadInputHandler sets the key commands for the file view.
// If any of the file views has the focus, then TAB is shortcut
// to step one forward and BACKTAB steps one backward.
func (fv *fileView) LoadInputHandler() {
	fv.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyTab {
			nextIndex := AdvanceNextFileViewForward(fv.allFileViews)
			if nextIndex > -1 {
				fv.allFileViews[nextIndex].SetDisplayText()
			}
		} else if key == tcell.KeyBacktab {
			prevIndex := AdvancePrevFileViewBackward(fv.allFileViews)
			if prevIndex > -1 {
				fv.allFileViews[prevIndex].SetDisplayText()
			}
		}
	})
}

// newFileView creates a new FileView for the given file and and logFilename
// and it also tells us what index we are in the list of all FileViews.
func newFileView(file *os.File, logFilename string, index int) *fileView {
	head, tail := filechunk.NewFileChunk(file)

	dataStr := string(tail.FileChunkBytes)
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetText(dataStr)

	textView.SetBorder(true)
	textView.SetTitle(logFilename)

	return &fileView{
		TextView:  textView,
		file:      file,
		headChunk: head,
		tailChunk: tail,
		currChunk: head,
		index:     index,
	}
}

// currCommand stores the current comment entered in the command edit
// box at the bottom
var currCommand string

// RunLogSync is the main tview function that builds the UI
// Right now, it consists of one text box for each file being viewed,
// and they are stacked on top of each other.
// Below that is an edit box for entering commands.
//
// Here are the valid commands:
// "head" jumps all files to the beginning
// "tail" jumps all files to the end
// Any positive number jumps that many steps, where each step chooses the next
// log line based on time stamp and advancing that file foward one.
// Any negative number goes back that many steps.
// Also it is possible to search based on a timestamp like
// "2020-05-25|08:47:33.663" to jump to the closest log line for all the files
func RunLogSync(args []string) {
	app := tview.NewApplication()
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow)
	flexRows := tview.NewFlex().SetDirection(tview.FlexRow)
	var fileViews []fileView
	for i, logFilename := range args {
		file, err := os.Open(logFilename)
		if err != nil {
			log.Fatal(err)
		}
		fv := newFileView(file, logFilename, i)
		fileViews = append(fileViews, *fv)
	}

	for i := range fileViews {
		fileViews[i].allFileViews = fileViews
		fileViews[i].LoadInputHandler()
		flexRows = flexRows.AddItem(fileViews[i], 0, 1, false)
	}

	inputField := tview.NewInputField().
		SetLabel("> ").
		SetFieldBackgroundColor(tcell.ColorBlack).
		SetFieldWidth(80).
		SetChangedFunc(func(text string) {
			currCommand = text
		}).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				numSteps, err := strconv.Atoi(currCommand)
				if err == nil {
					if numSteps > 0 {
						for i := 0; i < numSteps; i++ {
							next := AdvanceNextFileViewForward(fileViews)
							if next < 0 {
								break
							}
						}
					} else if numSteps < 0 {
						numSteps *= -1
						for i := 0; i < numSteps; i++ {
							prev := AdvancePrevFileViewBackward(fileViews)
							if prev < 0 {
								break
							}
						}
					}

					for i := range fileViews {
						fileViews[i].SetDisplayText()
					}
				} else {
					if currCommand == "tail" {
						MoveAllToEnd(fileViews)
					} else if currCommand == "head" {
						MoveAllToBeginning(fileViews)
					} else {
						timeStamp := filechunk.GetTimeStampFromLine(currCommand)
						if timeStamp > 1 {
							MoveAllToTime(fileViews, timeStamp)
						}
					}
				}
			}
		})

	mainFlex = mainFlex.AddItem(flexRows, 0, 1, false)
	mainFlex = mainFlex.AddItem(inputField, 1, 1, true)

	MoveAllToBeginning(fileViews)
	if err := app.SetRoot(mainFlex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
