# IPFS reverse proxy  module 
Service for public websites deployed to ipfs 

## Main config parameters
DomainList:
- solenopsys.org: https://alpha.node.solenopsys.org/ipfs/QmcbcZ18iAXFSwnci6GQYQnML8MtQMe92EppggJVDDhaEk
- auth.solenopsys.org: https://alpha.node.solenopsys.org/ipfs/QmcbcZ18iAXFSwnci6GQYQnML8MtQMe92EppggJVDDhaEk

## config map
reverse-proxy-mapping 

nerdctl build --platform=amd64  --progress=plain --output type=image,name=registry.*/solenopsys/ms-ipfs-revproxy:latest,push=true .