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

package of11

import (
	"github.com/contiv/libOpenflow/openflow/ofbase"
)

// OpenFlow11 implements the 1.1 OpenFlow protocol basic methods
var OpenFlow11 OpenFlow11Protocol

// OpenFlow11Protocol implements the basic methods for OpenFlow 1.1
type OpenFlow11Protocol struct {
}

// String returns the OpenFlow protocol version as a string
func (p OpenFlow11Protocol) String() string {
	return "OpenFlow 1.1"
}

// GetVersion returns the OpenFlow protocol wire version
func (p OpenFlow11Protocol) GetVersion() uint8 {
	return ofbase.VERSION_1_1
}

// NewHello returns a new hello message
func (p OpenFlow11Protocol) NewHello(versionBitmap uint32) ofbase.Message {
	return NewHello()
}

// NewEchoRequest returns a new echo request message
func (p OpenFlow11Protocol) NewEchoRequest() ofbase.Message {
	return NewEchoRequest()
}

// NewEchoReply returns a new echo reply message
func (p OpenFlow11Protocol) NewEchoReply() ofbase.Message {
	return NewEchoReply()
}

// NewBarrierRequest returns a new barrier request message
func (p OpenFlow11Protocol) NewBarrierRequest() ofbase.Message {
	return NewBarrierRequest()
}

// DecodeMessage parses an OpenFlow message
func (p OpenFlow11Protocol) DecodeMessage(data []byte) (ofbase.Message, error) {
	return DecodeMessage(data)
}
