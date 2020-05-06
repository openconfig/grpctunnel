# Failed New Session On Client

[TOC]

<!--*
# Document freshness: For more information, see go/fresh-source.
freshness: { owner: 'jprotzman' reviewed: '2019-03-15' }
*-->

```sequence-diagram
Title: Failed NewSession on Client
participant NewSession
Client-->Server: gRPC register
Server-->Client: gRPC register
Client->Server: client capabilities
Server->Client: server capabilities
NewSession->Client: NewSession request
Client->Server: Register Session
Server->Client: Register Session error
Client->NewSession: error
```
