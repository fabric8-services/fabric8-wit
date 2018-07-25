package controller

import (
	"testing"
	"time"

	"github.com/fabric8-services/fabric8-wit/account/tenant"
	"github.com/fabric8-services/fabric8-wit/app"
	"github.com/fabric8-services/fabric8-wit/ptr"

	goauuid "github.com/goadesign/goa/uuid"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

var (
	pts = ptr.String
	ptb = ptr.Bool
)

func Test_convert(t *testing.T) {
	nsTime := time.Now()
	nsInput := []*tenant.NamespaceAttributes{
		{
			CreatedAt:                &nsTime,
			UpdatedAt:                &nsTime,
			Name:                     pts("foons"),
			State:                    pts("created"),
			Version:                  pts("1.0"),
			Type:                     pts("che"),
			ClusterURL:               pts("http://test.org"),
			ClusterConsoleURL:        pts("https://console.example.com/console"),
			ClusterMetricsURL:        pts("https://metrics.example.com"),
			ClusterLoggingURL:        pts("https://console.example.com/console"),
			ClusterAppDomain:         pts("apps.example.com"),
			ClusterCapacityExhausted: ptb(true),
		},
	}

	tenantID := goauuid.NewV4()
	tenantCreated := time.Now()
	tenantSingle := &tenant.TenantSingle{
		Data: &tenant.Tenant{
			ID: &tenantID,
			Attributes: &tenant.TenantAttributes{
				CreatedAt:  &tenantCreated,
				Namespaces: nsInput,
			},
		},
	}

	nsOutput := []*app.NamespaceAttributes{
		{
			CreatedAt:                &nsTime,
			UpdatedAt:                &nsTime,
			Name:                     pts("foons"),
			State:                    pts("created"),
			Version:                  pts("1.0"),
			Type:                     pts("che"),
			ClusterURL:               pts("http://test.org"),
			ClusterConsoleURL:        pts("https://console.example.com/console"),
			ClusterMetricsURL:        pts("https://metrics.example.com"),
			ClusterLoggingURL:        pts("https://console.example.com/console"),
			ClusterAppDomain:         pts("apps.example.com"),
			ClusterCapacityExhausted: ptb(true),
		},
	}
	tenantIDConv, err := uuid.FromString(tenantID.String())
	require.NoError(t, err)
	expected := &app.UserServiceSingle{
		Data: &app.UserService{
			Attributes: &app.UserServiceAttributes{
				CreatedAt:  &tenantCreated,
				Namespaces: nsOutput,
			},
			ID:   &tenantIDConv,
			Type: "userservices",
		},
	}

	actual := convertTenant(tenantSingle)
	require.Equal(t, expected, actual)
}
