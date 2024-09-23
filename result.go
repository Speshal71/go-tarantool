package tarantool

import (
	"fmt"
	"github.com/tinylib/msgp/msgp"
)

type Result struct {
	ErrorCode uint
	Error     error
	Data      [][]interface{}
	DataBytes []byte

	marshaller msgp.Marshaler
}

func (r *Result) GetCommandID() uint {
	if r.Error != nil {
		return r.ErrorCode | ErrorFlag
	}
	return r.ErrorCode
}

// MarshalMsg implements msgp.Marshaler
func (r *Result) MarshalMsg(b []byte) (o []byte, err error) {
	if r.marshaller == nil {
		r.marshaller = defaultResultMarshaller{Result: r}
	}
	return r.marshaller.MarshalMsg(b)
}

func (r *Result) WithBytesMarshaller() *Result {
	r.marshaller = bytesResultMarshaller{Result: r}
	return r
}

type defaultResultMarshaller struct {
	*Result
}

func (r defaultResultMarshaller) MarshalMsg(b []byte) (o []byte, err error) {
	o = b
	if r.Error != nil {
		o = msgp.AppendMapHeader(o, 1)
		o = msgp.AppendUint(o, KeyError)
		o = msgp.AppendString(o, r.Error.Error())
	} else {
		o = msgp.AppendMapHeader(o, 1)
		o = msgp.AppendUint(o, KeyData)
		if r.Data != nil {
			if o, err = msgp.AppendIntf(o, r.Data); err != nil {
				return nil, err
			}
		} else {
			o = msgp.AppendArrayHeader(o, 0)
		}
	}

	return o, nil
}

type bytesResultMarshaller struct {
	*Result
}

func (r bytesResultMarshaller) MarshalMsg(b []byte) (o []byte, err error) {
	o = b
	if r.Error != nil {
		o = msgp.AppendMapHeader(o, 1)
		o = msgp.AppendUint(o, KeyError)
		o = msgp.AppendString(o, r.Error.Error())
	} else {
		o = msgp.AppendMapHeader(o, 1)
		o = msgp.AppendUint(o, KeyData)
		if len(r.DataBytes) != 0 {
			o = append(o, r.DataBytes...)
		} else {
			o = msgp.AppendArrayHeader(o, 0)
		}
	}

	return o, nil
}

// UnmarshalMsg implements msgp.Unmarshaler
func (r *Result) UnmarshalMsg(data []byte) (buf []byte, err error) {
	var l uint32
	var dl, tl uint32
	var errorMessage string
	var val interface{}

	buf = data

	// Tarantool >= 1.7.7 sends periodic heartbeat messages without body
	if len(buf) == 0 && r.ErrorCode == OKCommand {
		return buf, nil
	}
	l, buf, err = msgp.ReadMapHeaderBytes(buf)

	if err != nil {
		return
	}

	for ; l > 0; l-- {
		var cd uint

		if cd, buf, err = msgp.ReadUintBytes(buf); err != nil {
			return
		}

		switch cd {
		case KeyData:
			var i, j uint32

			bufData := buf

			if dl, buf, err = msgp.ReadArrayHeaderBytes(buf); err != nil {
				return
			}

			r.Data = make([][]interface{}, dl)
			for i = 0; i < dl; i++ {
				obuf := buf
				if tl, buf, err = msgp.ReadArrayHeaderBytes(buf); err != nil {
					buf = obuf
					if _, ok := err.(msgp.TypeError); ok {
						if val, buf, err = msgp.ReadIntfBytes(buf); err != nil {
							return
						}
						r.Data[i] = []interface{}{val}
						continue
					}
					return
				}

				r.Data[i] = make([]interface{}, tl)
				for j = 0; j < tl; j++ {
					if r.Data[i][j], buf, err = msgp.ReadIntfBytes(buf); err != nil {
						return
					}
				}
			}

			bufRead := len(bufData) - len(buf)
			bufData = bufData[:bufRead]
			r.DataBytes = make([]byte, len(bufData))
			copy(r.DataBytes, bufData)

		case KeyError:
			errorMessage, buf, err = msgp.ReadStringBytes(buf)
			if err != nil {
				return
			}
			r.Error = NewQueryError(r.ErrorCode, errorMessage)

		default:
			if buf, err = msgp.Skip(buf); err != nil {
				return
			}
		}
	}

	return
}

func (r *Result) String() string {
	switch {
	case r == nil:
		return "Result <nil>"
	case r.Error != nil:
		return fmt.Sprintf("Result ErrCode:%v, Err: %v", r.ErrorCode, r.Error)
	case r.Data != nil:
		return fmt.Sprintf("Result Data:%#v", r.Data)
	default:
		return ""
	}
}
