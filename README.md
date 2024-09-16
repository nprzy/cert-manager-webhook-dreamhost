<p align="center">
  <img src="https://raw.githubusercontent.com/cert-manager/cert-manager/d53c0b9270f8cd90d908460d69502694e1838f5f/logo/logo-small.png" height="256" width="256" alt="cert-manager project logo" />
</p>

# DreamHost `cert-manager` ACME Webhook Solver

**Work in Progress**

This project is intended to be an ACME DNS01 webhook solver for
[cert-manager](https://cert-manager.io/). It is not yet ready for use.

## Why not in core?

As the project & adoption has grown, there has been an influx of DNS provider
pull requests to our core codebase. As this number has grown, the test matrix
has become un-maintainable and so, it's not possible for us to certify that
providers work to a sufficient level.

By creating this 'interface' between cert-manager and DNS providers, we allow
users to quickly iterate and test out new integrations, and then packaging
those up themselves as 'extensions' to cert-manager.

We can also then provide a standardised 'testing framework', or set of
conformance tests, which allow us to validate that a DNS provider works as
expected.

### Running the test suite

All DNS providers **must** run the DNS01 provider conformance testing suite,
else they will have undetermined behaviour when used with cert-manager.

**It is essential that you configure and run the test suite when creating a
DNS01 webhook.**

An example Go test file has been provided in [main_test.go](https://github.com/cert-manager/webhook-example/blob/master/main_test.go).

You can run the test suite with:

```bash
$ TEST_ZONE_NAME=example.com. make test
```

The example file has a number of areas you must fill in and replace with your
own options in order for tests to pass.
