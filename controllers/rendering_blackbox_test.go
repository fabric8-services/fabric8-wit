package controllers

import (
	"testing"

	"github.com/almighty/almighty-core/app"
	"github.com/almighty/almighty-core/app/test"
	"github.com/almighty/almighty-core/rendering"
	"github.com/almighty/almighty-core/resource"
	"github.com/goadesign/goa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// a normal test function that will kick off MarkupRenderingSuite
func TestSuiteMarkupRendering(t *testing.T) {
	resource.Require(t, resource.Database)
	suite.Run(t, new(MarkupRenderingSuite))
}

// ========== MarkupRenderingSuite struct that implements SetupSuite, TearDownSuite, SetupTest, TearDownTest ==========
type MarkupRenderingSuite struct {
	suite.Suite
	controller app.RenderController
	svc        *goa.Service
}

func (s *MarkupRenderingSuite) SetupSuite() {
}

func (s *MarkupRenderingSuite) TearDownSuite() {
}

func (s *MarkupRenderingSuite) SetupTest() {
	s.svc = goa.New("Rendering-service-test")
	s.controller = NewRenderController(s.svc)
}

func (s *MarkupRenderingSuite) TearDownTest() {
}

func (s *MarkupRenderingSuite) TestRenderPlainText() {
	// given
	payload := app.MarkupRenderingPayload{Data: &app.MarkupRenderingPayloadData{
		Type: RenderingType,
		Attributes: &app.MarkupRenderingPayloadDataAttributes{
			Content: "foo",
			Markup:  rendering.SystemMarkupPlainText,
		}}}
	// when
	_, result := test.RenderRenderOK(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	// then
	require.NotNil(s.T(), result)
	require.NotNil(s.T(), result.Data)
	assert.Equal(s.T(), "foo", result.Data.Attributes.RenderedContent)
}

func (s *MarkupRenderingSuite) TestRenderMarkdown() {
	// given
	payload := app.MarkupRenderingPayload{Data: &app.MarkupRenderingPayloadData{
		Type: RenderingType,
		Attributes: &app.MarkupRenderingPayloadDataAttributes{
			Content: "foo",
			Markup:  rendering.SystemMarkupMarkdown,
		}}}

	// when
	_, result := test.RenderRenderOK(s.T(), s.svc.Context, s.svc, s.controller, &payload)
	// then
	require.NotNil(s.T(), result)
	require.NotNil(s.T(), result.Data)
	assert.Equal(s.T(), "<p>foo</p>\n", result.Data.Attributes.RenderedContent)
}

func (s *MarkupRenderingSuite) TestRenderUnsupportedMarkup() {
	// given
	payload := app.MarkupRenderingPayload{Data: &app.MarkupRenderingPayloadData{
		Type: RenderingType,
		Attributes: &app.MarkupRenderingPayloadDataAttributes{
			Content: "foo",
			Markup:  "bar",
		}}}

	// when/then
	test.RenderRenderBadRequest(s.T(), s.svc.Context, s.svc, s.controller, &payload)
}
