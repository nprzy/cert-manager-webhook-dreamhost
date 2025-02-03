<p align="center">
  <img src="https://raw.githubusercontent.com/cert-manager/cert-manager/d53c0b9270f8cd90d908460d69502694e1838f5f/logo/logo-small.png" height="256" width="256" alt="cert-manager project logo" />
</p>

# DreamHost `cert-manager` ACME Webhook Solver

**Work in Progress**

This project is intended to be an ACME DNS01 webhook solver for
[cert-manager](https://cert-manager.io/). It is not yet ready for use.

### Running the test suite

You can run the test suite against local API/DNS mocks with:

```bash
$ make test
```

To run against the real DreamHost API, edit `testdata/dreamhost-solver/secret.yaml`
and include your real API key. Then run the tests with additional environment variables.
Replace `subdomain.example.com.` with the actual domain name you want to test. The top
level domain must be something that is actually registered with your DreamHost account.
The IP address specified here for `TEST_DNS_SERVER` is the IP address resolved from
`ns1.dreamhost.com`. It's specified directly here to reduce DNS propagation delay. If
that IP ever changes you may need to update the command.

```bash
$ TEST_API_URL=https://api.dreamhost.com/ TEST_DNS_SERVER=162.159.26.14:53 TEST_ZONE_NAME=subdomain.example.com. make test
```