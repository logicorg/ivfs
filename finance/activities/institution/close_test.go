package institution

import (
	"fmt"
	"testing"
	"time"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/require"
)

// Test_Close checks that an institution closes only once each of its
// financial accounts is closed, keeps the ClosedAt it was first closed at, and
// closes now when no ClosedAt is given.
func (s *UnitTestSuite) Test_Close() {
	closedAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	earlier := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tcs := map[string]struct {
		// closedAt is when the institution is already closed, if it is.
		closedAt *time.Time
		// accountsClosed holds, for each financial account at the
		// institution, whether it is closed.
		accountsClosed []bool
		// findBy selects the institution to close, given the fixture.
		findBy func(institution *model.Institution) it.RetrieveInstitutionParams
		// at is the ClosedAt to close with.
		at     time.Time
		errStr string
		// wantClosedAt is the stored ClosedAt, unless now expects a time
		// during the test instead.
		wantClosedAt *time.Time
		now          bool
	}{
		"an institution with no accounts, by id": {
			findBy:       byID,
			at:           closedAt,
			wantClosedAt: &closedAt,
		},
		"an institution with no accounts, by name": {
			findBy:       byName,
			at:           closedAt,
			wantClosedAt: &closedAt,
		},
		"an institution whose accounts are all closed": {
			accountsClosed: []bool{true, true},
			findBy:         byID,
			at:             closedAt,
			wantClosedAt:   &closedAt,
		},
		"an institution with an open account": {
			accountsClosed: []bool{true, false},
			findBy:         byID,
			at:             closedAt,
			errStr:         "resource Institution conflict: institution has open financial accounts",
		},
		"an institution already closed keeps its closing date": {
			closedAt:     &earlier,
			findBy:       byID,
			at:           closedAt,
			wantClosedAt: &earlier,
		},
		"no closing date closes it now": {
			findBy: byID,
			now:    true,
		},
		"an unknown id": {
			findBy: func(*model.Institution) it.RetrieveInstitutionParams { return it.RetrieveInstitutionParams{ID: 999999} },
			at:     closedAt,
			errStr: "resource Institution not found with ID: 999999",
		},
		"an unknown name": {
			findBy: func(*model.Institution) it.RetrieveInstitutionParams {
				return it.RetrieveInstitutionParams{Name: "missing"}
			},
			at:     closedAt,
			errStr: "resource Institution not found with ID: missing",
		},
		"no id or name": {
			findBy: func(*model.Institution) it.RetrieveInstitutionParams { return it.RetrieveInstitutionParams{} },
			at:     closedAt,
			errStr: "either name or id must be provided",
		},
	}

	for name, tc := range tcs {
		institution := &model.Institution{Name: "Revolut", Kind: "bank", ClosedAt: tc.closedAt}
		temporaltesting.Insert(s.T(), s.DB, institution)
		accounts := make([]*model.FinancialAccount, 0, len(tc.accountsClosed))
		for i, closed := range tc.accountsClosed {
			account := &model.FinancialAccount{InstitutionID: institution.ID, Name: fmt.Sprintf("Account %d", i)}
			if closed {
				account.ClosedAt = ptr.NewPtr(earlier)
			}
			temporaltesting.Insert(s.T(), s.DB, account)
			accounts = append(accounts, account)
		}

		s.T().Run(name, func(t *testing.T) {
			before := time.Now()
			_, err := s.Env.ExecuteActivity(s.institution.Close, it.CloseParams{FindBy: tc.findBy(institution), ClosedAt: tc.at})

			stored := &model.Institution{}
			temporaltesting.Retrieve(t, s.DB, stored, institution.ID)
			switch {
			case tc.errStr != "":
				require.ErrorContains(t, err, tc.errStr)
				require.Equal(t, tc.closedAt == nil, stored.ClosedAt == nil, "a refused close changes nothing")
			case tc.now:
				require.NoError(t, err)
				require.NotNil(t, stored.ClosedAt)
				require.WithinRange(t, *stored.ClosedAt, before.Truncate(time.Microsecond), time.Now())
			default:
				require.NoError(t, err)
				require.NotNil(t, stored.ClosedAt)
				require.Equal(t, *tc.wantClosedAt, stored.ClosedAt.UTC())
			}
		})

		for _, account := range accounts {
			temporaltesting.Delete(s.T(), s.DB, (*model.FinancialAccount)(nil), account.ID)
		}
		temporaltesting.Delete(s.T(), s.DB, (*model.Institution)(nil), institution.ID)
	}
}

// Test_Close_Idempotency checks that closing an institution twice succeeds
// both times and leaves it closed at the first ClosedAt.
func (s *UnitTestSuite) Test_Close_Idempotency() {
	closedAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)

	params := it.CloseParams{FindBy: it.RetrieveInstitutionParams{ID: institution.ID}, ClosedAt: closedAt}
	for range 2 {
		_, err := s.Env.ExecuteActivity(s.institution.Close, params)
		require.NoError(s.T(), err)

		stored := &model.Institution{}
		temporaltesting.Retrieve(s.T(), s.DB, stored, institution.ID)
		require.NotNil(s.T(), stored.ClosedAt)
		require.Equal(s.T(), closedAt, stored.ClosedAt.UTC())
	}

	temporaltesting.Delete(s.T(), s.DB, (*model.Institution)(nil), institution.ID)
}

func byID(institution *model.Institution) it.RetrieveInstitutionParams {
	return it.RetrieveInstitutionParams{ID: institution.ID}
}

func byName(institution *model.Institution) it.RetrieveInstitutionParams {
	return it.RetrieveInstitutionParams{Name: institution.Name}
}
