package filechunk

import (
	"io"
	"os"
	"regexp"
	"strconv"
	"time"
)

const defaultChunkSize int64 = 262144

func check(e error) {
	if e != nil {
		panic(e)
	}
}

// FileChunk is how we can step through file and view chunks at a time
type FileChunk struct {
	FileToRead      *os.File
	FileChunkBytes  []byte
	FileOffsetStart int64
	FileOffsetEnd   int64
	LineTimeStamp   int64
	PrevChunk       *FileChunk
	NextChunk       *FileChunk
}

// LoadFileChunkForward loads the file chunk that would
// come after an existing file chunk, or from nothing if it
// is the head
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

	fc.FileOffsetEnd = fc.FileOffsetStart + newChunkSize - 1

	if err != io.EOF && originalChunkSize != newChunkSize {
		check(err)

		// Walk it back to end of log line
		didWalkBack := false
		for i := newChunkSize - 1; i >= 0; i-- {
			if fc.FileChunkBytes[i] == '\n' {
				break
			} else {
				didWalkBack = true
				fc.FileOffsetEnd--
				newChunkSize--
			}
		}

		if didWalkBack {
			fc.FileChunkBytes = fc.FileChunkBytes[0:newChunkSize]
		}
	}

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
		currNext.PrevChunk = fc.NextChunk
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
// is the tail
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
		didWalkForward := false
		var newChunkStartIndex int = 1
		for ; int64(newChunkStartIndex) < newChunkSize; newChunkStartIndex++ {
			if fc.FileChunkBytes[newChunkStartIndex-1] == '\n' {
				break
			} else {
				didWalkForward = true
				fc.FileOffsetStart++
				newChunkSize--
			}
		}

		if didWalkForward {
			if int64(newChunkStartIndex) == newChunkSize {
				newChunkStartIndex--
			}
			fc.FileChunkBytes = fc.FileChunkBytes[newChunkStartIndex:]
		}
	}

	prevChunkSize := originalChunkSize - newChunkSize
	if prevChunkSize > 0 {
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
		currPrev.NextChunk = fc.PrevChunk
	}

	fc = fc.SeparateFirstLogLine()
	var lastChunk *FileChunk
	if fc.NextChunk != nil && fc.NextChunk.FileChunkBytes != nil {
		lastChunk = fc.NextChunk.SeparateLastLogLine()
	}

	return fc, lastChunk
}

// NewFileChunk loads a new file chunk from the file
// and returns the first and last log line chunks
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

	_, tailEnd := headEnd.NextChunk.LoadFileChunkForward()

	return headStart, tailEnd
}

// GetTimeStampFromLine gets the time stamp from the regex
// This is hardcoded to be like the tendermint logs for now
func GetTimeStampFromLine(line string) int64 {
	compRegEx := *regexp.MustCompile(`\[(?P<Year>\d{4})-(?P<Month>\d{2})-(?P<Day>\d{2})\|(?P<Hour>\d{2})\:(?P<Minute>\d{2})\:(?P<Second>\d{2})\.(?P<Millisecond>\d{3})\]`)
	match := compRegEx.FindStringSubmatch(line)

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
	return nanos / 1000000
}

// SeparateFirstLogLine breaks off first log line in the loaded chunk
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

// SeparateLastLogLine breaks off the last log line in the loaded chunk
func (fc *FileChunk) SeparateLastLogLine() *FileChunk {
	originalChunkEndIndex := fc.FileOffsetEnd - fc.FileOffsetStart
	lastLineChunkIndex := originalChunkEndIndex - 1
	for ; lastLineChunkIndex >= 0; lastLineChunkIndex-- {
		if fc.FileChunkBytes[lastLineChunkIndex] == '\n' {
			lastLineChunkIndex++
			break
		}
	}

	if lastLineChunkIndex < 0 {
		lastLineChunkIndex = 0
	}

	if lastLineChunkIndex > 0 {
		// Only create a new prevChunk if the current prev chunk ends prior
		// to the new end that we found
		if fc.PrevChunk == nil || fc.PrevChunk.FileOffsetEnd != fc.FileOffsetStart+lastLineChunkIndex-1 {
			currPrevChunk := fc.PrevChunk
			fc.PrevChunk = &FileChunk{
				FileToRead:      fc.FileToRead,
				FileChunkBytes:  fc.FileChunkBytes[0 : lastLineChunkIndex-1],
				FileOffsetStart: fc.FileOffsetStart,
				FileOffsetEnd:   fc.FileOffsetStart + lastLineChunkIndex - 1,
				LineTimeStamp:   -1,
				PrevChunk:       fc.PrevChunk,
				NextChunk:       fc,
			}

			if currPrevChunk != nil {
				currPrevChunk.NextChunk = fc.PrevChunk
			}

			fc.FileOffsetStart = fc.FileOffsetStart + lastLineChunkIndex
		}
	}

	fc.FileChunkBytes = fc.FileChunkBytes[lastLineChunkIndex:]
	fc.LineTimeStamp = GetTimeStampFromLine(string(fc.FileChunkBytes))

	return fc
}

// GetNextFileChunk returns the next file chunk line
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

// GetPrevFileChunk returns the previous file chunk line
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
