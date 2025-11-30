# fabric offline signer cli

to foster better understand of how fabric-x offline signing is handled ,
a fabric interactive command line interface of client application for
offline signing is drafted .

# key materials

- a public x.509 certificate ( signcerts ) .
- a public tls ca root certificate ( tls/ca ) .
- a private key stored in a local drive ( keystore ) .
- a grpc connection host:port mapping .
- a msp id .

a client-side x.509 certificate can be generated using a locally generated keypair for
certificate signing request of the fabric-ca-client . then the certificate can be used
for validation and the private key can be used for offline signing context . note that
a client-side tls ca certificate could also be generated and validated using the form
of certificate signing request but it is optional for tls connection .

for testing purpose of offline signer cli of hyperledger fabric , a pre-stored key
materials is used instead of the process above to create the keys and certificates 
from scratch .

# offline signer

an offline signer could be implemented in any language , such as a node.js singer :

```js
// Source - https://stackoverflow.com/a
// Posted by bestbeforetoday, modified by community. See post 'Timeline' for change history
// Retrieved 2025-11-29, License - CC BY-SA 4.0

import { ec as EC } from 'elliptic';

const p256 = new EC('p256');

const { d } = nodePrivateKeyObj.export({ format: 'jwk' });
const privateKey = Buffer.from(d, 'base64url');

const signature = p256.sign(digest, privateKey, { canonical: true });
const signatureBytes = new Uint8Array(signature.toDER());
```

# run

```go
go run .
```