# fabric-x offline signer cli

the offline signer cli for hyperledger fabric x is built upon the 
fabric dlog sdk , fabric token sdk and fabric smart client node , thus
achieving high-level business logic and digital assets scenarios .

# differences from fabric in offline signing context

the main differences between fabric and fabric-x in an offline signing
setting are that : 

- fabric uses gateway protos to achieve offline signing ; while fabric-x
defines self-contained protos for signatures and verifications in
fabric-x-endorser , fabric-x-orderer , fabric-x-committer , respectively .
- fabric-x-common extends fabric msp config by adding known_certs field
of the msp.proto file . however , in offline signing context , a pre-registered
certificates are not useful especially for the offline certificates holder .
- idemix and public parameters are used predominately in fabric x , altough
it can be switched to normal identity and algorithm .

# common area of fabric x and fabric in offline signing

- they use lower than or equal to curve halved low-s signatures . this means
they are using the same curve values for the offline signing context .

# inceptive transaction flow

* issue
  - issuer creates an anonymous tx and inserts the issuer ' s signature locally .
  - the issuer sends the anonymous tx to the intended recipient .
  - the recipient verifies the anonymous tx with a well-known sender ' s public key
and signature in the tx .
  -  the recipient creates a redeem action and broadcast it to the ordering nodes for
tx committment .
* transfer
  - sender creates an anonymous tx and inserts the sender ' s signature locally .
  - the sender sends the anonymous tx to the intended receiver .
  - the receiver verifies the anonymous tx with a well-known sender ' s public key
and signature in the tx .
  - the receiver creates a redeem action and broadcast it to the ordering nodes for
tx committment .

in the above scenarios , the recipient may or may not insert their signature to the
anonymous tx , thus creating a double signature from the identities mentioned above .
if the recipients choose not to insert their signature , the recipients are called
anonymous identities .
there ' s no direct interation between the senders and recipients , which is intended
for offline signing context .
