package p2p

import (
	"github.com/line/ostracon/libs/service"
	"github.com/line/ostracon/p2p/conn"
)

// Reactor is responsible for handling incoming messages on one or more
// Channel. Switch calls GetChannels when reactor is added to it. When a new
// peer joins our node, InitPeer and AddPeer are called. RemovePeer is called
// when the peer is stopped. Receive is called when a message is received on a
// channel associated with this reactor.
//
// Peer#Send or Peer#TrySend should be used to send the message to a peer.
type Reactor interface {
	service.Service // Start, Stop

	// SetSwitch allows setting a switch.
	SetSwitch(*Switch)

	// GetChannels returns the list of MConnection.ChannelDescriptor. Make sure
	// that each ID is unique across all the reactors added to the switch.
	GetChannels() []*conn.ChannelDescriptor

	// InitPeer is called by the switch before the peer is started. Use it to
	// initialize data for the peer (e.g. peer state).
	//
	// NOTE: The switch won't call AddPeer nor RemovePeer if it fails to start
	// the peer. Do not store any data associated with the peer in the reactor
	// itself unless you don't want to have a state, which is never cleaned up.
	InitPeer(peer Peer) Peer

	// AddPeer is called by the switch after the peer is added and successfully
	// started. Use it to start goroutines communicating with the peer.
	AddPeer(peer Peer)

	// RemovePeer is called by the switch when the peer is stopped (due to error
	// or other reason).
	RemovePeer(peer Peer, reason interface{})

	// Receive is called by the switch when msgBytes is received from the peer.
	//
	// NOTE reactor can not keep msgBytes around after Receive completes without
	// copying.
	//
	// CONTRACT: msgBytes are not nil.
	Receive(chID byte, peer Peer, msgBytes []byte)

	// receive async version
	GetRecvChan() chan *BufferedMsg

	// receive routine per reactor
	RecvRoutine()
}

//--------------------------------------

type BaseReactor struct {
	service.BaseService // Provides Start, Stop, .Quit
	Switch              *Switch
	recvMsgBuf          chan *BufferedMsg
	impl                Reactor
}

func NewBaseReactor(name string, impl Reactor, async bool, recvBufSize int) *BaseReactor {
	baseReactor := &BaseReactor{
		BaseService: *service.NewBaseService(nil, name, impl),
		Switch:      nil,
		impl:        impl,
	}
	if async {
		baseReactor.recvMsgBuf = make(chan *BufferedMsg, recvBufSize)
	}
	return baseReactor
}

func (br *BaseReactor) SetSwitch(sw *Switch) {
	br.Switch = sw
}
func (*BaseReactor) GetChannels() []*conn.ChannelDescriptor        { return nil }
func (*BaseReactor) AddPeer(peer Peer)                             {}
func (*BaseReactor) RemovePeer(peer Peer, reason interface{})      {}
func (*BaseReactor) Receive(chID byte, peer Peer, msgBytes []byte) {}
func (*BaseReactor) InitPeer(peer Peer) Peer                       { return peer }

func (br *BaseReactor) OnStart() error {
	if br.recvMsgBuf != nil {
		// if it is async mode it starts RecvRoutine()
		go br.RecvRoutine()
	}
	return nil
}

func (br *BaseReactor) RecvRoutine() {
	for {
		select {
		case msg := <-br.recvMsgBuf:
			br.impl.Receive(msg.ChID, msg.Peer, msg.Msg)
		case <-br.Quit():
			return
		}
	}
}

func (br *BaseReactor) GetRecvChan() chan *BufferedMsg {
	if br.recvMsgBuf == nil {
		panic("It's not async reactor, but GetRecvChan() is called ")
	}
	return br.recvMsgBuf
}
