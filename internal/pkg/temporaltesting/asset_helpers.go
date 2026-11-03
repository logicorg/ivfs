package temporaltesting

import (
	"testing"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

// InsertAsset inserts an asset
func (s *TemporalDBUnitTestSuite) InsertAsset(t *testing.T, asset *model.Asset) int64 {
	t.Helper()

	_, err := s.DB.NewInsert().Model(asset).Returning("id").Exec(t.Context())
	require.NoError(s.T(), err)
	require.NotZero(s.T(), asset.ID)
	return asset.ID
}

// DeleteAsset deletes an asset
func (s *TemporalDBUnitTestSuite) DeleteAsset(t *testing.T, id int64) {
	t.Helper()

	_, err := s.DB.NewDelete().Model((*model.Asset)(nil)).Where("id = ?", id).Exec(t.Context())
	require.NoError(s.T(), err)
}

// InsertAssetValue inserts an asset value
func (s *TemporalDBUnitTestSuite) InsertAssetValue(t *testing.T, av *model.AssetValue) int64 {
	t.Helper()

	_, err := s.DB.NewInsert().Model(av).Returning("id").Exec(t.Context())
	require.NoError(s.T(), err)
	require.NotZero(s.T(), av.ID)
	return av.ID
}

// DeleteAssetValue deletes an asset value
func (s *TemporalDBUnitTestSuite) DeleteAssetValue(t *testing.T, id int64) {
	t.Helper()

	_, err := s.DB.NewDelete().Model((*model.AssetValue)(nil)).Where("id = ?", id).Exec(t.Context())
	require.NoError(s.T(), err)
}
