/*
 * Copyright (C) 2018 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy ofthe License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specificlanguage governing permissions and
 * limitations under the License.
 *
 */

package openflow

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/contiv/libOpenflow/openflow/of10"
	"github.com/contiv/libOpenflow/openflow/of11"
	"github.com/contiv/libOpenflow/openflow/of12"
	"github.com/contiv/libOpenflow/openflow/of13"
	"github.com/contiv/libOpenflow/openflow/of14"
	"github.com/contiv/libOpenflow/openflow/of15"
	"github.com/contiv/libOpenflow/openflow/ofbase"
)

const (
	echoDuration = 3
)

var (
	// ErrContextDone is returned what the context was done or canceled
	ErrContextDone = errors.New("Context was terminated")
	// ErrConnectionTimeout is returned when a timeout was reached when trying to connect
	ErrConnectionTimeout = errors.New("Timeout while connecting")
	// ErrReaderChannelClosed is returned when the read channel was closed
	ErrReaderChannelClosed = errors.New("Reader channel was closed")
)

// Client describes an OpenFlow client
type Client struct {
	sync.RWMutex
	conn               net.Conn
	addr               string
	tlsConfig          *tls.Config
	reader             *bufio.Reader
	msgChan            chan (ofbase.Message)
	listeners          []Listener
	xid                uint32
	protocol           ofbase.Protocol
	supportedProtocols []ofbase.Protocol
}

// Listener defines the interface implemented by monitor listeners
type Listener interface {
	OnMessage(ofbase.Message)
}

func (c *Client) connect(addr string) (net.Conn, error) {
	var protocol string

	parts := strings.SplitN(addr, ":", 2)
	if len(parts) > 1 {
		protocol = parts[0]
		addr = parts[1]
	} else {
		protocol = "unix"
		if !strings.HasPrefix(addr, "/") {
			addr = fmt.Sprintf("/var/run/openvswitch/%s.mgmt", addr)
		}
	}

	switch protocol {
	case "tcp":
		return net.Dial(protocol, addr)
	case "ssl":
		return tls.Dial("tcp", addr, c.tlsConfig)
	case "unix":
		raddr, err := net.ResolveUnixAddr("unix", addr)
		if err != nil {
			return nil, err
		}
		return net.DialUnix("unix", nil, raddr)
	default:
		return nil, fmt.Errorf("Unsupported connection scheme '%s'", protocol)
	}
}

func (c *Client) handshake() (ofbase.Protocol, error) {
	var ownBitmap uint32

	protocol := c.supportedProtocols[len(c.supportedProtocols)-1]
	for _, supportedProtocol := range c.supportedProtocols {
		ownBitmap |= 1 << supportedProtocol.GetVersion()
	}

	if err := c.SendMessage(protocol.NewHello(ownBitmap)); err != nil {
		return nil, err
	}

	header, data, err := c.readMessage()
	if err != nil {
		return nil, err
	}

	if header.Type != ofbase.OFPTHello {
		return nil, fmt.Errorf("Expected a first message of type Hello")
	}

	switch {
	case header.Version == protocol.GetVersion():
		return protocol, nil
	case header.Version < protocol.GetVersion():
		for _, protocol := range c.supportedProtocols {
			if header.Version == protocol.GetVersion() {
				return protocol, nil
			}
		}
	case header.Version > protocol.GetVersion():
		// Since OpenFlow 1.3, Hello message can include bitmaps of the supported versions.
		// If this bitmap is provided, the negotiated version is the highest one supported
		// by both sides
		if header.Version >= ofbase.VERSION_1_3 && len(data) > 8 {
			if msg, err := of13.DecodeHello(nil, ofbase.NewDecoder(data[8:])); err == nil {
				for _, element := range msg.GetElements() {
					if peerBitmaps, ok := element.(*of13.HelloElemVersionbitmap); ok && len(peerBitmaps.GetBitmaps()) > 0 {
						peerBitmap := peerBitmaps.GetBitmaps()[0].Value
						for i := uint8(31); i >= 0; i-- {
							if peerBitmap&(1<<i) != 0 {
								for _, supportedProtocol := range c.supportedProtocols {
									if i == supportedProtocol.GetVersion() {
										log.Printf("Negotiated version %d", i)
										return protocol, nil
									}
								}
							}
						}
					}
				}
			}
		} else {
			// Otherwise, the negotiated version is the lowest version
			return protocol, nil
		}
	}

	return nil, fmt.Errorf("Unsupported protocol version %d", protocol.GetVersion())
}

