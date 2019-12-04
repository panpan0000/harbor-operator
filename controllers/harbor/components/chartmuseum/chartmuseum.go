package chartmuseum

import (
	"context"

	containerregistryv1alpha1 "github.com/ovh/harbor-operator/api/v1alpha1"
)

type ChartMuseum struct {
	harbor *containerregistryv1alpha1.Harbor
	Option Option
}

type Option struct {
	Priority *int32
}

func New(ctx context.Context, harbor *containerregistryv1alpha1.Harbor, opt Option) (*ChartMuseum, error) {
	return &ChartMuseum{
		harbor: harbor,
		Option: opt,
	}, nil
}
