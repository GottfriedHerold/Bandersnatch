package testutils

import (
	"bytes"
	"fmt"
)

// This file provides an IO Reader and an IO Writer that simulate IO failures.
// This is intented to test error reporting

type FaultyBuffer struct {
	designatedErr  error
	faultThreshold int
	buf            bytes.Buffer
	alreadyRead    int
	alreadyWritten int
}

func (fb *FaultyBuffer) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if fb.alreadyRead > fb.faultThreshold {
		err = fmt.Errorf("repeated read call to already faulty reader, error %w", fb.designatedErr)
		return 0, err
	}
	L := len(p)
	fault := false
	if fb.alreadyRead+L > fb.faultThreshold {
		fault = true
		L = fb.faultThreshold - fb.alreadyRead
	}

	n, err = fb.buf.Read(p[0:L])
	fb.alreadyRead += n
	if err != nil {
		return
	}
	if fault {
		err = fb.designatedErr
		fb.alreadyRead += 1 // to differentiate repeated calls
	}
	return
}

func (fb *FaultyBuffer) Write(p []byte) (n int, err error) {
	if fb.alreadyWritten > fb.faultThreshold {
		err = fmt.Errorf("repeated write call to already faulty writer, error %w", fb.designatedErr)
		return 0, err
	}
	L := len(p)
	fault := false
	if fb.alreadyWritten+L > fb.faultThreshold {
		fault = true
		L = fb.faultThreshold - fb.alreadyWritten
	}
	n, err = fb.buf.Write(p[0:L])
	fb.alreadyWritten += n
	if err != nil {
		return
	}
	if fault {
		err = fb.designatedErr
		fb.alreadyWritten += 1
	}
	return
}

func (fb *FaultyBuffer) Reset() {
	fb.buf.Reset()
	fb.alreadyRead = 0
	fb.alreadyWritten = 0
}

func NewFaultyBuffer(faultThreshold int, designatedError error) *FaultyBuffer {
	if designatedError == nil {
		panic("Called NewFaultyBuffer with nil designated error") // This would result in a FaultyBuffer that would just stop after a certain point w/o indicating its error, breaking the io.Reader / io.Writer contract.
	}
	var fb FaultyBuffer // zero-initialize buf, alreadyRead, alreadyWritten
	fb.designatedErr = designatedError
	fb.faultThreshold = faultThreshold
	return &fb
}
