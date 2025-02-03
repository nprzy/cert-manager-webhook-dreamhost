package main

import (
	"context"
	"encoding/json"
	"fmt"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/nprzy/cert-manager-webhook-dreamhost/internal/dreamhost"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"os"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/client-go/rest"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	// This will register our custom DNS provider with the webhook serving
	// library, making it available as an API under the provided GroupName.
	// You can register multiple DNS provider implementations with a single
	// webhook, where the Name() method will be used to disambiguate between
	// the different implementations.
	cmd.RunWebhookServer(GroupName,
		&dreamHostDnsProviderSolver{},
	)
}

// dreamHostDnsProviderSolver implements the provider-specific logic needed to
// 'present' an ACME challenge TXT record for your own DNS provider.
// To do so, it must implement the `github.com/cert-manager/cert-manager/pkg/acme/webhook.Solver`
// interface.
type dreamHostDnsProviderSolver struct {
	baseUrl string
	client  kubernetes.Clientset
}

// dreamHostDnsProviderConfig is a structure that is used to decode into when
// solving a DNS01 challenge.
// This information is provided by cert-manager, and may be a reference to
// additional configuration that's needed to solve the challenge for this
// particular certificate or issuer.
// This typically includes references to Secret resources containing DNS
// provider credentials, in cases where a 'multi-tenant' DNS solver is being
// created.
// If you do *not* require per-issuer or per-certificate configuration to be
// provided to your webhook, you can skip decoding altogether in favour of
// using CLI flags or similar to provide configuration.
// You should not include sensitive information here. If credentials need to
// be used by your provider here, you should reference a Kubernetes Secret
// resource and fetch these credentials using a Kubernetes clientset.
type dreamHostDnsProviderConfig struct {
	// These fields will be set by users in the
	// `issuer.spec.acme.dns01.providers.webhook.config` field.
	APIKeySecretRef cmmeta.SecretKeySelector `json:"dreamhostApiKeyRef"`
}

// Name is used as the name for this DNS solver when referencing it on the ACME
// Issuer resource.
// This should be unique **within the group name**, i.e. you can have two
// solvers configured with the same Name() **so long as they do not co-exist
// within a single webhook deployment**.
// For example, `cloudflare` may be used as the name of a solver.
func (c *dreamHostDnsProviderSolver) Name() string {
	return "dreamhost"
}

// Present is responsible for actually presenting the DNS record with the
// DNS provider.
// This method should tolerate being called multiple times with the same value.
// cert-manager itself will later perform a self check to ensure that the
// solver has correctly configured the DNS provider.
func (c *dreamHostDnsProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	klog.Infof("Attempting to create TXT record %v with value %v", ch.ResolvedFQDN, ch.Key)

	client, err := c.loadDreamhostClient(ch)
	if err != nil {
		klog.Errorf("Failed to create Dream Host client: %v", err)
		return errors.Wrapf(err, "Failed to create Dream Host client")
	}

	name := trimTrailingDot(ch.ResolvedFQDN)
	err = client.CreateRecord(dreamhost.DNSRecordValue{Name: name, RecordType: "TXT", Value: ch.Key}, "")
	if err != nil {
		klog.Errorf("Dreamhost client failed to create DNS record: %v", err)
		return errors.Wrapf(err, "Dreamhost client failed to create DNS record")
	}

	klog.Infof("Created TXT record %v with value %v", ch.ResolvedFQDN, ch.Key)
	return nil
}

// CleanUp should delete the relevant TXT record from the DNS provider console.
// If multiple TXT records exist with the same record name (e.g.
// _acme-challenge.example.com) then **only** the record with the same `key`
// value provided on the ChallengeRequest should be cleaned up.
// This is in order to facilitate multiple DNS validations for the same domain
// concurrently.
func (c *dreamHostDnsProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	klog.Infof("Attempting to delete TXT record %v with value %v", ch.ResolvedFQDN, ch.Key)

	client, err := c.loadDreamhostClient(ch)
	if err != nil {
		klog.Errorf("Failed to create Dream Host client: %v", err)
		return errors.Wrapf(err, "Failed to create Dream Host client")
	}

	name := trimTrailingDot(ch.ResolvedFQDN)
	err = client.DeleteRecord(dreamhost.DNSRecordValue{Name: name, RecordType: "TXT", Value: ch.Key}, "")
	if err != nil {
		klog.Errorf("Dreamhost client failed to delete DNS record: %v", err)
		return errors.Wrapf(err, "Dreamhost client failed to delete DNS record")
	}

	klog.Infof("Deleted TXT record %v with value %v", ch.ResolvedFQDN, ch.Key)
	return nil
}

// Initialize will be called when the webhook first starts.
// This method can be used to instantiate the webhook, i.e. initialising
// connections or warming up caches.
// Typically, the kubeClientConfig parameter is used to build a Kubernetes
// client that can be used to fetch resources from the Kubernetes API, e.g.
// Secret resources containing credentials used to authenticate with DNS
// provider accounts.
// The stopCh can be used to handle early termination of the webhook, in cases
// where a SIGTERM or similar signal is sent to the webhook process.
func (c *dreamHostDnsProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	c.client = *cl
	return nil
}

func (c *dreamHostDnsProviderSolver) loadDreamhostClient(ch *v1alpha1.ChallengeRequest) (*dreamhost.DNSClient, error) {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return nil, err
	}

	secretName := cfg.APIKeySecretRef.LocalObjectReference.Name
	namespace := ch.ResourceNamespace

	secret, err := c.client.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to load secret %v %q", secretName, namespace+"/"+secretName)
	}

	data, ok := secret.Data[cfg.APIKeySecretRef.Key]
	if !ok {
		return nil, fmt.Errorf("key %q not found in secret \"%s/%s\"", cfg.APIKeySecretRef.Key,
			cfg.APIKeySecretRef.LocalObjectReference.Name, namespace)
	}
	apiKey := string(data)
	return dreamhost.NewClient(apiKey, nil, c.baseUrl)
}

// Remove the trailing "." from a string
func trimTrailingDot(name string) string {
	if string(name[len(name)-1:]) == "." {
		return name[:len(name)-1]
	}
	return name
}

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (dreamHostDnsProviderConfig, error) {
	cfg := dreamHostDnsProviderConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}
