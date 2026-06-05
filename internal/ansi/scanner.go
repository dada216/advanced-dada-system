package ansi

import (
	"bytes"
	"regexp"
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

var osc133Re = regexp.MustCompile(`\x1b\]133;([A-D])(?:;([0-9]+))?(?:\x07|\x1b\\)`)

func (s *OSCScanner) Write(p []byte) (n int, err error) {
	s.stream.Write(p)

	for {
		data := s.stream.Bytes()
		loc := osc133Re.FindIndex(data)
		if loc == nil {
			break
		}

		markerData := data[loc[0]:loc[1]]
		matches := osc133Re.FindSubmatch(markerData)
		markerType := string(matches[1])

		textBefore := data[:loc[0]]

		if s.state == 2 {
			s.commandBuf.Write(textBefore)
		}

		s.stream.Next(loc[1])

		switch markerType {
		case "A":
			s.state = 1 // prompt
		case "B":
			s.state = 2 // input
			s.commandBuf.Reset()
		case "C":
			s.state = 3 // execution
			s.startTs = time.Now()
		case "D":
			s.state = 0 // done
			exitCode := 0
			if len(matches[2]) > 0 {
				exitCode, _ = strconv.Atoi(string(matches[2]))
			}

			cmdText := Strip(s.commandBuf.Bytes())
			cmdText = strings.TrimSpace(cmdText)

			if cmdText != "" {
				_ = s.db.InsertCommand(cmdText, s.startTs, time.Now(), exitCode)
			}
			s.commandBuf.Reset()
		}
	}

	data := s.stream.Bytes()
	lastEsc := bytes.LastIndexByte(data, '\x1b')
	if lastEsc == -1 {
		if s.state == 2 {
			s.commandBuf.Write(data)
		}
		s.stream.Reset()
	} else {
		if s.state == 2 {
			s.commandBuf.Write(data[:lastEsc])
		}
		leftover := data[lastEsc:]
		s.stream.Reset()
		s.stream.Write(leftover)
	}

	return len(p), nil
}