func (c *Client) handleLoop(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	echoTicker := time.NewTicker(time.Second * echoDuration)
	defer echoTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Context was cancelled")
			return ErrContextDone
		case <-echoTicker.C:
			c.SendEcho()
		case msg, ok := <-c.msgChan:
			if !ok {
				log.Println(ErrReaderChannelClosed)
				return ErrReaderChannelClosed
			}

			c.dispatchMessage(msg)

			if msg.MessageType() == ofbase.OFPTEchoRequest {
				c.SendMessage(c.protocol.NewEchoReply())
			}
		}
	}
}

func (c *Client) dispatchMessage(msg ofbase.Message) {
	c.RLock()
	for _, listener := range c.listeners {
		listener.OnMessage(msg)
	}
	c.RUnlock()
}

func (c *Client) readMessage() (*ofbase.Header, []byte, error) {
	c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
	data, err := c.reader.Peek(8)
	if err != nil {
		return nil, nil, err
	}

	header := &ofbase.Header{}
	if err := header.Decode(ofbase.NewDecoder(data)); err != nil {
		return nil, nil, err
	}

	data = make([]byte, header.Length)
	_, err = io.ReadFull(c.reader, data)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to read full OpenFlow message: %s", err)
	}

	return header, data, nil
}

func (c *Client) readLoop() {
	for {
		_, data, err := c.readMessage()
		if err != nil {
			if err != io.EOF {
				log.Println(err)
			}
			return
		}

		msg, err := c.protocol.DecodeMessage(data)
		if err != nil {
			log.Printf("Failed to decode message on bridge %s with %s: %s", c.addr, err, c.protocol)
			continue
		}

		c.msgChan <- msg
	}
}

type barrier struct {
	c chan ofbase.Message
}

// OnMessage is called when an OpenFlow message is received
func (b *barrier) OnMessage(msg ofbase.Message) {
	if msg.MessageName() == "OFPTBarrierReply" {
		b.c <- msg
	}
}

// PrepareMessage set the message xid and increment it
func (c *Client) PrepareMessage(msg ofbase.Message) {
	msg.SetXid(atomic.AddUint32(&c.xid, 1))
}

// SendMessage sends a message to the switch
func (c *Client) SendMessage(msg ofbase.Message) error {
	if msg.GetXid() == 0 {
		c.PrepareMessage(msg)
	}

	isBarrier := msg.MessageName() == "OFPTBarrierRequest"
	encoder := ofbase.NewEncoder()

	if err := msg.Serialize(encoder); err != nil {
		return err
	}

	if isBarrier {
		b := &barrier{c: make(chan ofbase.Message, 1)}
		c.RegisterListener(b)

		_, err := c.conn.Write(encoder.Bytes())
		if err == nil {
			<-b.c
		}
		return nil
	}

	_, err := c.conn.Write(encoder.Bytes())
	return err
}

// SendEcho sends an OpenFlow echo message
func (c *Client) SendEcho() error {
	return c.SendMessage(c.protocol.NewEchoRequest())
}

// RegisterListener registers a new listener of the received messages
func (c *Client) RegisterListener(listener Listener) {
	c.Lock()
	defer c.Unlock()

	c.listeners = append(c.listeners, listener)
}

// Start monitoring the OpenFlow bridge
func (c *Client) Start(ctx context.Context) (err error) {
	c.conn, err = c.connect(c.addr)
	if err != nil {
		return err
	}

	c.reader = bufio.NewReader(c.conn)

	c.protocol, err = c.handshake()
	if err != nil {
		return err
	}

	go c.readLoop()
	go c.handleLoop(ctx)

	log.Printf("Successfully connected to OpenFlow switch %s using version %d", c.addr, c.protocol.GetVersion())

	return nil
}

// Stop the client
func (c *Client) Stop() error {
	return nil
}

// GetProtocol returns the current protocol
func (c *Client) GetProtocol() ofbase.Protocol {
	return c.protocol
}

// NewClient returns a new OpenFlow client using either a UNIX socket or a TCP socket
func NewClient(addr string, tlsConfig *tls.Config, protocols []ofbase.Protocol) (*Client, error) {
	if len(protocols) == 0 {
		protocols = []ofbase.Protocol{of10.OpenFlow10, of11.OpenFlow11,
			of12.OpenFlow12, of13.OpenFlow13, of14.OpenFlow14, of15.OpenFlow15}
	}
	client := &Client{
		addr:               addr,
		tlsConfig:          tlsConfig,
		msgChan:            make(chan ofbase.Message, 500),
		supportedProtocols: protocols,
	}
	return client, nil
}
