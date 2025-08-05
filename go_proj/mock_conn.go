package main

import (
	"log"

	"github.com/pion/webrtc/v3/pkg/media"
)

type MockConn struct {
	onValidated   func()
	onMessage     func(message interface{}, msgObj interface{})
	onOpen        func()
	onVideoSample func(sample media.Sample) error
}

func NewMockConn(onValidated func(), onMessage func(message interface{}, msgObj interface{}), onOpen func(), onVideoSample func(sample media.Sample) error) *MockConn {
	return &MockConn{
		onValidated:   onValidated,
		onMessage:     onMessage,
		onOpen:        onOpen,
		onVideoSample: onVideoSample,
	}
}

func (conn *MockConn) Connect(ip, token string) error {
	log.Printf("🤖 MockConn Connect: %s, %s", ip, token)
	conn.onValidated()
	conn.onOpen()

	go startRTMPServer(conn.onVideoSample)
	return nil
}

func (conn *MockConn) Close() error {
	log.Printf("🤖 MockConn Close")
	return nil
}

func (conn *MockConn) SendCommand(command string, data interface{}) {
	log.Printf("🤖 MockConn SendCommand: %s, %v", command, data)
}
