# route53ctl

```bash
route53ctl -zone=test.io -vpc vpc-abc123 -rule 1:ip:gw-pci.pismo.cloud -rule 2:ip:1.1.1.1,2.2.2.2 -rule 3:vpce:vpce-0abc123456789defg-abcdefgh.vpce-svc-0123456789abcdef.us-east-1.vpce.amazonaws.com

route53ctl -zone=test.io -purge
```
