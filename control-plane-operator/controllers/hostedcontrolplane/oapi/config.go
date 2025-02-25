package oapi

import (
	"encoding/json"
	"fmt"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/konnectivity"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/manifests"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path"

	configv1 "github.com/openshift/api/config/v1"
	openshiftcpv1 "github.com/openshift/api/openshiftcontrolplane/v1"

	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/config"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/kas"
	"github.com/openshift/hypershift/control-plane-operator/controllers/hostedcontrolplane/pki"
)

const (
	openshiftAPIServerConfigKey     = "config.yaml"
	oasKonnectivityProxyConfigKey   = "haproxy.cfg"
	defaultInternalRegistryHostname = "image-registry.openshift-image-registry.svc:5000"
	haproxyCombinedPemLocation = "/tmp/tls.pem"
)

func ReconcileConfig(cm *corev1.ConfigMap, ownerRef config.OwnerRef, etcdURL, ingressDomain, minTLSVersion string, cipherSuites []string) error {
	ownerRef.ApplyTo(cm)
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	openshiftAPIServerConfig := &openshiftcpv1.OpenShiftAPIServerConfig{}
	if configBytes, exist := cm.Data[openshiftAPIServerConfigKey]; exist {
		if err := json.Unmarshal([]byte(configBytes), openshiftAPIServerConfig); err != nil {
			return fmt.Errorf("failed to read existing config: %w", err)
		}
	}
	reconcileConfigObject(openshiftAPIServerConfig, etcdURL, ingressDomain, minTLSVersion, cipherSuites)
	serializedConfig, err := json.Marshal(openshiftAPIServerConfig)
	if err != nil {
		return fmt.Errorf("failed to serialize openshift apiserver config: %w", err)
	}
	cm.Data[openshiftAPIServerConfigKey] = string(serializedConfig)
	oasKonnectivityProxyConfig := reconcileOASKonnectivityProxyConfig()
	cm.Data[oasKonnectivityProxyConfigKey] = oasKonnectivityProxyConfig
	return nil
}

func reconcileConfigObject(cfg *openshiftcpv1.OpenShiftAPIServerConfig, etcdURL, ingressDomain, minTLSVersion string, cipherSuites []string) {
	cfg.TypeMeta = metav1.TypeMeta{
		Kind:       "OpenShiftAPIServerConfig",
		APIVersion: openshiftcpv1.GroupVersion.String(),
	}
	cpath := func(volume, file string) string {
		dir := volumeMounts.Path(oasContainerMain().Name, volume)
		return path.Join(dir, file)
	}
	cfg.APIServerArguments = map[string][]string{
		"shutdown-delay-duration": {"3s"},
		"audit-log-format":        {"json"},
		"audit-log-maxsize":       {"100"},
		"audit-log-path":          {cpath(oasVolumeWorkLogs().Name, "audit.log")},
		"audit-policy-file":       {cpath(oasVolumeAuditConfig().Name, auditPolicyConfigMapKey)},
	}
	cfg.KubeClientConfig.KubeConfig = cpath(oasVolumeKubeconfig().Name, kas.KubeconfigKey)
	cfg.ServingInfo = configv1.HTTPServingInfo{
		ServingInfo: configv1.ServingInfo{
			CertInfo: configv1.CertInfo{
				CertFile: cpath(oasVolumeServingCert().Name, corev1.TLSCertKey),
				KeyFile:  cpath(oasVolumeServingCert().Name, corev1.TLSPrivateKeyKey),
			},
			ClientCA:      cpath(oasVolumeServingCA().Name, pki.CASignerCertMapKey),
			CipherSuites:  cipherSuites,
			MinTLSVersion: minTLSVersion,
		},
	}

	if cfg.ImagePolicyConfig.InternalRegistryHostname == "" {
		cfg.ImagePolicyConfig.InternalRegistryHostname = defaultInternalRegistryHostname
	}

	cfg.RoutingConfig = openshiftcpv1.RoutingConfig{
		Subdomain: ingressDomain,
	}

	cfg.StorageConfig = configv1.EtcdStorageConfig{
		EtcdConnectionInfo: configv1.EtcdConnectionInfo{
			URLs: []string{etcdURL},
			CertInfo: configv1.CertInfo{
				CertFile: cpath(oasVolumeEtcdClientCert().Name, pki.EtcdClientCrtKey),
				KeyFile:  cpath(oasVolumeEtcdClientCert().Name, pki.EtcdClientKeyKey),
			},
			CA: cpath(oasVolumeEtcdClientCA().Name, pki.CASignerCertMapKey),
		},
	}
}

const haproxyTemplate = `
global
  maxconn 7000
  log stdout local0
  log stdout local1 notice

defaults
  mode http
  log global
  option log-health-checks
  balance roundrobin
  timeout client 10m
  timeout server 10m
  timeout connect 10s
  timeout client-fin 5s
  timeout server-fin 5s
  timeout queue 5s
  retries 3

frontend http-in
  bind *:%d
  default_backend be
backend be
  mode http
  server ks %s:%d ssl verify required ca-file %s crt %s
`

func reconcileOASKonnectivityProxyConfig() string {
	cpath := func(volume, file string) string {
		dir := volumeMounts.Path(oasKonnectivityProxyContainer().Name, volume)
		return path.Join(dir, file)
	}
	return fmt.Sprintf(haproxyTemplate, konnectivity.KonnectivityServerLocalPort, manifests.KonnectivityServerLocalService("").Name, konnectivity.KonnectivityServerLocalPort, cpath(oasVolumeKonnectivityProxyCert().Name,"ca.crt"), haproxyCombinedPemLocation)
}
