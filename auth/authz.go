package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fabric8-services/fabric8-wit/errors"
	"github.com/fabric8-services/fabric8-wit/log"
	"github.com/fabric8-services/fabric8-wit/rest"
	errs "github.com/pkg/errors"
)

// EntitlementResource represents a payload for obtaining entitlement for specific resource
type EntitlementResource struct {
	Permissions     []ResourceSet   `json:"permissions"`
	MetaInformation EntitlementMeta `json:"metadata"`
}

type EntitlementMeta struct {
	Limit string `json:"limit"`
}

// ResourceSet represents a resource set for Entitlement payload
type ResourceSet struct {
	Name string  `json:"resource_set_name"`
	ID   *string `json:"resource_set_id,omitempty"`
}

type entitlementResult struct {
	Rpt string `json:"rpt"`
}

// GetEntitlement obtains Entitlement for specific resource.
// If entitlementResource == nil then Entitlement for all resources available to the user is returned.
// Returns (nil, nil) if response status == Forbiden which means the user doesn't have permissions to obtain Entitlement
func GetEntitlement(ctx context.Context, entitlementEndpoint string, entitlementResource *EntitlementResource, userAccesToken string) (*string, error) {
	var req *http.Request
	var reqErr error
	if entitlementResource != nil {
		b, err := json.Marshal(entitlementResource)
		if err != nil {
			log.Error(ctx, map[string]interface{}{
				"entitlement_resource": entitlementResource,
				"err": err.Error(),
			}, "unable to marshal keycloak entitlement resource struct")
			return nil, errors.NewInternalError(ctx, errs.Wrap(err, "unable to marshal keycloak entitlement resource struct"))
		}

		req, reqErr = http.NewRequest("POST", entitlementEndpoint, strings.NewReader(string(b)))
		req.Header.Add("Content-Type", "application/json")
	} else {
		req, reqErr = http.NewRequest("GET", entitlementEndpoint, nil)
	}
	if reqErr != nil {
		log.Error(ctx, map[string]interface{}{
			"err": reqErr.Error(),
		}, "unable to create http request")
		return nil, errors.NewInternalError(ctx, errs.Wrap(reqErr, "unable to create http request"))
	}

	req.Header.Add("Authorization", "Bearer "+userAccesToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"entitlement_resource": entitlementResource,
			"err": err.Error(),
		}, "unable to obtain entitlement resource")
		return nil, errors.NewInternalError(ctx, errs.Wrap(err, "unable to obtain entitlement resource"))
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK:
		// OK
	case http.StatusForbidden:
		return nil, nil
	default:
		log.Error(ctx, map[string]interface{}{
			"entitlement_resource": entitlementResource,
			"response_status":      res.Status,
			"response_body":        rest.ReadBody(res.Body),
		}, "unable to update the Keycloak permission")
		return nil, errors.NewInternalError(ctx, errs.New("unable to obtain entitlement resource. Response status: "+res.Status+". Response body: "+rest.ReadBody(res.Body)))
	}
	jsonString := rest.ReadBody(res.Body)

	var r entitlementResult
	err = json.Unmarshal([]byte(jsonString), &r)
	if err != nil {
		log.Error(ctx, map[string]interface{}{
			"entitlement_resource": entitlementResource,
			"json_string":          jsonString,
		}, "unable to unmarshal json with the obtain entitlement request result")
		return nil, errors.NewInternalError(ctx, errs.Wrapf(err, "error when unmarshal json with the obtain entitlement request result %s ", jsonString))
	}

	return &r.Rpt, nil
}
