# Successful New Session On Server

[TOC]

<!--*
# Document freshness: For more information, see go/fresh-source.
freshness: { owner: 'jprotzman' reviewed: '2019-03-15' }
*-->

```sequence-diagram
Title: Successful NewSession on Server
participant NewSession
participant Server
Client-->Server: gRPC Register
Server-->Client: gRPC Register
Client->Server: Client Capabilities
Server->Client: Server Capabilities
NewSession->Server: NewSession Request
Server->Client: Register Session (ack)
participant ClientHandlerFunc
Client->ClientRegHandlerFunc: Can I handle request?
ClientRegHandlerFunc->Client: Yes
Client->Server: New Tunnel Session
Server->Client: Tunnel Session
Client->ClientHandlerFunc: Tunnel Session
Server->NewSession: Tunnel Session
```
