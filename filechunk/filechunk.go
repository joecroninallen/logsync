package filechunk

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"time"
)

// defaultChunkSize is the size of the block we try to read from the file
// We read a rather large chunk to minimize the number of reads.
const defaultChunkSize int64 = 262144

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// FileChunk is how we can step through file and view chunks at a time
// FileChunk is a node in a linked list of "chunks" in a file we are viewing.
// Each chunk represents a section of the file starting from FileOffsetStart
// going through FileOffsetEnd.
// We first read a large chunk defined by defaultChunkSize, and then as
// we step through, we break a part chunks such that a chunk represents
// a single line in the log file. Once we have done that, we can set the
// LineTimeStamp.
type FileChunk struct {
	FileToRead      *os.File   // file we are viewing
	FileChunkBytes  []byte     // the bytes will be read into memory here once chunk is loaded
	FileOffsetStart int64      // the file offset start, where we seek to in the file before reading
	FileOffsetEnd   int64      // we read up to and including the FileOffsetEnd
	LineTimeStamp   int64      // if this chunk represents a single log line, this will be set
	PrevChunk       *FileChunk // previous FileChunk in linked list
	NextChunk       *FileChunk // next FileChunk in linked list
}

// LoadFileChunkForward loads the file chunk that would
// come after an existing file chunk, or from nothing if it
// is the head. In other words, this performs an actual read from the
// disk to load a section of file that is of size defaultChunkSize,
// or less if its the end of the file.
// We don't read the whole file at once, we only read in chunks as needed.
// This function is used when we are navigating forward in the file
// and come to the end of the current chunk and realize we need
// to read in the next chunk from the file.
func (fc *FileChunk) LoadFileChunkForward() (*FileChunk, *FileChunk) {
	originalChunkSize := (fc.FileOffsetEnd - fc.FileOffsetStart) + 1
	newChunkSize := defaultChunkSize
	if newChunkSize > originalChunkSize {
		newChunkSize = originalChunkSize
	}

	actualSeekStart, err := fc.FileToRead.Seek(fc.FileOffsetStart, 0)
	check(err)
	if fc.FileOffsetStart != actualSeekStart {
		panic("failed to seek to desired position in file, yet there was no error")
	}

	fc.FileChunkBytes = make([]byte, newChunkSize)
	actualChunkSizeRead, err := fc.FileToRead.Read(fc.FileChunkBytes)
	if int64(actualChunkSizeRead) != newChunkSize {
		panic("Failed to read requested newChunkSize")
	}

	if err != io.EOF && originalChunkSize != newChunkSize {
		check(err)

		// Walk it back to end of log line
		for ; newChunkSize > 0; newChunkSize-- {
			if fc.FileChunkBytes[newChunkSize-1] == '\n' {
				break
			}
		}

		if newChunkSize == 0 {
			newChunkSize = 1
		}
	}

	fc.FileOffsetEnd = fc.FileOffsetStart + newChunkSize - 1
	fc.FileChunkBytes = fc.FileChunkBytes[0:newChunkSize]

	nextChunkSize := originalChunkSize - newChunkSize
	if nextChunkSize > 0 {
		currNext := fc.NextChunk
		fc.NextChunk = &FileChunk{
			FileToRead:      fc.FileToRead,
			FileChunkBytes:  nil,
			FileOffsetStart: fc.FileOffsetEnd + 1,
			FileOffsetEnd:   fc.FileOffsetEnd + nextChunkSize,
			LineTimeStamp:   -1,
			PrevChunk:       fc,
			NextChunk:       currNext,
		}
		if currNext != nil {
			currNext.PrevChunk = fc.NextChunk
		}
	}

	fc = fc.SeparateFirstLogLine()

	var lastChunk *FileChunk
	if fc.NextChunk != nil && fc.NextChunk.FileChunkBytes != nil {
		lastChunk = fc.NextChunk.SeparateLastLogLine()
	}

	return fc, lastChunk
}

