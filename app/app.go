package app

import (
	"fmt"
	"io/ioutil"
	"github.com/rivo/tview"
)

const corporate = `Leverage agile frameworks to provide a robust synopsis for high level overviews. Iterative approaches to corporate strategy foster collaborative thinking to further the overall value proposition. Organically grow the holistic world view of disruptive innovation via workplace diversity and empowerment.

Bring to the table win-win survival strategies to ensure proactive domination. At the end of the day, going forward, a new normal that has evolved from generation X is on the runway heading towards a streamlined cloud solution. User generated content in real-time will have multiple touchpoints for offshoring.

Capitalize on low hanging fruit to identify a ballpark value added activity to beta test. Override the digital divide with additional clickthroughs from DevOps. Nanotechnology immersion along the information highway will close the loop on focusing solely on the bottom line.

[yellow]Press Enter, then Tab/Backtab for word selections`

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func getFileView(logFilename string) tview.Primitive {
	dat, err := ioutil.ReadFile(logFilename)
	check(err)

        dataStr := fmt.Sprintf("%s", dat)

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetText(dataStr)

	textView.SetBorder(true)
	textView.SetTitle(logFilename)

	return textView
}

func RunLogSync(args []string) {
	app := tview.NewApplication()
	mainFlex := tview.NewFlex()
	flexRows := tview.NewFlex().SetDirection(tview.FlexRow)
	for _, logFilename := range args {
		flexRows = flexRows.AddItem(getFileView(logFilename), 0, 1, false)
	}
	mainFlex.AddItem(flexRows, 0, 1, false)
	if err := app.SetRoot(mainFlex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
