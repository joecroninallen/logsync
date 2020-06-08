package app

import (
	"log"
	"os"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"

	"github.com/joecroninallen/logsync/filechunk"
)

type fileView struct {
	*tview.TextView
	file      *os.File
	headChunk *filechunk.FileChunk
	tailChunk *filechunk.FileChunk
	currChunk *filechunk.FileChunk
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

func newFileView(file *os.File, logFilename string) *fileView {

	head, tail := filechunk.NewFileChunk(file)

	dataStr := string(tail.FileChunkBytes)
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

	return &fileView{
		TextView:  textView,
		file:      file,
		headChunk: head,
		tailChunk: tail,
		currChunk: head,
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

// RunLogSync is the main tview function that builds the UI
func RunLogSync(args []string) {
	app := tview.NewApplication()
	mainFlex := tview.NewFlex()
	flexRows := tview.NewFlex().SetDirection(tview.FlexRow)
	for _, logFilename := range args {
		file, err := os.Open(logFilename)
		if err != nil {
			log.Fatal(err)
		}
		// defer file.Close()
		flexRows = flexRows.AddItem(newFileView(file, logFilename), 0, 1, false)
	}
	mainFlex.AddItem(flexRows, 0, 1, false)
	if err := app.SetRoot(mainFlex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
	//app.SetInputCapture()
}