// LoadFileChunkBackward loads the file chunk that would
// come before an existing file chunk, or from nothing if it
// is the tail. In other words, this performs an actual read from the
// disk to load a section of file that is of size defaultChunkSize,
// or less if the preceedng chunk is found to be smaller.
// We don't read the whole file at once, we only read in chunks as needed.
// This function is used when we are navigating backward in the file
// and come to the beginning of the current chunk and realize we need
// to read in the previous chunk from the file.
func (fc *FileChunk) LoadFileChunkBackward() (*FileChunk, *FileChunk) {
	originalChunkSize := (fc.FileOffsetEnd - fc.FileOffsetStart) + 1
	newChunkSize := defaultChunkSize
	if newChunkSize > originalChunkSize {
		newChunkSize = originalChunkSize
	}

	newFileOffsetStart := (fc.FileOffsetEnd + 1) - newChunkSize
	actualSeekStart, err := fc.FileToRead.Seek(newFileOffsetStart, 0)
	check(err)
	if newFileOffsetStart != actualSeekStart {
		panic("failed to seek to desired position in file, yet there was no error")
	}

	fc.FileChunkBytes = make([]byte, newChunkSize)
	actualChunkSizeRead, err := fc.FileToRead.Read(fc.FileChunkBytes)
	if int64(actualChunkSizeRead) != newChunkSize {
		panic("Failed to read requested newChunkSize")
	}

	fc.FileOffsetStart = newFileOffsetStart

	if originalChunkSize != newChunkSize {
		if err != io.EOF {
			check(err)
		}

		// Walk it forward to end of beginning of next line
		var newChunkStartIndex int = 1
		for ; int64(newChunkStartIndex) < newChunkSize; newChunkStartIndex++ {
			if fc.FileChunkBytes[newChunkStartIndex-1] == '\n' {
				break
			}
		}

		if int64(newChunkStartIndex) == newChunkSize {
			newChunkStartIndex = 0
		}

		newChunkSize -= int64(newChunkStartIndex)

		fc.FileChunkBytes = fc.FileChunkBytes[newChunkStartIndex:]
	}

	fc.FileOffsetStart = 1 + fc.FileOffsetEnd - newChunkSize

	prevChunkSize := originalChunkSize - newChunkSize
	if prevChunkSize > 0 && fc.FileOffsetStart-1 != fc.PrevChunk.FileOffsetEnd {
		currPrev := fc.PrevChunk
		fc.PrevChunk = &FileChunk{
			FileToRead:      fc.FileToRead,
			FileChunkBytes:  nil,
			FileOffsetStart: fc.FileOffsetStart - prevChunkSize,
			FileOffsetEnd:   fc.FileOffsetStart - 1,
			LineTimeStamp:   -1,
			PrevChunk:       fc.PrevChunk,
			NextChunk:       fc,
		}
		if currPrev != nil {
			currPrev.NextChunk = fc.PrevChunk
		}
	}

	if !fc.ValidateFileChunkChain() {
		fmt.Println("Invalid before fc.SeparateFirstLogLine")
		fc.PrintFileChunkChain()
	}

	fc = fc.SeparateFirstLogLine()

	var lastChunk *FileChunk
	if fc.NextChunk != nil && fc.NextChunk.FileChunkBytes != nil {
		lastChunk = fc.NextChunk.SeparateLastLogLine()
	}

	return fc, lastChunk
}

// NewFileChunk loads a new file chunk from the file
// and returns the first and last log line chunks.
// This is called in the beginning to start our initial
// linked list of file chunks.
// First, it reads in the first defaultChunkSize at the head
// of the file. It then breaks off the first log line
// of that initial chunk and also the last log line of
// that initial chunk. It leaves a large chunk in between
// those two chunks which has the bytes from the file in between,
// but that haven't yet been broken down by log line.
// Next, we read in the last block in the file of size
// defaultChunkSize, and we do the same to this chunk and
// break off the first line and last line of this tail chunk.
// Thus, the tail chunk looks like the same as the first chunk
// in having the first and last line broken off and leaving
// a middle chunk in between that we have not yet broken apart
// into separate lines.
// In between the last log line of the head chunk and the first
// log line of the tail chunk, we have a large FileChunk
// that has not yet been read into memory. We only read
// parts of the file into memory if you navigate there.
// What gets returned from NeWFileChunk is the head log
// line and the tail log line, which allows for easily jumping
// to the head and tail of the file.
func NewFileChunk(f *os.File) (*FileChunk, *FileChunk) {
	fileInfo, err := f.Stat()
	check(err)
	fileSize := fileInfo.Size()

	// First create start chunk
	startChunk := &FileChunk{
		FileToRead:      f,
		FileChunkBytes:  nil,
		FileOffsetStart: 0,
		FileOffsetEnd:   fileSize - 1,
		LineTimeStamp:   -1,
		PrevChunk:       nil,
		NextChunk:       nil,
	}

	headStart, headEnd := startChunk.LoadFileChunkForward()

	if headEnd.NextChunk == nil {
		return headStart, headEnd
	}

	_, tailEnd := headEnd.NextChunk.LoadFileChunkBackward()

	return headStart, tailEnd
}

