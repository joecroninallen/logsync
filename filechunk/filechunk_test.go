// Package filechunk_test tests the filechunk code
package filechunk_test

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"

	"github.com/joecroninallen/logsync/filechunk"
)

func ExampleTheTestingFramework() {
	fmt.Println("hello")
	// Output: hello
}

func ExampleLoadShortFilechunk() {
	file, err := os.Open("../test_data/short-logs/node0-json.log")
	if err != nil {
		log.Fatal(err)
	}

	head, tail := filechunk.NewFileChunk(file)

	headStr := string(head.FileChunkBytes)
	tailStr := string(tail.FileChunkBytes)

	fmt.Println(headStr)
	fmt.Println(tailStr)
	// Output: {"log":"I[2020-05-25|08:45:31.749] Starting multiAppConn service                module=proxy impl=multiAppConn\n","stream":"stdout","time":"2020-05-25T08:45:31.749290288Z"}
	//
	// {"log":"I[2020-05-25|08:45:31.782] Started node                                 module=main nodeInfo=\"{ProtocolVersion:{P2P:7 Block:10 App:1} DefaultNodeID:2e352167f6097f6e0fbb31bd92f5c3b2b0bd78ec ListenAddr:tcp://0.0.0.0:26656 Network:chain-3DzG2d Version:0.33.4 Channels:40202122233038606100 Moniker:2168B76125ABCA7B Other:{TxIndex:on RPCAddress:tcp://0.0.0.0:26657}}\"\n","stream":"stdout","time":"2020-05-25T08:45:31.782198955Z"}
}

func ExamplePrintShortFilechunk() {
	file, err := os.Open("../test_data/short-logs/node0-json.log")
	if err != nil {
		log.Fatal(err)
	}

	head, tail := filechunk.NewFileChunk(file)

	head.PrintFileChunkChain()
	tail.PrintFileChunkChain()
	// Output: Printing file chunk chain for file with size: 6998
	// FileOffsetStart 0, FileOffsetEnd 172, LenFromIndices 173, len(FileChunkBytes) 173, LineTimeStamp 1590396331749000000
	// FileOffsetStart 173, FileOffsetEnd 6562, LenFromIndices 6390, len(FileChunkBytes) 6390, LineTimeStamp -1
	// FileOffsetStart 6563, FileOffsetEnd 6997, LenFromIndices 435, len(FileChunkBytes) 435, LineTimeStamp 1590396331782000000
	// Printing file chunk chain for file with size: 6998
	// FileOffsetStart 0, FileOffsetEnd 172, LenFromIndices 173, len(FileChunkBytes) 173, LineTimeStamp 1590396331749000000
	// FileOffsetStart 173, FileOffsetEnd 6562, LenFromIndices 6390, len(FileChunkBytes) 6390, LineTimeStamp -1
	// FileOffsetStart 6563, FileOffsetEnd 6997, LenFromIndices 435, len(FileChunkBytes) 435, LineTimeStamp 1590396331782000000
}

func ExampleIterateMediumFilechunkForward() {
	file, err := os.Open("../test_data/medium-logs/node0-json.log")
	if err != nil {
		log.Fatal(err)
	}

	head, _ := filechunk.NewFileChunk(file)

	var lineCount int = 1

	next := head
	for {
		next = next.GetNextFileChunk()
		if next != nil {
			lineCount++
		} else {
			break
		}
	}

	if !head.ValidateFileChunkChain() {
		fmt.Printf("failure at line count %v\n", lineCount)
	}

	fmt.Printf("%v\n", lineCount)
	// Output: 9999
}

func ExampleIterateMediumFilechunkBackward() {
	file, err := os.Open("../test_data/medium-logs/node0-json.log")
	if err != nil {
		log.Fatal(err)
	}

	_, tail := filechunk.NewFileChunk(file)

	var lineCount int = 1

	prev := tail
	for {
		prev = prev.GetPrevFileChunk()
		if prev != nil {
			lineCount++
		} else {
			break
		}
	}

	if !tail.ValidateFileChunkChain() {
		fmt.Printf("failure at line count %v\n", lineCount)
	}

	fmt.Printf("%v\n", lineCount)
	// Output: 9999
}

