<p align="center">
  <img src="https://raw.githubusercontent.com/cert-manager/cert-manager/d53c0b9270f8cd90d908460d69502694e1838f5f/logo/logo-small.png" height="256" width="256" alt="cert-manager project logo" />
</p>

# DreamHost `cert-manager` ACME Webhook Solver

This project is intended to be an ACME DNS01 webhook solver for
[cert-manager](https://cert-manager.io/).

## Usage

These steps can be followed to quickly get started with this solver. For more detailed documentation, refer
back to cert-manager's documentation. I recommend reading the
[ACME Issuer Introduction](https://cert-manager.io/docs/configuration/acme/),
[ACME DNS01 Introduction](https://cert-manager.io/docs/configuration/acme/dns01/), and the
[ACME DNS01 Webhook](https://cert-manager.io/docs/configuration/acme/dns01/webhook/) pages to start with.

1. Create a secret with the Dreamhost API key. Write the following YAML
   to a file and apply it with `kubectl apply -f dreamhost-secret.yaml`
    ```
    apiVersion: v1
    kind: Secret
    metadata:
      name: dreamhost-api-key
      namespace: cert-manager
    stringData:
      apikey: YOUR_ACCESS_KEY
    ```

1. Add the Helm repo
    ```
    $ helm repo add cert-manager-webhook-dreamhost https://nprzy.github.io/cert-manager-webhook-dreamhost
    ```

1. Install the Dreamhost solver application. Ensure that secretOverride matches the
    name of the secret you created above for the API key. If this does not match then
    the webhook service won't be granted access to your secret.
    ```
    $ helm install dreamhost-webhook \
        --namespace cert-manager \
        cert-manager-webhook-dreamhost/dreamhost-webhook
    ```
    You may optionally add `--set secretResourceName=dreamhost-api-key` to limit the webhook's access
    to the specific named secret resource. If omitted, it will be granted access to all secrets in the
    namespace where it's deployed.

1. Create a ClusterIssuer. Same as we did above, write the following YAML to a file. Customize
    the `email` at a minimum. You can read more about the ways to customize the issuer
    [here](https://cert-manager.io/docs/configuration/acme/). If you customized the `groupName` when you
    deployed the webhook-dreamhost helm chart, you'll need to ensure that the `groupName` below matches.
    The `groupName` is only used so that cert-manager can route webhook requests to the right solver, so
    it just has to be unique per solver. Beyond that the value doesn't matter. The last thing to note here
    is that the `letsencrypt-production` secret will be created automatically for you. When you're satisfied,
    apply it.
    `kubectl apply -f clusterissuer-lets-encrypt-production.yaml`
    ```
    apiVersion: cert-manager.io/v1
    kind: ClusterIssuer
    metadata:
      name: letsencrypt-production
      namespace: cert-manager
    spec:
      acme:
        server: https://acme-v02.api.letsencrypt.org/directory
        email: YOUR_EMAIL_ADDRESS
        privateKeySecretRef:
          name: letsencrypt-production
      solvers:
      - dns01:
          webhook:
            groupName: nprzy.github.io
            solverName: dreamhost
            config:
              dreamhostApiKeyRef:
                name: dreamhost-api-key
                key: apikey
    ```

1. Create a Certificate using your new ClusterIssuer. Once again, apply with `kubectl`. See
    [the cert-manager Certificate docs](https://cert-manager.io/docs/usage/certificate/) for more details.
    ```
    apiVersion: cert-manager.io/v1
    kind: Certificate
    metadata:
      name: www
    spec:
      secretName: www-tls
      renewBefore: 240h
      commonName: www.YOUR_DOMAIN
      dnsNames:
      - 'www.YOUR_DOMAIN'
      issuerRef:
        name: letsencrypt-production
        kind: ClusterIssuer
    ```

## Development

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