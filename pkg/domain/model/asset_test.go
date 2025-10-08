package model_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/secmon-lab/lycaon/pkg/domain/model"
	"github.com/secmon-lab/lycaon/pkg/domain/types"
)

func TestAsset_Validate(t *testing.T) {
	tests := []struct {
		name    string
		asset   model.Asset
		wantErr bool
	}{
		{
			name: "valid asset with all fields",
			asset: model.Asset{
				ID:          types.AssetID("web_frontend"),
				Name:        "Web Frontend",
				Description: "Customer-facing web application",
			},
			wantErr: false,
		},
		{
			name: "valid asset without description",
			asset: model.Asset{
				ID:   types.AssetID("api_gateway"),
				Name: "API Gateway",
			},
			wantErr: false,
		},
		{
			name: "invalid asset - empty ID",
			asset: model.Asset{
				ID:   types.AssetID(""),
				Name: "Test Asset",
			},
			wantErr: true,
		},
		{
			name: "invalid asset - empty Name",
			asset: model.Asset{
				ID:   types.AssetID("test_asset"),
				Name: "",
			},
			wantErr: true,
		},
		{
			name: "invalid asset - both ID and Name empty",
			asset: model.Asset{
				ID:   types.AssetID(""),
				Name: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.asset.Validate()
			if tt.wantErr {
				gt.Error(t, err)
			} else {
				gt.NoError(t, err)
			}
		})
	}
}
