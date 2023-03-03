package testutils

import (
	"bytes"
	"fmt"
)

// This file provides an IO Reader and an IO Writer that simulate IO failures.
// This is intented to test error reporting

// FaultyBuffer is an [io.Reader] and [io.Writer] with similar functionality as [bytes.Buffer].
// After either reading or writing faultThreshold many bytes, it will generate a customizable IO error.
//
// This is intended to be used by tests to check correct error handling.
//
// Note that the zero value of FaultyBuffer is considered an invalid object (as it has no designated error) and will panic on usage.
// Use NewFaultyBuffer to create a valid FaultyBuffer
// Note that (similary to [bytes.Buffer]), a FaultyBuffer must not be copied after its first usage.
type FaultyBuffer struct {
	designatedErr  error
	faultThreshold int
	buf            bytes.Buffer
	alreadyRead    int
	alreadyWritten int
}

// Read is provided to satify the [io.Reader] interface.
// After reading a total of faultThreshold bytes (and on subsequent read attempt of >0 bytes), we return the designated error.
func (fb *FaultyBuffer) Read(p []byte) (n int, err error) {
	if fb.designatedErr == nil {
		panic("FaultyBuffer without designated error")
	}
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

// Write is provided to satisfy the [io.Writer] interface
// After writing a total of faultThreshold bytes (and on subsequent write attempts of >0 bytes), we return the designated error.
func (fb *FaultyBuffer) Write(p []byte) (n int, err error) {
	if fb.designatedErr == nil {
		panic("FaultyBuffer without designated error")
	}
	if len(p) == 0 {
		return 0, nil
	}
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

// Reset resets the buffer. This clears whatever was written to the buffer and we can again write/read until faultThreshold is reached.
//
// The designated error is kept.
func (fb *FaultyBuffer) Reset() {
	if fb.designatedErr == nil {
		panic("FaultyBuffer without designated error")
	}
	fb.buf.Reset()
	fb.alreadyRead = 0
	fb.alreadyWritten = 0
}

// NewFaultyBuffer creates a new (pointer to) a [FaultyBuffer] with the given fault threshold and non-nil designated error.
//
// The result behaves similar to a [bytes.Buffer], but after *either* reading or writing faultThreshold many bytes (separate counters)
// we return the designated error as an IO error. This is intended to test error handling.
//
// Note: To trigger the faulty behaviour on reads, enough bytes must be present in the buffer beforehand.
// These cannot be written by Write, since that would trigger the faulty behaviour already on Write.
// Use the [SetContent] method to fill the FaultyBuffer.
//
// Calling NewFaultyBuffer with a nil designatedError is a bug and causes this function to panic.
func NewFaultyBuffer(faultThreshold int, designatedError error) *FaultyBuffer {
	if designatedError == nil {
		panic("Called NewFaultyBuffer with nil designated error") // This would result in a FaultyBuffer that would just stop after a certain point w/o indicating its error, breaking the io.Reader / io.Writer contract.
	}
	var fb FaultyBuffer // zero-initialize buf, alreadyRead, alreadyWritten
	fb.designatedErr = designatedError
	fb.faultThreshold = faultThreshold
	return &fb
}

// SetContent resets the buffer and sets its content (for reading) to content. Note that content's length may be larger than the fault threshold.
// The [FaultyBuffer] will trigger an IO error after reading faultThreshold bytes or writing faultThreshold *additional* bytes (separate counters).
func (fb *FaultyBuffer) SetContent(content []byte) {
	if fb.designatedErr == nil {
		panic("FaultyBuffer without designated error")
	}
	fb.Reset()
	L, err := fb.buf.Write(content)
	if err != nil {
		panic(fmt.Errorf("SetContent failed with error %v", err))
	}
	if L != len(content) {
		panic("Should be unreachable")
	}
}

// Bytes returns the underlying buffer's Bytes.
// This is a view into the unread portion, used to (readonly) look at what has been written.
func (fb *FaultyBuffer) Bytes() []byte {
	if fb.designatedErr == nil {
		panic("FaultyBuffer without designated error")
	}
	return fb.buf.Bytes()
}