func ExampleIterateLargeFilechunkForward() {
	file, err := os.Open("../test_data/logs/node0-json.log")
	if err != nil {
		log.Fatal(err)
	}

	head, _ := filechunk.NewFileChunk(file)

	var lineCount int = 1

	next := head
	for {
		next = next.GetNextFileChunk()
		if next != nil {
			lineCount++
		} else {
			break
		}
	}

	if !head.ValidateFileChunkChain() {
		fmt.Printf("failure at line count %v\n", lineCount)
	}

	fmt.Printf("%v\n", lineCount)
	// Output: 49540
}

func ExampleIterateLargeFilechunkBackward() {
	file, err := os.Open("../test_data/logs/node0-json.log")
	if err != nil {
		log.Fatal(err)
	}

	_, tail := filechunk.NewFileChunk(file)

	var lineCount int = 1

	prev := tail
	for {
		prev = prev.GetPrevFileChunk()
		if prev != nil {
			lineCount++
		} else {
			break
		}
	}

	if !tail.ValidateFileChunkChain() {
		fmt.Printf("failure at line count %v\n", lineCount)
	}

	fmt.Printf("%v\n", lineCount)
	// Output: 49540
}

func ExamplePrintLargeFilechunk() {
	file, err := os.Open("../test_data/logs/node0-json.log")
	if err != nil {
		log.Fatal(err)
	}

	head, tail := filechunk.NewFileChunk(file)

	head.PrintFileChunkChain()
	tail.PrintFileChunkChain()
	// Output: Printing file chunk chain for file with size: 17039577
	// FileOffsetStart 0, FileOffsetEnd 172, LenFromIndices 173, len(FileChunkBytes) 173, LineTimeStamp 1590396331749000000
	// FileOffsetStart 173, FileOffsetEnd 261742, LenFromIndices 261570, len(FileChunkBytes) 261570, LineTimeStamp -1
	// FileOffsetStart 261743, FileOffsetEnd 262104, LenFromIndices 362, len(FileChunkBytes) 362, LineTimeStamp 1590396334444000000
	// FileOffsetStart 262105, FileOffsetEnd 16777656, LenFromIndices 16515552, len(FileChunkBytes) 0, LineTimeStamp -1
	// FileOffsetStart 16777657, FileOffsetEnd 16777942, LenFromIndices 286, len(FileChunkBytes) 286, LineTimeStamp 1590396452039000000
	// FileOffsetStart 16777943, FileOffsetEnd 17039369, LenFromIndices 261427, len(FileChunkBytes) 261427, LineTimeStamp -1
	// FileOffsetStart 17039370, FileOffsetEnd 17039576, LenFromIndices 207, len(FileChunkBytes) 207, LineTimeStamp 1590396453663000000
	// Printing file chunk chain for file with size: 17039577
	// FileOffsetStart 0, FileOffsetEnd 172, LenFromIndices 173, len(FileChunkBytes) 173, LineTimeStamp 1590396331749000000
	// FileOffsetStart 173, FileOffsetEnd 261742, LenFromIndices 261570, len(FileChunkBytes) 261570, LineTimeStamp -1
	// FileOffsetStart 261743, FileOffsetEnd 262104, LenFromIndices 362, len(FileChunkBytes) 362, LineTimeStamp 1590396334444000000
	// FileOffsetStart 262105, FileOffsetEnd 16777656, LenFromIndices 16515552, len(FileChunkBytes) 0, LineTimeStamp -1
	// FileOffsetStart 16777657, FileOffsetEnd 16777942, LenFromIndices 286, len(FileChunkBytes) 286, LineTimeStamp 1590396452039000000
	// FileOffsetStart 16777943, FileOffsetEnd 17039369, LenFromIndices 261427, len(FileChunkBytes) 261427, LineTimeStamp -1
	// FileOffsetStart 17039370, FileOffsetEnd 17039576, LenFromIndices 207, len(FileChunkBytes) 207, LineTimeStamp 1590396453663000000
}

func ExampleLoadLargeFilechunk() {
	file, err := os.Open("../test_data/logs/node0-json.log")
	if err != nil {
		log.Fatal(err)
	}

	head, tail := filechunk.NewFileChunk(file)

	headStr := string(head.FileChunkBytes)
	tailStr := string(tail.FileChunkBytes)

	fmt.Println(headStr)
	fmt.Println(tailStr)
	// Output: {"log":"I[2020-05-25|08:45:31.749] Starting multiAppConn service                module=proxy impl=multiAppConn\n","stream":"stdout","time":"2020-05-25T08:45:31.749290288Z"}
	//
	// {"log":"D[2020-05-25|08:47:33.663] addVote                                      module=consensus voteHeight=91 voteType=2 valIndex=1 csHeight=92\n","stream":"stdout","time":"2020-05-25T08:47:33.663241978Z"}
}

