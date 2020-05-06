# Failed New Session On Server

[TOC]

<!--*
# Document freshness: For more information, see go/fresh-source.
freshness: { owner: 'jprotzman' reviewed: '2019-03-15' }
*-->

```sequence-diagram
Title: Failed NewSession on Server
participant NewSession
participant Server
Client-->Server: gRPC register
Server-->Client: gRPC register
Client->Server: Client Capabilities
Server->Client: Server Capabilities
NewSession->Server: NewSession Request
Server->Client: Register Session
Client->Server: Session Request Error
Server->NewSession: Error
```