// PrintFileChunkChain prints the meta info about the entire chain
// This is used for debugging
func (fc *FileChunk) PrintFileChunkChain() {
	head := fc
	for {
		if head.PrevChunk == nil {
			break
		}
		head = head.PrevChunk
	}

	tail := head

	fileInfo, err := tail.FileToRead.Stat()
	check(err)
	fileSize := fileInfo.Size()
	fmt.Printf("Printing file chunk chain for file with size: %v\n", fileSize)

	for {
		fmt.Printf("FileOffsetStart %v, FileOffsetEnd %v, LenFromIndices %v, len(FileChunkBytes) %v, LineTimeStamp %v\n", tail.FileOffsetStart, tail.FileOffsetEnd, 1+tail.FileOffsetEnd-tail.FileOffsetStart, len(tail.FileChunkBytes), tail.LineTimeStamp)
		tail = tail.NextChunk
		if tail == nil {
			break
		}
	}
}

// ValidateFileChunkChain prints the meta info about the entire chain
// This is used in testing to validate the state of the FleChunk chain
func (fc *FileChunk) ValidateFileChunkChain() bool {
	head := fc
	for {
		if head.PrevChunk == nil {
			break
		}
		head = head.PrevChunk
	}

	if head == nil {
		return false
	}

	if head.FileOffsetStart != 0 {
		return false
	}

	tail := head

	for {
		if tail.FileChunkBytes != nil {
			if 1+tail.FileOffsetEnd-tail.FileOffsetStart != int64(len(tail.FileChunkBytes)) {
				return false
			}
		}

		if tail.NextChunk != nil {
			if tail.NextChunk.FileOffsetStart != tail.FileOffsetEnd+1 {
				return false
			}
			tail = tail.NextChunk
		} else {
			fileInfo, err := tail.FileToRead.Stat()
			check(err)
			fileSize := fileInfo.Size()

			if tail.FileOffsetEnd != fileSize-1 {
				return false
			}
			break
		}
	}

	return true
}

// GetTimeStampFromLine gets the time stamp from the regex
// This is hardcoded to be like the tendermint Docker logs for now.
// TODO: add in the functionality to specify how the time stamp is for each
// each log file.
func GetTimeStampFromLine(line string) int64 {
	compRegEx := *regexp.MustCompile(`(?P<Year>\d{4})-(?P<Month>\d{2})-(?P<Day>\d{2})\|(?P<Hour>\d{2})\:(?P<Minute>\d{2})\:(?P<Second>\d{2})\.(?P<Millisecond>\d{3})`)
	match := compRegEx.FindStringSubmatch(line)

	if match == nil {
		return 1
	}

	paramsMap := make(map[string]int)
	for i, name := range compRegEx.SubexpNames() {
		if i > 0 && i <= len(match) {
			if val, err := strconv.Atoi(match[i]); err == nil {
				paramsMap[name] = val
			} else {
				panic(err)
			}
		}
	}

	t := time.Date(paramsMap["Year"], time.Month(paramsMap["Month"]), paramsMap["Day"], paramsMap["Hour"], paramsMap["Minute"], paramsMap["Second"], paramsMap["Millisecond"]*1000000, time.UTC)

	nanos := t.UnixNano()
	return nanos
}

