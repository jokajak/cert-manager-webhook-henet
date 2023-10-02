# ACME webhook for Hurricane Electric Hosted DNS

This solver can be used when you want to use cert-manager with [Hurricane Electric hosted DNS](https://dns.he.net/) zones.

**Note** This is almost direct copy of vadikim's [cert-manager-webhook-hetzner](https://github.com/vadimkim/cert-manager-webhook-hetzner) with heavy adjustments to accomodate using Hurricane Electric's Dynamic DNS interface.

This is not considered ready for production, nor has it been tested. I do not know enough Go to make heads or tails about the code but I know enough to be dangerous.

This version is provided as-is, without any guarantees that it won't wreck your coffee maker or kubernetes installation. **Use at your own risk!**

## Requirements

-   [go](https://golang.org/) >= 1.19.0
-   [helm](https://helm.sh/) >= v3.0.0
-   [kubernetes](https://kubernetes.io/) >= v1.25.0
-   [cert-manager](https://cert-manager.io/) >= 1.11

## Installation

### cert-manager

Follow the [instructions](https://cert-manager.io/docs/installation/) using the cert-manager documentation to install it within your cluster.

### Webhook

#### Using public helm chart

```bash
helm repo add cert-manager-webhook-henet https://jokajak.github.io/cert-manager-webhook-henet
# Replace the groupName value with your desired domain
helm install --namespace cert-manager cert-manager-webhook-henet cert-manager-webhook-henet/cert-manager-webhook-henet --set groupName=acme.yourdomain.tld
```

#### From local checkout

```bash
helm install --namespace cert-manager cert-manager-webhook-henet deploy/cert-manager-webhook-henet
```

**Note**: The kubernetes resources used to install the Webhook should be deployed within the same namespace as the cert-manager.

To uninstall the webhook run

```bash
helm uninstall --namespace cert-manager cert-manager-webhook-henet
```

## Issuer

Create a `ClusterIssuer` or `Issuer` resource as following:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    # The ACME server URL
    server: https://acme-staging-v02.api.letsencrypt.org/directory

    # Email address used for ACME registration
    email: mail@example.com # REPLACE THIS WITH YOUR EMAIL!!!

    # Name of a secret used to store the ACME account private key
    privateKeySecretRef:
      name: letsencrypt-staging

    solvers:
      - dns01:
          webhook:
            # This group needs to be configured when installing the helm package, otherwise the webhook won't have permission to create an ACME challenge for this API group.
            groupName: acme.yourdomain.tld
            solverName: hurricane-electric
            config:
              secretName: henet-secret
              apiUrl: https://dyn.dns.he.net
```

### Credentials
In order to access the henet API, the webhook needs an API token.

If you choose another name for the secret than `henet-secret`, ensure you modify the value of `secretName` in the `[Cluster]Issuer`.

The secret for the example above will look like this:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: henet-secret
  namespace: cert-manager
type: Opaque
data:
  password: your-key-base64-encoded
```

### Create a certificate

Finally you can create certificates, for example:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: example-cert
  namespace: cert-manager
spec:
  commonName: example.com
  dnsNames:
    - example.com
  issuerRef:
    name: letsencrypt-staging
    kind: ClusterIssuer
  secretName: example-cert
```

## Development

### Running the test suite

All DNS providers **must** run the DNS01 provider conformance testing suite,
else they will have undetermined behaviour when used with cert-manager.

**It is essential that you configure and run the test suite when creating a
DNS01 webhook.**

First, you need to have a Hurricane Electric account with access to the DNS control panel. You need to create a new TXT record with the name of `cert-manager-dns01-tests`, use TTL of 5 minutes and ensure **Enable entry for dynamic dns** is checked. Use the circular arrows icon to set or generate the key for dynamic updates.
You can either set the `zoneName` parameter in `testdata/henet/config.json` to your zone name or use the `TEST_ZONE_NAME` as in the example below.
You also must encode your key into base64 and put the hash into `testdata/henet/henet-secret.yml` file.

You can run the test suite with:

```bash
TEST_ZONE_NAME=example.com. make test
```

## Credits

This was forked from <https://github.com/Diftraku/cert-manager-webhook-henet> who did all the work of making it work
with Hurricane Electric. I just updated the repo to have tests and newer libraries.
