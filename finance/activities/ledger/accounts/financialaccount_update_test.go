package accounts

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) Test_UpdateFinancialAccountMetadata() {
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)

	tests := map[string]struct {
		params   lt.UpdateFinancialAccountMetadataInput
		expErr   bool
		expName  string
		expExtID *string
	}{
		"rename": {
			params:  lt.UpdateFinancialAccountMetadataInput{Name: ptr.NewPtr("Revolut EUR renamed")},
			expName: "Revolut EUR renamed",
		},
		"set external id": {
			params:   lt.UpdateFinancialAccountMetadataInput{ExternalID: ptr.NewPtr("ext-1")},
			expName:  "Revolut EUR",
			expExtID: ptr.NewPtr("ext-1"),
		},
		"no-op": {
			params:  lt.UpdateFinancialAccountMetadataInput{},
			expName: "Revolut EUR",
		},
		"blank name": {
			params: lt.UpdateFinancialAccountMetadataInput{Name: ptr.NewPtr("   ")},
			expErr: true,
		},
		"invalid id": {
			params: lt.UpdateFinancialAccountMetadataInput{FinancialAccountID: 0, Name: ptr.NewPtr("x")},
			expErr: true,
		},
	}

	for name, tc := range tests {
		account := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Revolut EUR"}
		temporaltesting.Insert(s.T(), s.DB, account)

		s.Run(name, func() {
			if tc.params.FinancialAccountID == 0 && name != "invalid id" {
				tc.params.FinancialAccountID = account.ID
			}

			_, err := s.Env.ExecuteActivity(s.aa.UpdateFinancialAccountMetadata, tc.params)
			if tc.expErr {
				require.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)

			var stored model.FinancialAccount
			require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", account.ID).Scan(s.T().Context()))
			require.Equal(s.T(), tc.expName, stored.Name)
			if tc.expExtID != nil {
				require.NotNil(s.T(), stored.ExternalID)
				require.Equal(s.T(), *tc.expExtID, *stored.ExternalID)
			}
		})

		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.FinancialAccount)(nil), account.ID)
	}
}

func (s *UnitTestSuite) Test_UpdateFinancialAccountMetadata_NotFound() {
	_, err := s.Env.ExecuteActivity(s.aa.UpdateFinancialAccountMetadata, lt.UpdateFinancialAccountMetadataInput{
		FinancialAccountID: 999999,
		Name:               ptr.NewPtr("x"),
	})
	require.Error(s.T(), err)
}

func (s *UnitTestSuite) Test_UpdateFinancialAccountMetadata_Idempotency() {
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	account := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Revolut EUR"}
	temporaltesting.Insert(s.T(), s.DB, account)

	params := lt.UpdateFinancialAccountMetadataInput{FinancialAccountID: account.ID, Name: ptr.NewPtr("Revolut EUR v2")}

	_, err := s.Env.ExecuteActivity(s.aa.UpdateFinancialAccountMetadata, params)
	require.NoError(s.T(), err)
	_, err = s.Env.ExecuteActivity(s.aa.UpdateFinancialAccountMetadata, params)
	require.NoError(s.T(), err)

	var stored model.FinancialAccount
	require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", account.ID).Scan(s.T().Context()))
	require.Equal(s.T(), "Revolut EUR v2", stored.Name)
}

// Test_UpdateFinancialAccountMetadata_Taken checks that a financial account
// cannot take the name, compared case-insensitively, or the external ID of
// another account of its institution (ADR 0005), and is left as it was, while
// renaming it to its own name in another casing is allowed.
func (s *UnitTestSuite) Test_UpdateFinancialAccountMetadata_Taken() {
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)

	tests := map[string]struct {
		// target picks the account to update, of Personal and Joint.
		target func(personal, joint *model.FinancialAccount) *model.FinancialAccount
		update lt.UpdateFinancialAccountMetadataInput
		errStr string
		stored string
	}{
		"the name of another account": {
			target: func(_, joint *model.FinancialAccount) *model.FinancialAccount { return joint },
			update: lt.UpdateFinancialAccountMetadataInput{Name: ptr.NewPtr(" personal ")},
			errStr: "resource FinancialAccount conflict",
			stored: "Joint",
		},
		"the external id of another account": {
			target: func(_, joint *model.FinancialAccount) *model.FinancialAccount { return joint },
			update: lt.UpdateFinancialAccountMetadataInput{ExternalID: ptr.NewPtr("IBAN-1")},
			errStr: "resource FinancialAccount conflict",
			stored: "Joint",
		},
		"its own name in another casing": {
			target: func(personal, _ *model.FinancialAccount) *model.FinancialAccount { return personal },
			update: lt.UpdateFinancialAccountMetadataInput{Name: ptr.NewPtr("PERSONAL")},
			stored: "PERSONAL",
		},
	}

	for name, tc := range tests {
		personal := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Personal", ExternalID: ptr.NewPtr("IBAN-1")}
		temporaltesting.Insert(s.T(), s.DB, personal)
		joint := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Joint"}
		temporaltesting.Insert(s.T(), s.DB, joint)
		target := tc.target(personal, joint)

		s.Run(name, func() {
			update := tc.update
			update.FinancialAccountID = target.ID
			_, err := s.Env.ExecuteActivity(s.aa.UpdateFinancialAccountMetadata, update)

			if tc.errStr != "" {
				require.ErrorContains(s.T(), err, tc.errStr)
			} else {
				require.NoError(s.T(), err)
			}
			var stored model.FinancialAccount
			temporaltesting.Retrieve(s.T(), s.DB, &stored, target.ID)
			require.Equal(s.T(), tc.stored, stored.Name)
			require.Equal(s.T(), target.ExternalID, stored.ExternalID)
		})

		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.FinancialAccount)(nil), personal.ID)
		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.FinancialAccount)(nil), joint.ID)
	}
}
