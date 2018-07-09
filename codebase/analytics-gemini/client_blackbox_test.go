package gemini_test

import (
	"context"
	"net/http"
	"testing"

	gemini "github.com/fabric8-services/fabric8-wit/codebase/analytics-gemini"
	"github.com/fabric8-services/fabric8-wit/configuration"
	testjwt "github.com/fabric8-services/fabric8-wit/test/jwt"
	testrecorder "github.com/fabric8-services/fabric8-wit/test/recorder"

	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	// given
	repoURL := "https://github.com/fabric8-services/fabric8-wit"

	config, err := configuration.New("")
	require.NoError(t, err)
	url := config.GetAnalyticsGeminiServiceURL()

	t.Run("ok response", func(t *testing.T) {
		// given
		r, err := testrecorder.New(
			"../../test/data/gemini-scan/register-200",
			testrecorder.WithJWTMatcher("../../test/jwt/public_key.pem"),
		)
		require.NoError(t, err)
		defer r.Stop()

		httpClient := &http.Client{
			Transport: r.Transport,
		}

		cli := gemini.NewScanRepoClient(url, httpClient, "", nil, false)

		req := gemini.NewScanRepoRequest(repoURL)
		ctx, err := testjwt.NewJWTContext("bcdd0b29-123d-11e8-a8bc-b69930b94f5c", "../../test/jwt/private_key.pem")
		require.NoError(t, err)

		// then
		err = cli.Register(ctx, req)
		require.NoError(t, err)
	})

	t.Run("fail on no jwt token", func(t *testing.T) {
		r, err := testrecorder.New(
			"../../test/data/gemini-scan/register-401",
		)
		require.NoError(t, err)
		defer r.Stop()

		httpClient := &http.Client{
			Transport: r.Transport,
		}

		cli := gemini.NewScanRepoClient(url, httpClient, "", nil, false)
		req := gemini.NewScanRepoRequest(repoURL)

		// give a context that has no token
		err = cli.Register(context.Background(), req)
		require.Error(t, err)
		t.Log("successfully errored as: ", err)
	})

	t.Run("fail on parsing url", func(t *testing.T) {
		// the url for this request is invalid
		url := "%"
		cli := gemini.NewScanRepoClient(url, &http.Client{}, "", nil, false)

		req := gemini.NewScanRepoRequest(repoURL)
		// then
		err = cli.Register(context.Background(), req)
		require.Error(t, err)
		t.Log("successfully errored as: ", err)
	})

	t.Run("fail on wrong URL", func(t *testing.T) {
		// url in this request does not mention what scheme to use
		// for e.g. http or https
		url := "foo.bar"
		cli := gemini.NewScanRepoClient(url, &http.Client{}, "", nil, false)

		req := gemini.NewScanRepoRequest(repoURL)

		ctx, err := testjwt.NewJWTContext("bcdd0b29-123d-11e8-a8bc-b69930b94f5c", "../../test/jwt/private_key.pem")
		require.NoError(t, err)

		// then
		err = cli.Register(ctx, req)
		require.Error(t, err)
		t.Log("successfully errored as: ", err)
	})

	t.Run("fail to unmarshal response", func(t *testing.T) {
		// server responds 200 but empty body which is not
		// json format, so the unmarshaller will complain
		r, err := testrecorder.New(
			"../../test/data/gemini-scan/register-weird-response",
			testrecorder.WithJWTMatcher("../../test/jwt/public_key.pem"),
		)
		require.NoError(t, err)
		defer r.Stop()

		httpClient := &http.Client{
			Transport: r.Transport,
		}

		cli := gemini.NewScanRepoClient(url, httpClient, "", nil, false)

		req := gemini.NewScanRepoRequest(repoURL)
		ctx, err := testjwt.NewJWTContext("bcdd0b29-123d-11e8-a8bc-b69930b94f5c", "../../test/jwt/private_key.pem")
		require.NoError(t, err)

		// then
		err = cli.Register(ctx, req)
		require.Error(t, err)
		t.Log("successfully errored as: ", err)
	})

	t.Run("unknown response from the server", func(t *testing.T) {
		// given
		// server responds with 500 here
		r, err := testrecorder.New(
			"../../test/data/gemini-scan/register-unknown-response",
			testrecorder.WithJWTMatcher("../../test/jwt/public_key.pem"),
		)
		require.NoError(t, err)
		defer r.Stop()

		httpClient := &http.Client{
			Transport: r.Transport,
		}

		cli := gemini.NewScanRepoClient(url, httpClient, "", nil, false)

		req := gemini.NewScanRepoRequest(repoURL)
		ctx, err := testjwt.NewJWTContext("bcdd0b29-123d-11e8-a8bc-b69930b94f5c", "../../test/jwt/private_key.pem")
		require.NoError(t, err)

		// then
		err = cli.Register(ctx, req)
		require.Error(t, err)
		t.Log("successfully errored as: ", err)
	})

	t.Run("unknown response on 200", func(t *testing.T) {
		// given
		// the server returns 200 but some other output
		r, err := testrecorder.New(
			"../../test/data/gemini-scan/register-unknown-response-200",
			testrecorder.WithJWTMatcher("../../test/jwt/public_key.pem"),
		)
		require.NoError(t, err)
		defer r.Stop()

		httpClient := &http.Client{
			Transport: r.Transport,
		}

		cli := gemini.NewScanRepoClient(url, httpClient, "", nil, false)

		req := gemini.NewScanRepoRequest(repoURL)
		ctx, err := testjwt.NewJWTContext("bcdd0b29-123d-11e8-a8bc-b69930b94f5c", "../../test/jwt/private_key.pem")
		require.NoError(t, err)

		err = cli.Register(ctx, req)
		require.Error(t, err)
		t.Log("successfully errored as: ", err)
	})
}

