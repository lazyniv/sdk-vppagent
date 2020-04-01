// Copyright (c) 2020 Cisco and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package xconnectns provides an Endpoint that implements the cross connect networks service for use as a Forwarder
package xconnectns

import (
	"net"
	"net/url"

	"github.com/open-policy-agent/opa/rego"
	"go.ligato.io/vpp-agent/v3/proto/ligato/configurator"

	"github.com/networkservicemesh/sdk-vppagent/pkg/networkservice/mechanisms/directmemif"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk-vppagent/pkg/networkservice/commit"
	"github.com/networkservicemesh/sdk-vppagent/pkg/networkservice/connectioncontext/ipcontext/ipaddress"
	"github.com/networkservicemesh/sdk-vppagent/pkg/networkservice/connectioncontextkernel/ipcontext/routes"
	"github.com/networkservicemesh/sdk-vppagent/pkg/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/sdk-vppagent/pkg/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/sdk-vppagent/pkg/networkservice/mechanisms/srv6"
	"github.com/networkservicemesh/sdk-vppagent/pkg/networkservice/mechanisms/vxlan"
	"github.com/networkservicemesh/sdk-vppagent/pkg/networkservice/vppagent"
	"github.com/networkservicemesh/sdk-vppagent/pkg/networkservice/xconnect/l2xconnect"
)

type xconnectNSServer struct {
	endpoint.Endpoint
}

// NewServer - returns a new vppagent based Endpoint implementing the XConnect Network Service for use as a Forwarder
//             name - name of the Forwarder
//             authzPolicy - policy for allowing or rejecting requests
//             vppagentCC - grpc.ClientConnInterface of the vppagent
//             baseDir - baseDir for sockets
//             tunnelIP - IP we can use for originating and terminating tunnels
//             vxlanInitFunc - function to perform initial configuration of vppagent
//             clientUrl - *url.URL for the talking to the NSMgr
//             ...clientDialOptions - dialOptions for dialing the NSMgr
func NewServer(name string, authzPolicy *rego.PreparedEvalQuery, vppagentCC grpc.ClientConnInterface, baseDir string, tunnelIP net.IP, vxlanInitFunc func(conf *configurator.Config) error, clientURL *url.URL, clientDialOptions ...grpc.DialOption) endpoint.Endpoint {
	rv := xconnectNSServer{}
	rv.Endpoint = endpoint.NewServer(
		name,
		authzPolicy,
		// Make sure we have a fresh empty config for everyone in the chain to use
		vppagent.NewServer(),
		directmemif.NewServer(),
		// Preference ordered list of mechanisms we support for incoming connections
		memif.NewServer(baseDir),
		kernel.NewServer(),
		vxlan.NewServer(tunnelIP, vxlanInitFunc),
		srv6.NewServer(),
		// Statically set the url we use to the unix file socket for the NSMgr
		clienturl.NewServer(clientURL),
		connect.NewServer(client.NewClientFactory(
			name,
			// What to call onHeal
			addressof.NetworkServiceClient(adapters.NewServerToClient(rv)),
			// Preference ordered list of mechanisms we support for outgoing connections
			memif.NewClient(baseDir),
			kernel.NewClient(),
			vxlan.NewClient(tunnelIP, vxlanInitFunc),
			srv6.NewClient(),
			// l2 cross connect (xconnect) between incoming and outgoing connections
			// TODO - properly support l3xconnect for IP payload
			l2xconnect.NewClient()),
			clientDialOptions...,
		),
		ipaddress.NewServer(),
		routes.NewServer(),
		commit.NewServer(vppagentCC),
	)
	return rv
}
