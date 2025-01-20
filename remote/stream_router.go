package remote

import (
	"log/slog"

	"github.com/anthdm/hollywood/actor"
)

type streamDeliver struct {
	sender *actor.PID
	target *actor.PID
	msg    any
}

type streamRouter struct {
	engine *actor.Engine
	// streams is a map of remote address to stream writer pid.
	streams map[string]*actor.PID
	pid     *actor.PID
	Config
}

func newStreamRouter(e *actor.Engine, config Config) actor.Producer {
	return func() actor.Receiver {
		return &streamRouter{
			streams: make(map[string]*actor.PID),
			engine:  e,
			Config:  config,
		}
	}
}

func (s *streamRouter) Receive(ctx *actor.Context) {
	switch msg := ctx.Message().(type) {
	case actor.Started:
		s.pid = ctx.PID()
	case *streamDeliver:
		s.deliverStream(msg)
	case actor.RemoteUnreachableEvent:
		s.handleTerminateStream(msg)
	}
}

func (s *streamRouter) handleTerminateStream(msg actor.RemoteUnreachableEvent) {
	streamWriterPID := s.streams[msg.ListenAddr]
	delete(s.streams, msg.ListenAddr)
	slog.Debug("stream terminated",
		"remote", msg.ListenAddr,
		"pid", streamWriterPID,
	)
}

func (s *streamRouter) deliverStream(msg *streamDeliver) {
	var (
		swpid   *actor.PID
		ok      bool
		address = msg.target.Address
	)

	swpid, ok = s.streams[address]
	if !ok {
		swpid = s.engine.SpawnProc(newStreamWriter(s.engine, s.pid, address, s.Config))
		s.streams[address] = swpid
	}

	s.engine.Send(swpid, msg)
}
