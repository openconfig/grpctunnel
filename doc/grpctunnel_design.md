# TCP over gRPC Tunnel

**Contributors**:
James Protzman, Carl Lebsack, Rob Shakir

**February, 2019**
*Updated*: April 2020

<!-- MarkdownTOC -->

- [TCP over gRPC Tunnel](#tcp-over-grpc-tunnel)
  - [Objective](#objective)
  - [Background](#background)
  - [Overview](#overview)
  - [Detailed Design](#detailed-design)
    - [Tunnel Proto Design](#tunnel-proto-design)
    - [Data Message Fields](#data-message-fields)
      - [Tag Field](#tag-field)
      - [Data Field](#data-field)
      - [Close Field](#close-field)
    - [Session Message Fields](#session-message-fields)
      - [Tag Field](#tag-field-1)
      - [Accept Field](#accept-field)
      - [Capabilities Field](#capabilities-field)
      - [Target ID Field](#target-id-field)
      - [Error Field](#error-field)
    - [Capabilities Message Fields](#capabilities-message-fields)
      - [Handler Field](#handler-field)
    - [Handlers](#handlers)
    - [Client](#client)
    - [Server](#server)

<!-- /MarkdownTOC -->


## Objective

To create a transparent, bi-directional TCP-over-gRPC tunnel.


## Background

gRPC is a service based strictly around a client-server model in order to send a
gRPC request. gRPC Requests are executed on the server, and the result(s) are
returned to the client. If the client is unable to reach the server for any
reason, then it is not possible to send the request.

Common reasons that a gRPC client would be unable to communicate with a server
include:

*   the server is behind a firewall with ACLs preventing inbound connections
*   the server is behind a router implementing address translation (NAT)
*   other operational requirements which prevent outside connections

See the figure below for an example:

![gRPC Request Is Blocked at the Firewall / Router](images/grpctunnel-blocked-example.png "gRPC Request Is Blocked at the Firewall / Router")

Additional references and discussion:

[gRPC tunneling (Github issue)](https://github.com/grpc/grpc/issues/14101)
[gRPC-in-gRPC tunneling](https://github.com/jhump/grpctunnel)

## Overview

In the cases outlined above, it could be preferable to instantiate a connection
in reverse (server to client) between the gRPC client and server rather than
modifying the infrastructure between them. However, there is no mechanism built
into the gRPC library which supports reverse connections.

This document proposes an approach which will allow a gRPC client and server to
communicate using a TCP over gRPC tunnel. The tunnel will support external
connections from either endpoint over TCP and forward them using gRPC streams.
It’s possible that this tunnel could be used to forward more than gRPC traffic.

An overview of how the tunnel works from both client and server sides follows:

**Server side**:

1.  A new session, identified by a unique string, is requested from the tunnel
    server.
1.  The tunnel server forwards a request through the register stream.
1.  If the tunnel client can handle this request, it will do 2 things:
    1.  create a new tunnel stream to the server, and
    1.  call the client defined handler on the stream.
1.  The tunnel server will accept and return the stream.

More detail can be found in the [server section](#server) below.

**Client side**:

1.  A new session, identified by a unique string, is requested from the tunnel
    client.
1.  The tunnel client forwards a request through the register stream.
1.  The tunnel server will send an accepted request back to the client on the
    register stream.
1.  The tunnel client will
    1.  create a new tunnel stream to the server, and
    1.  return the stream.
1.  The tunnel server will accept the stream, and call the client defined
    handler.

More detail can be found in the [client section](#client) below.

The figure below illustrates the communication:

![gRPC Client and Server Communicate Through Tunnel](images/grpctunnel-client-server.png "gRPC Client and Server Communicate Through Tunnel")

## Detailed Design

### Tunnel Proto Design

```
service Tunnel {
  rpc Register(stream Session) returns (stream Session);
  rpc Tunnel(stream Data) returns (stream Data);
}

message Data {
  int32 tag = 1;
  bytes data = 2;
  bool close = 3;
}

message Session {
  int32 tag = 1;
  bool accept = 2;
  Capabilities capabilities = 3;
  string target_id = 4;
  string error = 5;
}

message Capabilities {
  bool handler = 1;
}
```


**Tunnel protobuf definition**

The tunnel proto defines a gRPC service, `Tunnel`, which defines two rpc
methods:

1. a `register` method which allows the tunnel server to request new tunnel
streams from the client using[ session messages](#session-message-fields), and
2. a `tunnel` method, which is used to create new tunnel streams. These streams
forward the data from TCP streams over a bi-directional gRPC stream of [data
messages](#data-message-fields).

Each field is described in the next section.

### Data Message Fields

#### Tag Field

The tunnel `tag` field is used by the tunnel endpoints to ensure the correct
tunnel streams are used to forward the correct TCP client connections.

#### Data Field

The `data` field is used to encapsulate raw bytes received from the client code.
This data will be forwarded, unchanged through the tunnel endpoints, to the
proper destination. This allows the tunnel to potentially handle non-gRPC
connections.

#### Close Field

The `close` field is used by the tunnel endpoints to know when forwarding has
finished - usually signalled by the TCP connection reaching EOF or going idle.
Once this boolean is set, the client and server will clean up the associated
tunnel connections.

### Session Message Fields

#### Tag Field

The `tag` field in the register service is used by the tunnel endpoints to
request new tunnel streams for a certain tag.  This tag defines the set of tags
used by the [data message tag field](#tag-field). By default, the tunnel server
uses positive tags, and the tunnel client uses negative tags. They start at 1,
and -1, respectively to disambiguate 0.

#### Accept Field

The `accept` field is sent by the tunnel server to indicate that it added an
associated connection in its map. This allows a new tunnel session from the
client to be handled correctly.

#### Capabilities Field

`Capabilities` is a separate message to allow future extensions. It is defined
in more detail in the [next section](#capabilities-message-fields)

#### Target ID Field

The `target_id` field is used by the register handler on either the tunnel
server or client to define what the handler function can handle. This is
explained in more detail in the [handlers](#handler-field) sections below.

#### Error Field

The `error` field is used by the client and server to exchange errors
encountered when requesting new sessions.

### Capabilities Message Fields

Capabilities are exchanged between endpoints at start up to allow them to make
decisions based on their neighbors’ settings.

#### Handler Field

The `handler` field is true when the client defined register handler and handler
functions are present on the endpoint. If no handlers are defined (either
remotely, or locally), then the server will not run. Having handlers defined on
one side, will allow the server to run.

### Handlers

There are two handlers used by the tunnel: the register handler function, and
the handler function itself. The register handler function defines what the
handler function will accept. A handler function and register handler are
required on one end of the tunnel in order to forward traffic, as they handle
what is done with a stream on the opposing side of a new session request.

### Client

A new client will be created and started using the `NewClient`. When the client
is started, it will attempt to connect to the tunnel server over the register
gRPC service. If it is successful, the client and server exchange
[capabilities](#capabilities-message-fields). At this point, the server and
client will wait until they receive a request for a new session. The [server
section](#server) below will cover new session requests from the server
perspective.

When `NewSession` is called on the client, it will send a register request to
the server. The server will check if it can handle the register request via the
server register handler. This handler will return an error if it cannot handle
the requested session. If it can handle it, the server will send a register
session ack back to the client and the client will start a new tunnel stream,
using the tunnel gRPC service. At this time, all register requests sent to the
client should contain the accept message. The rationale is that the client will
only receive requests from the server which are ready to have a new tunnel
session created for them.

This stream is passed to the server handler function on the server, and returned
to `NewSession` on the client side. See the timing diagram below.

![Successful NewSession on Client](images/grpctunnel-client-newsession.png "Successful NewSession on Client")

### Server

A server will need to be created using `grpc.NewServer` and the exported
`NewServer` method from this tunnel package.  Once the server is listening, it
will accept a connection from a client and exchange capabilities. The new server
will then wait for new session requests.

When `NewSession` is called on the server, it will send a register request, with
an ack to the tunnel client. The client will then check if it can handle the
request, and if it can, it will create a new tunnel session to the server. The
tunnel session is then forwarded to the client handler function on the client,
and returned to the `NewSession` request on the server.