// SeparateFirstLogLine breaks off first log line in the loaded chunk
// This is when we have read in a large chunk of the file and we want to
// break off just the first line of that chunk. The result is that our
// linked list will add a node for the beginning log line of the chunk,
// and then the remaining chunk after that first log line was broken off.
// This is used when we are navigating forward and reach a FileChunk that
// has not yet been broken a part.
func (fc *FileChunk) SeparateFirstLogLine() *FileChunk {
	originalChunkEndIndex := fc.FileOffsetEnd - fc.FileOffsetStart
	var firstLineChunkIndex int64 = 0
	for ; firstLineChunkIndex <= originalChunkEndIndex; firstLineChunkIndex++ {
		if fc.FileChunkBytes[firstLineChunkIndex] == '\n' {
			break
		}
	}

	if firstLineChunkIndex > originalChunkEndIndex {
		firstLineChunkIndex = originalChunkEndIndex
	}

	if firstLineChunkIndex < originalChunkEndIndex {
		// Only create a new next chunk if the current next chunk
		// is not already properly set
		if fc.NextChunk == nil || fc.NextChunk.FileOffsetStart != fc.FileOffsetStart+firstLineChunkIndex+1 {
			currNextChunk := fc.NextChunk
			fc.NextChunk = &FileChunk{
				FileToRead:      fc.FileToRead,
				FileChunkBytes:  fc.FileChunkBytes[(firstLineChunkIndex + 1):],
				FileOffsetStart: fc.FileOffsetStart + firstLineChunkIndex + 1,
				FileOffsetEnd:   fc.FileOffsetEnd,
				LineTimeStamp:   -1,
				PrevChunk:       fc,
				NextChunk:       fc.NextChunk,
			}

			if currNextChunk != nil {
				currNextChunk.PrevChunk = fc.NextChunk
			}

			fc.FileOffsetEnd = fc.FileOffsetStart + firstLineChunkIndex
		}
	}

	fc.FileChunkBytes = fc.FileChunkBytes[0:(firstLineChunkIndex + 1)]
	fc.LineTimeStamp = GetTimeStampFromLine(string(fc.FileChunkBytes))

	return fc
}

// SeparateLastLogLine breaks off the last log line in the loaded chunk.
// This is when we have read in a large chunk of the file and we want to
// break off just the last line of that chunk. The result is that our
// linked list will add a node for the last log line of the chunk,
// and then the remaining chunk before that last log line was broken off.
// This is used when we are navigating backward and reach a FileChunk that
// has not yet been broken a part.
func (fc *FileChunk) SeparateLastLogLine() *FileChunk {
	originalChunkEndIndex := fc.FileOffsetEnd - fc.FileOffsetStart
	newPrevEndIndex := originalChunkEndIndex - 1
	for ; newPrevEndIndex >= 0; newPrevEndIndex-- {
		if fc.FileChunkBytes[newPrevEndIndex] == '\n' {
			break
		}
	}

	if newPrevEndIndex >= 0 {
		// Only create a new prevChunk if the current prev chunk ends prior
		// to the new end that we found
		if fc.PrevChunk == nil || fc.PrevChunk.FileOffsetEnd != fc.FileOffsetStart+newPrevEndIndex {
			currPrevChunk := fc.PrevChunk
			fc.PrevChunk = &FileChunk{
				FileToRead:      fc.FileToRead,
				FileChunkBytes:  fc.FileChunkBytes[0 : newPrevEndIndex+1],
				FileOffsetStart: fc.FileOffsetStart,
				FileOffsetEnd:   fc.FileOffsetStart + newPrevEndIndex,
				LineTimeStamp:   -1,
				PrevChunk:       fc.PrevChunk,
				NextChunk:       fc,
			}

			if currPrevChunk != nil {
				currPrevChunk.NextChunk = fc.PrevChunk
			}

			fc.FileOffsetStart = fc.PrevChunk.FileOffsetEnd + 1
		}
	}

	fc.FileChunkBytes = fc.FileChunkBytes[newPrevEndIndex+1:]
	fc.LineTimeStamp = GetTimeStampFromLine(string(fc.FileChunkBytes))

	return fc
}

