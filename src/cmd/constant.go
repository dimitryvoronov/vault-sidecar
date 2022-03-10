/* Constant values for vaultsidecar */
package cmd

import (
	"fmt"
	"net/http"
	"os"
)

var (
	awsSecrets       = "environment/aws/"
	basePath         = "/tmp/vault/roles/vault-store/files/"
	checkInterval    = "60"
	clusterAws       = "cluster/aws/"
	clusterOpenshift = "cluster/openshift/"
	httpClient       http.Client
	infraKeys               = "infrastructure/keys/"
	k8sAnsibleKey    string = "gv-ansible-vault-key"
	k8sSecrets       []string
	ocSecrets               = "gigabit-tm*"
	clusterOne              = fmt.Sprint(basePath + "environment/cluster/" + "clusterOne")
	clusterTwo              = fmt.Sprint(basePath + "environment/cluster/" + "clusterTwo")
	clusterThree            = fmt.Sprint(basePath + "environment/cluster/" + "clusterThree")
	vaultAddr               = "http://127.0.0.1:8200"
	vaultInitKeys    string = "vault-init-keys"
	vaultKeysPath    []string
	vaultNamespace   string = os.Args[1]
	vaultRootToken   string = "vault-root-token"
	workFlowGard            = "workflow/gard/"
)

// InitRequest holds a Vault init request.
type InitRequest struct {
	SecretShares    int `json:"secret_shares"`
	SecretThreshold int `json:"secret_threshold"`
}

// InitResponse holds a Vault init response.
type InitResponse struct {
	Keys       []string `json:"keys"`
	KeysBase64 []string `json:"keys_base64"`
	RootToken  string   `json:"root_token"`
}

// UnsealRequest holds a Vault unseal request.
type UnsealRequest struct {
	Key   string `json:"key"`
	Reset bool   `json:"reset"`
}

// UnsealResponse holds a Vault unseal response.
type UnsealResponse struct {
	Sealed   bool `json:"sealed"`
	T        int  `json:"t"`
	N        int  `json:"n"`
	Progress int  `json:"progress"`
}

type K8sSecret struct {
	Token     string `json:"token"`
	KeySecret string `json:"key"`
}
type VaultKeys struct {
	KeyOne   string `json:"vault-key-0"`
	KeyTwo   string `json:"vault-key-1"`
	KeyThree string `json:"vault-key-2"`
}
