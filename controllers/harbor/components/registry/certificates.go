package registry

import (
	"context"

	certv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	"github.com/ovh/configstore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	containerregistryv1alpha1 "github.com/ovh/harbor-operator/api/v1alpha1"
	"github.com/ovh/harbor-operator/pkg/factories/application"
	"github.com/ovh/harbor-operator/pkg/factories/logger"
)

const (
	defaultKeyAlgorithm = certv1.RSAKeyAlgorithm
)

type certificateEncryption struct {
	KeySize      int
	KeyAlgorithm certv1.KeyAlgorithm
}

func (r *Registry) GetCertificates(ctx context.Context) []*certv1.Certificate {
	operatorName := application.GetName(ctx)
	harborName := r.harbor.Name

	url := r.harbor.Spec.PublicURL

	encryption := &certificateEncryption{
		KeyAlgorithm: defaultKeyAlgorithm,
	}

	item, err := configstore.Filter().Slice("certificate-encryption").Unmarshal(func() interface{} { return &certificateEncryption{} }).GetFirstItem()
	if err == nil {
		l := logger.Get(ctx)

		// todo
		encryptionConfig, err := item.Unmarshaled()
		if err != nil {
			l.Error(err, "Invalid encryption certificate config: use default value")
		} else {
			encryption = encryptionConfig.(*certificateEncryption)
		}
	}

	return []*certv1.Certificate{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      r.harbor.NormalizeComponentName(containerregistryv1alpha1.RegistryName),
				Namespace: r.harbor.Namespace,
				Labels: map[string]string{
					"app":      containerregistryv1alpha1.RegistryName,
					"harbor":   harborName,
					"operator": operatorName,
				},
			},
			Spec: certv1.CertificateSpec{
				CommonName:   url,
				Organization: []string{"Harbor Operator"},
				SecretName:   r.harbor.NormalizeComponentName(containerregistryv1alpha1.CertificateName),
				KeySize:      encryption.KeySize,
				KeyAlgorithm: encryption.KeyAlgorithm,
				KeyEncoding:  certv1.PKCS8, // TODO check that Harbor & registry Handle this format
				DNSNames:     []string{url},
				IssuerRef:    r.harbor.Spec.CertificateIssuerRef,
			},
		},
	}
}
