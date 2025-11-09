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

## Releasing

This project uses automated CI/CD via GitHub Actions to build and publish releases. When a version tag
is pushed, the workflow automatically builds the Docker image, publishes it to GitHub Container Registry,
and releases the Helm chart.

### Release Process

1. **Update the Helm chart version**

   Edit `deploy/dreamhost-webhook/Chart.yaml` and update both the `version` and `appVersion` fields
   to the new version number (following [semantic versioning](https://semver.org/)):

   ```yaml
   apiVersion: v2
   appVersion: "X.Y.Z"
   description: Dreamhost DNS-01 solver for Cert Manager
   name: dreamhost-webhook
   type: application
   version: X.Y.Z
   ```

   Also update the image tag in `deploy/dreamhost-webhook/values.yaml` to match:

   ```yaml
   image:
     repository: ghcr.io/nprzy/cert-manager-webhook-dreamhost
     tag: X.Y.Z
     pullPolicy: IfNotPresent
   ```

2. **Commit the version changes**

   ```bash
   $ git add deploy/dreamhost-webhook/Chart.yaml deploy/dreamhost-webhook/values.yaml
   $ git commit -m "Bump version to X.Y.Z"
   ```

3. **Create and push a version tag**

   Create a git tag with the `v` prefix matching the version number:

   ```bash
   $ git tag vX.Y.Z
   $ git push origin vX.Y.Z
   ```

4. **Automated release workflow**

   Once the tag is pushed, the GitHub Actions workflow (`.github/workflows/release.yaml`) will automatically:

   - Run the test suite and generate coverage reports
   - Build the Docker image and push it to `ghcr.io/nprzy/cert-manager-webhook-dreamhost` with tags:
     - `vX.Y.Z` (full semantic version)
     - `X.Y` (major.minor version)
     - `X` (major version, only for versions >= 1.0.0)
   - Generate artifact attestations for supply chain security
   - Release the Helm chart to GitHub releases using [chart-releaser](https://github.com/helm/chart-releaser)
   - Update the Helm chart repository at `https://nprzy.github.io/cert-manager-webhook-dreamhost`

5. **Verify the release**

   After the workflow completes:
   - Check the [GitHub Releases page](https://github.com/nprzy/cert-manager-webhook-dreamhost/releases) for the new release
   - Verify the Docker image is available at `ghcr.io/nprzy/cert-manager-webhook-dreamhost:vX.Y.Z`
   - Verify the Helm chart can be installed:
     ```bash
     $ helm repo update
     $ helm search repo cert-manager-webhook-dreamhost
     ```

### Version Numbering Guidelines

Follow [semantic versioning](https://semver.org/) (MAJOR.MINOR.PATCH):

- **MAJOR**: Increment for incompatible API changes or breaking changes
- **MINOR**: Increment for new functionality in a backwards-compatible manner
- **PATCH**: Increment for backwards-compatible bug fixes