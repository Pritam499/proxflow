package graphqlfederation

import (
	"testing"

	"github.com/Pritam499/proxflow/v2"
)

func TestGateway_Provision(t *testing.T) {
	g := &Gateway{
		Services: []Service{
			{
				Name: "test",
				URL:  "http://example.com/graphql",
				SchemaSDL: `
					type User {
						id: ID!
						name: String!
					}
					type Query {
						user(id: ID!): User
					}
				`,
			},
		},
	}

	ctx := caddy.ActiveContext()
	err := g.Provision(ctx)
	if err != nil {
		t.Fatalf("Provision failed: %v", err)
	}

	if g.schemaCache == nil {
		t.Error("Schema cache not initialized")
	}

	if len(g.schemaCache.Types) == 0 {
		t.Error("Schema types not parsed")
	}
}

func TestGateway_Validate(t *testing.T) {
	tests := []struct {
		name    string
		gateway Gateway
		wantErr bool
	}{
		{
			name: "valid config",
			gateway: Gateway{
				Services: []Service{
					{Name: "svc1", URL: "http://svc1.com"},
				},
			},
			wantErr: false,
		},
		{
			name: "no services",
			gateway: Gateway{
				Services: []Service{},
			},
			wantErr: true,
		},
		{
			name: "missing name",
			gateway: Gateway{
				Services: []Service{
					{URL: "http://svc1.com"},
				},
			},
			wantErr: true,
		},
		{
			name: "missing URL",
			gateway: Gateway{
				Services: []Service{
					{Name: "svc1"},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate names",
			gateway: Gateway{
				Services: []Service{
					{Name: "svc1", URL: "http://svc1.com"},
					{Name: "svc1", URL: "http://svc2.com"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.gateway.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}