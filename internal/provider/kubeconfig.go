package provider

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type kubeConfig struct {
	Clusters []kubeConfigCluster `yaml:"clusters"`
	Users    []kubeConfigUser    `yaml:"users"`
}

type kubeConfigCluster struct {
	Cluster struct {
		Server                   string `yaml:"server"`
		CertificateAuthorityData string `yaml:"certificate-authority-data"`
	} `yaml:"cluster"`
}

type kubeConfigUser struct {
	User struct {
		ClientCertificateData string `yaml:"client-certificate-data"`
		ClientKeyData         string `yaml:"client-key-data"`
	} `yaml:"user"`
}

type parsedKubeconfig struct {
	Endpoint             string
	ClusterCACertificate string
	ClientCertificate    string
	ClientKey            string
}

func parseKubeconfig(kubeconfigYAML string) (*parsedKubeconfig, error) {
	var cfg kubeConfig
	if err := yaml.Unmarshal([]byte(kubeconfigYAML), &cfg); err != nil {
		return nil, fmt.Errorf("parsing kubeconfig: %w", err)
	}

	result := &parsedKubeconfig{}

	if len(cfg.Clusters) > 0 {
		result.Endpoint = cfg.Clusters[0].Cluster.Server
		result.ClusterCACertificate = cfg.Clusters[0].Cluster.CertificateAuthorityData
	}

	if len(cfg.Users) > 0 {
		result.ClientCertificate = cfg.Users[0].User.ClientCertificateData
		result.ClientKey = cfg.Users[0].User.ClientKeyData
	}

	return result, nil
}

func writeKubeconfigFile(path, kubeconfig string) error {
	if err := os.MkdirAll(fsDir(path), 0750); err != nil {
		return fmt.Errorf("creating parent directories: %w", err)
	}
	if err := os.WriteFile(path, []byte(kubeconfig), 0600); err != nil {
		return fmt.Errorf("writing kubeconfig file: %w", err)
	}
	return nil
}

func fsDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}
