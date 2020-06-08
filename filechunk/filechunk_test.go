package filechunk_test

import (
	"fmt"
	"log"
	"os"

	"github.com/joecroninallen/logsync/filechunk"
)

func ExampleTheTestingFramework() {
	fmt.Println("hello")
	// Output: hello
}

func ExampleLoadShortFilechunk() {
	file, err := os.Open("../short-logs/node0-json.log")
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
