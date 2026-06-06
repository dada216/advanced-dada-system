package ansi

import (
	"bytes"
	"strconv"
	"strings"
	"time"
)

type CommandInserter interface {
	InsertCommand(commandText string, startTs, endTs time.Time, exitCode int) error
}

type OSCScanner struct {
	db         CommandInserter
	stream     bytes.Buffer
	state      int // 0: unknown, 1: prompt, 2: input, 3: execution
	startTs    time.Time
	commandBuf bytes.Buffer
}

func NewOSCScanner(db CommandInserter) *OSCScanner {
	return &OSCScanner{
		db: db,
	}
}

var oscPrefix = []byte("\x1b]133;")

func (s *OSCScanner) Write(p []byte) (n int, err error) {
	s.stream.Write(p)

	for {
		data := s.stream.Bytes()

		startIdx := bytes.Index(data, oscPrefix)
		if startIdx == -1 {
			partialIdx := -1
			for i := len(oscPrefix) - 1; i > 0; i-- {
				if len(data) >= i && bytes.Equal(data[len(data)-i:], oscPrefix[:i]) {
					partialIdx = len(data) - i
					break
				}
			}

			if partialIdx == -1 {
				if s.state == 2 {
					s.commandBuf.Write(data)
				}
				s.stream.Reset()
			} else {
				if s.state == 2 {
					s.commandBuf.Write(data[:partialIdx])
				}
				leftover := data[partialIdx:]
				s.stream.Reset()
				s.stream.Write(leftover)
			}
			break
		}

		afterPrefix := data[startIdx+len(oscPrefix):]

		stIdx := bytes.IndexByte(afterPrefix, '\x07')
		stLen := 1

		escBackslashIdx := bytes.Index(afterPrefix, []byte("\x1b\\"))
		if stIdx == -1 || (escBackslashIdx != -1 && escBackslashIdx < stIdx) {
			stIdx = escBackslashIdx
			stLen = 2
		}

		if stIdx == -1 {
			if s.state == 2 {
				s.commandBuf.Write(data[:startIdx])
			}
			leftover := data[startIdx:]
			s.stream.Reset()
			s.stream.Write(leftover)
			break
		}

		markerPayload := afterPrefix[:stIdx]
		totalMarkerLen := startIdx + len(oscPrefix) + stIdx + stLen

		if len(markerPayload) > 0 {
			markerType := markerPayload[0]

			if s.state == 2 {
				s.commandBuf.Write(data[:startIdx])
			}

			s.stream.Next(totalMarkerLen)

			switch markerType {
			case 'A':
				s.state = 1
			case 'B':
				s.state = 2
				s.commandBuf.Reset()
			case 'C':
				s.state = 3
				s.startTs = time.Now()
				if len(markerPayload) > 2 && markerPayload[1] == ';' {
					s.commandBuf.Reset()
					s.commandBuf.Write(markerPayload[2:])
				}
			case 'D':
				s.state = 0
				exitCode := 0
				if len(markerPayload) > 2 && markerPayload[1] == ';' {
					exitCode, _ = strconv.Atoi(string(markerPayload[2:]))
				}

				cmdText := Strip(s.commandBuf.Bytes())
				cmdText = strings.TrimSpace(cmdText)

				if cmdText != "" {
					_ = s.db.InsertCommand(cmdText, s.startTs, time.Now(), exitCode)
				}
				s.commandBuf.Reset()
			}
		} else {
			if s.state == 2 {
				s.commandBuf.Write(data[:startIdx+len(oscPrefix)])
			}
			s.stream.Next(startIdx + len(oscPrefix))
		}
	}

	return len(p), nil
}
