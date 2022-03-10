package qerrors

import "fmt"

type Error struct {
	Code int32
	Msg  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s(%d)", e.Msg, e.Code)
}

func (e *Error) ErrCode() int32 {
	return e.Code
}

func (e *Error) ErrMsg() string {
	return e.Msg
}

func (e *Error) SetMsg(msg string) error {
	if msg != "" {
		return &Error{e.Code, msg}
	}

	return e
}

func (e *Error) SetError(err error) error {
	if err != nil {
		return &Error{e.Code, err.Error()}
	}

	return e
}

func (e *Error) AddMsg(errMsg string) error {
	if errMsg != "" {
		return &Error{e.Code, e.Msg + ":" + errMsg}
	}

	return e
}

func (e *Error) AddError(err error) error {
	if err != nil {
		return &Error{e.Code, e.Msg + ":" + err.Error()}
	}

	return err
}
