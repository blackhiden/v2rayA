package main

import (
	"encoding/json"
	"io"
	"os"
	"time"

	xlog "github.com/xtls/xray-core/app/log"
	"github.com/xtls/xray-core/common"
	clog "github.com/xtls/xray-core/common/log"
	"github.com/xtls/xray-core/common/serial"
)

const liveFlowEventPrefix = "V2RAYA_LIVE_FLOW "

// liveFlowEvent is emitted to stdout for v2rayA consumption when V2RAYA_LIVE_FLOW=1.
type liveFlowEvent struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Detour      string `json:"detour,omitempty"`
	Timestamp   int64  `json:"timestamp"`
}

type liveFlowWrapper struct {
	inner  clog.Handler
	output io.Writer
}

func (w *liveFlowWrapper) Handle(msg clog.Message) {
	if accessMsg, ok := msg.(*clog.AccessMessage); ok {
		if os.Getenv("V2RAYA_LIVE_FLOW") == "1" && accessMsg != nil && accessMsg.Status == clog.AccessAccepted {
			source := serial.ToString(accessMsg.From)
			destination := serial.ToString(accessMsg.To)
			detour := accessMsg.Detour
			if source != "" && destination != "" && len(source) <= 512 && len(destination) <= 512 && len(detour) <= 512 {
				event := liveFlowEvent{
					Source:      source,
					Destination: destination,
					Detour:      detour,
					Timestamp:   time.Now().UnixMilli(),
				}
				if encoded, err := json.Marshal(event); err == nil {
					out := w.output
					if out == nil {
						out = os.Stdout
					}
					line := append(append([]byte(liveFlowEventPrefix), encoded...), '\n')
					_, _ = out.Write(line)
				}
			}
		}
	}
	if w.inner != nil {
		w.inner.Handle(msg)
	}
}

func init() {
	common.Must(xlog.RegisterHandlerCreator(xlog.LogType_Console, func(lt xlog.LogType, options xlog.HandlerCreatorOptions) (clog.Handler, error) {
		return &liveFlowWrapper{inner: clog.NewLogger(clog.CreateStdoutLogWriter())}, nil
	}))

	common.Must(xlog.RegisterHandlerCreator(xlog.LogType_File, func(lt xlog.LogType, options xlog.HandlerCreatorOptions) (clog.Handler, error) {
		creator, err := clog.CreateFileLogWriter(options.Path)
		if err != nil {
			return nil, err
		}
		return &liveFlowWrapper{inner: clog.NewLogger(creator)}, nil
	}))

	common.Must(xlog.RegisterHandlerCreator(xlog.LogType_None, func(lt xlog.LogType, options xlog.HandlerCreatorOptions) (clog.Handler, error) {
		if os.Getenv("V2RAYA_LIVE_FLOW") == "1" {
			return &liveFlowWrapper{inner: nil}, nil
		}
		return nil, nil
	}))
}
