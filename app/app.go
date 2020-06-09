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

type fileView struct {
	*tview.TextView
	file           *os.File
	headChunk      *filechunk.FileChunk
	tailChunk      *filechunk.FileChunk
	currChunk      *filechunk.FileChunk
	index          int
	lastScrollTime int64
	allFileViews   []fileView
}

/*
func getNewFileTextView(logFilename string) tview.Primitive {


	dataStr := fmt.Sprintf("%s", dat)

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetText(dataStr)

	textView.SetBorder(true)
	textView.SetTitle(logFilename)
	textView.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			textView.Clear()
		}
	})

	return textView
}
*/

// AdvanceNextFileViewForward figures out which fileview is next
// in line and advances its current chunk.
// This is based on who has the most recent
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

// AdvancePrevFileViewBackward figures out which fileview is closest prev
// in line and advances its current chunk backward.
// This is based on who has the latest prev time
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

// MoveAllToBeginning should be private
func MoveAllToBeginning(fileViews []fileView) {
	for i := range fileViews {
		fileViews[i].currChunk = fileViews[i].headChunk
		fileViews[i].SetDisplayText()
	}
}

// MoveAllToEnd should be private
func MoveAllToEnd(fileViews []fileView) {
	for i := range fileViews {
		fileViews[i].currChunk = fileViews[i].tailChunk
		fileViews[i].SetDisplayText()
	}
}

// MoveAllToTime should be private
func MoveAllToTime(fileViews []fileView, searchTime int64) {
	for i := range fileViews {
		closestChunk := fileViews[i].currChunk.GetFileChunkClosestToTime(searchTime)
		if closestChunk != nil {
			fileViews[i].currChunk = closestChunk
			fileViews[i].SetDisplayText()
		}
	}
}

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
		} else if key == tcell.KeyEscape {
			MoveAllToBeginning(fv.allFileViews)
		} else if key == tcell.KeyF2 {
			MoveAllToEnd(fv.allFileViews)
		}
	})
}

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

/*
type logSyncApplication struct {
	*tview.Application
}

func newLogSyncApplication(args []string) *logSyncApplication {
	rd := bufio.NewReader(f)
	dataStr, err := rd.ReadString('\n')
	if err == io.EOF {
		fmt.Print(line)
		break
	}

	// loop termination condition 2: some other error.
	// Errors happen, so check for them and do something with them.
	if err != nil {
		log.Fatal(err)
	}

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetText(dataStr)

	textView.SetBorder(true)
	textView.SetTitle(logFilename)
	textView.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyEscape {
			textView.Clear()
		}
	})

	return &FileView{
		TextView: textVeiw,
		file:     fileToView,
	}
}
*/

var currCommand string

// RunLogSync is the main tview function that builds the UI
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
