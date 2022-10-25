package main

import (
	"encoding/json"
	"fmt"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"os"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jetstack/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/jetstack/cert-manager/pkg/acme/webhook/cmd"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

var GroupName = os.Getenv("GROUP_NAME")

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	cmd.RunWebhookServer(GroupName,
		&HEnetDNSProviderSolver{},
	)
}

type HEnetDNSProviderSolver struct {
	client *kubernetes.Clientset
}

type HEnetDNSProviderConfig struct {
	ApiUrl string	 `json:"apiUrl"`
	SecretRef string `json:"secretName"`
}

type InternalConfig struct {
	Password, ApiUrl string
}

func (c *HEnetDNSProviderSolver) Name() string {
	return "hurricane-electric"
}

func (c *HEnetDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	klog.V(6).Infof("call function Present: namespace=%s, zone=%s, fqdn=%s", ch.ResourceNamespace, ch.ResolvedZone, ch.ResolvedFQDN)

	config, err := clientConfig(c, ch)

	if err != nil {
		return fmt.Errorf("unable to get secret `%s`; %v", ch.ResourceNamespace, err)
	}

	apiUrl := fmt.Sprintf(`%s/nic/update`, config.ApiUrl)
	method := "POST"
	data := url.Values{}
	// The hostname should be the full hostname without a trailing dot
	data.Set("hostname", strings.TrimRight(ch.ResolvedFQDN, "."))
	data.Set("password", config.Password)
	data.Set("txt", ch.Key)

	req, err := http.NewRequest(method, apiUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("unable to execute request %v", err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			klog.Fatal(err)
		}
	}()

	respBody, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		text := "Error calling API status:" + resp.Status + " url: " +  apiUrl + " method: " + method
		klog.Error(text)
		return errors.New(text)
	}
	text := "API responded:" + resp.Status + " url: " +  apiUrl + " method: " + method + " body: " + string(respBody)
	klog.Info(text)

	klog.Infof("Succesfully presented txt record %v", ch.ResolvedFQDN)

	return nil
}

func (c *HEnetDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	// We don't handle cleanup as all dynamic records are created manually in the dns.he.net control panel
	return nil
}

func (c *HEnetDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	k8sClient, err := kubernetes.NewForConfig(kubeClientConfig)
	klog.V(6).Infof("Input variable stopCh is %d length", len(stopCh))
	if err != nil {
		return err
	}

	c.client = k8sClient

	return nil
}

// loadConfig is a small helper function that decodes JSON configuration into
// the typed config struct.
func loadConfig(cfgJSON *extapi.JSON) (HEnetDNSProviderConfig, error) {
	cfg := HEnetDNSProviderConfig{}
	// handle the 'base case' where no configuration has been provided
	if cfgJSON == nil {
		return cfg, nil
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}

	return cfg, nil
}

func clientConfig(c *HEnetDNSProviderSolver, ch *v1alpha1.ChallengeRequest) (InternalConfig, error) {
	var config InternalConfig

	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return config, err
	}
	config.ApiUrl = cfg.ApiUrl

	secretName := cfg.SecretRef
	sec, err := c.client.CoreV1().Secrets(ch.ResourceNamespace).Get(secretName, metav1.GetOptions{})

	if err != nil {
		return config, fmt.Errorf("unable to get secret `%s/%s`; %v", secretName, ch.ResourceNamespace, err)
	}

	password, err := stringFromSecretData(&sec.Data, "password")
	config.Password = password

	if err != nil {
		return config, fmt.Errorf("unable to get password from secret `%s/%s`; %v", secretName, ch.ResourceNamespace, err)
	}

	return config, nil
}

func stringFromSecretData(secretData *map[string][]byte, key string) (string, error) {
	data, ok := (*secretData)[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret data", key)
	}
	return string(data), nil
}