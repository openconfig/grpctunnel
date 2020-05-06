# Successful New Session On Client

[TOC]

<!--*
# Document freshness: For more information, see go/fresh-source.
freshness: { owner: 'jprotzman' reviewed: '2019-03-08' }
*-->

```sequence-diagram
Title: Successful NewSession on Client
participant NewSession
Client-->Server: gRPC register
Server-->Client: gRPC register
Client->Server: Client Capabilities
Server->Client: Server Capabilities
NewSession->Client: NewSession Request
Client->Server: Register Session
participant ServerHandlerFunc
Server->ServerRegHandlerFunc: Can I handle this request?
ServerRegHandlerFunc->Server: Yes
Server->Client: Register Session (ack)
Client->Server: New Tunnel Session
Server->Client: Tunnel Session
Server->ServerHandlerFunc: Tunnel Session
Client->NewSession: Tunnel Session
```