func TestDeRegister(t *testing.T) {
	// given
	repoURL := "https://github.com/fabric8-services/fabric8-wit"

	config, err := configuration.New("")
	require.NoError(t, err)
	geminiURL := config.GetAnalyticsGeminiServiceURL()
	codebaseURL := config.GetCodebaseServiceURL()

	// in this test, the call to deregister is made but there are some codebases
	// available with the same URL so further call to the gemini service to deregister
	// is not made
	t.Run("ok response no call to gemini", func(t *testing.T) {
		// given
		r, err := testrecorder.New(
			"../../test/data/gemini-scan/deregister-200",
		)
		require.NoError(t, err)
		defer r.Stop()

		httpClient := &http.Client{Transport: r.Transport}
		cli := gemini.NewScanRepoClient(geminiURL, httpClient, codebaseURL, httpClient, false)

		req := gemini.NewScanRepoRequest(repoURL)
		ctx, err := testjwt.NewJWTContext("bcdd0b29-123d-11e8-a8bc-b69930b94f5c", "../../test/jwt/private_key.pem")
		require.NoError(t, err)

		// then
		err = cli.DeRegister(ctx, req)
		require.NoError(t, err)
	})

	// in this test, call to deregister is made and there are no codebases
	// left with same codebase URL so the call to gemini service is made
	t.Run("ok response and call to gemini", func(t *testing.T) {
		// given for codebases
		recordCodebases, err := testrecorder.New(
			"../../test/data/gemini-scan/deregister-call-gemini-codebases-200",
		)
		require.NoError(t, err)
		defer recordCodebases.Stop()
		codebaseClient := &http.Client{Transport: recordCodebases.Transport}

		recordGemini, err := testrecorder.New(
			"../../test/data/gemini-scan/deregister-call-gemini-200",
			testrecorder.WithJWTMatcher("../../test/jwt/public_key.pem"),
		)
		require.NoError(t, err)
		defer recordGemini.Stop()
		geminiClient := &http.Client{Transport: recordGemini.Transport}

		cli := gemini.NewScanRepoClient(geminiURL, geminiClient, codebaseURL, codebaseClient, false)

		req := gemini.NewScanRepoRequest(repoURL)
		ctx, err := testjwt.NewJWTContext("bcdd0b29-123d-11e8-a8bc-b69930b94f5c", "../../test/jwt/private_key.pem")
		require.NoError(t, err)

		// then
		err = cli.DeRegister(ctx, req)
		require.NoError(t, err)
	})

	t.Run("malformed url error for the codebase", func(t *testing.T) {
		codebaseAlterURL := "%"
		cli := gemini.NewScanRepoClient(geminiURL, &http.Client{}, codebaseAlterURL, &http.Client{}, false)

		req := gemini.NewScanRepoRequest(repoURL)
		ctx, err := testjwt.NewJWTContext("bcdd0b29-123d-11e8-a8bc-b69930b94f5c", "../../test/jwt/private_key.pem")
		require.NoError(t, err)

		// then
		err = cli.DeRegister(ctx, req)
		require.Error(t, err)
		t.Log("successfully errored as: ", err)
	})

	t.Run("should fail on making call to core", func(t *testing.T) {
		cli := gemini.NewScanRepoClient(geminiURL, &http.Client{}, codebaseURL, &http.Client{}, false)

		req := gemini.NewScanRepoRequest(repoURL)
		ctx, err := testjwt.NewJWTContext("bcdd0b29-123d-11e8-a8bc-b69930b94f5c", "../../test/jwt/private_key.pem")
		require.NoError(t, err)

		// then
		err = cli.DeRegister(ctx, req)
		require.Error(t, err)
		t.Log("successfully errored as: ", err)
	})

	t.Run("400 from codebases", func(t *testing.T) {
		// given
		r, err := testrecorder.New(
			"../../test/data/gemini-scan/deregister-400",
		)
		require.NoError(t, err)
		defer r.Stop()

		httpClient := &http.Client{Transport: r.Transport}
		cli := gemini.NewScanRepoClient(geminiURL, httpClient, codebaseURL, httpClient, false)

		req := gemini.NewScanRepoRequest(repoURL)
		ctx, err := testjwt.NewJWTContext("bcdd0b29-123d-11e8-a8bc-b69930b94f5c", "../../test/jwt/private_key.pem")
		require.NoError(t, err)

		// then
		err = cli.DeRegister(ctx, req)
		require.Error(t, err)
		t.Log("successfully errored as: ", err)
	})

	t.Run("could not decode codebases", func(t *testing.T) {
		// given
		r, err := testrecorder.New(
			"../../test/data/gemini-scan/deregister-bad-output",
		)
		require.NoError(t, err)
		defer r.Stop()

		httpClient := &http.Client{Transport: r.Transport}
		cli := gemini.NewScanRepoClient(geminiURL, httpClient, codebaseURL, httpClient, false)

		req := gemini.NewScanRepoRequest(repoURL)
		ctx, err := testjwt.NewJWTContext("bcdd0b29-123d-11e8-a8bc-b69930b94f5c", "../../test/jwt/private_key.pem")
		require.NoError(t, err)

		// then
		err = cli.DeRegister(ctx, req)
		require.Error(t, err)
		t.Log("successfully errored as: ", err)
	})
}
