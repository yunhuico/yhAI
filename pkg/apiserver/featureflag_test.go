package apiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/payload"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/apiserver/response"
	"jihulab.com/jihulab/ultrafox/ultrafox/pkg/featureflag"
)

func (s *testServer) testFeatureFlag(name string, context featureflag.ContextData) bool {
	req := &payload.TestFeatureFlagReq{
		Name:    name,
		Context: context,
	}
	b, err := json.Marshal(req)
	s.NoError(err)
	resp := s.request("POST", "/internal/test/featureFlag", bytes.NewReader(b))
	s.assertResponseOK(resp)

	r := &R{
		Data: &response.TestFeatureFlagResp{},
	}
	err = unmarshalResponse(resp, r)
	s.NoError(err)
	return r.Data.(*response.TestFeatureFlagResp).Enabled
}

func TestUsingFeatureFlagWhenDontInitFeatureFlag(t *testing.T) {
	server := newTestServer(t)
	enabled := server.testFeatureFlag("not_exists_feature", featureflag.ContextData{})
	assert.False(t, enabled)
}

func TestUsingFeatureFlagWhenInitFeatureFlag(t *testing.T) {
	server := newTestServer(t)

	t.Run("test feature1", func(t *testing.T) {
		featureflag.TestSuite(t, []featureflag.FeatureName{"feature1"}, func(t *testing.T) {
			enabled := server.testFeatureFlag("feature1", featureflag.ContextData{})
			assert.True(t, enabled)
		})
	})
	t.Run("test feature2", func(t *testing.T) {
		featureflag.TestSuite(t, []featureflag.FeatureName{"feature2"}, func(t *testing.T) {
			enabled := server.testFeatureFlag("feature2", featureflag.ContextData{})
			assert.True(t, enabled)
		})
	})
	t.Run("test not_exists_feature", func(t *testing.T) {
		featureflag.TestSuite(t, []featureflag.FeatureName{}, func(t *testing.T) {
			enabled := server.testFeatureFlag("not_exists_feature", featureflag.ContextData{})
			assert.False(t, enabled)
		})
	})
}

type buz1 struct {
}

var errBuz1 = errors.New("buz1 error")

func (buz1) handle() error {
	if featureflag.IsEnabled(context.Background(), "buz1_pass", featureflag.ContextData{}) {
		return nil
	}
	return errBuz1
}

type buz2 struct {
}

var errBuz2 = errors.New("buz2 error")

func (buz2) handle() error {
	if featureflag.IsEnabled(context.Background(), "buz2_pass", featureflag.ContextData{}) {
		return nil
	}
	return errBuz2
}

func TestUsingFeatureFlagInBusinessCode(t *testing.T) {
	b1 := buz1{}
	assert.ErrorIs(t, errBuz1, b1.handle())

	featureflag.TestSuite(t, []featureflag.FeatureName{"buz1_pass"}, func(t *testing.T) {
		assert.NoError(t, b1.handle())

		b2 := buz2{}
		assert.ErrorIs(t, errBuz2, b2.handle())
	})
}