func ExampleGetNextChunk() {
	file, err := os.Open("../test_data/logs/node0-json.log")
	if err != nil {
		log.Fatal(err)
	}

	head, _ := filechunk.NewFileChunk(file)
	nextChunk := head.GetNextFileChunk()

	headStr := string(head.FileChunkBytes)
	nextStr := string(nextChunk.FileChunkBytes)

	fmt.Println(headStr)
	fmt.Println(nextStr)

	// Output: {"log":"I[2020-05-25|08:45:31.749] Starting multiAppConn service                module=proxy impl=multiAppConn\n","stream":"stdout","time":"2020-05-25T08:45:31.749290288Z"}
	//
	// {"log":"I[2020-05-25|08:45:31.749] Starting localClient service                 module=abci-client connection=query impl=localClient\n","stream":"stdout","time":"2020-05-25T08:45:31.749312239Z"}
}

func ExampleGetPrevChunk() {
	file, err := os.Open("../test_data/logs/node0-json.log")
	if err != nil {
		log.Fatal(err)
	}

	_, tail := filechunk.NewFileChunk(file)
	prevChunk := tail.GetPrevFileChunk()
	prevPrevChunk := prevChunk.GetPrevFileChunk()

	tailStr := string(tail.FileChunkBytes)
	prevStr := string(prevChunk.FileChunkBytes)
	prevPrevStr := string(prevPrevChunk.FileChunkBytes)

	fmt.Println(prevPrevStr)
	fmt.Println(prevStr)
	fmt.Println(tailStr)

	// Output: {"log":"D[2020-05-25|08:47:33.662] setHasVote                                   module=consensus peerH/R=92/0 H/R=91/0 type=2 index=1\n","stream":"stdout","time":"2020-05-25T08:47:33.663229828Z"}
	//
	// {"log":"D[2020-05-25|08:47:33.663] addVote                                      module=consensus voteHeight=91 voteType=2 valIndex=2 csHeight=92\n","stream":"stdout","time":"2020-05-25T08:47:33.663233503Z"}
	//
	// {"log":"D[2020-05-25|08:47:33.663] addVote                                      module=consensus voteHeight=91 voteType=2 valIndex=1 csHeight=92\n","stream":"stdout","time":"2020-05-25T08:47:33.663241978Z"}
}

func ExampleGetTimeStampFromLine() {
	//GetTimeStampFromLine(line string) int64
	exampleLine := "{\"log\":\"I[2020-05-25|08:45:33.068] Starting PEX service                         module=pex impl=PEX\n\",\"stream\":\"stdout\",\"time\":\"2020-05-25T08:45:31.769886329Z\"}"

	lineTime := filechunk.GetTimeStampFromLine(exampleLine)
	fmt.Printf("lineTime is: %v\n", lineTime)

	compRegEx := *regexp.MustCompile(`(?P<Year>\d{4})-(?P<Month>\d{2})-(?P<Day>\d{2})\|(?P<Hour>\d{2})\:(?P<Minute>\d{2})\:(?P<Second>\d{2})\.(?P<Millisecond>\d{3})`)
	match := compRegEx.FindStringSubmatch(exampleLine)

	sampleTimeStamp := filechunk.GetTimeStampFromLine("2020-05-25|08:45:33.068")
	fmt.Printf("sampleTimeStamp is: %v\n", sampleTimeStamp)

	if match == nil {
		fmt.Println("No match found")
	}

	paramsMap := make(map[string]int)
	for i, name := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			if val, err := strconv.Atoi(match[i]); err == nil {
				fmt.Printf("%s --> %v\n", name, val)
				paramsMap[name] = val
			} else {
				panic(err)
			}
		}
	}

	// Output: lineTime is: 1590396333068000000
	// sampleTimeStamp is: 1590396333068000000
	// Year --> 2020
	// Month --> 5
	// Day --> 25
	// Hour --> 8
	// Minute --> 45
	// Second --> 33
	// Millisecond --> 68
}