// GetNextFileChunk returns the next file chunk line.
// This is used when navigating forward. The easy case is
// when the next log line has already been broken off
// and we can just return it. Otherwise, the next easiest case
// is when the next FileChunk has been loaded into memory
// but we just need to break off the first log line of the chunk.
// After that, we enter a chunk that requires us to read from disk.
// After reading from disk, we handle it just like
// the second case where we have a large chunk that needs to
// have the first log broken off.
func (fc *FileChunk) GetNextFileChunk() *FileChunk {
	if fc == nil || fc.NextChunk == nil {
		return nil
	}

	if fc.NextChunk.LineTimeStamp > -1 {
		return fc.NextChunk
	}

	if fc.NextChunk.FileChunkBytes == nil {
		front, _ := fc.NextChunk.LoadFileChunkForward()
		return front
	}

	return fc.NextChunk.SeparateFirstLogLine()
}

// GetNextTimestampedFileChunk returns the next file chunk line with a timestamp.
// Because not all log lines have timestamps, sometimes we need to just skip over
// log lines that don't have a time stamp until we get to one that does.
func (fc *FileChunk) GetNextTimestampedFileChunk() *FileChunk {
	nextFileChunk := fc.GetNextFileChunk()
	if nextFileChunk == nil {
		return nil
	}

	for {
		if nextFileChunk.LineTimeStamp > 1 {
			return nextFileChunk
		}
		nextFileChunk = nextFileChunk.GetNextFileChunk()
		if nextFileChunk == nil {
			return nil
		}
	}

	return nil
}

// GetPrevFileChunk returns the previous file chunk line.
// This is used when navigating backward. The easy case is
// when the previous log line has already been broken off
// and we can just return it. Otherwise, the next easiest case
// is when the previous FileChunk has been loaded into memory
// but we just need to break off the last log line of the chunk.
// After that, we enter a chunk that requires us to read from disk.
// After reading from disk, we handle it just like
// the second case where we have a large chunk that needs to
// have the last log line broken off.
func (fc *FileChunk) GetPrevFileChunk() *FileChunk {
	if fc == nil || fc.PrevChunk == nil {
		return nil
	}

	if fc.PrevChunk.LineTimeStamp > -1 {
		return fc.PrevChunk
	}

	if fc.PrevChunk.FileChunkBytes == nil {
		_, back := fc.PrevChunk.LoadFileChunkBackward()
		return back
	}

	return fc.PrevChunk.SeparateLastLogLine()
}

// GetPrevTimestampedFileChunk returns the previous file chunk line with a timestamp.
// Because not all log lines have timestamps, sometimes we need to just skip over
// log lines that don't have a time stamp until we get to one that does.
func (fc *FileChunk) GetPrevTimestampedFileChunk() *FileChunk {
	prevFileChunk := fc.GetPrevFileChunk()
	if prevFileChunk == nil {
		return nil
	}

	for {
		if prevFileChunk.LineTimeStamp > 1 {
			return prevFileChunk
		}
		prevFileChunk = prevFileChunk.GetPrevFileChunk()
		if prevFileChunk == nil {
			return nil
		}
	}

	return nil
}

// GetFileChunkClosestToTime returns the previous file chunk line just before time.
// This is used for searching for a particular time in the log file.
// Note that right now, this just does a brute force linear search.
// TODO: Implement binary search which will run in logarithmic time, and more
// importanly not load the whole file into memory when its not needed.
func (fc *FileChunk) GetFileChunkClosestToTime(searchTime int64) *FileChunk {
	currFileChunk := fc.GetPrevTimestampedFileChunk()
	if currFileChunk == nil {
		currFileChunk = fc.GetNextTimestampedFileChunk()
	}

	if currFileChunk.LineTimeStamp <= searchTime {
		// Search forward until we get the next chunk just after
		// search time and then return the previous chunk
		for {
			nextChunk := currFileChunk.GetNextTimestampedFileChunk()
			if nextChunk == nil || nextChunk.LineTimeStamp > searchTime {
				return currFileChunk
			}

			currFileChunk = nextChunk
		}
	} else if currFileChunk.LineTimeStamp > searchTime {
		// Search backward until we get a chunk before search time and return that
		for {
			prevChunk := currFileChunk.GetPrevTimestampedFileChunk()
			if prevChunk == nil || prevChunk.LineTimeStamp < searchTime {
				if prevChunk == nil {
					return currFileChunk
				}
				return prevChunk
			}

			currFileChunk = prevChunk
		}
	}
	return nil
}
